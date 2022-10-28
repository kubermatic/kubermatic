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
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/utils/pointer"
)

// KubeVirtGenericReq represent a request with common parameters for KubeVirt.
// swagger:parameters listKubevirtStorageClasses listKubeVirtIPreferences
type KubeVirtGenericReq struct {
	// in: header
	// name: Kubeconfig (provided credential)
	Kubeconfig string
	// in: header
	// name: Credential (predefined Kubermatic credential name from the Kubermatic presets)
	Credential string
}

// KubeVirtListInstanceOrVMIPresetsReq represent a request to list presets or instance types for KubeVirt.
// swagger:parameters listKubeVirtVMIPresets listKubeVirtInstancetypes
type KubeVirtListInstanceOrVMIPresetsReq struct {
	KubeVirtGenericReq

	// in: header
	// DatacenterName datacenter name
	DatacenterName string
}

// KubeVirtGenericNoCredentialReq represent a generic KubeVirt request with cluster credentials.
// swagger:parameters listKubevirtStorageClassesNoCredentials listKubeVirtPreferencesNoCredentials
type KubeVirtGenericNoCredentialReq struct {
	cluster.GetClusterReq
}

// KubeVirtListInstanceOrVMIPresetsNoCredentialReq represent a request to list presets or instance types for KubeVirt.
// swagger:parameters listKubeVirtVMIPresetsNoCredentials listKubeVirtInstancetypesNoCredentials
type KubeVirtListInstanceOrVMIPresetsNoCredentialReq struct {
	cluster.GetClusterReq

	// in: header
	// DatacenterName datacenter name
	DatacenterName string
}

func getKubeconfig(ctx context.Context, kubeconfig string, credential string, presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) (string, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return "", common.KubernetesErrorToHTTPError(err)
	}
	if len(credential) > 0 {
		preset, err := presetsProvider.GetPreset(ctx, userInfo, pointer.String(""), credential)
		if err != nil {
			return "", utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", credential, userInfo.Email))
		}
		if credentials := preset.Spec.Kubevirt; credentials != nil {
			kubeconfig = credentials.Kubeconfig
		}
	}
	return kubeconfig, nil
}

// KubeVirtVMIPresetsEndpoint handles the request to list available KubeVirtVMIPresets (provided credentials).
//
// Deprecated: in favor of KubeVirtInstancetypesEndpoint.
func KubeVirtVMIPresetsEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(KubeVirtListInstanceOrVMIPresetsReq)
		if !ok {
			return "", utilerrors.NewBadRequest("invalid request")
		}

		kubeconfig, err := getKubeconfig(ctx, req.Kubeconfig, req.Credential, presetsProvider, userInfoGetter)
		if err != nil {
			return nil, err
		}
		return providercommon.KubeVirtVMIPresets(ctx, kubeconfig, req.DatacenterName, nil, settingsProvider, userInfoGetter, seedsGetter)
	}
}

// KubeVirtInstancetypesEndpoint handles the request to list available KubeVirtInstancetypes (provided credentials).
func KubeVirtInstancetypesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(KubeVirtListInstanceOrVMIPresetsReq)
		if !ok {
			return "", utilerrors.NewBadRequest("invalid request")
		}

		kubeconfig, err := getKubeconfig(ctx, req.Kubeconfig, req.Credential, presetsProvider, userInfoGetter)
		if err != nil {
			return nil, err
		}
		return providercommon.KubeVirtInstancetypes(ctx, kubeconfig, req.DatacenterName, nil, settingsProvider, userInfoGetter, seedsGetter)
	}
}

// KubeVirtPreferencesEndpoint handles the request to list available KubeVirtPreferences (provided credentials).
func KubeVirtPreferencesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(KubeVirtGenericReq)
		if !ok {
			return "", utilerrors.NewBadRequest("invalid request")
		}

		kubeconfig, err := getKubeconfig(ctx, req.Kubeconfig, req.Credential, presetsProvider, userInfoGetter)
		if err != nil {
			return nil, err
		}
		return providercommon.KubeVirtPreferences(ctx, kubeconfig, nil, settingsProvider)
	}
}

// KubeVirtVMIPresetsWithClusterCredentialsEndpoint handles the request to list available KubeVirtVMIPresets (cluster credentials).
//
// Deprecated: in favor of KubeVirtInstancetypesWithClusterCredentialsEndpoint.
func KubeVirtVMIPresetsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(KubeVirtListInstanceOrVMIPresetsNoCredentialReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		return providercommon.KubeVirtVMIPresetsWithClusterCredentialsEndpoint(ctx, seedsGetter, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.DatacenterName, settingsProvider)
	}
}

// KubeVirtInstacetypesWithClusterCredentialsEndpoint handles the request to list available KubeVirtInstancetypes (cluster credentials).
func KubeVirtInstancetypesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(KubeVirtListInstanceOrVMIPresetsNoCredentialReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		return providercommon.KubeVirtInstancetypesWithClusterCredentialsEndpoint(ctx, userInfoGetter, seedsGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.DatacenterName, settingsProvider)
	}
}

// KubeVirtPreferencesWithClusterCredentialsEndpoint handles the request to list available KubeVirtPreferences (cluster credentials).
func KubeVirtPreferencesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(KubeVirtGenericNoCredentialReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		return providercommon.KubeVirtPreferencesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, settingsProvider)
	}
}

// KubeVirtStorageClassesEndpoint handles the request to list available k8s StorageClasses (provided credentials).
func KubeVirtStorageClassesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(KubeVirtGenericReq)
		if !ok {
			return "", utilerrors.NewBadRequest("invalid request")
		}

		kubeconfig, err := getKubeconfig(ctx, req.Kubeconfig, req.Credential, presetsProvider, userInfoGetter)
		if err != nil {
			return nil, err
		}
		return providercommon.KubeVirtStorageClasses(ctx, kubeconfig)
	}
}

// KubeVirtStorageClassesWithClusterCredentialsEndpoint handles the request to list storage classes (cluster credentials).
func KubeVirtStorageClassesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(KubeVirtGenericNoCredentialReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		return providercommon.KubeVirtStorageClassesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

// Decoders

func DecodeKubeVirtGenericReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubeVirtGenericReq
	req.Kubeconfig = r.Header.Get("Kubeconfig")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeKubeVirtListInstanceOrVMIPresetsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubeVirtListInstanceOrVMIPresetsReq
	req.Kubeconfig = r.Header.Get("Kubeconfig")
	req.Credential = r.Header.Get("Credential")
	req.DatacenterName = r.Header.Get("DatacenterName")

	return req, nil
}

func DecodeKubeVirtGenericNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubeVirtGenericNoCredentialReq
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

func DecodeKubeVirtListInstanceOrVMIPresetsNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubeVirtListInstanceOrVMIPresetsNoCredentialReq
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
	req.DatacenterName = r.Header.Get("DatacenterName")

	return req, nil
}
