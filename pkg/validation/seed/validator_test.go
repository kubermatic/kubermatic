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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Cluster{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Cluster{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Cluster{
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
				&kubermaticv1.Seed{
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
				&kubermaticv1.Seed{
					ObjectMeta: metav1.ObjectMeta{
						Name: "myseed",
					},
				},
			},
			existingClusters: []*kubermaticv1.Cluster{
				&kubermaticv1.Cluster{},
			},
			isDelete:    true,
			errExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var obj []runtime.Object
			for _, c := range tc.existingClusters {
				obj = append(obj, c)
			}
			for _, s := range tc.existingSeeds {
				obj = append(obj, s)
			}
			client := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, obj...)
			sv := &validator{
				lock:     &sync.Mutex{},
				listOpts: &ctrlruntimeclient.ListOptions{},
				client:   client,
				seedClientGetter: func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
					return client, nil
				},
			}

			op := admissionv1beta1.Create
			if tc.isDelete {
				op = admissionv1beta1.Delete
			}
			err := sv.Validate(context.Background(), tc.seedToValidate, op)

			if (err != nil) != tc.errExpected {
				t.Fatalf("Expected err: %t, but got err: %v", tc.errExpected, err)
			}
		})
	}

}

func TestSingleSeedValidateFunc(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		seed      *kubermaticv1.Seed
		op        admissionv1beta1.Operation
		wantErr   bool
	}{
		{
			name:      "Matching name and namespace",
			namespace: "kubermatic",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      provider.DefaultSeedName,
					Namespace: "kubermatic",
				},
			},
			op:      admissionv1beta1.Create,
			wantErr: false,
		},
		{
			name:      "Non Matching namespace",
			namespace: "kubermatic",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      provider.DefaultSeedName,
					Namespace: "kube-system",
				},
			},
			op:      admissionv1beta1.Create,
			wantErr: true,
		},
		{
			name:      "Non Matching name",
			namespace: "kubermatic",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-seed",
					Namespace: "kubermatic",
				},
			},
			op:      admissionv1beta1.Create,
			wantErr: true,
		},
		{
			name:      "my-seed",
			namespace: "kubermatic",
			seed: &kubermaticv1.Seed{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "custom-seed",
					Namespace: "kube-system",
				},
			},
			op:      admissionv1beta1.Delete,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (ensureSingleSeedValidatorWrapper{validateFunc: validationSuccess, Namespace: tt.namespace, Name: "kubermatic"}).Validate(context.Background(), tt.seed, tt.op); (got == nil) == tt.wantErr {
				t.Errorf("Expected validation error = %v, but got: %v", tt.wantErr, got)
			}
		})
	}
}

func validationSuccess(ctx context.Context, seed *kubermaticv1.Seed, op admissionv1beta1.Operation) error {
	return nil
}
