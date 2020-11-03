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
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	compute "google.golang.org/api/compute/v1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/dc"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// GCPZoneReq represent a request for GCP zones.
// swagger:parameters listGCPZones
type GCPZoneReq struct {
	GCPCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
}

// GCPTypesReq represent a request for GCP machine or disk types.
// swagger:parameters listGCPDiskTypes listGCPSizes
type GCPTypesReq struct {
	GCPCommonReq
	// in: header
	// name: Zone
	Zone string
}

// GCPSubnetworksReq represent a request for GCP subnetworks.
// swagger:parameters listGCPSubnetworks
type GCPSubnetworksReq struct {
	GCPCommonReq
	// in: header
	// name: Network
	Network string
	// in: path
	// required: true
	DC string `json:"dc"`
}

// GCPCommonReq represent a request with common parameters for GCP.
type GCPCommonReq struct {
	// in: header
	// name: ServiceAccount
	ServiceAccount string
	// in: header
	// name: Credential
	Credential string
}

// GCPTypesNoCredentialReq represent a request for GCP machine or disk types.
// swagger:parameters listGCPSizesNoCredentials listGCPDiskTypesNoCredentials
type GCPTypesNoCredentialReq struct {
	common.GetClusterReq
	// in: header
	// name: Zone
	Zone string
}

// GCPSubnetworksNoCredentialReq represent a request for GCP subnetworks.
// swagger:parameters listGCPSubnetworksNoCredentials
type GCPSubnetworksNoCredentialReq struct {
	common.GetClusterReq
	// in: header
	// name: Network
	Network string
}

func DecodeGCPTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPTypesReq

	commonReq, err := DecodeGCPCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GCPCommonReq = commonReq.(GCPCommonReq)
	req.Zone = r.Header.Get("Zone")

	return req, nil
}

func DecodeGCPZoneReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPZoneReq

	commonReq, err := DecodeGCPCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GCPCommonReq = commonReq.(GCPCommonReq)

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return req, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

func DecodeGCPSubnetworksReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPSubnetworksReq

	commonReq, err := DecodeGCPCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GCPCommonReq = commonReq.(GCPCommonReq)
	req.Network = r.Header.Get("Network")

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return req, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

func DecodeGCPCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPCommonReq

	req.ServiceAccount = r.Header.Get("ServiceAccount")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeGCPTypesNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPTypesNoCredentialReq

	commonReq, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = commonReq.(common.GetClusterReq)
	req.Zone = r.Header.Get("Zone")

	return req, nil
}

func DecodeGCPSubnetworksNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPSubnetworksNoCredentialReq

	commonReq, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = commonReq.(common.GetClusterReq)
	req.Network = r.Header.Get("Network")

	return req, nil
}

func GCPDiskTypesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesReq)

		zone := req.Zone
		sa := req.ServiceAccount
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount
			}
		}
		return listGCPDiskTypes(ctx, sa, zone)
	}
}

func GCPDiskTypesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.GCP == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
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

		return listGCPDiskTypes(ctx, sa, req.Zone)
	}
}

func listGCPDiskTypes(ctx context.Context, sa string, zone string) (apiv1.GCPDiskTypeList, error) {
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

func GCPSizeEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesReq)

		zone := req.Zone
		sa := req.ServiceAccount

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount
			}
		}

		return listGCPSizes(ctx, sa, zone)
	}
}

func GCPSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.GCP == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
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
		return listGCPSizes(ctx, sa, req.Zone)
	}
}

func listGCPSizes(ctx context.Context, sa string, zone string) (apiv1.GCPMachineSizeList, error) {
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

func GCPZoneEndpoint(presetsProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPZoneReq)
		sa := req.ServiceAccount

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount
			}
		}

		return listGCPZones(ctx, userInfo, sa, req.DC, seedsGetter)
	}
}

func GCPZoneWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.GCP == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
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
		return listGCPZones(ctx, userInfo, sa, cluster.Spec.Cloud.DatacenterName, seedsGetter)
	}
}

func listGCPZones(ctx context.Context, userInfo *provider.UserInfo, sa, datacenterName string, seedsGetter provider.SeedsGetter) (apiv1.GCPZoneList, error) {
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

func GCPNetworkEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPCommonReq)
		sa := req.ServiceAccount

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount
			}
		}

		return listGCPNetworks(ctx, sa)
	}
}

func GCPNetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.GCP == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
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
		return listGCPNetworks(ctx, sa)
	}
}

func listGCPNetworks(ctx context.Context, sa string) (apiv1.GCPNetworkList, error) {
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

func GCPSubnetworkEndpoint(presetsProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPSubnetworksReq)
		sa := req.ServiceAccount

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount
			}
		}

		return listGCPSubnetworks(ctx, userInfo, req.DC, sa, req.Network, seedsGetter)
	}
}

func GCPSubnetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPSubnetworksNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.GCP == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
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
		return listGCPSubnetworks(ctx, userInfo, req.DC, sa, req.Network, seedsGetter)
	}
}

func listGCPSubnetworks(ctx context.Context, userInfo *provider.UserInfo, datacenterName string, sa string, networkName string, seedsGetter provider.SeedsGetter) (apiv1.GCPSubnetworkList, error) {
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
