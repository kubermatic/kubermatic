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

package presets

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
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
	"alibaba",
	"anexia",
}

// providerReq represents a request for provider name
// swagger:parameters listCredentials
type providerReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: query
	Datacenter string `json:"datacenter,omitempty"`
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
			//	Alibaba      Alibaba
			//  Anexia       Anexia
			// }
			providersRaw := reflect.ValueOf(preset.Spec)
			if providersRaw.Kind() == reflect.Struct {
				providers := reflect.Indirect(providersRaw)
				providerItem := providers.Field(providerN)

				// append preset name if specific provider is not empty:
				if !providerItem.IsNil() {
					var datacenterValue string
					item := reflect.Indirect(providerItem)
					datacenter := item.FieldByName("Datacenter")

					if datacenter.Kind() == reflect.String {
						datacenterValue = datacenter.String()
					}
					if datacenterValue == req.Datacenter || datacenterValue == "" {
						names = append(names, preset.Name)
					}
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
		Datacenter:   r.URL.Query().Get("datacenter"),
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
