/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

const (
	DefaultNetwork                   = "global/networks/default"
	computeAPIEndpoint               = "https://www.googleapis.com/compute/v1/"
	firewallSelfCleanupFinalizer     = "kubermatic.k8c.io/cleanup-gcp-firewall-self"
	firewallICMPCleanupFinalizer     = "kubermatic.k8c.io/cleanup-gcp-firewall-icmp"
	firewallNodePortCleanupFinalizer = "kubermatic.k8c.io/cleanup-gcp-firewall-nodeport"
	routesCleanupFinalizer           = "kubermatic.k8c.io/cleanup-gcp-routes"

	k8sNodeRouteTag          = "k8s-node-route"
	k8sNodeRoutePrefixRegexp = "kubernetes-.*"
)

type gcp struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
	log               *zap.SugaredLogger
}

// NewCloudProvider creates a new gcp provider.
func NewCloudProvider(secretKeyGetter provider.SecretKeySelectorValueFunc) provider.CloudProvider {
	return &gcp{
		secretKeySelector: secretKeyGetter,
		log:               log.Logger,
	}
}

var _ provider.ReconcilingCloudProvider = &gcp{}

// InitializeCloudProvider initializes a cluster.
func (g *gcp) InitializeCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return g.reconcileCluster(ctx, cluster, update, false, true)
}

// ReconcileCluster enforces the existence of the needed resources in the cloud provider.
func (g *gcp) ReconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return g.reconcileCluster(ctx, cluster, update, true, true)
}

func (*gcp) ClusterNeedsReconciling(cluster *kubermaticv1.Cluster) bool {
	return false
}

func (g *gcp) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, force, setTags bool) (*kubermaticv1.Cluster, error) {
	var err error
	if cluster.Spec.Cloud.GCP.Network == "" && cluster.Spec.Cloud.GCP.Subnetwork == "" {
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.Cloud.GCP.Network = DefaultNetwork
		})
		if err != nil {
			return nil, err
		}
	}

	serviceAccount, err := GetCredentialsForCluster(cluster.Spec.Cloud, g.secretKeySelector)
	if err != nil {
		return nil, err
	}

	svc, projectID, err := ConnectToComputeService(ctx, serviceAccount)
	if err != nil {
		return nil, err
	}

	if err := reconcileFirewallRules(ctx, cluster, update, svc, projectID); err != nil {
		return nil, err
	}

	// add the routes cleanup finalizer
	if !kuberneteshelper.HasFinalizer(cluster, routesCleanupFinalizer) {
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, routesCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add %s finalizer: %w", routesCleanupFinalizer, err)
		}
	}
	return cluster, nil
}

// DefaultCloudSpec adds defaults to the cloud spec.
func (g *gcp) DefaultCloudSpec(ctx context.Context, spec *kubermaticv1.ClusterSpec) error {
	if spec.Cloud.GCP == nil {
		return errors.New("no GCP cloud spec found")
	}

	if spec.Cloud.GCP.NodePortsAllowedIPRanges == nil {
		switch spec.ClusterNetwork.IPFamily {
		case kubermaticv1.IPFamilyIPv4:
			spec.Cloud.GCP.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{resources.IPv4MatchAnyCIDR},
			}
		case kubermaticv1.IPFamilyDualStack:
			spec.Cloud.GCP.NodePortsAllowedIPRanges = &kubermaticv1.NetworkRanges{
				CIDRBlocks: []string{resources.IPv4MatchAnyCIDR, resources.IPv6MatchAnyCIDR},
			}
		}
	}
	return nil
}

// ValidateCloudSpec validates the given CloudSpec.
func (g *gcp) ValidateCloudSpec(ctx context.Context, spec kubermaticv1.CloudSpec) error {
	sa, err := GetCredentialsForCluster(spec, g.secretKeySelector)
	if err != nil {
		return err
	}
	if sa == "" {
		return fmt.Errorf("serviceAccount cannot be empty")
	}
	return nil
}

// CleanUpCloudProvider removes firewall rules and related finalizer.
func (g *gcp) CleanUpCloudProvider(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	serviceAccount, err := GetCredentialsForCluster(cluster.Spec.Cloud, g.secretKeySelector)
	if err != nil {
		return nil, err
	}

	svc, projectID, err := ConnectToComputeService(ctx, serviceAccount)
	if err != nil {
		return nil, err
	}

	return deleteFirewallRules(ctx, cluster, update, g.log, svc, projectID)
}

func ValidateCredentials(ctx context.Context, serviceAccount string) error {
	svc, project, err := ConnectToComputeService(ctx, serviceAccount)
	if err != nil {
		return err
	}
	req := svc.Regions.List(project)
	err = req.Pages(ctx, func(list *compute.RegionList) error {
		return nil
	})
	return err
}

// ConnectToComputeService establishes a service connection to the Compute Engine.
func ConnectToComputeService(ctx context.Context, serviceAccount string) (*compute.Service, string, error) {
	client, projectID, err := createClient(ctx, serviceAccount, compute.ComputeScope)
	if err != nil {
		return nil, "", fmt.Errorf("cannot create Google Cloud client: %w", err)
	}
	svc, err := compute.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, "", fmt.Errorf("cannot connect to Google Cloud: %w", err)
	}
	return svc, projectID, nil
}

func createClient(ctx context.Context, serviceAccount string, scope string) (*http.Client, string, error) {
	b, err := base64.StdEncoding.DecodeString(serviceAccount)
	if err != nil {
		return nil, "", fmt.Errorf("error decoding service account: %w", err)
	}
	sam := map[string]string{}
	err = json.Unmarshal(b, &sam)
	if err != nil {
		return nil, "", fmt.Errorf("failed unmarshalling service account: %w", err)
	}

	projectID := sam["project_id"]
	if projectID == "" {
		return nil, "", errors.New("empty project_id")
	}
	conf, err := google.JWTConfigFromJSON(b, scope)
	if err != nil {
		return nil, "", err
	}

	client := conf.Client(ctx)

	return client, projectID, nil
}

// ValidateCloudSpecUpdate verifies whether an update of cloud spec is valid and permitted.
func (g *gcp) ValidateCloudSpecUpdate(_ context.Context, _ kubermaticv1.CloudSpec, _ kubermaticv1.CloudSpec) error {
	return nil
}

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error.
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
	var gerr *googleapi.Error

	return errors.As(err, &gerr) && gerr.Code == status
}

func GetGCPNetwork(ctx context.Context, sa, networkName string) (apiv1.GCPNetwork, error) {
	computeService, project, err := ConnectToComputeService(ctx, sa)
	if err != nil {
		return apiv1.GCPNetwork{}, err
	}

	req := computeService.Networks.Get(project, networkName)
	network, err := req.Do()
	if err != nil {
		return apiv1.GCPNetwork{}, err
	}

	return ToGCPNetworkAPIModel(network), nil
}

var networkRegex = regexp.MustCompile(`(global\/.+)$`)

func ToGCPNetworkAPIModel(network *compute.Network) apiv1.GCPNetwork {
	networkPath := networkRegex.FindString(network.SelfLink)
	return apiv1.GCPNetwork{
		ID:                    network.Id,
		Name:                  network.Name,
		AutoCreateSubnetworks: network.AutoCreateSubnetworks,
		Subnetworks:           network.Subnetworks,
		Kind:                  network.Kind,
		Path:                  networkPath,
	}
}

// GCPSubnetworkGetter is a function to retrieve a single subnetwork.
type GCPSubnetworkGetter = func(ctx context.Context, sa, region, subnetworkName string) (apiv1.GCPSubnetwork, error)

func GetGCPSubnetwork(ctx context.Context, sa, region, subnetworkName string) (apiv1.GCPSubnetwork, error) {
	computeService, project, err := ConnectToComputeService(ctx, sa)
	if err != nil {
		return apiv1.GCPSubnetwork{}, err
	}

	req := computeService.Subnetworks.Get(project, region, subnetworkName)
	subnetwork, err := req.Do()
	if err != nil {
		return apiv1.GCPSubnetwork{}, err
	}

	return ToGCPSubnetworkAPIModel(subnetwork), nil
}

var subnetworkRegex = regexp.MustCompile(`(projects\/.+)$`)

func ToGCPSubnetworkAPIModel(subnetwork *compute.Subnetwork) apiv1.GCPSubnetwork {
	subnetworkPath := subnetworkRegex.FindString(subnetwork.SelfLink)
	net := apiv1.GCPSubnetwork{
		ID:                    subnetwork.Id,
		Name:                  subnetwork.Name,
		Network:               subnetwork.Network,
		IPCidrRange:           subnetwork.IpCidrRange,
		GatewayAddress:        subnetwork.GatewayAddress,
		Region:                subnetwork.Region,
		SelfLink:              subnetwork.SelfLink,
		PrivateIPGoogleAccess: subnetwork.PrivateIpGoogleAccess,
		Kind:                  subnetwork.Kind,
		Path:                  subnetworkPath,
	}

	switch subnetwork.StackType {
	case "IPV4_ONLY":
		net.IPFamily = kubermaticv1.IPFamilyIPv4
	case "IPV4_IPV6":
		net.IPFamily = kubermaticv1.IPFamilyDualStack
	default:
		net.IPFamily = kubermaticv1.IPFamilyUnspecified
	}

	return net
}

func GetMachineSize(ctx context.Context, machineType, sa, zone string) (*provider.NodeCapacity, error) {
	computeService, project, err := ConnectToComputeService(ctx, sa)
	if err != nil {
		return nil, err
	}

	req := computeService.MachineTypes.Get(project, zone, machineType)
	m, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP machine size: %w", err)
	}

	capacity := provider.NewNodeCapacity()
	capacity.WithCPUCount(int(m.GuestCpus))

	if err := capacity.WithMemory(int(m.MemoryMb), "M"); err != nil {
		return nil, fmt.Errorf("failed to parse memory size: %w", err)
	}

	return capacity, nil
}
