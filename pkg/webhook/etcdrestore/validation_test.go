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

package etcdrestore

import (
	"context"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlertest "k8c.io/kubermatic/v2/pkg/handler/test"
	helpertest "k8c.io/kubermatic/v2/pkg/test"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = kubermaticv1.AddToScheme(testScheme)
}

func TestEtcdLauncherFeatureGate(t *testing.T) {
	seed := kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				"datacenterName": {
					Spec: kubermaticv1.DatacenterSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{},
					},
				},
			},
		},
	}

	seedGetter := helpertest.NewSeedGetter(&seed)

	makeCluster := func(enableEtcdLauncher bool) *kubermaticv1.Cluster {
		c := handlertest.GenDefaultCluster()
		c.Name = fmt.Sprintf("%s-%t", c.Name, enableEtcdLauncher)
		if c.Spec.Features == nil {
			c.Spec.Features = make(map[string]bool)
		}
		c.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] = enableEtcdLauncher
		return c
	}

	seedClientGetter := func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
		return ctrlruntimefakeclient.
			NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(seed, makeCluster(true), makeCluster(false)).
			Build(), nil
	}

	makeEtcdRestore := func(c *kubermaticv1.Cluster) *kubermaticv1.EtcdRestore {
		return &kubermaticv1.EtcdRestore{
			Spec: kubermaticv1.EtcdRestoreSpec{
				Cluster: v1.ObjectReference{
					Namespace: c.Namespace,
					Name:      c.Name,
				},
			},
		}
	}

	tests := []struct {
		name                string
		etcdLauncherEnabled bool
		expectError         bool
	}{
		{
			"create with etcdlauncher enabled",
			true,
			false,
		},
		{
			"create with etcdlauncher disabled",
			false,
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := makeEtcdRestore(makeCluster(test.etcdLauncherEnabled))
			v := NewValidator(seedGetter, seedClientGetter)
			err := v.ValidateCreate(context.Background(), e)
			if test.expectError != (err != nil) {
				t.Fatalf("expectedError: %t, got: %s", test.expectError, err)
			}
		})
	}
}
