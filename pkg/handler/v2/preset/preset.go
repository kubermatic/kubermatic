package preset

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	crdapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// listPresetsReq represents a request for a list of presets
// swagger:parameters listPresets
type listPresetsReq struct {
	// in: query
	Disabled bool `json:"disabled,omitempty"`
}

func DecodeListPresets(_ context.Context, r *http.Request) (interface{}, error) {
	return listPresetsReq{
		Disabled: r.URL.Query().Get("disabled") == "true",
	}, nil
}

// ListProviderPresets returns a list of preset names for the provider
func ListPresets(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listPresetsReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		credentialList := apiv1.CredentialList{Names: make([]string, 0)}
		presets, err := presetsProvider.GetPresets(userInfo)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		for _, preset := range presets {
			if !preset.Spec.IsEnabled() && !req.Disabled {
				continue
			}

			credentialList.Names = append(credentialList.Names, preset.Name)
		}

		return credentialList, nil
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
	Body struct {Enabled bool}
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

// Validate validates updatePresetStatusReq request
func (r updatePresetStatusReq) Validate() error {
	if len(r.PresetName) == 0 {
		return fmt.Errorf("the preset name cannot be empty")
	}

	if len(r.Provider) > 0 && !crdapiv1.IsProviderSupported(r.Provider) {
		return fmt.Errorf("invalid provider name %s", r.Provider)
	}

	return nil
}

// UpdatePresetStatus updates the status of a preset. It can enable or disable it, so that it won't be listed by the list endpoints.
func UpdatePresetStatus(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updatePresetStatusReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		preset, err := presetsProvider.GetPreset(userInfo, req.PresetName)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		if len(req.Provider) == 0 {
			preset.Spec.SetPresetStatus(req.Body.Enabled)
			_, err := presetsProvider.UpdatePreset(userInfo, preset)
			return nil, err
		}

		if hasProvider, _ := preset.Spec.HasProvider(crdapiv1.ProviderType(req.Provider)); !hasProvider {
			return nil, fmt.Errorf("missing provider configuration for: %s", req.Provider)
		}

		preset.Spec.SetPresetProviderStatus(crdapiv1.ProviderType(req.Provider), req.Body.Enabled)
		_, err = presetsProvider.UpdatePreset(userInfo, preset)
		return nil, err
	}
}

// listProviderPresetsReq represents a request for a list of presets
// swagger:parameters listPresets
type listProviderPresetsReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: query
	Datacenter string `json:"datacenter,omitempty"`
}

func (l listProviderPresetsReq) matchesDatacenter(datacenter string) bool {
	return len(l.Datacenter) == 0 || strings.EqualFold(l.Datacenter, datacenter)
}

// Validate validates listProviderPresetsReq request
func (r listProviderPresetsReq) Validate() error {
	if len(r.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	if !crdapiv1.IsProviderSupported(r.ProviderName) {
		return fmt.Errorf("invalid provider name %s", r.ProviderName)
	}

	return nil
}

func DecodeListProviderPresets(_ context.Context, r *http.Request) (interface{}, error) {
	return listProviderPresetsReq{
		ProviderName: mux.Vars(r)["provider_name"],
		Datacenter:   r.URL.Query().Get("datacenter"),
	}, nil
}

// ListProviderPresets returns a list of preset names for the provider
func ListProviderPresets(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listProviderPresetsReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		credentialList := apiv1.CredentialList{Names: make([]string, 0)}
		presets, err := presetsProvider.GetPresets(userInfo)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		for _, preset := range presets {
			providerType := crdapiv1.ProviderType(req.ProviderName)
			presetProvider := preset.Spec.GetPresetProvider(providerType)

			if !preset.Spec.IsEnabled() ||
				presetProvider == nil ||
				!presetProvider.IsEnabled() ||
				!req.matchesDatacenter(presetProvider.Datacenter) {
				continue
			}

			credentialList.Names = append(credentialList.Names, preset.Name)
		}

		return credentialList, nil
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
	Body crdapiv1.Preset
}

// Validate validates createPresetReq request
func (r createPresetReq) Validate() error {
	if len(r.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	if !crdapiv1.IsProviderSupported(r.ProviderName) {
		return fmt.Errorf("invalid provider name %s", r.ProviderName)
	}

	if len(r.Body.Name) == 0 {
		return fmt.Errorf("preset name cannot be empty")
	}

	if hasProvider, _ := r.Body.Spec.HasProvider(crdapiv1.ProviderType(r.ProviderName)); !hasProvider {
		return fmt.Errorf("missing provider configuration for: %s", r.ProviderName)
	}

	err := r.Body.Spec.Validate(crdapiv1.ProviderType(r.ProviderName))
	if err != nil {
		return err
	}

	for _, providerType := range crdapiv1.SupportedProviders() {
		if string(providerType) == r.ProviderName {
			continue
		}

		if hasProvider, _ := r.Body.Spec.HasProvider(providerType); hasProvider {
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

// CreatePreset creates a preset for the selected provider and returns the name if successful, error otherwise
func CreatePreset(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createPresetReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		preset, err := presetsProvider.GetPreset(userInfo, req.Body.Name)
		if k8serrors.IsNotFound(err) {
			return presetsProvider.CreatePreset(userInfo, &req.Body)
		}

		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, err
		}

		if hasProvider, _ := preset.Spec.HasProvider(crdapiv1.ProviderType(req.ProviderName)); hasProvider {
			return nil, fmt.Errorf("preset with name %s already exists for the provider %s", preset.Name, req.ProviderName)
		}

		preset = mergePresets(preset, &req.Body, crdapiv1.ProviderType(req.ProviderName))
		return presetsProvider.UpdatePreset(userInfo, preset)
	}
}

// updatePresetReq represents a request to update a preset
// swagger:parameters updatePreset
type updatePresetReq struct {
	createPresetReq
}

// Validate validates updatePresetReq request
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

// UpdatePreset updates a preset for the selected provider and returns the name if successful, error otherwise
func UpdatePreset(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updatePresetReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		preset, err := presetsProvider.GetPreset(userInfo, req.Body.Name)
		if err != nil {
			return nil, err
		}

		preset = mergePresets(preset, &req.Body, crdapiv1.ProviderType(req.ProviderName))
		return presetsProvider.UpdatePreset(userInfo, preset)
	}
}

func mergePresets(oldPreset *crdapiv1.Preset, newPreset *crdapiv1.Preset, providerType crdapiv1.ProviderType) *crdapiv1.Preset {
	oldPreset.Spec.OverrideProvider(providerType, &newPreset.Spec)
	return oldPreset
}
