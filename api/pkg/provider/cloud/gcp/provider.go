package gcp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

const (
	defaultNetwork               = "global/networks/default"
	firewallSelfCleanupFinalizer = "kubermatic.io/cleanup-gcp-firewall-self"
	firewallICMPCleanupFinalizer = "kubermatic.io/cleanup-gcp-firewall-icmp"
)

type gcp struct {
	dcs map[string]provider.DatacenterMeta
}

// NewCloudProvider creates a new gcp provider.
func NewCloudProvider(dcs map[string]provider.DatacenterMeta) provider.CloudProvider {
	return &gcp{
		dcs: dcs,
	}
}

// TODO: update behaviour of all these methods
// InitializeCloudProvider initializes a cluster.
func (g *gcp) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var err error
	dc, ok := g.dcs[cluster.Spec.Cloud.DatacenterName]
	if !ok {
		return nil, fmt.Errorf("could not find datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	if dc.Spec.GCP == nil {
		return nil, fmt.Errorf("datacenter %q is not a valid GCP datacenter", cluster.Spec.Cloud.DatacenterName)
	}

	if cluster.Spec.Cloud.GCP.Network == "" && cluster.Spec.Cloud.GCP.Subnetwork == "" {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.GCP.Network = defaultNetwork
		})
		if err != nil {
			return nil, err
		}
	}

	err = ensureFirewallRules(cluster, update)
	if err != nil {
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
	svc, projectID, err := connectToComputeService(cluster.Spec.Cloud.GCP.ServiceAccount)
	if err != nil {
		return nil, err
	}

	firewallService := compute.NewFirewallsService(svc)

	selfRuleName := fmt.Sprintf("firewall-%s-self", cluster.Name)
	icmpRuleName := fmt.Sprintf("firewall-%s-icmp", cluster.Name)

	if kuberneteshelper.HasFinalizer(cluster, firewallSelfCleanupFinalizer) {
		_, err = firewallService.Delete(projectID, selfRuleName).Do()
		if err != nil {
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
		if err != nil {
			return nil, fmt.Errorf("failed to delete firewall rule %s: %v", icmpRuleName, err)
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

func (g *gcp) Migrate(_ *kubermaticv1.Cluster) error {
	return nil
}

// connectToComputeService establishes a service connection to the Compute Engine.
func connectToComputeService(serviceAccount string) (*compute.Service, string, error) {
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
	client := conf.Client(oauth2.NoContext)
	svc, err := compute.New(client)
	if err != nil {
		return nil, "", fmt.Errorf("cannot connect to Google Cloud: %v", err)
	}
	return svc, projectID, nil
}

func ensureFirewallRules(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) error {
	svc, projectID, err := connectToComputeService(cluster.Spec.Cloud.GCP.ServiceAccount)
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
		if err != nil {
			// we ignore a Google API "already exists" error
			if ge, ok := err.(*googleapi.Error); !ok || ge.Code != http.StatusConflict {
				return err
			}
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, firewallSelfCleanupFinalizer)
		})
		if err != nil {
			return err
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
		if err != nil {
			// we ignore a Google API "already exists" error
			if ge, ok := err.(*googleapi.Error); !ok || ge.Code != http.StatusConflict {
				return err
			}
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, firewallICMPCleanupFinalizer)
		})
		if err != nil {
			return err
		}
	}

	return err
}
