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

	"k8s.io/apimachinery/pkg/util/sets"
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

	if cluster.Spec.Cloud.GCP.FirewallRuleNames == nil || len(cluster.Spec.Cloud.GCP.FirewallRuleNames) == 0 {
		firewallRuleNames, err := createFirewallRules(*cluster.Spec.Cloud.GCP, cluster.Name)
		if err != nil {
			return nil, err
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.GCP.FirewallRuleNames = firewallRuleNames
		})
		if err != nil {
			return nil, err
		}
	}

	if !kuberneteshelper.HasFinalizer(cluster, firewallCleanupFinalizer) {
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, firewallCleanupFinalizer)
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
	if kuberneteshelper.HasFinalizer(cluster, firewallCleanupFinalizer) {
		svc, projectID, err := connectToComputeService(cluster.Spec.Cloud.GCP.ServiceAccount)
		if err != nil {
			return nil, err
		}

		firewallService := compute.NewFirewallsService(svc)

		// Iterate over each rule and remove them one by one, updating the cluster object each time.
		set := sets.NewString(cluster.Spec.Cloud.GCP.FirewallRuleNames...)
		for _, ruleName := range cluster.Spec.Cloud.GCP.FirewallRuleNames {
			_, err = firewallService.Delete(projectID, ruleName).Do()
			if err != nil {
				return nil, fmt.Errorf("failed to delete firewall rule %s: %v", ruleName, err)
			}

			set.Delete(ruleName)

			cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
				cluster.Spec.Cloud.GCP.FirewallRuleNames = set.List()
			})
			if err != nil {
				return nil, fmt.Errorf("failed to remove firewall rule %q from cluster: %v", ruleName, err)
			}
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, firewallCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %v", firewallCleanupFinalizer, err)
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

func createFirewallRules(spec kubermaticv1.GCPCloudSpec, clusterName string) ([]string, error) {
	svc, projectID, err := connectToComputeService(spec.ServiceAccount)
	if err != nil {
		return nil, err
	}

	firewallService := compute.NewFirewallsService(svc)
	tag := fmt.Sprintf("kubernetes-cluster-%s", clusterName)
	selfRuleName := fmt.Sprintf("firewall-%s-self", clusterName)
	icmpRuleName := fmt.Sprintf("firewall-%s-icmp", clusterName)

	_, err = firewallService.Insert(projectID, &compute.Firewall{
		Name:    selfRuleName,
		Network: spec.Network,
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
		return nil, err
	}

	// allow ICMP from everywhere
	_, err = firewallService.Insert(projectID, &compute.Firewall{
		Name:    icmpRuleName,
		Network: spec.Network,
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "icmp",
			},
		},
		TargetTags: []string{tag},
	}).Do()
	if err != nil {
		return nil, err
	}

	return []string{selfRuleName, icmpRuleName}, err
}
