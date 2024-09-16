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

package seed

import (
	"context"
	"sync"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	providerconfig "k8c.io/machine-controller/pkg/providerconfig/types"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestValidate(t *testing.T) {
	fakeProviderSpec := kubermaticv1.DatacenterSpec{
		Fake: &kubermaticv1.DatacenterSpecFake{},
	}

	testCases := []struct {
		name             string
		seedToValidate   *kubermaticv1.Seed
		existingSeeds    []*kubermaticv1.Seed
		existingClusters []*kubermaticv1.Cluster
		features         features.FeatureGate
		isDelete         bool
		errExpected      bool
	}{
		{
			name:           "Adding an empty seed should be possible",
			seedToValidate: &kubermaticv1.Seed{},
		},
		{
			name: "Adding a seed with a single datacenter and valid provider should succeed",
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"dc1": {
							Spec: fakeProviderSpec,
						},
					},
				},
			},
		},
		{
			name: "No changes, no error",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"dc1": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"dc1": {
							Spec: fakeProviderSpec,
						},
					},
				},
			},
		},
		{
			name: "Clusters from other seeds should have no effect on new empty seeds",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "europe-west3-c",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"do-fra1": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			existingClusters: []*kubermaticv1.Cluster{
				{
					Spec: kubermaticv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "do-fra1",
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "asia-south1-a",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{},
				},
			},
		},
		{
			name: "Clusters from other seeds should have no effect when deleting seeds",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "europe-west3-c",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"do-fra1": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "asia-south1-a",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"aws-asia-south1-a": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			existingClusters: []*kubermaticv1.Cluster{
				{
					Spec: kubermaticv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "do-fra1",
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "asia-south1-a",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"aws-asia-south1-a": {
							Spec: fakeProviderSpec,
						},
					},
				},
			},
			isDelete: true,
		},
		{
			name: "Adding new datacenter should be possible",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"dc1": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"dc1": {
							Spec: fakeProviderSpec,
						},
						"dc2": {
							Spec: fakeProviderSpec,
						},
					},
				},
			},
		},
		{
			name: "Should be able to remove unused datacenters from a seed",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"dc1": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{},
				},
			},
		},
		{
			name: "Datacenters must have a provider defined",
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myseed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"a": {},
					},
				},
			},
			errExpected: true,
		},
		{
			name: "Datacenters cannot have multiple providers",
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myseed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"a": {
							Spec: kubermaticv1.DatacenterSpec{
								AWS:   &kubermaticv1.DatacenterSpecAWS{},
								Azure: &kubermaticv1.DatacenterSpecAzure{},
							},
						},
					},
				},
			},
			errExpected: true,
		},
		{
			name: "It should not be possible to change a datacenter's provider",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"dc1": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"dc1": {
							Spec: kubermaticv1.DatacenterSpec{
								AWS: &kubermaticv1.DatacenterSpecAWS{},
							},
						},
					},
				},
			},
			errExpected: true,
		},
		{
			name: "Datacenter names are unique across all seeds",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"in-use": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed-two",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"foo": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"in-use": {
							Spec: fakeProviderSpec,
						},
					},
				},
			},
			errExpected: true,
		},
		{
			name: "Cannot remove datacenters that are used by clusters",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"dc1": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			existingClusters: []*kubermaticv1.Cluster{
				{
					Spec: kubermaticv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "dc1",
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-seed",
				},
			},
			errExpected: true,
		},
		{
			name:           "Should be able to delete empty seeds",
			seedToValidate: &kubermaticv1.Seed{},
			isDelete:       true,
		},
		{
			name: "Should be able to delete seeds with no used datacenters",
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-seed",
					},
					Spec: kubermaticv1.SeedSpec{
						Datacenters: map[string]kubermaticv1.Datacenter{
							"dc1": {
								Spec: fakeProviderSpec,
							},
						},
					},
				},
			},
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"dc1": {
							Spec: fakeProviderSpec,
						},
					},
				},
			},
			isDelete: true,
		},
		{
			name: "Cannot delete a seed when there are still clusters left",
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myseed",
				},
			},
			existingSeeds: []*kubermaticv1.Seed{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "myseed",
					},
				},
			},
			existingClusters: []*kubermaticv1.Cluster{
				{},
			},
			isDelete:    true,
			errExpected: true,
		},
		{
			name: "Adding a seed with Tunneling ExposeStrategy should succeed",
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					ExposeStrategy: kubermaticv1.ExposeStrategyTunneling,
				},
			},
		},
		{
			name: "Adding a seed with invalid cron expression",
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Metering: &kubermaticv1.MeteringConfiguration{
						ReportConfigurations: map[string]kubermaticv1.MeteringReportConfiguration{
							"daily": {
								Schedule: "*/invalid * * * *",
							},
						},
					},
				},
			},
			features:    features.FeatureGate{},
			errExpected: true,
		},
		{
			name: "Adding a seed with kubevirt datacenter should fail with not supported operating-system",
			seedToValidate: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-seed",
				},
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"test-kv": {
							Spec: kubermaticv1.DatacenterSpec{
								Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
									Images: kubermaticv1.KubeVirtImageSources{HTTP: &kubermaticv1.KubeVirtHTTPSource{
										OperatingSystems: map[providerconfig.OperatingSystem]kubermaticv1.OSVersions{
											"invalid-os": map[string]string{"v1": "https://test.com"},
										},
									}},
								},
							},
						},
					},
				},
			},
			features:    features.FeatureGate{},
			errExpected: true,
		},
	}

	scheme := fake.NewScheme()
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to register scheme: %v", err)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				obj []ctrlruntimeclient.Object
				err error
			)

			clusterCRD, err := crd.CRDForGVK(kubermaticv1.SchemeGroupVersion.WithKind("Cluster"))
			if err != nil {
				t.Fatalf("Failed to load Cluster CRD: %v", err)
			}

			obj = append(obj, clusterCRD)
			for _, c := range tc.existingClusters {
				obj = append(obj, c)
			}
			for _, s := range tc.existingSeeds {
				obj = append(obj, s)
			}
			client := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(obj...).
				Build()

			sv := &validator{
				lock:        &sync.Mutex{},
				features:    tc.features,
				seedsGetter: test.NewSeedsGetter(tc.existingSeeds...),
				seedClientGetter: func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
					return client, nil
				},
			}

			if tc.isDelete {
				_, err = sv.ValidateDelete(context.Background(), tc.seedToValidate)
			} else {
				_, err = sv.ValidateCreate(context.Background(), tc.seedToValidate)
			}

			if (err != nil) != tc.errExpected {
				t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
			}
		})
	}
}
