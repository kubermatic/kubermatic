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

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
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

		return providercommon.GetVsphereNetworks(userInfo, seedsGetter, username, password, req.DatacenterName)
	}
}

func VsphereNetworksWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VSphereNetworksNoCredentialsReq)
		return providercommon.VsphereNetworksWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)

	}
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

		return providercommon.GetVsphereFolders(userInfo, seedsGetter, username, password, req.DatacenterName)
	}
}

func VsphereFoldersWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(VSphereFoldersNoCredentialsReq)
		return providercommon.VsphereFoldersWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
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

func DecodeVSphereNetworksReq(_ context.Context, r *http.Request) (interface{}, error) {
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

func DecodeVSphereFoldersReq(_ context.Context, r *http.Request) (interface{}, error) {
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
