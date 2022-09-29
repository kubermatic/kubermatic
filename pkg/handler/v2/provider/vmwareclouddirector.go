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
	"k8s.io/utils/pointer"

	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	vcd "k8c.io/kubermatic/v2/pkg/provider/cloud/vmwareclouddirector"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

type VMwareCloudDirectorCommonReq struct {
	// KKP Datacenter to use for endpoint
	// in: path
	// required: true
	DC string `json:"dc"`

	// in: header
	// name: Username
	Username string

	// in: header
	// name: Password
	Password string

	// in: header
	// name: Organization
	Organization string

	// in: header
	// name: VDC
	VDC string

	// in: header
	// name: Credential
	Credential string
}

func DecodeVMwareCloudDirectorCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VMwareCloudDirectorCommonReq

	req.Username = r.Header.Get("Username")
	req.Password = r.Header.Get("Password")
	req.Organization = r.Header.Get("Organization")
	req.VDC = r.Header.Get("VDC")
	req.Credential = r.Header.Get("Credential")

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return nil, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

// VMwareCloudDirectorCatalogReq represents a request for listing catalogs.
// swagger:parameters listVMwareCloudDirectorCatalogs
type VMwareCloudDirectorCatalogReq struct {
	VMwareCloudDirectorCommonReq
}

// VMwareCloudDirectorNetworkReq represents a request for listing OVDC networks.
// swagger:parameters listVMwareCloudDirectorNetworks
type VMwareCloudDirectorNetworkReq struct {
	VMwareCloudDirectorCommonReq
}

// VMwareCloudDirectorStorageProfileReq represents a request for listing storage profiles.
// swagger:parameters listVMwareCloudDirectorStorageProfiles
type VMwareCloudDirectorStorageProfileReq struct {
	VMwareCloudDirectorCommonReq
}

// VMwareCloudDirectorNoCredentialsReq represent a request for VMwareCloudDirector information with cluster-provided credentials
// swagger:parameters listVMwareCloudDirectorNetworksNoCredentials listVMwareCloudDirectorStorageProfilesNoCredentials listVMwareCloudDirectorCatalogsNoCredentials
type VMwareCloudDirectorNoCredentialsReq struct {
	cluster.GetClusterReq
}

func DecodeVMwareCloudDirectorNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VMwareCloudDirectorNoCredentialsReq

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

// VMwareCloudDirectoTemplateReq defines HTTP request for listing templates.
// swagger:parameters listVMwareCloudDirectorTemplates
type VMwareCloudDirectorTemplateReq struct {
	VMwareCloudDirectorCommonReq

	// Catalog name to fetch the templates from
	// in: path
	// required: true
	CatalogName string `json:"catalog_name"`
}

// Validate validates listTemplatesReq request.
func (r VMwareCloudDirectorTemplateReq) Validate() error {
	if len(r.CatalogName) == 0 {
		return fmt.Errorf("catalog name cannot be empty")
	}
	return nil
}

func DecodeListTemplatesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VMwareCloudDirectorTemplateReq

	commonReq, err := DecodeVMwareCloudDirectorCommonReq(c, r)
	if err != nil {
		return nil, err
	}

	req.VMwareCloudDirectorCommonReq = commonReq.(VMwareCloudDirectorCommonReq)

	CatalogName, ok := mux.Vars(r)["catalog_name"]
	if !ok {
		return nil, fmt.Errorf("'catalog_name' parameter is required")
	}

	req.CatalogName = CatalogName

	return req, nil
}

// VMwareCloudDirectorTemplateNoCredentialsReq represents a request for VMware Cloud Director templates values with cluster-provided credentials
// swagger:parameters listVMwareCloudDirectorTemplatesNoCredentials
type VMwareCloudDirectorTemplateNoCredentialsReq struct {
	VMwareCloudDirectorNoCredentialsReq

	// Catalog name to fetch the templates from
	// in: path
	// required: true
	CatalogName string `json:"catalog_name"`
}

func DecodeVMwareCloudDirectorTemplateNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req VMwareCloudDirectorTemplateNoCredentialsReq

	noCredsReq, err := DecodeVMwareCloudDirectorNoCredentialsReq(c, r)
	if err != nil {
		return nil, err
	}
	req.VMwareCloudDirectorNoCredentialsReq = noCredsReq.(VMwareCloudDirectorNoCredentialsReq)

	CatalogName, ok := mux.Vars(r)["catalog_name"]
	if !ok {
		return nil, fmt.Errorf("'catalog_name' parameter is required")
	}
	req.CatalogName = CatalogName

	return req, nil
}

func getVMwareCloudDirectorCredentialsFromReq(ctx context.Context, req VMwareCloudDirectorCommonReq, userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter) (*vcd.Auth, error) {
	username := req.Username
	password := req.Password
	organization := req.Organization
	vdc := req.VDC

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if len(req.Credential) > 0 {
		preset, err := presetProvider.GetPreset(ctx, userInfo, pointer.String(""), req.Credential)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
		}
		if credentials := preset.Spec.VMwareCloudDirector; credentials != nil {
			username = credentials.Username
			password = credentials.Password
			organization = credentials.Organization
			vdc = credentials.VDC
		}
	}

	_, dc, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, req.DC)
	if err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}
	if dc.Spec.VMwareCloudDirector == nil {
		return nil, utilerrors.NewBadRequest("datacenter '%s' is not a VMware Cloud Director datacenter", req.DC)
	}

	return &vcd.Auth{
		Username:      username,
		Password:      password,
		Organization:  organization,
		VDC:           vdc,
		URL:           dc.Spec.VMwareCloudDirector.URL,
		AllowInsecure: dc.Spec.VMwareCloudDirector.AllowInsecure,
	}, nil
}

func getVMwareCloudDirectorCredentialsFromCluster(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, projectID, clusterID string) (*vcd.Auth, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.VMwareCloudDirector == nil {
		return nil, utilerrors.NewNotFound("no cloud spec for %s", clusterID)
	}

	datacenterName := cluster.Spec.Cloud.DatacenterName
	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, utilerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	_, datacenter, err := provider.DatacenterFromSeedMap(userInfo, seedsGetter, datacenterName)
	if err != nil {
		return nil, fmt.Errorf("failed to find Datacenter %q: %w", datacenterName, err)
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	return vcd.GetAuthInfo(cluster.Spec.Cloud, secretKeySelector, datacenter.Spec.VMwareCloudDirector)
}

func VMwareCloudDirectorNetworksEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorCommonReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMwareCloudDirectorCredentialsFromReq(ctx, req, userInfoGetter, presetProvider, seedsGetter)
		if err != nil {
			return nil, err
		}

		return vcd.ListOVDCNetworks(ctx, *creds)
	}
}

func VMwareCloudDirectorStorageProfilesEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorCommonReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMwareCloudDirectorCredentialsFromReq(ctx, req, userInfoGetter, presetProvider, seedsGetter)
		if err != nil {
			return nil, err
		}

		return vcd.ListStorageProfiles(ctx, *creds)
	}
}

func VMwareCloudDirectorCatalogsEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorCommonReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMwareCloudDirectorCredentialsFromReq(ctx, req, userInfoGetter, presetProvider, seedsGetter)
		if err != nil {
			return nil, err
		}

		return vcd.ListCatalogs(ctx, *creds)
	}
}

func VMwareCloudDirectorTemplatesEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorTemplateReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMwareCloudDirectorCredentialsFromReq(ctx, req.VMwareCloudDirectorCommonReq, userInfoGetter, presetProvider, seedsGetter)
		if err != nil {
			return nil, err
		}

		return vcd.ListTemplates(ctx, *creds, req.CatalogName)
	}
}

func VMwareCloudDirectorNetworksWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorNoCredentialsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMwareCloudDirectorCredentialsFromCluster(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		return vcd.ListOVDCNetworks(ctx, *creds)
	}
}

func VMwareCloudDirectorStorageProfilesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorNoCredentialsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMwareCloudDirectorCredentialsFromCluster(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		return vcd.ListStorageProfiles(ctx, *creds)
	}
}

func VMwareCloudDirectorCatalogsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorNoCredentialsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMwareCloudDirectorCredentialsFromCluster(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		return vcd.ListCatalogs(ctx, *creds)
	}
}

func VMwareCloudDirectorTemplatesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorTemplateNoCredentialsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMwareCloudDirectorCredentialsFromCluster(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, seedsGetter, req.ProjectID, req.ClusterID)
		if err != nil {
			return nil, err
		}

		return vcd.ListTemplates(ctx, *creds, req.CatalogName)
	}
}
