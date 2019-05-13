package gcp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

const (
	defaultNetwork           = "global/networks/default"
	firewallCleanupFinalizer = "kubermatic.io/cleanup-gcp-firewall"
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

	if cluster.Spec.Cloud.GCP.FirewallRuleName == "" {
		firewallName, err := createFirewallRule(*cluster.Spec.Cloud.GCP, cluster.Name)
		if err != nil {
			return nil, err
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.GCP.FirewallRuleName = firewallName
		})
		if err != nil {
			return nil, err
		}
	}

	if !kuberneteshelper.HasFinalizer(cluster, firewallCleanupFinalizer) {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Finalizers = append(cluster.Finalizers, firewallCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
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
	for _, finalizer := range cluster.Finalizers {
		if finalizer == firewallCleanupFinalizer {
			svc, projectID, err := connectToComputeService(cluster.Spec.Cloud.GCP.ServiceAccount)
			if err != nil {
				return nil, err
			}

			firewallService := compute.NewFirewallsService(svc)
			_, err = firewallService.Delete(projectID, cluster.Spec.Cloud.GCP.FirewallRuleName).Do()
			if err != nil {
				return nil, fmt.Errorf("failed to delete firewall %s: %v", cluster.Spec.Cloud.GCP.FirewallRuleName, err)
			}

			cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Finalizers = kuberneteshelper.RemoveFinalizer(cluster.Finalizers, firewallCleanupFinalizer)
			})
			if err != nil {
				return nil, fmt.Errorf("failed to remove %s finalizer: %v", firewallCleanupFinalizer, err)
			}
		}
	}

	return cluster, nil
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

func createFirewallRule(spec kubermaticv1.GCPCloudSpec, clusterName string) (string, error) {
	svc, projectID, err := connectToComputeService(spec.ServiceAccount)
	if err != nil {
		return "", err
	}

	firewallService := compute.NewFirewallsService(svc)
	tag := fmt.Sprintf("kubernetes-cluster-%s", clusterName)
	firewallName := fmt.Sprintf("firewall-%s", clusterName)

	_, err = firewallService.Insert(projectID, &compute.Firewall{
		Name:    firewallName,
		Network: spec.Network,
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
			},
			{
				IPProtocol: "udp",
			},
		},
		TargetTags: []string{tag},
		SourceTags: []string{tag},
	}).Do()
	if err != nil {
		return "", err
	}

	return firewallName, err
}
