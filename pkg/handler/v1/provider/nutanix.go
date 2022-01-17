/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"strconv"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"

	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

type NutanixCommonReq struct {
	// in: header
	// name: NutanixEndpoint
	Endpoint string

	// in: header
	// name: NutanixPort
	Port int32

	// in: header
	// name: AllowInsecure
	AllowInsecure *bool

	// in: header
	// name: NutanixUsername
	Username string

	// in: header
	// name: NutanixPassword
	Password string

	// in: header
	// name: ProxyURL
	ProxyURL string

	// in: header
	// name: Credential
	Credential string
}

// NutanixClusterReq represents a request for Nutanix clusters
// swagger:parameters listNutanixClusters
type NutanixClusterReq struct {
	NutanixCommonReq
}

// NutanixProjectReq represents a request for Nutanix projects
type NutanixProjectReq struct {
	NutanixCommonReq
}

// NutanixSubnetReq represents a request for Nutanix subnets
// swagger:parameters listNutanixSubnets
type NutanixSubnetReq struct {
	NutanixCommonReq

	// in: path
	// required: true
	ClusterName string `json:"cluster_name"`

	// Project query parameter. Can be omitted to query subnets without project scope
	// in: query
	ProjectName string `json:"project_name,omitempty"`
}

// NutanixNoCredentialReq represent a request for Nutanix information with cluster-provided credentials
// swagger:parameters listNutanixSubnetsNoCredentials
type NutanixNoCredentialReq struct {
	common.GetClusterReq
}

func DecodeNutanixCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NutanixCommonReq

	req.Endpoint = r.Header.Get("NutanixEndpoint")
	req.Username = r.Header.Get("NutanixUsername")
	req.Password = r.Header.Get("NutanixPassword")
	req.ProxyURL = r.Header.Get("ProxyURL")
	req.Credential = r.Header.Get("Credential")

	portHeader := r.Header.Get("NutanixPort")
	if portHeader != "" {
		port, err := strconv.Atoi(r.Header.Get("NutanixPort"))
		if err != nil {
			return nil, err
		}
		req.Port = int32(port)
	}

	allowInsecureHeader := r.Header.Get("AllowInsecure")
	if allowInsecureHeader != "" {
		allowInsecure, err := strconv.ParseBool(r.Header.Get("AllowInsecure"))
		if err != nil {
			return nil, err
		}
		req.AllowInsecure = &allowInsecure
	}

	return req, nil
}

func DecodeNutanixSubnetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NutanixSubnetReq

	commonReq, err := DecodeNutanixCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.NutanixCommonReq = commonReq.(NutanixCommonReq)
	req.ProjectName = r.URL.Query().Get("project_name")

	cluster, ok := mux.Vars(r)["cluster_name"]
	if !ok {
		return nil, fmt.Errorf("'cluster_name' parameter is required")
	}
	req.ClusterName = cluster

	return req, nil
}

func DecodeNutanixNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NutanixNoCredentialReq

	commonReq, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = commonReq.(common.GetClusterReq)

	return req, nil
}

// NutanixClusterEndpoint handles the request for a list of clusters, using provided credentials
func NutanixClusterEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NutanixCommonReq)

		creds := providercommon.NutanixCredentials{
			Endpoint:      req.Endpoint,
			AllowInsecure: req.AllowInsecure,
			ProxyURL:      req.ProxyURL,
			Username:      req.Username,
			Password:      req.Password,
		}

		if req.Port != 0 {
			creds.Port = &req.Port
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credential := preset.Spec.Nutanix; credential != nil {
				creds.ProxyURL = credential.ProxyURL
				creds.Username = credential.Username
				creds.Password = credential.Password
			}
		}

		clusters, err := providercommon.ListNutanixClusters(creds)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("cannot list clusters: %s", err.Error()))
		}

		return clusters, nil
	}
}

// NutanixProjectEndpoint handles the request for a list of projects, using provided credentials
func NutanixProjectEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NutanixCommonReq)

		creds := providercommon.NutanixCredentials{
			Endpoint:      req.Endpoint,
			AllowInsecure: req.AllowInsecure,
			ProxyURL:      req.ProxyURL,
			Username:      req.Username,
			Password:      req.Password,
		}

		if req.Port != 0 {
			creds.Port = &req.Port
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credential := preset.Spec.Nutanix; credential != nil {
				creds.ProxyURL = credential.ProxyURL
				creds.Username = credential.Username
				creds.Password = credential.Password
			}
		}

		projects, err := providercommon.ListNutanixProjects(creds)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("cannot list projects: %s", err.Error()))
		}

		return projects, nil
	}
}

// NutanixSubnetEndpoint handles the request for a list of subnets on a specific Nutanix cluster, using provided credentials
func NutanixSubnetEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NutanixSubnetReq)

		creds := providercommon.NutanixCredentials{
			Endpoint:      req.Endpoint,
			AllowInsecure: req.AllowInsecure,
			ProxyURL:      req.ProxyURL,
			Username:      req.Username,
			Password:      req.Password,
		}

		if req.Port != 0 {
			creds.Port = &req.Port
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credential := preset.Spec.Nutanix; credential != nil {
				creds.ProxyURL = credential.ProxyURL
				creds.Username = credential.Username
				creds.Password = credential.Password
			}
		}

		subnets, err := providercommon.ListNutanixSubnets(creds, req.ClusterName, req.ProjectName)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("cannot list subnets: %s", err.Error()))
		}

		return subnets, nil
	}
}

func NutanixSubnetsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NutanixNoCredentialReq)
		return providercommon.NutanixSubnetsWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
	}
}
