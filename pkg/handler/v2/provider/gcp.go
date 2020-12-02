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

	"github.com/go-kit/kit/endpoint"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
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

// GetSeedCluster returns the SeedCluster object
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

// GetSeedCluster returns the SeedCluster object
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

func GCPSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(gcpTypesNoCredentialReq)
		return providercommon.GCPSizeWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.Zone)
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
