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

package datacenter

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/crd"
	"k8c.io/kubermatic/v3/pkg/test"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidate(t *testing.T) {
	testCases := []struct {
		name             string
		newDatacenter    *kubermaticv1.Datacenter
		oldDatacenter    *kubermaticv1.Datacenter
		existingClusters []*kubermaticv1.Cluster
		errExpected      bool
	}{
		{
			name:          "Adding a datacenter with a single datacenter and valid provider should succeed",
			oldDatacenter: nil,
			newDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-datacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						Fake: &kubermaticv1.DatacenterSpecFake{},
					},
				},
			},
		},
		{
			name: "No changes, no error",
			oldDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-datacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						Fake: &kubermaticv1.DatacenterSpecFake{},
					},
				},
			},
			newDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-datacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						Fake: &kubermaticv1.DatacenterSpecFake{},
					},
				},
			},
		},
		{
			name: "Should be able to remove unused datacenters",
			oldDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "existing-datacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						Fake: &kubermaticv1.DatacenterSpecFake{},
					},
				},
			},
			newDatacenter: nil,
		},
		{
			name: "Datacenters must have a provider defined",
			newDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mydatacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{},
				},
			},
			errExpected: true,
		},
		{
			name: "Datacenters cannot have multiple providers",
			newDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mydatacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						Fake:         &kubermaticv1.DatacenterSpecFake{},
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{},
					},
				},
			},
			errExpected: true,
		},
		{
			name: "It should not be possible to change a datacenter's provider",
			oldDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mydatacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{},
					},
				},
			},
			newDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mydatacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						Fake: &kubermaticv1.DatacenterSpecFake{},
					},
				},
			},
			errExpected: true,
		},
		{
			name: "Cannot remove datacenters that are used by clusters",
			existingClusters: []*kubermaticv1.Cluster{
				{
					Spec: kubermaticv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							DatacenterName: "mydatacenter",
						},
					},
				},
			},
			oldDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "mydatacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{},
					},
				},
			},
			newDatacenter: nil,
			errExpected:   true,
		},
		{
			name:          "Adding a seed with kubevirt datacenter should fail with not supported operating-system",
			oldDatacenter: nil,
			newDatacenter: &kubermaticv1.Datacenter{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-datacenter",
				},
				Spec: kubermaticv1.DatacenterSpec{
					Provider: kubermaticv1.DatacenterProviderSpec{
						KubeVirt: &kubermaticv1.DatacenterSpecKubeVirt{
							Images: &kubermaticv1.KubeVirtImageSources{
								HTTP: &kubermaticv1.KubeVirtHTTPSource{
									OperatingSystems: map[kubermaticv1.OperatingSystem]kubermaticv1.OSVersions{
										"invalid-os": map[string]string{"v1": "https://test.com"},
									},
								},
							},
						},
					},
				},
			},
			errExpected: true,
		},
	}

	if err := apiextensionsv1.AddToScheme(scheme.Scheme); err != nil {
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
			existingDatacenters := []*kubermaticv1.Datacenter{}
			if tc.oldDatacenter != nil {
				obj = append(obj, tc.oldDatacenter)
				existingDatacenters = append(existingDatacenters, tc.oldDatacenter)
			}

			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(obj...).
				Build()

			sv := &validator{
				seedClient:        client,
				datacentersGetter: test.NewDatacentersGetter(existingDatacenters...),
			}

			if tc.oldDatacenter != nil && tc.newDatacenter != nil {
				err = sv.ValidateUpdate(context.Background(), tc.oldDatacenter, tc.newDatacenter)
			} else if tc.oldDatacenter != nil {
				err = sv.ValidateDelete(context.Background(), tc.oldDatacenter)
			} else if tc.newDatacenter != nil {
				err = sv.ValidateCreate(context.Background(), tc.newDatacenter)
			} else {
				t.Fatal("invalid test case: either old or new or both datacenters must be set")
			}

			if (err != nil) != tc.errExpected {
				t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
			}
		})
	}
}
