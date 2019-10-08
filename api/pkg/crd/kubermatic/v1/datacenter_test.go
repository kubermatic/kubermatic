package v1

import (
	"testing"

	"github.com/go-test/deep"

	utilpointer "k8s.io/utils/pointer"
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
						HTTPProxy: utilpointer.StringPtr("seed-proxy"),
						NoProxy:   utilpointer.StringPtr("seed-no-proxy"),
					},
					Datacenters: map[string]Datacenter{
						"a": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
							HTTPProxy: utilpointer.StringPtr("dc-proxy"),
							NoProxy:   utilpointer.StringPtr("dc-no-proxy"),
						}}},
						"b": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
							HTTPProxy: utilpointer.StringPtr("dc-proxy"),
							NoProxy:   utilpointer.StringPtr("dc-no-proxy"),
						}}},
					},
				},
			},
			expected: map[string]Datacenter{
				"a": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: utilpointer.StringPtr("dc-proxy"),
					NoProxy:   utilpointer.StringPtr("dc-no-proxy"),
				}}},
				"b": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: utilpointer.StringPtr("dc-proxy"),
					NoProxy:   utilpointer.StringPtr("dc-no-proxy"),
				}}},
			},
		},
		{
			name: "DC settings are being set",
			seed: &Seed{
				Spec: SeedSpec{
					ProxySettings: &ProxySettings{
						HTTPProxy: utilpointer.StringPtr("seed-proxy"),
						NoProxy:   utilpointer.StringPtr("seed-no-proxy"),
					},
					Datacenters: map[string]Datacenter{
						"a": Datacenter{},
						"b": Datacenter{},
					},
				},
			},
			expected: map[string]Datacenter{
				"a": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: utilpointer.StringPtr("seed-proxy"),
					NoProxy:   utilpointer.StringPtr("seed-no-proxy"),
				}}},
				"b": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: utilpointer.StringPtr("seed-proxy"),
					NoProxy:   utilpointer.StringPtr("seed-no-proxy"),
				}}},
			},
		},
		{
			name: "Only http_proxy is set",
			seed: &Seed{
				Spec: SeedSpec{
					ProxySettings: &ProxySettings{
						HTTPProxy: utilpointer.StringPtr("seed-proxy"),
						NoProxy:   utilpointer.StringPtr("seed-no-proxy"),
					},
					Datacenters: map[string]Datacenter{
						"a": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
							NoProxy: utilpointer.StringPtr("dc-no-proxy"),
						}}},
						"b": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
							NoProxy: utilpointer.StringPtr("dc-no-proxy"),
						}}},
					},
				},
			},
			expected: map[string]Datacenter{
				"a": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: utilpointer.StringPtr("seed-proxy"),
					NoProxy:   utilpointer.StringPtr("dc-no-proxy"),
				}}},
				"b": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: utilpointer.StringPtr("seed-proxy"),
					NoProxy:   utilpointer.StringPtr("dc-no-proxy"),
				}}},
			},
		},
		{
			name: "Only no_proxy is set",
			seed: &Seed{
				Spec: SeedSpec{
					ProxySettings: &ProxySettings{
						HTTPProxy: utilpointer.StringPtr("seed-proxy"),
						NoProxy:   utilpointer.StringPtr("seed-no-proxy"),
					},
					Datacenters: map[string]Datacenter{
						"a": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
							HTTPProxy: utilpointer.StringPtr("dc-proxy"),
						}}},
						"b": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
							HTTPProxy: utilpointer.StringPtr("dc-proxy"),
						}}},
					},
				},
			},
			expected: map[string]Datacenter{
				"a": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: utilpointer.StringPtr("dc-proxy"),
					NoProxy:   utilpointer.StringPtr("seed-no-proxy"),
				}}},
				"b": Datacenter{Node: NodeSettings{ProxySettings: ProxySettings{
					HTTPProxy: utilpointer.StringPtr("dc-proxy"),
					NoProxy:   utilpointer.StringPtr("seed-no-proxy"),
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
