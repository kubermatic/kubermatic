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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	clusterv2 "k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	kubermaticprovider "k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// getNetworkDefaultsReq represents a request for retrieving the cluster networking defaults for the given provider.
// swagger:parameters getNetworkDefaults
type getNetworkDefaultsReq struct {
	// in: path
	// required: true
	ProviderName string `json:"provider_name"`
	// in: path
	// required: true
	DC string `json:"dc"`

	// private field for the seed name. Needed for the cluster provider.
	seedName string
}

// GetSeedCluster returns the SeedCluster object.
func (req getNetworkDefaultsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.seedName,
	}
}

// Validate validates getNetworkDefaultsReq request.
func (r getNetworkDefaultsReq) Validate() error {
	if r.ProviderName == "" {
		return fmt.Errorf("the provider name cannot be empty")
	}
	if !kubermaticv1.IsProviderSupported(r.ProviderName) {
		return fmt.Errorf("unsupported provider: %q", r.ProviderName)
	}
	if r.DC == "" {
		return fmt.Errorf("the datacenter cannot be empty")
	}
	return nil
}

func DecodeGetNetworkDefaultsReq(ctx context.Context, r *http.Request) (interface{}, error) {
	req := getNetworkDefaultsReq{
		ProviderName: mux.Vars(r)["provider_name"],
		DC:           mux.Vars(r)["dc"],
	}

	seedName, err := clusterv2.FindSeedNameForDatacenter(ctx, req.DC)
	if err != nil {
		return nil, err
	}
	req.seedName = seedName

	return req, nil
}

// GetNetworkDefaultsEndpoint returns the cluster networking defaults for the given provider.
func GetNetworkDefaultsEndpoint(
	seedsGetter kubermaticprovider.SeedsGetter,
	userInfoGetter kubermaticprovider.UserInfoGetter,
) endpoint.Endpoint {
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
		datacenter := req.DC

		networkDefaults := apiv2.NetworkDefaults{
			IPv4: &apiv2.NetworkDefaultsIPFamily{
				PodsCIDR:                resources.GetDefaultPodCIDRIPv4(provider),
				ServicesCIDR:            resources.GetDefaultServicesCIDRIPv4(provider),
				NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv4,
				NodePortsAllowedIPRange: resources.IPv4MatchAnyCIDR,
			},
			IPv6: &apiv2.NetworkDefaultsIPFamily{
				PodsCIDR:                resources.DefaultClusterPodsCIDRIPv6,
				ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv6,
				NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv6,
				NodePortsAllowedIPRange: resources.IPv6MatchAnyCIDR,
			},
			ProxyMode:                resources.GetDefaultProxyMode(provider),
			NodeLocalDNSCacheEnabled: resources.DefaultNodeLocalDNSCacheEnabled,
		}

		// Fetching the defaulting ClusterTemplate.

		privilegedClusterProvider := ctx.Value(middleware.PrivilegedClusterProviderContextKey).(kubermaticprovider.PrivilegedClusterProvider)
		seedClient := privilegedClusterProvider.GetSeedClusterAdminRuntimeClient()

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		seed, _, err := kubermaticprovider.DatacenterFromSeedMap(adminUserInfo, seedsGetter, datacenter)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		defaultingTemplate, err := defaulting.GetDefaultingClusterTemplate(ctx, seedClient, seed)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Using network defaults from the template defaults when it's available
		if defaultingTemplate != nil {
			networkDefaults = overrideNetworkDefaultsByDefaultingTemplate(networkDefaults, defaultingTemplate.Spec.ClusterNetwork, provider)
		}

		return networkDefaults, nil
	}
}

func overrideNetworkDefaultsByDefaultingTemplate(networkDefaults apiv2.NetworkDefaults, templateClusterNetwork kubermaticv1.ClusterNetworkingConfig, provider kubermaticv1.ProviderType) apiv2.NetworkDefaults {
	defaultClusterNetwork := defaulting.DefaultClusterNetwork(templateClusterNetwork, provider)

	if defaultClusterNetwork.ProxyMode != "" {
		networkDefaults.ProxyMode = defaultClusterNetwork.ProxyMode
	}

	if defaultClusterNetwork.NodeLocalDNSCacheEnabled != nil {
		networkDefaults.NodeLocalDNSCacheEnabled = *defaultClusterNetwork.NodeLocalDNSCacheEnabled
	}

	// IPv4

	podsIPv4CIDR := defaultClusterNetwork.Pods.GetIPv4CIDR()
	if podsIPv4CIDR != "" {
		networkDefaults.IPv4.PodsCIDR = podsIPv4CIDR
	}

	servicesIPv4CIDR := defaultClusterNetwork.Services.GetIPv4CIDR()
	if servicesIPv4CIDR != "" {
		networkDefaults.IPv4.ServicesCIDR = servicesIPv4CIDR
	}

	if defaultClusterNetwork.NodeCIDRMaskSizeIPv4 != nil {
		networkDefaults.IPv4.NodeCIDRMaskSize = *defaultClusterNetwork.NodeCIDRMaskSizeIPv4
	}

	// IPv6

	podsIPv6CIDR := defaultClusterNetwork.Pods.GetIPv6CIDR()
	if podsIPv6CIDR != "" {
		networkDefaults.IPv6.PodsCIDR = podsIPv6CIDR
	}

	servicesIPv6CIDR := defaultClusterNetwork.Services.GetIPv6CIDR()
	if servicesIPv6CIDR != "" {
		networkDefaults.IPv6.ServicesCIDR = servicesIPv6CIDR
	}

	if defaultClusterNetwork.NodeCIDRMaskSizeIPv6 != nil {
		networkDefaults.IPv6.NodeCIDRMaskSize = *defaultClusterNetwork.NodeCIDRMaskSizeIPv6
	}

	return networkDefaults
}
