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

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gke"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// gcpTypesNoCredentialReq represent a request for GCP machine or disk types.
// swagger:parameters listGCPSizesNoCredentialsV2 listGCPDiskTypesNoCredentialsV2
type gcpTypesNoCredentialReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: header
	// name: Zone
	Zone string
}

// GetSeedCluster returns the SeedCluster object.
func (req gcpTypesNoCredentialReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

// gcpSubnetworksNoCredentialReq represent a request for GCP subnetworks.
// swagger:parameters listGCPSubnetworksNoCredentialsV2
type gcpSubnetworksNoCredentialReq struct {
	common.ProjectReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: header
	// name: Network
	Network string
}

// GetSeedCluster returns the SeedCluster object.
func (req gcpSubnetworksNoCredentialReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		ClusterID: req.ClusterID,
	}
}

func DecodeGCPTypesNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req gcpTypesNoCredentialReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}

	req.ProjectReq = pr.(common.ProjectReq)
	req.Zone = r.Header.Get("Zone")

	return req, nil
}

func DecodeGCPSubnetworksNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req gcpSubnetworksNoCredentialReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)
	req.Network = r.Header.Get("Network")

	return req, nil
}

func GCPDiskTypesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(gcpTypesNoCredentialReq)
		return providercommon.GCPDiskTypesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.Zone)
	}
}

func GCPSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(gcpTypesNoCredentialReq)
		return providercommon.GCPSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, settingsProvider, req.ProjectID, req.ClusterID, req.Zone)
	}
}

func GCPZoneWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(cluster.GetClusterReq)
		return providercommon.GCPZoneWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}

func GCPNetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(cluster.GetClusterReq)
		return providercommon.GCPNetworkWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

func GCPSubnetworkWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(gcpSubnetworksNoCredentialReq)
		return providercommon.GCPSubnetworkWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID, req.Network)
	}
}

// GKECommonReq represent a request with common parameters for GKE.
type GKECommonReq struct {
	// The plain GCP service account
	// in: header
	// name: ServiceAccount
	ServiceAccount string
	// The credential name used in the preset for the GCP provider
	// in: header
	// name: Credential
	Credential string
}

// GKEClusterListReq represent a request for GKE cluster list.
// swagger:parameters listGKEClusters
type GKEClusterListReq struct {
	common.ProjectReq
	GKECommonReq
}

func DecodeGKEClusterListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKEClusterListReq

	commonReq, err := DecodeGKECommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GKECommonReq = commonReq.(GKECommonReq)
	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// GKETypesReq represent a request for GKE types.
// swagger:parameters validateGKECredentials
type GKETypesReq struct {
	GKECommonReq
}

func DecodeGKECommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKECommonReq

	req.ServiceAccount = r.Header.Get("ServiceAccount")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeGKETypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKETypesReq

	commonReq, err := DecodeGKECommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GKECommonReq = commonReq.(GKECommonReq)

	return req, nil
}

func GKEClustersEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GKEClusterListReq)
		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return gke.ListGKEClusters(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, clusterProvider, req.ProjectID, sa)
	}
}

func GKEImagesEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GKEImagesReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return gke.ListGKEImages(ctx, sa, req.Zone)
	}
}

func GKEZonesEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GKECommonReq)
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return gke.ListGKEZones(ctx, sa)
	}
}

func GKEValidateCredentialsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GKETypesReq)

		var err error
		sa := req.ServiceAccount
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return nil, gke.ValidateGKECredentials(ctx, sa)
	}
}

func getSAFromPreset(ctx context.Context,
	userInfoGetter provider.UserInfoGetter,
	presetProvider provider.PresetProvider,
	presetName string,
) (string, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return "", common.KubernetesErrorToHTTPError(err)
	}
	preset, err := presetProvider.GetPreset(ctx, userInfo, presetName)
	if err != nil {
		return "", utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", presetName, userInfo.Email))
	}
	credentials := preset.Spec.GKE
	if credentials == nil {
		return "", fmt.Errorf("gke credentials not present in the preset %s", presetName)
	}
	return credentials.ServiceAccount, nil
}

// GKEImagesReq represent a request for GKE images.
// swagger:parameters listGKEImages
type GKEImagesReq struct {
	GKECommonReq
	// The zone name
	// in: header
	// name: Zone
	Zone string
}

// Validate validates GKECommonReq request.
func (req GKECommonReq) Validate() error {
	if len(req.ServiceAccount) == 0 && len(req.Credential) == 0 {
		return fmt.Errorf("GKE credentials cannot be empty")
	}
	return nil
}

// Validate validates GKEImagesReq request.
func (req GKEImagesReq) Validate() error {
	if err := req.GKECommonReq.Validate(); err != nil {
		return err
	}
	if len(req.Zone) == 0 {
		return fmt.Errorf("GKE Zone cannot be empty")
	}
	return nil
}

func DecodeGKEImagesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKEImagesReq

	commonReq, err := DecodeGKECommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GKECommonReq = commonReq.(GKECommonReq)

	req.Zone = r.Header.Get("Zone")

	return req, nil
}
