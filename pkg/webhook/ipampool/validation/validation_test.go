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

package validation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	admissionv1 "k8s.io/api/admission/v1"
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

func TestValidator(t *testing.T) {
	testCases := []struct {
		name          string
		op            admissionv1.Operation
		ipamPool      *kubermaticv1.IPAMPool
		oldIPAMPool   *kubermaticv1.IPAMPool
		objects       []ctrlruntimeclient.Object
		expectedError error
	}{
		{
			name: "deletion always allowed",
			op:   admissionv1.Delete,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
			},
			expectedError: nil,
		},
		{
			name: "invalid pool cidr",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "",
							AllocationRange: 8,
						},
					},
				},
			},
			expectedError: &net.ParseError{Type: "CIDR address"},
		},
		{
			name: "allowed range creation",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "missing allocation range",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:     "range",
							PoolCIDR: "192.168.1.0/28",
						},
					},
				},
			},
			expectedError: errors.New("allocation range should be greater than zero"),
		},
		{
			name: "allocation range greater than subnet size",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 17,
						},
					},
				},
			},
			expectedError: errors.New("allocation range cannot be greater than the pool subnet possible number of IP addresses"),
		},
		{
			name: "pool too big for range allocation",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "2001:db8:abcd:0012::0/64",
							AllocationRange: 10000,
						},
					},
				},
			},
			expectedError: errors.New("the pool is too big to be processed"),
		},
		{
			name: "pool too big for range allocation 2",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "192.168.0.1/19",
							AllocationRange: 8,
						},
					},
				},
			},
			expectedError: errors.New("pool prefix is too low for range allocation type"),
		},
		{
			name: "allowed prefix creation",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/28",
							AllocationPrefix: 29,
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "prefix smaller than subnet mask size",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "2001:db8:abcd:0012::0/64",
							AllocationPrefix: 63,
						},
					},
				},
			},
			expectedError: errors.New("allocation prefix cannot be smaller than the pool subnet mask size"),
		},
		{
			name: "invalid allocation prefix for IP version",
			op:   admissionv1.Create,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/32",
							AllocationPrefix: 64,
						},
					},
				},
			},
			expectedError: errors.New("invalid allocation prefix for IP version"),
		},
		{
			name: "allowed to update adding a new datacenter pool",
			op:   admissionv1.Update,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/27",
							AllocationPrefix: 28,
						},
						"dc2": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			oldIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
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
			name: "not allowed CIDR update",
			op:   admissionv1.Update,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/29",
							AllocationRange: 8,
						},
					},
				},
			},
			oldIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			expectedError: errors.New("it's not allowed to update the pool CIDR for a datacenter"),
		},
		{
			name: "not allowed type update",
			op:   admissionv1.Update,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/28",
							AllocationPrefix: 29,
						},
					},
				},
			},
			oldIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			expectedError: errors.New("it's not allowed to update the allocation type for a datacenter"),
		},
		{
			name: "not allowed to update the allocation range",
			op:   admissionv1.Update,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 10,
						},
					},
				},
			},
			oldIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			expectedError: errors.New("it's not allowed to update the allocation range for a datacenter"),
		},
		{
			name: "not allowed to update the allocation prefix",
			op:   admissionv1.Update,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/27",
							AllocationPrefix: 29,
						},
					},
				},
			},
			oldIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/27",
							AllocationPrefix: 28,
						},
					},
				},
			},
			expectedError: errors.New("it's not allowed to update the allocation prefix for a datacenter"),
		},
		{
			name: "allowed to remove a datacenter pool if no allocations",
			op:   admissionv1.Update,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc2": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			oldIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-ipam-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/27",
							AllocationPrefix: 28,
						},
						"dc2": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "not allowed to remove a datacenter pool if it has allocations",
			op:   admissionv1.Update,
			ipamPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc2": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			oldIPAMPool: &kubermaticv1.IPAMPool{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pool",
				},
				Spec: kubermaticv1.IPAMPoolSpec{
					Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
						"dc": {
							Type:             "prefix",
							PoolCIDR:         "192.168.1.0/27",
							AllocationPrefix: 28,
						},
						"dc2": {
							Type:            "range",
							PoolCIDR:        "192.168.1.0/28",
							AllocationRange: 8,
						},
					},
				},
			},
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "dc",
						CIDR: "192.168.1.0/28",
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "dc2",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
			},
			expectedError: errors.New("cannot delete some datacenter IPAMPool because there is existing IPAMAllocation in namespaces (cluster-test-cluster-1)"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			seedClient := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(tc.objects...).
				Build()

			validator := NewValidator(
				func() (*kubermaticv1.Seed, error) {
					return &kubermaticv1.Seed{}, nil
				},
				func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
					return seedClient, nil
				})

			ctx := context.Background()
			var err error

			switch tc.op {
			case admissionv1.Create:
				err = validator.ValidateCreate(ctx, tc.ipamPool)
			case admissionv1.Update:
				err = validator.ValidateUpdate(ctx, tc.oldIPAMPool, tc.ipamPool)
			case admissionv1.Delete:
				err = validator.ValidateDelete(ctx, tc.ipamPool)
			}

			assert.Equal(t, tc.expectedError, err)
		})
	}
}
