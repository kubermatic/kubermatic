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

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

type NutanixCommonReq struct {
	// KKP Datacenter to use for endpoint
	// in: path
	// required: true
	DC string `json:"dc"`

	// in: header
	// name: NutanixUsername
	NutanixUsername string

	// in: header
	// name: NutanixPassword
	NutanixPassword string

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
// swagger:parameters listNutanixProjects
type NutanixProjectReq struct {
	NutanixCommonReq
}

// NutanixSubnetReq represents a request for Nutanix subnets
// swagger:parameters listNutanixSubnets
type NutanixSubnetReq struct {
	NutanixCommonReq

	// in: header
	// name: NutanixCluster
	// required: true
	NutanixCluster string

	// Project query parameter. Can be omitted to query subnets without project scope
	// in: header
	// name: NutanixProject
	NutanixProject string
}

// NutanixNoCredentialReq represent a request for Nutanix information with cluster-provided credentials
// swagger:parameters listNutanixSubnetsNoCredentials
type NutanixNoCredentialReq struct {
	cluster.GetClusterReq
}

func DecodeNutanixCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NutanixCommonReq

	req.NutanixUsername = r.Header.Get("NutanixUsername")
	req.NutanixPassword = r.Header.Get("NutanixPassword")
	req.ProxyURL = r.Header.Get("NutanixProxyURL")
	req.Credential = r.Header.Get("Credential")

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return nil, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

func DecodeNutanixSubnetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NutanixSubnetReq

	commonReq, err := DecodeNutanixCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.NutanixCommonReq = commonReq.(NutanixCommonReq)
	req.NutanixCluster = r.Header.Get("NutanixCluster")
	req.NutanixProject = r.Header.Get("NutanixProject")

	return req, nil
}

func DecodeNutanixNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req NutanixNoCredentialReq

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
	return req, nil
}

// NutanixClusterEndpoint handles the request for a list of clusters, using provided credentials.
func NutanixClusterEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NutanixCommonReq)

		creds := providercommon.NutanixCredentials{
			ProxyURL: req.ProxyURL,
			Username: req.NutanixUsername,
			Password: req.NutanixPassword,
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if len(req.Credential) > 0 {
			preset, err := presetProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("cannot get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credential := preset.Spec.Nutanix; credential != nil {
				creds.ProxyURL = credential.ProxyURL
				creds.Username = credential.Username
				creds.Password = credential.Password
			}
		}

		if creds.Username == "" || creds.Password == "" {
			return nil, errors.NewBadRequest("no valid credentials found")
		}

		_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		if dc.Spec.Nutanix == nil {
			return nil, errors.NewBadRequest("datacenter '%s' is not a Nutanix datacenter", req.DC)
		}

		clusters, err := providercommon.NewNutanixClient(dc.Spec.Nutanix, &creds).ListNutanixClusters()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("cannot list clusters: %s", err.Error()))
		}

		return clusters, nil
	}
}

// NutanixProjectEndpoint handles the request for a list of projects, using provided credentials.
func NutanixProjectEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NutanixCommonReq)

		creds := providercommon.NutanixCredentials{
			ProxyURL: req.ProxyURL,
			Username: req.NutanixUsername,
			Password: req.NutanixPassword,
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

		if creds.Username == "" || creds.Password == "" {
			return nil, errors.NewBadRequest("no valid credentials found")
		}

		_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		if dc.Spec.Nutanix == nil {
			return nil, errors.NewBadRequest("datacenter '%s' is not a Nutanix datacenter", req.DC)
		}

		projects, err := providercommon.NewNutanixClient(dc.Spec.Nutanix, &creds).ListNutanixProjects()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("cannot list projects: %s", err.Error()))
		}

		return projects, nil
	}
}

// NutanixSubnetEndpoint handles the request for a list of subnets on a specific Nutanix cluster, using provided credentials.
func NutanixSubnetEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NutanixSubnetReq)

		creds := providercommon.NutanixCredentials{
			ProxyURL: req.ProxyURL,
			Username: req.NutanixUsername,
			Password: req.NutanixPassword,
		}

		cluster := req.NutanixCluster
		project := req.NutanixProject

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
				cluster = credential.ClusterName
				project = credential.ProjectName
			}
		}

		if creds.Username == "" || creds.Password == "" {
			return nil, errors.NewBadRequest("no valid credentials found")
		}

		_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}
		if dc.Spec.Nutanix == nil {
			return nil, errors.NewBadRequest("datacenter '%s' is not a Nutanix datacenter", req.DC)
		}

		subnets, err := providercommon.NewNutanixClient(dc.Spec.Nutanix, &creds).ListNutanixSubnets(cluster, project)
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
