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
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/utils/pointer"
)

// GCPZoneReq represent a request for GCP zones.
// swagger:parameters listGCPZones
type GCPZoneReq struct {
	GCPCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
}

// GCPMachineTypesReq represent a request for GCP machine types.
// swagger:parameters listGCPSizes
type GCPMachineTypesReq struct {
	GCPCommonReq
	// in: header
	// name: Zone
	Zone string
	// in: header
	// DatacenterName datacenter name
	DatacenterName string
}

// GCPDiskTypesReq represent a request for GCP disk types.
// swagger:parameters listGCPDiskTypes
type GCPDiskTypesReq struct {
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

func DecodeGCPMachineTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPMachineTypesReq

	commonReq, err := DecodeGCPCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GCPCommonReq = commonReq.(GCPCommonReq)
	req.Zone = r.Header.Get("Zone")
	req.DatacenterName = r.Header.Get("DatacenterName")

	return req, nil
}

func DecodeGCPDiskTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPDiskTypesReq

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

func GCPDiskTypesEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPDiskTypesReq)

		zone := req.Zone
		sa := req.ServiceAccount
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(ctx, userInfo, pointer.String(""), req.Credential)
			if err != nil {
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount
			}
		}
		return providercommon.ListGCPDiskTypes(ctx, sa, zone)
	}
}

func GCPDiskTypesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesNoCredentialReq)
		return providercommon.GCPDiskTypesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.Zone)
	}
}

func GCPSizeEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPMachineTypesReq)

		zone := req.Zone
		sa := req.ServiceAccount

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(ctx, userInfo, pointer.String(""), req.Credential)
			if err != nil {
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount
			}
		}
		settings, err := settingsProvider.GetGlobalSettings(ctx)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		filter := *settings.Spec.MachineDeploymentVMResourceQuota
		datacenterName := req.DatacenterName
		if datacenterName != "" {
			_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
			if err != nil {
				return nil, fmt.Errorf("error getting dc: %w", err)
			}
			filter = handlercommon.DetermineMachineFlavorFilter(datacenter.Spec.MachineFlavorFilter, settings.Spec.MachineDeploymentVMResourceQuota)
		}
		return providercommon.ListGCPSizes(ctx, filter, sa, zone)
	}
}

func GCPSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesNoCredentialReq)
		return providercommon.GCPSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, settingsProvider, req.ProjectID, req.ClusterID, req.Zone)
	}
}

func GCPZoneEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPZoneReq)
		sa := req.ServiceAccount

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(ctx, userInfo, pointer.String(""), req.Credential)
			if err != nil {
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount
			}
		}

		return providercommon.ListGCPZones(ctx, userInfo, sa, req.DC, seedsGetter)
	}
}

func GCPZoneWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		return providercommon.GCPZoneWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}

func GCPNetworkEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPCommonReq)
		sa := req.ServiceAccount

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(ctx, userInfo, pointer.String(""), req.Credential)
			if err != nil {
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount

				if credentials.Network != "" {
					networkParts := strings.Split(credentials.Network, "/")
					if len(networkParts) != 3 {
						return nil, utilerrors.New(http.StatusBadRequest, "invalid GCP network path for preset in credential header")
					}
					networkName := networkParts[2]
					presetNetwork, err := gcp.GetGCPNetwork(ctx, sa, networkName)
					if err != nil {
						return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("cannot get network %s", credentials.Network))
					}
					return apiv1.GCPNetworkList{presetNetwork}, nil
				}
			}
		}

		return providercommon.ListGCPNetworks(ctx, sa)
	}
}

func GCPNetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
		return providercommon.GCPNetworkWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

func GCPSubnetworkEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPSubnetworksReq)
		sa := req.ServiceAccount

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(ctx, userInfo, pointer.String(""), req.Credential)
			if err != nil {
				return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.GCP; credentials != nil {
				sa = credentials.ServiceAccount

				if credentials.Subnetwork != "" {
					subnetworkParts := strings.Split(credentials.Subnetwork, "/")
					if len(subnetworkParts) != 6 {
						return nil, utilerrors.New(http.StatusBadRequest, "invalid GCP subnetwork path for preset in credential header")
					}
					subnetworkRegion := subnetworkParts[3]
					subnetworkName := subnetworkParts[5]
					presetSubnetwork, err := gcp.GetGCPSubnetwork(ctx, sa, subnetworkRegion, subnetworkName)
					if err != nil {
						return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("cannot get subnetwork %s", credentials.Subnetwork))
					}
					return apiv1.GCPSubnetworkList{presetSubnetwork}, nil
				}
			}
		}

		return providercommon.ListGCPSubnetworks(ctx, userInfo, req.DC, sa, req.Network, seedsGetter)
	}
}

func GCPSubnetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPSubnetworksNoCredentialReq)
		return providercommon.GCPSubnetworkWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, req.Network)
	}
}
