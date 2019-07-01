package presets

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gorilla/mux"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// supportedProvidersWithNetwork holds a list of providers which have network field.
var supportedProvidersWithNetwork = []string{
	"azure",
	"vsphere",
	"aws",
	"openstack",
	"gcp",
}

// providerNetworkReq represents a request for supported provider name
// swagger:parameters getProviderNetwork
type providerNetworkReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
}

func NetworkEndpoint(credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(providerNetworkReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		network := apiv1.ProviderNetwork{}
		providerN := parseProvider(req.ProviderName)
		providers := reflect.ValueOf(credentialManager.GetPresets()).Elem()
		providerItems := providers.Field(providerN)
		networkItem := providerItems.FieldByName("Network")

		if networkItem.Kind() == reflect.Struct {
			v := reflect.Indirect(networkItem)
			if _, ok := v.Type().FieldByName("Name"); ok {
				rawName := v.FieldByName("Name").Interface()
				name := rawName.(string)
				network.Name = name
			}
		}

		return network, nil
	}
}

func DecodeProviderNetworkReq(c context.Context, r *http.Request) (interface{}, error) {
	return providerNetworkReq{
		ProviderName: mux.Vars(r)["provider_name"],
	}, nil
}

// Validate validates providerNetworkReq request
func (r providerNetworkReq) Validate() error {
	if len(r.ProviderName) == 0 {
		return fmt.Errorf("the provider name cannot be empty")
	}

	for _, existingProviders := range supportedProvidersWithNetwork {
		if existingProviders == r.ProviderName {
			return nil
		}
	}
	return fmt.Errorf("invalid provider name %s", r.ProviderName)
}
