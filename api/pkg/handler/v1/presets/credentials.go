package presets

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// providerNames holds a list of providers. They must stay in this order.
var providerNames = []string{
	"digitalocean",
	"hetzner",
	"azure",
	"vsphere",
	"aws",
	"openstack",
	"packet",
	"gcp",
	"kubevirt",
}

// providerReq represents a request for provider name
// swagger:parameters listCredentials
type providerReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
}

// CredentialEndpoint returns custom credential list name for the provider
func CredentialEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(providerReq)
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

		providerN := parseProvider(req.ProviderName)
		presets, err := presetsProvider.GetPresets(userInfo)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, err.Error())
		}

		for _, preset := range presets {
			// get specific provider by name from the Preset spec struct:
			// type PresetSpec struct {
			//	Digitalocean Digitalocean
			//	Hetzner      Hetzner
			//	Azure        Azure
			//	VSphere      VSphere
			//	AWS          AWS
			//	Openstack    Openstack
			//	Packet       Packet
			//	GCP          GCP
			//	Kubevirt     Kubevirt
			// }
			providersRaw := reflect.ValueOf(preset.Spec)
			if providersRaw.Kind() == reflect.Struct {
				providers := reflect.Indirect(providersRaw)
				providerItem := providers.Field(providerN)

				// append preset name if specific provider is not empty:
				if !providerItem.IsNil() {
					names = append(names, preset.Name)
				}
			}
		}

		credentials.Names = names
		return credentials, nil
	}
}

func parseProvider(p string) int {
	elementMap := make(map[string]int)
	for i, s := range providerNames {
		elementMap[s] = i
	}

	return elementMap[p]
}

func DecodeProviderReq(c context.Context, r *http.Request) (interface{}, error) {
	return providerReq{
		ProviderName: mux.Vars(r)["provider_name"],
	}, nil
}

// Validate validates providerReq request
func (r providerReq) Validate() error {
	if len(r.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	for _, existingProviders := range providerNames {
		if existingProviders == r.ProviderName {
			return nil
		}
	}
	return fmt.Errorf("invalid provider name %s", r.ProviderName)
}
