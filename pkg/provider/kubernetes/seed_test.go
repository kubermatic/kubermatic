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

package kubernetes

import (
	"context"
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSeedGetterFactorySetsDefaults(t *testing.T) {
	t.Parallel()
	initSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      provider.DefaultSeedName,
			Namespace: "my-ns",
		},
		Spec: kubermaticv1.SeedSpec{
			ProxySettings: &kubermaticv1.ProxySettings{
				HTTPProxy: pointer.String("seed-proxy"),
			},
			Datacenters: map[string]kubermaticv1.Datacenter{"a": {}},
		},
	}

	client := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(initSeed).
		Build()

	seedGetter, err := SeedGetterFactory(context.Background(), client, provider.DefaultSeedName, "my-ns")
	if err != nil {
		t.Fatalf("failed getting seedGetter: %v", err)
	}
	seed, err := seedGetter()
	if err != nil {
		t.Fatalf("failed calling seedGetter: %v", err)
	}

	nodeSettings := seed.Spec.Datacenters["a"].Node
	if nodeSettings == nil {
		t.Fatal("expected the datacenter's node setting to be set, but it's nil")
	}
	if val := pointer.StringDeref(nodeSettings.ProxySettings.HTTPProxy, ""); val != "seed-proxy" {
		t.Errorf("expected the datacenters http proxy setting to get set but was %v", val)
	}
}

func TestSeedsGetterFactorySetsDefaults(t *testing.T) {
	t.Parallel()
	initSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      provider.DefaultSeedName,
			Namespace: "my-ns",
		},
		Spec: kubermaticv1.SeedSpec{
			ProxySettings: &kubermaticv1.ProxySettings{
				HTTPProxy: pointer.String("seed-proxy"),
			},
			Datacenters: map[string]kubermaticv1.Datacenter{"a": {}},
		},
	}
	client := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(initSeed).
		Build()

	seedsGetter, err := SeedsGetterFactory(context.Background(), client, "my-ns")
	if err != nil {
		t.Fatalf("failed getting seedsGetter: %v", err)
	}
	seeds, err := seedsGetter()
	if err != nil {
		t.Fatalf("failed calling seedsGetter: %v", err)
	}
	if _, exists := seeds[provider.DefaultSeedName]; !exists || len(seeds) != 1 {
		t.Fatalf("expected to get a map with exactly one key `my-seed`, got %v", seeds)
	}

	seed := seeds[provider.DefaultSeedName]
	nodeSettings := seed.Spec.Datacenters["a"].Node
	if nodeSettings == nil {
		t.Fatal("expected the datacenter's node setting to be set, but it's nil")
	}
	if val := pointer.StringDeref(nodeSettings.ProxySettings.HTTPProxy, ""); val != "seed-proxy" {
		t.Errorf("expected the datacenters http proxy setting to get set but was %v", val)
	}
}

func TestSeedsGetterFactoryNoSeed(t *testing.T) {
	t.Parallel()
	// No seed is returned by the fake client
	client := fakectrlruntimeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	seedsGetter, err := SeedsGetterFactory(context.Background(), client, "my-ns")
	if err != nil {
		t.Fatalf("failed getting seedsGetter: %v", err)
	}
	seeds, err := seedsGetter()
	if err != nil {
		t.Fatalf("error occurred while calling seedsGetter: %v", err)
	}
	if !reflect.DeepEqual(seeds, emptySeedMap) {
		t.Errorf("Expected no seed, but got %d: %v", len(seeds), seeds)
	}
}

func TestApplySeedProxyToDatacenters(t *testing.T) {
	testCases := []struct {
		name     string
		seed     *kubermaticv1.Seed
		expected map[string]kubermaticv1.Datacenter
	}{
		{
			name: "DC settings are being respected",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					ProxySettings: &kubermaticv1.ProxySettings{
						HTTPProxy: pointer.String("seed-proxy"),
						NoProxy:   pointer.String("seed-no-proxy"),
					},
					Datacenters: map[string]kubermaticv1.Datacenter{
						"a": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
							HTTPProxy: pointer.String("dc-proxy"),
							NoProxy:   pointer.String("dc-no-proxy"),
						}}},
						"b": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
							HTTPProxy: pointer.String("dc-proxy"),
							NoProxy:   pointer.String("dc-no-proxy"),
						}}},
					},
				},
			},
			expected: map[string]kubermaticv1.Datacenter{
				"a": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: pointer.String("dc-proxy"),
					NoProxy:   pointer.String("dc-no-proxy"),
				}}},
				"b": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: pointer.String("dc-proxy"),
					NoProxy:   pointer.String("dc-no-proxy"),
				}}},
			},
		},
		{
			name: "DC settings are being set",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					ProxySettings: &kubermaticv1.ProxySettings{
						HTTPProxy: pointer.String("seed-proxy"),
						NoProxy:   pointer.String("seed-no-proxy"),
					},
					Datacenters: map[string]kubermaticv1.Datacenter{
						"a": {},
						"b": {},
					},
				},
			},
			expected: map[string]kubermaticv1.Datacenter{
				"a": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: pointer.String("seed-proxy"),
					NoProxy:   pointer.String("seed-no-proxy"),
				}}},
				"b": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: pointer.String("seed-proxy"),
					NoProxy:   pointer.String("seed-no-proxy"),
				}}},
			},
		},
		{
			name: "Only http_proxy is set",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					ProxySettings: &kubermaticv1.ProxySettings{
						HTTPProxy: pointer.String("seed-proxy"),
						NoProxy:   pointer.String("seed-no-proxy"),
					},
					Datacenters: map[string]kubermaticv1.Datacenter{
						"a": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
							NoProxy: pointer.String("dc-no-proxy"),
						}}},
						"b": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
							NoProxy: pointer.String("dc-no-proxy"),
						}}},
					},
				},
			},
			expected: map[string]kubermaticv1.Datacenter{
				"a": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: pointer.String("seed-proxy"),
					NoProxy:   pointer.String("dc-no-proxy"),
				}}},
				"b": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: pointer.String("seed-proxy"),
					NoProxy:   pointer.String("dc-no-proxy"),
				}}},
			},
		},
		{
			name: "Only no_proxy is set",
			seed: &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					ProxySettings: &kubermaticv1.ProxySettings{
						HTTPProxy: pointer.String("seed-proxy"),
						NoProxy:   pointer.String("seed-no-proxy"),
					},
					Datacenters: map[string]kubermaticv1.Datacenter{
						"a": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
							HTTPProxy: pointer.String("dc-proxy"),
						}}},
						"b": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
							HTTPProxy: pointer.String("dc-proxy"),
						}}},
					},
				},
			},
			expected: map[string]kubermaticv1.Datacenter{
				"a": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: pointer.String("dc-proxy"),
					NoProxy:   pointer.String("seed-no-proxy"),
				}}},
				"b": {Node: &kubermaticv1.NodeSettings{ProxySettings: kubermaticv1.ProxySettings{
					HTTPProxy: pointer.String("dc-proxy"),
					NoProxy:   pointer.String("seed-no-proxy"),
				}}},
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ApplySeedProxyToDatacenters(tc.seed)
			if diff := diff.ObjectDiff(tc.expected, tc.seed.Spec.Datacenters); diff != "" {
				t.Errorf("seed.Spec.Datacenter differs from expected:\n%v", diff)
			}
		})
	}
}
