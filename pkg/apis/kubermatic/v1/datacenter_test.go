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

package v1

import (
	"testing"

	"github.com/go-test/deep"
)

func TestSetSeedDefaults(t *testing.T) {
	testCases := []struct {
		name     string
		seed     *Seed
		expected map[string]Datacenter
	}{
		{
			name: "DC settings are being respected",
			seed: &Seed{
				Spec: SeedSpec{
					ProxySettings: &ProxySettings{
						HTTPProxy: NewProxyValue("seed-proxy"),
						NoProxy:   NewProxyValue("seed-no-proxy"),
					},
					Datacenters: map[string]Datacenter{
						"a": {Node: &NodeSettings{ProxySettings: ProxySettings{
							HTTPProxy: NewProxyValue("dc-proxy"),
							NoProxy:   NewProxyValue("dc-no-proxy"),
						}}},
						"b": {Node: &NodeSettings{ProxySettings: ProxySettings{
							HTTPProxy: NewProxyValue("dc-proxy"),
							NoProxy:   NewProxyValue("dc-no-proxy"),
						}}},
					},
				},
			},
			expected: map[string]Datacenter{
				"a": {Node: &NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: NewProxyValue("dc-proxy"),
					NoProxy:   NewProxyValue("dc-no-proxy"),
				}}},
				"b": {Node: &NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: NewProxyValue("dc-proxy"),
					NoProxy:   NewProxyValue("dc-no-proxy"),
				}}},
			},
		},
		{
			name: "DC settings are being set",
			seed: &Seed{
				Spec: SeedSpec{
					ProxySettings: &ProxySettings{
						HTTPProxy: NewProxyValue("seed-proxy"),
						NoProxy:   NewProxyValue("seed-no-proxy"),
					},
					Datacenters: map[string]Datacenter{
						"a": {},
						"b": {},
					},
				},
			},
			expected: map[string]Datacenter{
				"a": {Node: &NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: NewProxyValue("seed-proxy"),
					NoProxy:   NewProxyValue("seed-no-proxy"),
				}}},
				"b": {Node: &NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: NewProxyValue("seed-proxy"),
					NoProxy:   NewProxyValue("seed-no-proxy"),
				}}},
			},
		},
		{
			name: "Only http_proxy is set",
			seed: &Seed{
				Spec: SeedSpec{
					ProxySettings: &ProxySettings{
						HTTPProxy: NewProxyValue("seed-proxy"),
						NoProxy:   NewProxyValue("seed-no-proxy"),
					},
					Datacenters: map[string]Datacenter{
						"a": {Node: &NodeSettings{ProxySettings: ProxySettings{
							NoProxy: NewProxyValue("dc-no-proxy"),
						}}},
						"b": {Node: &NodeSettings{ProxySettings: ProxySettings{
							NoProxy: NewProxyValue("dc-no-proxy"),
						}}},
					},
				},
			},
			expected: map[string]Datacenter{
				"a": {Node: &NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: NewProxyValue("seed-proxy"),
					NoProxy:   NewProxyValue("dc-no-proxy"),
				}}},
				"b": {Node: &NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: NewProxyValue("seed-proxy"),
					NoProxy:   NewProxyValue("dc-no-proxy"),
				}}},
			},
		},
		{
			name: "Only no_proxy is set",
			seed: &Seed{
				Spec: SeedSpec{
					ProxySettings: &ProxySettings{
						HTTPProxy: NewProxyValue("seed-proxy"),
						NoProxy:   NewProxyValue("seed-no-proxy"),
					},
					Datacenters: map[string]Datacenter{
						"a": {Node: &NodeSettings{ProxySettings: ProxySettings{
							HTTPProxy: NewProxyValue("dc-proxy"),
						}}},
						"b": {Node: &NodeSettings{ProxySettings: ProxySettings{
							HTTPProxy: NewProxyValue("dc-proxy"),
						}}},
					},
				},
			},
			expected: map[string]Datacenter{
				"a": {Node: &NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: NewProxyValue("dc-proxy"),
					NoProxy:   NewProxyValue("seed-no-proxy"),
				}}},
				"b": {Node: &NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: NewProxyValue("dc-proxy"),
					NoProxy:   NewProxyValue("seed-no-proxy"),
				}}},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.seed.SetDefaults()
			if diff := deep.Equal(tc.seed.Spec.Datacenters, tc.expected); diff != nil {
				t.Errorf("seed.Spec.Datacenter differs from expected, diff: %v", diff)
			}
		})
	}
}
