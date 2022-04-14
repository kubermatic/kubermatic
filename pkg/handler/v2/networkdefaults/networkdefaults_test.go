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

package networkdefaults_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/resources"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetEndpoint(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Name                      string
		Provider                  kubermaticv1.ProviderType
		CNI                       kubermaticv1.CNIPluginType
		ExistingKubermaticObjects []ctrlruntimeclient.Object
		ExistingAPIUser           *apiv1.User
		ExpectedResponse          *apiv2.NetworkDefaults
		ExpectedHTTPStatusCode    int
	}{
		{
			Name:                      "AWS + Canal network settings",
			Provider:                  kubermaticv1.AWSCloudProvider,
			CNI:                       kubermaticv1.CNIPluginTypeCanal,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode:    http.StatusOK,
			ExpectedResponse: &apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                resources.DefaultClusterPodsCIDRIPv4,
					ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv4,
					NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv4,
					NodePortsAllowedIPRange: resources.IPv4MatchAnyCIDR,
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                resources.DefaultClusterPodsCIDRIPv6,
					ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv6,
					NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv6,
					NodePortsAllowedIPRange: resources.IPv6MatchAnyCIDR,
				},
				ProxyMode:                resources.IPVSProxyMode,
				NodeLocalDNSCacheEnabled: true,
			},
		},
		{
			Name:                      "AWS + Cilium network settings",
			Provider:                  kubermaticv1.AWSCloudProvider,
			CNI:                       kubermaticv1.CNIPluginTypeCilium,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode:    http.StatusOK,
			ExpectedResponse: &apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                resources.DefaultClusterPodsCIDRIPv4,
					ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv4,
					NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv4,
					NodePortsAllowedIPRange: resources.IPv4MatchAnyCIDR,
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                resources.DefaultClusterPodsCIDRIPv6,
					ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv6,
					NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv6,
					NodePortsAllowedIPRange: resources.IPv6MatchAnyCIDR,
				},
				ProxyMode:                resources.IPVSProxyMode,
				NodeLocalDNSCacheEnabled: true,
			},
		},
		{
			Name:                      "Kubevirt + Canal network settings",
			Provider:                  kubermaticv1.KubevirtCloudProvider,
			CNI:                       kubermaticv1.CNIPluginTypeCanal,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode:    http.StatusOK,
			ExpectedResponse: &apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                resources.DefaultClusterPodsCIDRIPv4KubeVirt,
					ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv4KubeVirt,
					NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv4,
					NodePortsAllowedIPRange: resources.IPv4MatchAnyCIDR,
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                resources.DefaultClusterPodsCIDRIPv6,
					ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv6,
					NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv6,
					NodePortsAllowedIPRange: resources.IPv6MatchAnyCIDR,
				},
				ProxyMode:                resources.IPVSProxyMode,
				NodeLocalDNSCacheEnabled: true,
			},
		},
		{
			Name:                      "Hetzner + Canal network settings",
			Provider:                  kubermaticv1.HetznerCloudProvider,
			CNI:                       kubermaticv1.CNIPluginTypeCanal,
			ExistingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			ExistingAPIUser:           test.GenDefaultAPIUser(),
			ExpectedHTTPStatusCode:    http.StatusOK,
			ExpectedResponse: &apiv2.NetworkDefaults{
				IPv4: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                resources.DefaultClusterPodsCIDRIPv4,
					ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv4,
					NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv4,
					NodePortsAllowedIPRange: resources.IPv4MatchAnyCIDR,
				},
				IPv6: &apiv2.NetworkDefaultsIPFamily{
					PodsCIDR:                resources.DefaultClusterPodsCIDRIPv6,
					ServicesCIDR:            resources.DefaultClusterServicesCIDRIPv6,
					NodeCIDRMaskSize:        resources.DefaultNodeCIDRMaskSizeIPv6,
					NodePortsAllowedIPRange: resources.IPv6MatchAnyCIDR,
				},
				ProxyMode:                resources.IPTablesProxyMode,
				NodeLocalDNSCacheEnabled: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/providers/%s/cni/%s/networkdefaults", tc.Provider, tc.CNI)
			req := httptest.NewRequest(http.MethodGet, requestURL, nil)
			resp := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, nil, tc.ExistingKubermaticObjects, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint: %v", err)
			}
			ep.ServeHTTP(resp, req)

			if resp.Code != tc.ExpectedHTTPStatusCode {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.ExpectedHTTPStatusCode, resp.Code, resp.Body.String())
			}
			if resp.Code == http.StatusOK {
				b, err := json.Marshal(tc.ExpectedResponse)
				if err != nil {
					t.Fatalf("failed to marshal expected response: %v", err)
				}
				test.CompareWithResult(t, resp, string(b))
			}
		})
	}
}
