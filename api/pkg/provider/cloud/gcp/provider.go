package gcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
)

const (
	defaultNetwork               = "global/networks/default"
	firewallSelfCleanupFinalizer = "kubermatic.io/cleanup-gcp-firewall-self"
	firewallICMPCleanupFinalizer = "kubermatic.io/cleanup-gcp-firewall-icmp"
)

type gcp struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
}

// NewCloudProvider creates a new gcp provider.
func NewCloudProvider(secretKeyGetter provider.SecretKeySelectorValueFunc) provider.CloudProvider {
	return &gcp{
		secretKeySelector: secretKeyGetter,
	}
}

// TODO: update behaviour of all these methods
// InitializeCloudProvider initializes a cluster.
func (g *gcp) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error
	if cluster.Spec.Cloud.GCP.Network == "" && cluster.Spec.Cloud.GCP.Subnetwork == "" {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.GCP.Network = defaultNetwork
		})
		if err != nil {
			return nil, err
		}
	}

	if err := g.ensureFirewallRules(cluster, update); err != nil {
		return nil, err
	}

	return cluster, nil
}

// DefaultCloudSpec adds defaults to the cloud spec.
func (g *gcp) DefaultCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

// ValidateCloudSpec validates the given CloudSpec.
func (g *gcp) ValidateCloudSpec(spec kubermaticv1.CloudSpec) error {
	if spec.GCP.ServiceAccount == "" {
		return fmt.Errorf("serviceAccount cannot be empty")
	}
	return nil
}

// CleanUpCloudProvider removes firewall rules and related finalizer.
func (g *gcp) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	serviceAccount, err := GetCredentialsForCluster(cluster.Spec.Cloud, g.secretKeySelector)
	if err != nil {
		return nil, err
	}

	svc, projectID, err := ConnectToComputeService(serviceAccount)
	if err != nil {
		return nil, err
	}

	firewallService := compute.NewFirewallsService(svc)

	selfRuleName := fmt.Sprintf("firewall-%s-self", cluster.Name)
	icmpRuleName := fmt.Sprintf("firewall-%s-icmp", cluster.Name)

	if kuberneteshelper.HasFinalizer(cluster, firewallSelfCleanupFinalizer) {
		_, err = firewallService.Delete(projectID, selfRuleName).Do()
		// we ignore a Google API "not found" error
		if err != nil && !isHTTPError(err, http.StatusNotFound) {
			return nil, fmt.Errorf("failed to delete firewall rule %s: %v", selfRuleName, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, firewallSelfCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %v", firewallSelfCleanupFinalizer, err)
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, firewallICMPCleanupFinalizer) {
		_, err = firewallService.Delete(projectID, icmpRuleName).Do()
		// we ignore a Google API "not found" error
		if err != nil && !isHTTPError(err, http.StatusNotFound) {
			return nil, fmt.Errorf("failed to delete firewall rule %s: %v", selfRuleName, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, firewallICMPCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %v", firewallICMPCleanupFinalizer, err)
		}
	}

	return cluster, nil
}

// ConnectToComputeService establishes a service connection to the Compute Engine.
func ConnectToComputeService(serviceAccount string) (*compute.Service, string, error) {
	b, err := base64.StdEncoding.DecodeString(serviceAccount)
	if err != nil {
		return nil, "", fmt.Errorf("error decoding service account: %v", err)
	}
	sam := map[string]string{}
	err = json.Unmarshal(b, &sam)
	if err != nil {
		return nil, "", fmt.Errorf("failed unmarshalling service account: %v", err)
	}

	projectID := sam["project_id"]
	if projectID == "" {
		return nil, "", errors.New("empty project_id")
	}
	conf, err := google.JWTConfigFromJSON(b, compute.ComputeScope)
	if err != nil {
		return nil, "", err
	}
	client := conf.Client(context.Background())
	svc, err := compute.New(client)
	if err != nil {
		return nil, "", fmt.Errorf("cannot connect to Google Cloud: %v", err)
	}
	return svc, projectID, nil
}

func (g *gcp) ensureFirewallRules(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) error {
	serviceAccount, err := GetCredentialsForCluster(cluster.Spec.Cloud, g.secretKeySelector)
	if err != nil {
		return err
	}

	svc, projectID, err := ConnectToComputeService(serviceAccount)
	if err != nil {
		return err
	}

	firewallService := compute.NewFirewallsService(svc)
	tag := fmt.Sprintf("kubernetes-cluster-%s", cluster.Name)
	selfRuleName := fmt.Sprintf("firewall-%s-self", cluster.Name)
	icmpRuleName := fmt.Sprintf("firewall-%s-icmp", cluster.Name)

	// allow traffic within the same cluster
	if !kuberneteshelper.HasFinalizer(cluster, firewallSelfCleanupFinalizer) {
		_, err = firewallService.Insert(projectID, &compute.Firewall{
			Name:    selfRuleName,
			Network: cluster.Spec.Cloud.GCP.Network,
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
				},
				{
					IPProtocol: "udp",
				},
				{
					IPProtocol: "icmp",
				},
				{
					IPProtocol: "esp",
				},
				{
					IPProtocol: "ah",
				},
				{
					IPProtocol: "sctp",
				},
				{
					IPProtocol: "ipip",
				},
			},
			TargetTags: []string{tag},
			SourceTags: []string{tag},
		}).Do()
		// we ignore a Google API "already exists" error
		if err != nil && !isHTTPError(err, http.StatusConflict) {
			return fmt.Errorf("failed to create firewall rule %s: %v", selfRuleName, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, firewallSelfCleanupFinalizer)
		})
		if err != nil {
			return fmt.Errorf("failed to add %s finalizer: %v", firewallSelfCleanupFinalizer, err)
		}
	}

	// allow ICMP from everywhere
	if !kuberneteshelper.HasFinalizer(cluster, firewallICMPCleanupFinalizer) {
		_, err = firewallService.Insert(projectID, &compute.Firewall{
			Name:    icmpRuleName,
			Network: cluster.Spec.Cloud.GCP.Network,
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "icmp",
				},
			},
			TargetTags: []string{tag},
		}).Do()
		// we ignore a Google API "already exists" error
		if err != nil && !isHTTPError(err, http.StatusConflict) {
			return fmt.Errorf("failed to create firewall rule %s: %v", icmpRuleName, err)
		}

		newCluster, err := update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, firewallICMPCleanupFinalizer)
		})
		if err != nil {
			return fmt.Errorf("failed to add %s finalizer: %v", firewallICMPCleanupFinalizer, err)
		}
		*cluster = *newCluster
	}

	return err
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted
func (g *gcp) ValidateCloudSpecUpdate(oldSpec kubermaticv1.CloudSpec, newSpec kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
func GetCredentialsForCluster(cloud kubermaticv1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (serviceAccount string, err error) {
	serviceAccount = cloud.GCP.ServiceAccount

	if serviceAccount == "" {
		if cloud.GCP.CredentialsReference == nil {
			return "", errors.New("no credentials provided")
		}
		serviceAccount, err = secretKeySelector(cloud.GCP.CredentialsReference, resources.GCPServiceAccount)
		if err != nil {
			return "", err
		}
	}

	return serviceAccount, nil
}

// isHTTPError returns true if the given error is of a specific HTTP status code.
func isHTTPError(err error, status int) bool {
	gerr, ok := err.(*googleapi.Error)
	return ok && gerr.Code == status
}
