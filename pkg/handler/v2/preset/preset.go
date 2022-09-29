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

package preset

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// listPresetsReq represents a request for a list of presets
// swagger:parameters listPresets
type listPresetsReq struct {
	// in: query
	Disabled bool `json:"disabled,omitempty"`
}

// listProjectPresetsReq represents a request for a list of presets in a specific project
// swagger:parameters listProjectPresets
type listProjectPresetsReq struct {
	common.ProjectReq
	listPresetsReq
}

func DecodeListPresets(_ context.Context, r *http.Request) (interface{}, error) {
	return listPresetsReq{
		Disabled: r.URL.Query().Get("disabled") == "true",
	}, nil
}

func DecodeListProjectPresets(ctx context.Context, r *http.Request) (interface{}, error) {
	listReq, err := DecodeListPresets(ctx, r)
	if err != nil {
		return nil, err
	}

	projectReq, err := common.DecodeProjectRequest(ctx, r)
	if err != nil {
		return nil, err
	}

	return listProjectPresetsReq{
		listPresetsReq: listReq.(listPresetsReq),
		ProjectReq:     projectReq.(common.ProjectReq),
	}, nil
}

// ListPresets returns a list of presets.
func ListPresets(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listPresetsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		presetList := &apiv2.PresetList{Items: make([]apiv2.Preset, 0)}
		presets, err := presetProvider.GetPresets(ctx, userInfo, nil)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}

		for _, preset := range presets {
			enabled := preset.Spec.IsEnabled()

			if !preset.Spec.IsEnabled() && !req.Disabled {
				continue
			}

			presetList.Items = append(presetList.Items, newAPIPreset(&preset, enabled))
		}

		return presetList, nil
	}
}

// ListProjectPresets returns a list of presets for a specific project.
func ListProjectPresets(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listProjectPresetsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		presetList := &apiv2.PresetList{Items: make([]apiv2.Preset, 0)}
		presets, err := presetProvider.GetPresets(ctx, userInfo, &req.ProjectID)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}

		for _, preset := range presets {
			enabled := preset.Spec.IsEnabled()

			if !preset.Spec.IsEnabled() && !req.Disabled {
				continue
			}

			presetList.Items = append(presetList.Items, newAPIPreset(&preset, enabled))
		}

		return presetList, nil
	}
}

// updatePresetStatusReq represents a request to update preset status
// swagger:parameters updatePresetStatus
type updatePresetStatusReq struct {
	// in: path
	// required: true
	PresetName string `json:"preset_name"`
	// in: query
	Provider string `json:"provider,omitempty"`
	// in: body
	// required: true
	Body struct {
		Enabled bool `json:"enabled"`
	}
}

func DecodeUpdatePresetStatus(_ context.Context, r *http.Request) (interface{}, error) {
	var req updatePresetStatusReq

	req.PresetName = mux.Vars(r)["preset_name"]
	req.Provider = r.URL.Query().Get("provider")
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// Validate validates updatePresetStatusReq request.
func (r updatePresetStatusReq) Validate() error {
	if len(r.PresetName) == 0 {
		return fmt.Errorf("the preset name cannot be empty")
	}

	if len(r.Provider) > 0 && !kubermaticv1.IsProviderSupported(r.Provider) {
		return fmt.Errorf("invalid provider name %s", r.Provider)
	}

	return nil
}

// UpdatePresetStatus updates the status of a preset. It can enable or disable it, so that it won't be listed by the list endpoints.
func UpdatePresetStatus(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updatePresetStatusReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, "only admins can update presets")
		}

		preset, err := presetProvider.GetPreset(ctx, userInfo, nil, req.PresetName)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}

		if len(req.Provider) == 0 {
			preset.Spec.SetEnabled(req.Body.Enabled)
			_, err = presetProvider.UpdatePreset(ctx, preset)
			return nil, err
		}

		if hasProvider, _ := kubermaticv1helper.HasProvider(preset, kubermaticv1.ProviderType(req.Provider)); !hasProvider {
			return nil, utilerrors.New(http.StatusConflict, fmt.Sprintf("trying to update preset with missing provider configuration for: %s", req.Provider))
		}

		kubermaticv1helper.SetProviderEnabled(preset, kubermaticv1.ProviderType(req.Provider), req.Body.Enabled)
		_, err = presetProvider.UpdatePreset(ctx, preset)
		return nil, err
	}
}

// listProviderPresetsReq represents a request for a list of presets
// swagger:parameters listProviderPresets
type listProviderPresetsReq struct {
	listPresetsReq `json:",inline"`

	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: query
	Datacenter string `json:"datacenter,omitempty"`
}

// listProjectProviderPresetsReq represents a request for a list of presets for a specific project
// swagger:parameters listProjectProviderPresets
type listProjectProviderPresetsReq struct {
	common.ProjectReq
	listProviderPresetsReq
}

func (l listProviderPresetsReq) matchesDatacenter(datacenter string) bool {
	return len(datacenter) == 0 || len(l.Datacenter) == 0 || strings.EqualFold(l.Datacenter, datacenter)
}

// Validate validates listProviderPresetsReq request.
func (l listProviderPresetsReq) Validate() error {
	if len(l.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	if !kubermaticv1.IsProviderSupported(l.ProviderName) {
		return fmt.Errorf("invalid provider name %s", l.ProviderName)
	}

	return nil
}

func DecodeListProviderPresets(ctx context.Context, r *http.Request) (interface{}, error) {
	listReq, err := DecodeListPresets(ctx, r)
	if err != nil {
		return nil, err
	}

	return listProviderPresetsReq{
		listPresetsReq: listReq.(listPresetsReq),
		ProviderName:   mux.Vars(r)["provider_name"],
		Datacenter:     r.URL.Query().Get("datacenter"),
	}, nil
}

func DecodeListProjectProviderPresets(ctx context.Context, r *http.Request) (interface{}, error) {
	listReq, err := DecodeListProviderPresets(ctx, r)
	if err != nil {
		return nil, err
	}

	projectReq, err := common.DecodeProjectRequest(ctx, r)
	if err != nil {
		return nil, err
	}

	return listProjectProviderPresetsReq{
		listProviderPresetsReq: listReq.(listProviderPresetsReq),
		ProjectReq:             projectReq.(common.ProjectReq),
	}, nil
}

// ListProviderPresets returns a list of preset names for the provider.
func ListProviderPresets(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listProviderPresetsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		presetList := &apiv2.PresetList{Items: make([]apiv2.Preset, 0)}
		presets, err := presetProvider.GetPresets(ctx, userInfo, nil)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}

		for _, preset := range presets {
			providerType := kubermaticv1.ProviderType(req.ProviderName)
			providerPreset := kubermaticv1helper.GetProviderPreset(&preset, providerType)

			// Preset does not contain requested provider configuration
			if providerPreset == nil {
				continue
			}

			// Preset does not contain requested datacenter
			if !req.matchesDatacenter(providerPreset.Datacenter) {
				continue
			}

			// Skip disabled presets when not requested
			enabled := preset.Spec.IsEnabled() && providerPreset.IsEnabled()
			if !req.Disabled && !enabled {
				continue
			}

			presetList.Items = append(presetList.Items, newAPIPreset(&preset, enabled))
		}

		return presetList, nil
	}
}

// ListProjectProviderPresets returns a list of presets for a specific provider in a specific project.
func ListProjectProviderPresets(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listProjectProviderPresetsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, req.ProjectID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		presetList := &apiv2.PresetList{Items: make([]apiv2.Preset, 0)}
		presets, err := presetProvider.GetPresets(ctx, userInfo, &req.ProjectID)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}

		for _, preset := range presets {
			providerType := kubermaticv1.ProviderType(req.ProviderName)
			providerPreset := kubermaticv1helper.GetProviderPreset(&preset, providerType)

			// Preset does not contain requested provider configuration
			if providerPreset == nil {
				continue
			}

			// Preset does not contain requested datacenter
			if !req.matchesDatacenter(providerPreset.Datacenter) {
				continue
			}

			// Skip disabled presets when not requested
			enabled := preset.Spec.IsEnabled() && providerPreset.IsEnabled()
			if !req.Disabled && !enabled {
				continue
			}

			presetList.Items = append(presetList.Items, newAPIPreset(&preset, enabled))
		}

		return presetList, nil
	}
}

// createPresetReq represents a request to create a new preset
// swagger:parameters createPreset
type createPresetReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: body
	// required: true
	Body apiv2.PresetBody
}

// Validate validates createPresetReq request.
func (r createPresetReq) Validate() error {
	if len(r.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	if !kubermaticv1.IsProviderSupported(r.ProviderName) {
		return fmt.Errorf("invalid provider name %s", r.ProviderName)
	}

	if len(r.Body.Name) == 0 {
		return fmt.Errorf("preset name cannot be empty")
	}

	if hasProvider, _ := kubermaticv1helper.HasProvider(convertAPIToInternalPreset(r.Body), kubermaticv1.ProviderType(r.ProviderName)); !hasProvider {
		return fmt.Errorf("missing provider configuration for: %s", r.ProviderName)
	}

	err := kubermaticv1helper.Validate(convertAPIToInternalPreset(r.Body), kubermaticv1.ProviderType(r.ProviderName))
	if err != nil {
		return err
	}

	for _, providerType := range kubermaticv1.SupportedProviders {
		if string(providerType) == r.ProviderName {
			continue
		}

		if hasProvider, _ := kubermaticv1helper.HasProvider(convertAPIToInternalPreset(r.Body), providerType); hasProvider {
			return fmt.Errorf("found unexpected provider configuration for: %s", providerType)
		}
	}

	return nil
}

func DecodeCreatePreset(_ context.Context, r *http.Request) (interface{}, error) {
	var req createPresetReq

	req.ProviderName = mux.Vars(r)["provider_name"]
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// CreatePreset creates a preset for the selected provider and returns the name if successful, error otherwise.
func CreatePreset(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createPresetReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if !userInfo.IsAdmin {
			return "", utilerrors.New(http.StatusForbidden, "only admins can update presets")
		}

		preset, err := presetProvider.GetPreset(ctx, userInfo, nil, req.Body.Name)
		if apierrors.IsNotFound(err) {
			return presetProvider.CreatePreset(ctx, convertAPIToInternalPreset(req.Body))
		}

		if err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}

		if hasProvider, _ := kubermaticv1helper.HasProvider(preset, kubermaticv1.ProviderType(req.ProviderName)); hasProvider {
			return nil, utilerrors.New(http.StatusConflict, fmt.Sprintf("%s provider configuration already exists for preset %s", req.ProviderName, preset.Name))
		}

		preset = mergePresets(preset, convertAPIToInternalPreset(req.Body), kubermaticv1.ProviderType(req.ProviderName))
		preset, err = presetProvider.UpdatePreset(ctx, preset)
		if err != nil {
			return nil, err
		}

		providerType := kubermaticv1.ProviderType(req.ProviderName)
		enabled := preset.Spec.IsEnabled() && kubermaticv1helper.IsProviderEnabled(preset, providerType)
		return newAPIPreset(preset, enabled), nil
	}
}

// updatePresetReq represents a request to update a preset
// swagger:parameters updatePreset
type updatePresetReq struct {
	createPresetReq
}

// Validate validates updatePresetReq request.
func (r updatePresetReq) Validate() error {
	return r.createPresetReq.Validate()
}

func DecodeUpdatePreset(_ context.Context, r *http.Request) (interface{}, error) {
	var req updatePresetReq

	req.ProviderName = mux.Vars(r)["provider_name"]
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// UpdatePreset updates a preset for the selected provider and returns the name if successful, error otherwise.
func UpdatePreset(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updatePresetReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if !userInfo.IsAdmin {
			return "", utilerrors.New(http.StatusForbidden, "only admins can update presets")
		}

		preset, err := presetProvider.GetPreset(ctx, userInfo, req.Body.Name)
		if err != nil {
			return nil, err
		}

		preset = mergePresets(preset, convertAPIToInternalPreset(req.Body), kubermaticv1.ProviderType(req.ProviderName))
		preset, err = presetProvider.UpdatePreset(ctx, preset)
		if err != nil {
			return nil, err
		}

		providerType := kubermaticv1.ProviderType(req.ProviderName)
		enabled := preset.Spec.IsEnabled() && kubermaticv1helper.IsProviderEnabled(preset, providerType)
		return newAPIPreset(preset, enabled), nil
	}
}

// deletePresetReq represents a request to delete a preset
// swagger:parameters deletePreset
type deletePresetReq struct {
	// in: path
	// required: true
	PresetName string `json:"preset_name"`
}

// Validate validates deletePresetReq request.
func (r deletePresetReq) Validate() error {
	if len(r.PresetName) == 0 {
		return fmt.Errorf("preset name cannot be empty")
	}
	return nil
}

func DecodeDeletePreset(_ context.Context, r *http.Request) (interface{}, error) {
	var req deletePresetReq

	req.PresetName = mux.Vars(r)["preset_name"]

	return req, nil
}

// DeletePreset deletes preset.
func DeletePreset(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deletePresetReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if !userInfo.IsAdmin {
			return "", utilerrors.New(http.StatusForbidden, "only admins can delete presets")
		}

		preset, err := presetProvider.GetPreset(ctx, userInfo, nil, req.PresetName)
		if apierrors.IsNotFound(err) {
			return nil, utilerrors.NewNotFound("Preset", "preset was not found.")
		}

		if err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}

		_, err = presetProvider.DeletePreset(ctx, preset)

		return nil, err
	}
}

// deletePresetProviderReq represents a request to delete preset provider
// swagger:parameters deletePresetProvider
type deletePresetProviderReq struct {
	// in: path
	// required: true
	PresetName string `json:"preset_name"`
	// in: path
	// required: true
	ProviderName string `json:"provider_name,omitempty"`
}

func DecodeDeletePresetProvider(_ context.Context, r *http.Request) (interface{}, error) {
	var req deletePresetProviderReq

	req.PresetName = mux.Vars(r)["preset_name"]
	req.ProviderName = mux.Vars(r)["provider_name"]

	return req, nil
}

// Validate validates deletePresetProviderReq request.
func (r deletePresetProviderReq) Validate() error {
	if len(r.PresetName) == 0 {
		return fmt.Errorf("the preset name cannot be empty")
	}

	if len(r.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	if !kubermaticv1.IsProviderSupported(r.ProviderName) {
		return fmt.Errorf("invalid provider name %s", r.ProviderName)
	}

	return nil
}

func DeletePresetProvider(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deletePresetProviderReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, "only admins can delete preset providers")
		}

		preset, err := presetProvider.GetPreset(ctx, userInfo, req.PresetName)
		if apierrors.IsNotFound(err) {
			return nil, utilerrors.NewNotFound("Preset", "preset was not found.")
		}

		if err != nil && !apierrors.IsNotFound(err) {
			return nil, utilerrors.New(http.StatusInternalServerError, err.Error())
		}

		providerName := kubermaticv1.ProviderType(req.ProviderName)
		if hasProvider, _ := kubermaticv1helper.HasProvider(preset, providerName); !hasProvider {
			return nil, utilerrors.NewNotFound("Preset", fmt.Sprintf("preset %s does not contain %s provider", req.PresetName, req.ProviderName))
		}

		preset = kubermaticv1helper.RemoveProvider(preset, providerName)
		_, err = presetProvider.UpdatePreset(ctx, preset)

		return preset, err
	}
}

// deleteProviderPresetReq represents a request to delete a preset or one of its providers
// swagger:parameters deleteProviderPreset
type deleteProviderPresetReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: path
	// required: true
	PresetName string `json:"preset_name"`
}

// Validate validates deleteProviderPresetReq request.
func (r deleteProviderPresetReq) Validate() error {
	if len(r.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	if !kubermaticv1.IsProviderSupported(r.ProviderName) {
		return fmt.Errorf("invalid provider name %s", r.ProviderName)
	}

	if len(r.PresetName) == 0 {
		return fmt.Errorf("preset name cannot be empty")
	}
	return nil
}

func DecodeDeleteProviderPreset(_ context.Context, r *http.Request) (interface{}, error) {
	var req deleteProviderPresetReq

	req.ProviderName = mux.Vars(r)["provider_name"]
	req.PresetName = mux.Vars(r)["preset_name"]

	return req, nil
}

// DeleteProviderPreset deletes the given provider from the preset AND if there is only one provider left, the preset gets deleted.
// Deprecated: This function has been deprecated; use DeletePreset or DeletePresetProvider.
func DeleteProviderPreset(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteProviderPresetReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if !userInfo.IsAdmin {
			return "", utilerrors.New(http.StatusForbidden, "only admins can delete presets")
		}

		preset, err := presetProvider.GetPreset(ctx, userInfo, req.PresetName)
		if apierrors.IsNotFound(err) {
			return nil, utilerrors.NewBadRequest("preset was not found.")
		}

		if err != nil && !apierrors.IsNotFound(err) {
			return nil, err
		}

		// remove provider from preset
		preset = kubermaticv1helper.RemoveProvider(preset, kubermaticv1.ProviderType(req.ProviderName))

		existingProviders := kubermaticv1helper.GetProviderList(preset)
		if len(existingProviders) > 0 {
			// Case: Remove provider from the preset
			preset, err = presetProvider.UpdatePreset(ctx, preset)
			if err != nil {
				return nil, err
			}
		} else {
			preset, err = presetProvider.DeletePreset(ctx, preset)
			if err != nil {
				return nil, err
			}
		}

		enabled := preset.Spec.IsEnabled()
		return newAPIPreset(preset, enabled), nil
	}
}

// getPresetSatsReq represents a request to get preset stats
// swagger:parameters getPresetStats
type getPresetSatsReq struct {
	// in: path
	// required: true
	PresetName string `json:"preset_name"`
}

// Validate validates getPresetSatsReq request.
func (r getPresetSatsReq) Validate() error {
	if len(r.PresetName) == 0 {
		return fmt.Errorf("preset name cannot be empty")
	}
	return nil
}

func DecodeGetPresetStats(_ context.Context, r *http.Request) (interface{}, error) {
	var req getPresetSatsReq

	req.PresetName = mux.Vars(r)["preset_name"]

	return req, nil
}

// GetPresetStats gets preset stats.
func GetPresetStats(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter, clusterProviderGetter provider.ClusterProviderGetter, seedsGetter provider.SeedsGetter, clusterTemplateProvider provider.ClusterTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(getPresetSatsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		preset, err := presetProvider.GetPreset(ctx, userInfo, req.PresetName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, utilerrors.NewBadRequest("preset was not found.")
			}
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		stats := apiv2.PresetStats{}

		seeds, err := seedsGetter()
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		presetLabelRequirement, err := labels.NewRequirement(kubermaticv1.IsCredentialPresetLabelKey, selection.Equals, []string{"true"})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		for datacenter, seed := range seeds {
			clusterProvider, err := clusterProviderGetter(seed)
			if err != nil {
				return nil, utilerrors.NewNotFound("cluster-provider", datacenter)
			}
			clusters, err := clusterProvider.ListAll(ctx, labels.NewSelector().Add(*presetLabelRequirement))
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
			for _, cluster := range clusters.Items {
				if cluster.Annotations != nil {
					if presetName := cluster.Annotations[kubermaticv1.PresetNameAnnotation]; presetName == preset.Name {
						stats.AssociatedClusters++
					}
				}
			}
		}

		templates, err := clusterTemplateProvider.ListALL(ctx, labels.NewSelector().Add(*presetLabelRequirement))
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		for _, template := range templates {
			if template.Annotations != nil {
				if presetName := template.Annotations[kubermaticv1.PresetNameAnnotation]; presetName == preset.Name {
					stats.AssociatedClusterTemplates++
				}
			}
		}

		return stats, err
	}
}

func mergePresets(oldPreset *kubermaticv1.Preset, newPreset *kubermaticv1.Preset, providerType kubermaticv1.ProviderType) *kubermaticv1.Preset {
	oldPreset = kubermaticv1helper.OverrideProvider(oldPreset, providerType, newPreset)
	oldPreset.Spec.RequiredEmails = newPreset.Spec.RequiredEmails
	return oldPreset
}

func newAPIPreset(preset *kubermaticv1.Preset, enabled bool) apiv2.Preset {
	providers := make([]apiv2.PresetProvider, 0)
	for _, providerType := range kubermaticv1.SupportedProviders {
		if hasProvider, _ := kubermaticv1helper.HasProvider(preset, providerType); hasProvider {
			providers = append(providers, apiv2.PresetProvider{
				Name:    providerType,
				Enabled: kubermaticv1helper.IsProviderEnabled(preset, providerType),
			})
		}
	}

	return apiv2.Preset{Name: preset.Name, Enabled: enabled, Providers: providers}
}

func convertAPIToInternalPreset(preset apiv2.PresetBody) *kubermaticv1.Preset {
	return &kubermaticv1.Preset{
		ObjectMeta: metav1.ObjectMeta{
			Name: preset.Name,
		},
		Spec: preset.Spec,
	}
}
