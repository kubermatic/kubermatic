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
	"github.com/gorilla/mux"

	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// KubevirtCommonReq represent a request with common parameters for Kubevirt.
type KubevirtCommonReq struct {
	// in: header
	// name: KvKubeconfig (provided credential)
	KvKubeconfig string
	// in: header
	// name: Credential (predefined Kubermatic credential name from the Kubermatic presets)
	Credential string
}

// KubevirtVmiPresetsReq represent a request to list KubevirtVmiPreset with provided credentials.
// swagger:parameters listKubevirtVmiPresets
type KubevirtVmiPresetsReq struct {
	KubevirtCommonReq
}

// KubevirtVmiPresetReq represent a request to get a KubevirtVmiPreset with provided credentials
// swagger:parameters getKubevirtVmiPreset
type KubevirtVmiPresetReq struct {
	KubevirtCommonReq
	// in: path
	// required: true
	PresetName string `json:"preset_id"`
}

// KubevirtVmiPresetsNoCredentialReq represent a request to list KubevirtVmiPreset from cluster credentials.
// swagger:parameters listKubevirtVmiPresetsNoCredentials
type KubevirtVmiPresetsNoCredentialReq struct {
	cluster.GetClusterReq
}

// KubevirtVmiPresetNoCredentialReq represent a request to get a KubevirtVmiPreset with cluster credentials
// swagger:parameters getKubevirtVmiPresetNoCredentials
type KubevirtVmiPresetNoCredentialReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	PresetName string `json:"preset_id"`
}

// KubevirtStorageClassessReq represent a request to list k8s StorageClasses with provided credentials.
// swagger:parameters listKubevirtStorageClasses
type KubevirtStorageClassesReq struct {
	KubevirtCommonReq
}

// KubevirtStorageClassesNoCredentialReq represent a request to list StorageClasses from cluster credentials.
// swagger:parameters listKubevirtStorageClassessNoCredentials
type KubevirtStorageClassesNoCredentialReq struct {
	cluster.GetClusterReq
}

// KubevirtStorageClassReq represent a request to get a StorageClass with provided credentials
// swagger:parameters getKubevirtStorageClass
type KubevirtStorageClassReq struct {
	KubevirtCommonReq
	// in: path
	// required: true
	StorageClass string `json:"storageclass_id"`
}

// KubevirtStorageClassNoCredentialReq represent a request to get a StorageClass with cluster credentials
// swagger:parameters getKubevirtStorageClassNoCredentials
type KubevirtStorageClassNoCredentialReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	StorageClass string `json:"storageclass_id"`
}

// KubevirtVmiPresetsEndpoint handles the request to list available KubevirtVmiPresets (provided credentials)
func KubevirtVmiPresetsEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(KubevirtVmiPresetsReq)
		kvKubeconfig := req.KvKubeconfig

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Kubevirt; credentials != nil {
				kvKubeconfig = credentials.Kubeconfig
			}
		}
		return providercommon.KubevirtVmiPresets(kvKubeconfig)
	}
}

// KubevirtVmiPresetsWithClusterCredentialsEndpoint handles the request to list available KubevirtVmiPresets (cluster credentials)
func KubevirtVmiPresetsWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(KubevirtVmiPresetsNoCredentialReq)
		return providercommon.KubevirtVmiPresetsWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

// GET VmiPreset

// KubevirtVmiPresetEndpoint handles the request to get a KubevirtVmiPreset (provided credentials)
func KubevirtVmiPresetEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(KubevirtVmiPresetReq)
		kvKubeconfig := req.KvKubeconfig

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Kubevirt; credentials != nil {
				kvKubeconfig = credentials.Kubeconfig
			}
		}

		return providercommon.KubevirtVmiPreset(kvKubeconfig, req.PresetName)
	}
}

// KubevirtVmiPresetWithClusterCredentialsEndpoint handles the request a KubevirtVmiPreset (cluster credentials)
func KubevirtVmiPresetWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(KubevirtVmiPresetNoCredentialReq)
		return providercommon.KubevirtVmiPresetWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.PresetName)
	}
}

// LIST StorageClass

// KubevirtStorageClassesEndpoint handles the request to list available k8s StorageClasses (provided credentials)
func KubevirtStorageClassesEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(KubevirtStorageClassesReq)
		kvKubeconfig := req.KvKubeconfig

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Kubevirt; credentials != nil {
				kvKubeconfig = credentials.Kubeconfig
			}
		}
		return providercommon.KubevirtStorageClasses(kvKubeconfig)
	}
}

// KubevirtStorageClassesWithClusterCredentialsEndpoint handles the request to list storage classes (cluster credentials)
func KubevirtStorageClassesWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(KubevirtStorageClassesNoCredentialReq)
		return providercommon.KubevirtStorageClassesWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID)
	}
}

// GET StorageClass

// KubevirtStorageClassEndpoint handles the request to get a StorageClasses (provided credentials)
func KubevirtStorageClassEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(KubevirtStorageClassReq)
		kvKubeconfig := req.KvKubeconfig

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Kubevirt; credentials != nil {
				kvKubeconfig = credentials.Kubeconfig
			}
		}
		return providercommon.KubevirtStorageClass(kvKubeconfig, req.StorageClass)
	}
}

// KubevirtStorageClassWithClusterCredentialsEndpoint handles the request to list storage classes (cluster credentials)
func KubevirtStorageClassWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(KubevirtStorageClassNoCredentialReq)
		return providercommon.KubevirtStorageClassWithClusterCredentialsEndpoint(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, req.ClusterID, req.StorageClass)
	}
}

// Decoders

func DecodeKubevirtVmiPresetsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubevirtVmiPresetsReq
	req.KvKubeconfig = r.Header.Get("KvKubeconfig")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeKubevirtVmiPresetsNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubevirtVmiPresetsNoCredentialReq
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

func DecodeKubevirtVmiPresetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubevirtVmiPresetReq
	req.KvKubeconfig = r.Header.Get("KvKubeconfig")
	req.Credential = r.Header.Get("Credential")
	presetName := mux.Vars(r)["preset_id"]
	if presetName == "" {
		return "", fmt.Errorf("'preset_id' parameter is required but was not provided")
	}
	req.PresetName = presetName
	return req, nil
}

func DecodeKubevirtVmiPresetNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubevirtVmiPresetNoCredentialReq
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
	presetName := mux.Vars(r)["preset_id"]
	if presetName == "" {
		return "", fmt.Errorf("'preset_id' parameter is required but was not provided")
	}
	req.PresetName = presetName
	return req, nil
}

func DecodeKubevirtStorageClassesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubevirtStorageClassesReq
	req.KvKubeconfig = r.Header.Get("KvKubeconfig")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeKubevirtStorageClassesNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubevirtStorageClassesNoCredentialReq
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

func DecodeKubevirtStorageClassReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubevirtStorageClassReq
	req.KvKubeconfig = r.Header.Get("KvKubeconfig")
	req.Credential = r.Header.Get("Credential")
	storageClassName := mux.Vars(r)["storageclass_id"]
	if storageClassName == "" {
		return "", fmt.Errorf("'storageclass_id' parameter is required but was not provided")
	}
	req.StorageClass = storageClassName
	return req, nil
}

func DecodeKubevirtStorageClassNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req KubevirtStorageClassNoCredentialReq
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
	storageClass := mux.Vars(r)["storageclass_id"]
	if storageClass == "" {
		return "", fmt.Errorf("'storageclass_id' parameter is required but was not provided")
	}
	req.StorageClass = storageClass
	return req, nil
}
