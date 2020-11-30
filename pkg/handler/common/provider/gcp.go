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

package provider

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/dc"

	compute "google.golang.org/api/compute/v1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func GCPSizeWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, zone string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.GCP == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	sa, err := gcp.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	return ListGCPSizes(ctx, sa, zone)
}

func GCPZoneWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.GCP == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	sa, err := gcp.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return ListGCPZones(ctx, userInfo, sa, cluster.Spec.Cloud.DatacenterName, seedsGetter)
}

func GCPNetworkWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.GCP == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	sa, err := gcp.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	return ListGCPNetworks(ctx, sa)
}

func GCPSubnetworkWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID, network string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.GCP == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	sa, err := gcp.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return ListGCPSubnetworks(ctx, userInfo, cluster.Spec.Cloud.DatacenterName, sa, network, seedsGetter)
}

func GCPDiskTypesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, zone string) (interface{}, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.GCP == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	sa, err := gcp.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	return ListGCPDiskTypes(ctx, sa, zone)
}

func ListGCPDiskTypes(ctx context.Context, sa string, zone string) (apiv1.GCPDiskTypeList, error) {
	diskTypes := apiv1.GCPDiskTypeList{}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return diskTypes, err
	}

	req := computeService.DiskTypes.List(project, zone)
	err = req.Pages(ctx, func(page *compute.DiskTypeList) error {
		for _, diskType := range page.Items {
			if diskType.Name != "local-ssd" {
				// TODO: There are some issues at the moment with local-ssd, that's why it is disabled at the moment.
				dt := apiv1.GCPDiskType{
					Name:        diskType.Name,
					Description: diskType.Description,
				}
				diskTypes = append(diskTypes, dt)
			}
		}
		return nil
	})

	return diskTypes, err
}

func ListGCPSubnetworks(ctx context.Context, userInfo *provider.UserInfo, datacenterName string, sa string, networkName string, seedsGetter provider.SeedsGetter) (apiv1.GCPSubnetworkList, error) {
	datacenter, err := dc.GetDatacenter(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, errors.NewBadRequest("%v", err)
	}

	if datacenter.Spec.GCP == nil {
		return nil, errors.NewBadRequest("%s is not a GCP datacenter", datacenterName)
	}

	subnetworks := apiv1.GCPSubnetworkList{}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return subnetworks, err
	}

	req := computeService.Subnetworks.List(project, datacenter.Spec.GCP.Region)
	err = req.Pages(ctx, func(page *compute.SubnetworkList) error {
		subnetworkRegex := regexp.MustCompile(`(projects\/.+)$`)
		for _, subnetwork := range page.Items {
			// subnetworks.Network are a url e.g. https://www.googleapis.com/compute/v1/[...]/networks/default"
			// we just get the path of the network, instead of the url
			// therefore we can't use regular Filter function and need to check on our own
			if strings.Contains(subnetwork.Network, networkName) {
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

				subnetworks = append(subnetworks, net)
			}

		}
		return nil
	})

	return subnetworks, err
}

func ListGCPNetworks(ctx context.Context, sa string) (apiv1.GCPNetworkList, error) {
	networks := apiv1.GCPNetworkList{}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return networks, err
	}

	req := computeService.Networks.List(project)
	err = req.Pages(ctx, func(page *compute.NetworkList) error {
		networkRegex := regexp.MustCompile(`(global\/.+)$`)
		for _, network := range page.Items {
			networkPath := networkRegex.FindString(network.SelfLink)

			net := apiv1.GCPNetwork{
				ID:                    network.Id,
				Name:                  network.Name,
				AutoCreateSubnetworks: network.AutoCreateSubnetworks,
				Subnetworks:           network.Subnetworks,
				Kind:                  network.Kind,
				Path:                  networkPath,
			}

			networks = append(networks, net)
		}
		return nil
	})

	return networks, err
}

func ListGCPZones(ctx context.Context, userInfo *provider.UserInfo, sa, datacenterName string, seedsGetter provider.SeedsGetter) (apiv1.GCPZoneList, error) {
	datacenter, err := dc.GetDatacenter(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, errors.NewBadRequest("%v", err)
	}

	if datacenter.Spec.GCP == nil {
		return nil, errors.NewBadRequest("the %s is not GCP datacenter", datacenterName)
	}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return nil, err
	}

	zones := apiv1.GCPZoneList{}
	req := computeService.Zones.List(project)
	err = req.Pages(ctx, func(page *compute.ZoneList) error {
		for _, zone := range page.Items {

			if strings.HasPrefix(zone.Name, datacenter.Spec.GCP.Region) {
				apiZone := apiv1.GCPZone{Name: zone.Name}
				zones = append(zones, apiZone)
			}
		}
		return nil
	})

	return zones, err
}

func ListGCPSizes(ctx context.Context, sa string, zone string) (apiv1.GCPMachineSizeList, error) {
	sizes := apiv1.GCPMachineSizeList{}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return sizes, err
	}

	req := computeService.MachineTypes.List(project, zone)
	err = req.Pages(ctx, func(page *compute.MachineTypeList) error {
		for _, machineType := range page.Items {
			// TODO: Make the check below more generic, working for all the providers. It is needed as the pods
			//  with memory under 2 GB will be full with required pods like kube-proxy, CNI etc.
			if machineType.MemoryMb >= 2048 {
				mt := apiv1.GCPMachineSize{
					Name:        machineType.Name,
					Description: machineType.Description,
					Memory:      machineType.MemoryMb,
					VCPUs:       machineType.GuestCpus,
				}

				sizes = append(sizes, mt)
			}
		}
		return nil
	})

	return sizes, err
}
