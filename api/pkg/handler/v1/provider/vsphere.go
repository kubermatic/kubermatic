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

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/vsphere"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func VsphereNetworksEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VSphereNetworksReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = VSphereNetworksReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		username := req.Username
		password := req.Password

		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.VSphere; credentials != nil {
				username = credentials.Username
				password = credentials.Password
			}
		}

		return getVsphereNetworks(userInfo, seedsGetter, username, password, req.DatacenterName)
	}
}

func VsphereNetworksWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VSphereNetworksNoCredentialsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.VSphere == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName

		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
		}

		username, password, err := vsphere.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector, datacenter.Spec.VSphere)
		if err != nil {
			return nil, err
		}
		return getVsphereNetworks(userInfo, seedsGetter, username, password, datacenterName)
	}
}

func getVsphereNetworks(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, username, password, datacenterName string) ([]apiv1.VSphereNetwork, error) {
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}

	networks, err := vsphere.GetNetworks(datacenter.Spec.VSphere, username, password)
	if err != nil {
		return nil, err
	}

	var apiNetworks []apiv1.VSphereNetwork
	for _, net := range networks {
		apiNetworks = append(apiNetworks, apiv1.VSphereNetwork{
			Name:         net.Name,
			Type:         net.Type,
			RelativePath: net.RelativePath,
			AbsolutePath: net.AbsolutePath,
		})
	}

	return apiNetworks, nil
}

func VsphereFoldersEndpoint(seedsGetter provider.SeedsGetter, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VSphereFoldersReq)
		if !ok {
			return nil, fmt.Errorf("incorrect type of request, expected = VSphereFoldersReq, got = %T", request)
		}
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		username := req.Username
		password := req.Password

		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.VSphere; credentials != nil {
				username = credentials.Username
				password = credentials.Password
			}
		}

		return getVsphereFolders(userInfo, seedsGetter, username, password, req.DatacenterName)
	}
}

func VsphereFoldersWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VSphereFoldersNoCredentialsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}
		if cluster.Spec.Cloud.VSphere == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		datacenterName := cluster.Spec.Cloud.DatacenterName
		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
		if err != nil {
			return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
		}

		username, password, err := vsphere.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector, datacenter.Spec.VSphere)
		if err != nil {
			return nil, err
		}
		return getVsphereFolders(userInfo, seedsGetter, username, password, datacenterName)
	}
}

func getVsphereFolders(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, username, password, datacenterName string) ([]apiv1.VSphereFolder, error) {
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %v", datacenterName, err)
	}

	folders, err := vsphere.GetVMFolders(datacenter.Spec.VSphere, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %v", err)
	}

	var apiFolders []apiv1.VSphereFolder
	for _, folder := range folders {
		apiFolders = append(apiFolders, apiv1.VSphereFolder{Path: folder.Path})
	}

	return apiFolders, nil
}

// VSphereNetworksReq represent a request for vsphere networks
// swagger:parameters listVSphereNetworks
type VSphereNetworksReq struct {
	// in: header
	Username string
	// in: header
	Password string
	// in: header
	DatacenterName string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
}

func DecodeVSphereNetworksReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereNetworksReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

// VSphereNetworksNoCredentialsReq represent a request for vsphere networks
// swagger:parameters listVSphereNetworksNoCredentials
type VSphereNetworksNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodeVSphereNetworksNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereNetworksNoCredentialsReq
	lr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = lr.(common.GetClusterReq)
	return req, nil
}

// VSphereFoldersReq represent a request for vsphere folders
// swagger:parameters listVSphereFolders
type VSphereFoldersReq struct {
	// in: header
	Username string
	// in: header
	Password string
	// in: header
	DatacenterName string
	// in: header
	// Credential predefined Kubermatic credential name from the presets
	Credential string
}

func DecodeVSphereFoldersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereFoldersReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.DatacenterName = r.Header.Get("DatacenterName")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

// VSphereFoldersNoCredentialsReq represent a request for vsphere folders
// swagger:parameters listVSphereFoldersNoCredentials
type VSphereFoldersNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodeVSphereFoldersNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VSphereFoldersNoCredentialsReq
	lr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = lr.(common.GetClusterReq)
	return req, nil
}
