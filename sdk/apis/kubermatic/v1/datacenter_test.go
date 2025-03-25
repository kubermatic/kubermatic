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
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEnsureThatProvidersAreSorted(t *testing.T) {
	stringValues := []string{}
	for _, provider := range SupportedProviders {
		stringValues = append(stringValues, string(provider))
	}

	sort.Strings(stringValues)

	for i, provider := range SupportedProviders {
		if string(provider) != stringValues[i] {
			t.Fatalf("The variable SupportedProviders is not sorted alphabetically. This will lead to all sorts of other tests failing.")
		}
	}
}

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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.seed.SetDefaults()
			if diff := cmp.Diff(tc.expected, tc.seed.Spec.Datacenters); diff != "" {
				t.Errorf("seed.Spec.Datacenter differs from expected:\n%v", diff)
			}
		})
	}
}
