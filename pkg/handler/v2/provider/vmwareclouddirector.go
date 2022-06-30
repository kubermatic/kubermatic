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

	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	vcd "k8c.io/kubermatic/v2/pkg/provider/cloud/vmwareclouddirector"
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

// VMwareCloudDirectoTemplateReq defines HTTP request for listing templates.
// swagger:parameters listVMwareCloudDirectorTemplates
type VMwareCloudDirectorTemplateReq struct {
	VMwareCloudDirectorCommonReq

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

func getVMareCloudDirectorCredentialsFromReq(ctx context.Context, req VMwareCloudDirectorCommonReq, userInfoGetter provider.UserInfoGetter, presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter) (*vcd.Auth, error) {
	username := req.Username
	password := req.Password
	organization := req.Organization
	vdc := req.VDC

	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	if len(req.Credential) > 0 {
		preset, err := presetProvider.GetPreset(ctx, userInfo, req.Credential)
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

func VMwareCloudDirectorNetworksEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorNetworkReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMareCloudDirectorCredentialsFromReq(ctx, req.VMwareCloudDirectorCommonReq, userInfoGetter, presetProvider, seedsGetter)
		if err != nil {
			return nil, err
		}

		return vcd.ListOVDCNetworks(ctx, *creds)
	}
}

func VMwareCloudDirectorStorageProfilesEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorStorageProfileReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMareCloudDirectorCredentialsFromReq(ctx, req.VMwareCloudDirectorCommonReq, userInfoGetter, presetProvider, seedsGetter)
		if err != nil {
			return nil, err
		}

		return vcd.ListStorageProfiles(ctx, *creds)
	}
}

func VMwareCloudDirectorCatalogsEndpoint(presetProvider provider.PresetProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(VMwareCloudDirectorCatalogReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		creds, err := getVMareCloudDirectorCredentialsFromReq(ctx, req.VMwareCloudDirectorCommonReq, userInfoGetter, presetProvider, seedsGetter)
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

		creds, err := getVMareCloudDirectorCredentialsFromReq(ctx, req.VMwareCloudDirectorCommonReq, userInfoGetter, presetProvider, seedsGetter)
		if err != nil {
			return nil, err
		}

		return vcd.ListTemplates(ctx, *creds, req.CatalogName)
	}
}
