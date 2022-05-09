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

package networkdefaults

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	v2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/version/cni"
)

// getNetworkDefaultsReq represents a request for retrieving the cluster networking defaults for the given provider.
// swagger:parameters getNetworkDefaults
type getNetworkDefaultsReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: path
	// required: true
	CNIPluginType string `json:"cni_plugin_type"`
}

// Validate validates getNetworkDefaultsReq request.
func (r getNetworkDefaultsReq) Validate() error {
	if r.ProviderName == "" {
		return fmt.Errorf("the provider name cannot be empty")
	}
	if !kubermaticv1.IsProviderSupported(r.ProviderName) {
		return fmt.Errorf("unsupported provider: %q", r.ProviderName)
	}
	if r.CNIPluginType == "" {
		return fmt.Errorf("CNI plugin type cannot be empty")
	}
	if !cni.GetSupportedCNIPlugins().Has(r.CNIPluginType) {
		return fmt.Errorf("CNI plugin type %q not supported. Supported types: %v", r.CNIPluginType, cni.GetSupportedCNIPlugins().List())
	}
	return nil
}

func DecodeGetNetworkDefaultsReq(ctx context.Context, r *http.Request) (interface{}, error) {
	return getNetworkDefaultsReq{
		ProviderName:  mux.Vars(r)["provider_name"],
		CNIPluginType: mux.Vars(r)["cni_plugin_type"],
	}, nil
}

// GetNetworkDefaultsEndpoint returns the cluster networking defaults for the given provider.
func GetNetworkDefaultsEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(getNetworkDefaultsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		provider := kubermaticv1.ProviderType(req.ProviderName)
		cni := kubermaticv1.CNIPluginType(req.CNIPluginType)

		return v2.NetworkDefaults{
			IPv4: &v2.NetworkDefaultsIPFamily{
				PodsCIDR:                resources.GetDefaultPodCIDRIPv4(provider),
				ServicesCIDR:            resources.GetDefaultServicesCIDRIPv4(provider),
				NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv4,
				NodePortsAllowedIPRange: resources.IPv4MatchAnyCIDR,
			},
			IPv6: &v2.NetworkDefaultsIPFamily{
				PodsCIDR:                resources.DefaultClusterPodsCIDRIPv6,
				ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv6,
				NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv6,
				NodePortsAllowedIPRange: resources.IPv6MatchAnyCIDR,
			},
			ProxyMode:                resources.GetDefaultProxyMode(provider, cni),
			NodeLocalDNSCacheEnabled: resources.DefaultNodeLocalDNSCacheEnabled,
		}, nil
	}
}
