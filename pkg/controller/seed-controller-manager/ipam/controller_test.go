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

package ipam

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func generateTestCluster(clusterName, dc string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: dc,
			},
		},
		Status: kubermaticv1.ClusterStatus{NamespaceName: fmt.Sprintf("cluster-%s", clusterName)},
	}
}

func TestReconcileCluster(t *testing.T) {
	testCases := []struct {
		name                       string
		objects                    []ctrlruntimeclient.Object
		cluster                    *kubermaticv1.Cluster
		expectedClusterAllocations *kubermaticv1.IPAMAllocationList
		expectedError              error
	}{
		{
			name:    "no pools",
			cluster: generateTestCluster("test-cluster-1", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{},
			},
		},
		{
			name:    "ignore pools for different dc",
			cluster: generateTestCluster("test-cluster-1", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
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
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{},
			},
		},
		{
			name:    "ignore allocations not relevant to the new pool",
			cluster: generateTestCluster("test-cluster-1", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
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
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-2",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/27",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-3",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
							"test-dc-2": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/27",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-2",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-2"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-1",
						CIDR: "192.168.1.0/28",
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
							DC:        "test-dc-1",
							Addresses: []string{"192.168.1.0-192.168.1.7"},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-2",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-2"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
							DC:   "test-dc-1",
							CIDR: "192.168.1.0/28",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-3",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-3"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
							DC:        "test-dc-1",
							Addresses: []string{"192.168.1.0-192.168.1.7"},
						},
					},
				},
			},
		},
		{
			name:    "delete allocation with datacenter not present in the IPAM pool spec",
			cluster: generateTestCluster("test-cluster-1", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{},
			},
		},
		{
			name:    "skipping already allocated cluster for pool",
			cluster: generateTestCluster("test-cluster-1", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
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
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.8-192.168.1.15"},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
							DC:        "test-dc-1",
							Addresses: []string{"192.168.1.8-192.168.1.15"},
						},
					},
				},
			},
		},
		{
			name:    "range: single pool",
			cluster: generateTestCluster("test-cluster-2", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
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
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
							DC:        "test-dc-1",
							Addresses: []string{"192.168.1.8-192.168.1.15"},
						},
					},
				},
			},
		},
		{
			name:    "range: single pool, multiple clusters in same dc",
			cluster: generateTestCluster("test-cluster-3", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/27",
								AllocationRange: 8,
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.8-192.168.1.15"},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-3"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
							DC:        "test-dc-1",
							Addresses: []string{"192.168.1.16-192.168.1.23"},
						},
					},
				},
			},
		},
		{
			name:    "range: single pool, not enough IPs from pool",
			cluster: generateTestCluster("test-cluster-1", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/30",
								AllocationRange: 9,
							},
						},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{},
			},
			expectedError: errors.New("failed to ensure IPAM Pool Allocation for IPAM Pool test-pool-1 in cluster test-cluster-1: failed to ensure IPAMAllocation cluster-test-cluster-1/test-pool-1: failed to generate object: there is no enough free IPs available for IPAM pool \"test-pool-1\""),
		},
		{
			name:    "range: single pool, not enough IPs from pool (2)",
			cluster: generateTestCluster("test-cluster-3", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/31",
								AllocationRange: 1,
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.0"},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.1"},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{},
			},
			expectedError: errors.New("failed to ensure IPAM Pool Allocation for IPAM Pool test-pool-1 in cluster test-cluster-3: failed to ensure IPAMAllocation cluster-test-cluster-3/test-pool-1: failed to generate object: there is no enough free IPs available for IPAM pool \"test-pool-1\""),
		},
		{
			name:    "prefix: single pool",
			cluster: generateTestCluster("test-cluster-2", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/28",
								AllocationPrefix: 29,
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-1",
						CIDR: "192.168.1.0/29",
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
							DC:   "test-dc-1",
							CIDR: "192.168.1.8/29",
						},
					},
				},
			},
		},
		{
			name:    "prefix: single pool, multiple clusters in same dc",
			cluster: generateTestCluster("test-cluster-3", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/26",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-1",
						CIDR: "192.168.1.0/28",
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-1",
						CIDR: "192.168.1.16/28",
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-3"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
							DC:   "test-dc-1",
							CIDR: "192.168.1.32/28",
						},
					},
				},
			},
		},
		{
			name:    "prefix: single pool, invalid prefix for subnet",
			cluster: generateTestCluster("test-cluster-1", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/28",
								AllocationPrefix: 27,
							},
						},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{},
			},
			expectedError: errors.New("failed to ensure IPAM Pool Allocation for IPAM Pool test-pool-1 in cluster test-cluster-1: failed to ensure IPAMAllocation cluster-test-cluster-1/test-pool-1: failed to generate object: invalid prefix for subnet"),
		},
		{
			name:    "prefix: single pool, invalid prefix for subnet 2",
			cluster: generateTestCluster("test-cluster-1", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/28",
								AllocationPrefix: 33,
							},
						},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{},
			},
			expectedError: errors.New("failed to ensure IPAM Pool Allocation for IPAM Pool test-pool-1 in cluster test-cluster-1: failed to ensure IPAMAllocation cluster-test-cluster-1/test-pool-1: failed to generate object: invalid prefix for subnet"),
		},
		{
			name:    "prefix: single pool, no free subnet",
			cluster: generateTestCluster("test-cluster-3", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/31",
								AllocationPrefix: 32,
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-1",
						CIDR: "192.168.1.0/32",
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-1",
						CIDR: "192.168.1.1/32",
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{},
			},
			expectedError: errors.New("failed to ensure IPAM Pool Allocation for IPAM Pool test-pool-1 in cluster test-cluster-3: failed to ensure IPAMAllocation cluster-test-cluster-3/test-pool-1: failed to generate object: there is no free subnet available for IPAM Pool \"test-pool-1\""),
		},
		{
			name:    "multiple pools, clusters and DCs",
			cluster: generateTestCluster("test-cluster-3", "test-dc-2"),
			objects: []ctrlruntimeclient.Object{
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
							"test-dc-3": {
								Type:            "range",
								PoolCIDR:        "193.170.2.0/28",
								AllocationRange: 16,
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
							"test-dc-2": {
								Type:             "prefix",
								PoolCIDR:         "192.167.1.0/27",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-3",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/28",
								AllocationRange: 8,
							},
							"test-dc-2": {
								Type:             "prefix",
								PoolCIDR:         "192.169.1.0/27",
								AllocationPrefix: 28,
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-3",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-3"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-2",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-2"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-2",
						CIDR: "192.167.1.0/28",
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-3",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-3"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-2",
						CIDR: "192.169.1.0/28",
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-2",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-3"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-2"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
							DC:   "test-dc-2",
							CIDR: "192.167.1.16/28",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-3",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-3"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-3"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
							DC:   "test-dc-2",
							CIDR: "192.169.1.16/28",
						},
					},
				},
			},
		},
		{
			name:    "prefix: exclude prefixes",
			cluster: generateTestCluster("test-cluster-2", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/27",
								AllocationPrefix: 29,
								ExcludePrefixes:  []kubermaticv1.SubnetCIDR{"192.168.1.8/29", "192.168.1.16/29"},
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-1",
						CIDR: "192.168.1.0/29",
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
							DC:   "test-dc-1",
							CIDR: "192.168.1.24/29",
						},
					},
				},
			},
		},
		{
			name:    "range: exclude ranges",
			cluster: generateTestCluster("test-cluster-2", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/27",
								AllocationRange: 8,
								ExcludeRanges:   []string{"192.168.1.9", "192.168.1.11-192.168.1.11", "192.168.1.15-192.168.1.17"},
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-1"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
							ResourceVersion: "1",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
							DC:        "test-dc-1",
							Addresses: []string{"192.168.1.8-192.168.1.8", "192.168.1.10-192.168.1.10", "192.168.1.12-192.168.1.14", "192.168.1.18-192.168.1.20"},
						},
					},
				},
			},
		},
		{
			name:    "range: increase allocation range",
			cluster: generateTestCluster("test-cluster-2", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:            "range",
								PoolCIDR:        "192.168.1.0/27",
								AllocationRange: 10,
								ExcludeRanges:   []string{"192.168.1.9"},
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
						DC:        "test-dc-1",
						Addresses: []string{"192.168.1.0-192.168.1.7"},
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
							ResourceVersion: "2",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type:      kubermaticv1.IPAMPoolAllocationTypeRange,
							DC:        "test-dc-1",
							Addresses: []string{"192.168.1.0-192.168.1.7", "192.168.1.8-192.168.1.8", "192.168.1.10-192.168.1.10"},
						},
					},
				},
			},
		},
		{
			name:    "prefix: lower allocation prefix",
			cluster: generateTestCluster("test-cluster-2", "test-dc-1"),
			objects: []ctrlruntimeclient.Object{
				&kubermaticv1.IPAMPool{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-pool-1",
					},
					Spec: kubermaticv1.IPAMPoolSpec{
						Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
							"test-dc-1": {
								Type:             "prefix",
								PoolCIDR:         "192.168.1.0/26",
								AllocationPrefix: 29,
								ExcludePrefixes:  []kubermaticv1.SubnetCIDR{"192.168.1.8/29", "192.168.1.16/29"},
							},
						},
					},
				},
				&kubermaticv1.IPAMAllocation{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "test-pool-1",
						Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
						ResourceVersion: "1",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
					},
					Spec: kubermaticv1.IPAMAllocationSpec{
						Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
						DC:   "test-dc-1",
						CIDR: "192.168.1.0/30",
					},
				},
			},
			expectedClusterAllocations: &kubermaticv1.IPAMAllocationList{
				Items: []kubermaticv1.IPAMAllocation{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "test-pool-1",
							Namespace:       fmt.Sprintf("cluster-%s", "test-cluster-2"),
							ResourceVersion: "2",
							OwnerReferences: []metav1.OwnerReference{{APIVersion: "kubermatic.k8c.io/v1", Kind: "IPAMPool", Name: "test-pool-1"}},
						},
						Spec: kubermaticv1.IPAMAllocationSpec{
							Type: kubermaticv1.IPAMPoolAllocationTypePrefix,
							DC:   "test-dc-1",
							CIDR: "192.168.1.0/29",
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			reconciler := &Reconciler{
				Client: fake.
					NewClientBuilder().
					WithObjects(tc.objects...).
					Build(),
			}

			_, err := reconciler.reconcile(ctx, tc.cluster)
			if tc.expectedError != nil {
				assert.EqualError(t, err, tc.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			ipamAllocationList := &kubermaticv1.IPAMAllocationList{}
			err = reconciler.List(ctx, ipamAllocationList, ctrlruntimeclient.InNamespace(tc.cluster.Status.NamespaceName))
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedClusterAllocations, ipamAllocationList)
		})
	}
}
