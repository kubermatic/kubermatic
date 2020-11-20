package preset

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: query
	Datacenter string `json:"datacenter,omitempty"`
}

// Validate validates listPresetsReq request
func (r listPresetsReq) Validate() error {
	if len(r.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	if !crdapiv1.IsProviderSupported(r.ProviderName) {
		return fmt.Errorf("invalid provider name %s", r.ProviderName)
	}

	return nil
}

func DecodeListPresets(_ context.Context, r *http.Request) (interface{}, error) {
	return listPresetsReq{
		ProviderName: mux.Vars(r)["provider_name"],
		Datacenter:   r.URL.Query().Get("datacenter"),
	}, nil
}

// ListPresets returns custom credential list name for the provider
func ListPresets(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(listPresetsReq)
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

		credentials := apiv1.CredentialList{}
		names := make([]string, 0)
		presets, err := presetsProvider.GetPresets(userInfo)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		for _, preset := range presets {
			providerType := crdapiv1.ProviderType(req.ProviderName)
			presetProvider := preset.Spec.GetPresetProvider(providerType)
			if presetProvider == nil || !preset.Spec.IsEnabled() || !presetProvider.IsEnabled() {
				continue
			}

			if presetProvider.Datacenter == req.Datacenter || presetProvider.Datacenter == "" {
				names = append(names, preset.Name)
			}
		}

		credentials.Names = names
		return credentials, nil
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
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, err
		}

		if k8serrors.IsNotFound(err) {
			return presetsProvider.CreatePreset(userInfo, &req.Body)
		}

		if hasProvider, _ := preset.Spec.HasProvider(crdapiv1.ProviderType(req.ProviderName)); hasProvider {
			return nil, fmt.Errorf("preset with name %s already exists for the provider %s", preset.Name, req.ProviderName)
		}

		return presetsProvider.UpdatePreset(userInfo, &req.Body)
	}
}

// updatePresetReq represents a request to update a preset
// swagger:parameters updatePreset
type updatePresetReq struct {
	createPresetReq
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

		_, err = presetsProvider.GetPreset(userInfo, req.Body.Name)
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, err
		}

		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("cannot update preset: preset %s does not exist", req.Body.Name)
		}

		return presetsProvider.UpdatePreset(userInfo, &req.Body)
	}
}
