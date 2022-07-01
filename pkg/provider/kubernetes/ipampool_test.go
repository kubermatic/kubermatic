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

package kubernetes_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestListIPAMPools(t *testing.T) {
	testCases := []struct {
		name             string
		existingObjects  []ctrlruntimeclient.Object
		expectedResponse *kubermaticv1.IPAMPoolList
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-2": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
							"test-dc-3": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/27",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-2",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-4": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/30",
								AllocationRange: 2,
							},
						},
					},
				},
			},
			expectedResponse: &kubermaticv1.IPAMPoolList{
				TypeMeta: metav1.TypeMeta{Kind: "IPAMPoolList", APIVersion: "kubermatic.k8c.io/v1"},
				Items: []kubermaticv1.IPAMPool{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							ResourceVersion: "999",
						},
						Spec: kubermaticv1.IPAMPoolSpec{
							Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
								"test-dc-2": {
									Type:            "range",
									PoolCIDR:        "192.168.1.0/28",
									AllocationRange: 8,
								},
								"test-dc-3": {
									Type:             "prefix",
									PoolCIDR:         "192.168.1.0/27",
									AllocationPrefix: 28,
								},
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-2",
							ResourceVersion: "999",
						},
						Spec: kubermaticv1.IPAMPoolSpec{
							Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
								"test-dc-4": {
									Type:            "range",
									PoolCIDR:        "192.168.1.0/30",
									AllocationRange: 2,
								},
							},
						},
					},
				},
			},
		},
		{
			name:            "empty list",
			existingObjects: []ctrlruntimeclient.Object{},
			expectedResponse: &kubermaticv1.IPAMPoolList{
				TypeMeta: metav1.TypeMeta{Kind: "IPAMPoolList", APIVersion: "kubermatic.k8c.io/v1"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			ipamPoolProvider := kubernetes.NewIPAMPoolProvider(client)

			resp, err := ipamPoolProvider.ListUnsecured(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedResponse, resp)
		})
	}
}

func TestGetIPAMPool(t *testing.T) {
	testCases := []struct {
		name             string
		existingObjects  []ctrlruntimeclient.Object
		ipamPoolName     string
		expectedResponse *kubermaticv1.IPAMPool
		expectedError    error
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-2": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
							"test-dc-3": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/27",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-2",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-4": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/30",
								AllocationRange: 2,
							},
						},
					},
				},
			},
			ipamPoolName: "test-pool-1",
			expectedResponse: &kubermaticv1.IPAMPool{
				TypeMeta: metav1.TypeMeta{Kind: "IPAMPool", APIVersion: "kubermatic.k8c.io/v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test-pool-1",
					ResourceVersion: "999",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"test-dc-2": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
						"test-dc-3": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/27",
							AllocationPrefix: 28,
						},
					},
				},
			},
		},
		{
			name:             "empty list",
			existingObjects:  []ctrlruntimeclient.Object{},
			ipamPoolName:     "test-pool-1",
			expectedResponse: nil,
			expectedError:    apierrors.NewNotFound(schema.GroupResource{Group: "kubermatic.k8c.io", Resource: "ipampools"}, "test-pool-1"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			ipamPoolProvider := kubernetes.NewIPAMPoolProvider(client)

			resp, err := ipamPoolProvider.GetUnsecured(context.Background(), tc.ipamPoolName)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedResponse, resp)
		})
	}
}

func TestCreateIPAMPool(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		ipamPool        *kubermaticv1.IPAMPool
		expectedError   error
	}{
		{
			name:            "base case",
			existingObjects: []ctrlruntimeclient.Object{},
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pool-1",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"test-dc-2": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
						"test-dc-3": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/27",
							AllocationPrefix: 28,
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "already exists",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pool-1",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"test-dc-2": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/28",
							AllocationPrefix: 29,
						},
					},
				},
			},
			expectedError: apierrors.NewAlreadyExists(schema.GroupResource{Group: "kubermatic.k8c.io", Resource: "ipampools"}, "test-pool-1"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			ipamPoolProvider := kubernetes.NewIPAMPoolProvider(client)

			err := ipamPoolProvider.CreateUnsecured(context.Background(), tc.ipamPool)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestDeleteIPAMPool(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		ipamPoolName    string
		expectedError   error
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			ipamPoolName:  "test-pool-1",
			expectedError: nil,
		},
		{
			name:            "not found",
			existingObjects: []ctrlruntimeclient.Object{},
			ipamPoolName:    "test-pool-1",
			expectedError:   apierrors.NewNotFound(schema.GroupResource{Group: "kubermatic.k8c.io", Resource: "ipampools"}, "test-pool-1"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			ipamPoolProvider := kubernetes.NewIPAMPoolProvider(client)

			err := ipamPoolProvider.DeleteUnsecured(context.Background(), tc.ipamPoolName)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestPatchIPAMPool(t *testing.T) {
	testCases := []struct {
		name             string
		existingObjects  []ctrlruntimeclient.Object
		newIPAMPool      *kubermaticv1.IPAMPool
		expectedGetError error
		expectedError    error
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			newIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pool-1",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"test-dc-1": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
						"test-dc-2": {
							Type:            "prefix",
							PoolCIDR:        "192.168.1.0/27",
							AllocationRange: 28,
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "not found",
			existingObjects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
						},
					},
				},
			},
			newIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pool-2",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"test-dc-2": {
							Type:            "prefix",
							PoolCIDR:        "192.168.1.0/27",
							AllocationRange: 28,
						},
					},
				},
			},
			expectedGetError: apierrors.NewNotFound(schema.GroupResource{Group: "kubermatic.k8c.io", Resource: "ipampools"}, "test-pool-2"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			ipamPoolProvider := kubernetes.NewIPAMPoolProvider(client)

			originalIPAMPool, err := ipamPoolProvider.GetUnsecured(ctx, tc.newIPAMPool.Name)
			assert.Equal(t, tc.expectedGetError, err)
			if err != nil {
				return
			}

			newIPAMPool := originalIPAMPool.DeepCopy()
			newIPAMPool.Spec = tc.newIPAMPool.Spec

			err = ipamPoolProvider.PatchUnsecured(ctx, originalIPAMPool, newIPAMPool)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}
