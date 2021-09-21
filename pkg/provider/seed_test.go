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

package provider

import (
	"context"
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSeedGetterFactorySetsDefaults(t *testing.T) {
	t.Parallel()
	initSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultSeedName,
			Namespace: "my-ns",
		},
		Spec: kubermaticv1.SeedSpec{
			ProxySettings: &kubermaticv1.ProxySettings{
				HTTPProxy: kubermaticv1.NewProxyValue("seed-proxy"),
			},
			Datacenters: map[string]kubermaticv1.Datacenter{"a": {}},
		},
	}

	client := fakectrlruntimeclient.
		NewClientBuilder().
		WithScheme(scheme.Scheme).
		WithObjects(initSeed).
		Build()

	seedGetter, err := SeedGetterFactory(context.Background(), client, DefaultSeedName, "my-ns")
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
	if nodeSettings.ProxySettings.HTTPProxy.String() != "seed-proxy" {
		t.Errorf("expected the datacenters http proxy setting to get set but was %v",
			nodeSettings.ProxySettings.HTTPProxy)
	}
}

func TestSeedsGetterFactorySetsDefaults(t *testing.T) {
	t.Parallel()
	initSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DefaultSeedName,
			Namespace: "my-ns",
		},
		Spec: kubermaticv1.SeedSpec{
			ProxySettings: &kubermaticv1.ProxySettings{
				HTTPProxy: kubermaticv1.NewProxyValue("seed-proxy"),
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
	if _, exists := seeds[DefaultSeedName]; !exists || len(seeds) != 1 {
		t.Fatalf("expceted to get a map with exactly one key `my-seed`, got %v", seeds)
	}

	seed := seeds[DefaultSeedName]
	nodeSettings := seed.Spec.Datacenters["a"].Node
	if nodeSettings == nil {
		t.Fatal("expected the datacenter's node setting to be set, but it's nil")
	}
	if nodeSettings.ProxySettings.HTTPProxy.String() != "seed-proxy" {
		t.Errorf("expected the datacenters http proxy setting to get set but was %v",
			nodeSettings.ProxySettings.HTTPProxy)
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
