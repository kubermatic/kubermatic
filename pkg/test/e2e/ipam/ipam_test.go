//go:build ipam

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
	"fmt"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	datacenter = "kubermatic"
	location   = "hetzner-hel1"
	version    = utils.KubernetesVersion()
	credential = "e2e-hetzner"
)

func TestIPAM(t *testing.T) {
	ctx := context.Background()

	seedClient, _, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// login
	masterToken, err := utils.RetrieveMasterToken(ctx)
	if err != nil {
		t.Fatalf("failed to get master token: %v", err)
	}
	masterClient := utils.NewTestClient(masterToken, t)

	// create dummy project
	t.Log("creating project...")
	project, err := masterClient.CreateProject(rand.String(10))
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	defer masterClient.CleanupProject(t, project.ID)

	t.Log("creating first cluster...")
	cluster1, err := createNewCluster(ctx, masterClient, seedClient, project.ID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("creating first IPAM Pool...")
	ipamPool1, err := createNewIPAMPool(ctx, seedClient, "192.168.1.0/28", "range", 8)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("checking IPAM Pool 1 allocation on first cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster1, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        location,
		Addresses: []string{"192.168.1.0-192.168.1.7"},
	}) {
		t.Fatalf("IPAM Allocation 1 wasn't created on cluster 1")
	}

	t.Log("creating second IPAM Pool...")
	ipamPool2, err := createNewIPAMPool(ctx, seedClient, "192.168.1.0/27", "prefix", 28)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("checking IPAM Pool 2 allocation on first cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster1, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   location,
		CIDR: "192.168.1.0/28",
	}) {
		t.Fatalf("IPAM Allocation 2 wasn't created on cluster 1")
	}

	t.Log("creating second cluster...")
	cluster2, err := createNewCluster(ctx, masterClient, seedClient, project.ID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("checking IPAM Pool 1 allocation on second cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster2, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        location,
		Addresses: []string{"192.168.1.8-192.168.1.15"},
	}) {
		t.Fatalf("IPAM Allocation 1 wasn't created on cluster 2")
	}

	t.Log("checking IPAM Pool 2 allocation on second cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster2, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   location,
		CIDR: "192.168.1.16/28",
	}) {
		t.Fatalf("IPAM Allocation 2 wasn't created on cluster 2")
	}

	t.Log("deleting first cluster...")
	masterClient.CleanupCluster(t, project.ID, datacenter, cluster1.Name)

	t.Log("checking that the first cluster allocations were gone...")
	if !checkIPAMAllocationIsGone(t, ctx, seedClient, cluster1, ipamPool1.Name) ||
		!checkIPAMAllocationIsGone(t, ctx, seedClient, cluster1, ipamPool2.Name) {
		t.Fatalf("IPAM Allocations in first cluster are still persisted")
	}

	t.Log("creating third IPAM Pool...")
	ipamPool3, err := createNewIPAMPool(ctx, seedClient, "193.169.1.0/28", "prefix", 29)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("checking IPAM Pool 3 allocation on second cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster2, ipamPool3.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   location,
		CIDR: "193.169.1.0/29",
	}) {
		t.Fatalf("IPAM Allocation 3 wasn't created on cluster 2")
	}

	t.Log("creating third cluster...")
	cluster3, err := createNewCluster(ctx, masterClient, seedClient, project.ID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("checking IPAM Pool 3 allocation on third cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster3, ipamPool3.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   location,
		CIDR: "193.169.1.8/29",
	}) {
		t.Fatalf("IPAM Allocation 3 wasn't created on cluster 3")
	}

	t.Log("checking IPAM Pool 1 allocation on third cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster3, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        location,
		Addresses: []string{"192.168.1.0-192.168.1.7"},
	}) {
		t.Fatalf("IPAM Allocation 1 wasn't created on cluster 3")
	}

	t.Log("checking IPAM Pool 2 allocation on third cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster3, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   location,
		CIDR: "192.168.1.0/28",
	}) {
		t.Fatalf("IPAM Allocation 2 wasn't created on cluster 3")
	}

	masterClient.CleanupCluster(t, project.ID, datacenter, cluster2.Name)
	masterClient.CleanupCluster(t, project.ID, datacenter, cluster3.Name)

	if !checkIPAMAllocationIsGone(t, ctx, seedClient, cluster2, ipamPool1.Name) ||
		!checkIPAMAllocationIsGone(t, ctx, seedClient, cluster2, ipamPool2.Name) ||
		!checkIPAMAllocationIsGone(t, ctx, seedClient, cluster2, ipamPool3.Name) ||
		!checkIPAMAllocationIsGone(t, ctx, seedClient, cluster3, ipamPool1.Name) ||
		!checkIPAMAllocationIsGone(t, ctx, seedClient, cluster3, ipamPool2.Name) ||
		!checkIPAMAllocationIsGone(t, ctx, seedClient, cluster3, ipamPool3.Name) {
		t.Fatalf("Some IPAM Allocation is still persisted")
	}
}

func createNewCluster(ctx context.Context, masterClient *utils.TestClient, seedClient ctrlruntimeclient.Client, projectID string) (*kubermaticv1.Cluster, error) {
	apiCluster, err := masterClient.CreateHetznerCluster(projectID, datacenter, rand.String(10), credential, version, location, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// wait for the cluster to become healthy
	if err := masterClient.WaitForClusterHealthy(projectID, datacenter, apiCluster.ID); err != nil {
		return nil, fmt.Errorf("cluster did not become healthy: %w", err)
	}

	// get the cluster object (the CRD, not the API's representation)
	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: apiCluster.ID}, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	return cluster, nil
}

func createNewIPAMPool(ctx context.Context, seedClient ctrlruntimeclient.Client, poolCIDR kubermaticv1.SubnetCIDR, allocationType kubermaticv1.IPAMPoolAllocationType, allocationValue int) (*kubermaticv1.IPAMPool, error) {
	ipamPoolName := rand.String(10)

	if err := seedClient.Create(ctx, &kubermaticv1.IPAMPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ipamPoolName,
			Labels: map[string]string{},
		},
		Spec: kubermaticv1.IPAMPoolSpec{
			Datacenters: map[string]kubermaticv1.IPAMPoolDatacenterSettings{
				location: {
					Type:             allocationType,
					PoolCIDR:         poolCIDR,
					AllocationRange:  allocationValue,
					AllocationPrefix: allocationValue,
				},
			},
		},
	}); err != nil {
		return nil, err
	}

	ipamPool := &kubermaticv1.IPAMPool{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: ipamPoolName}, ipamPool); err != nil {
		return nil, fmt.Errorf("failed to get IPAM pool: %w", err)
	}

	return ipamPool, nil
}

func checkIPAMAllocation(t *testing.T, ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string, expectedIPAMAllocationSpec kubermaticv1.IPAMAllocationSpec) bool {
	return utils.WaitFor(10*time.Second, 5*time.Minute, func() bool {
		ipamAllocation := &kubermaticv1.IPAMAllocation{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: ipamAllocationName, Namespace: cluster.Status.NamespaceName}, ipamAllocation); err != nil {
			t.Logf("Error getting IPAM allocation for cluster %s (namespace %s): %v", cluster.Name, cluster.Status.NamespaceName, err)
			return false
		}
		if ipamAllocation.Spec.Type != expectedIPAMAllocationSpec.Type ||
			ipamAllocation.Spec.CIDR != expectedIPAMAllocationSpec.CIDR ||
			ipamAllocation.Spec.DC != expectedIPAMAllocationSpec.DC ||
			len(ipamAllocation.Spec.Addresses) != len(expectedIPAMAllocationSpec.Addresses) {
			t.Logf("Expected IPAM allocation: %+v\nActual IPAM allocation: %+v", expectedIPAMAllocationSpec, ipamAllocation.Spec)
			return false
		}
		for i, address := range ipamAllocation.Spec.Addresses {
			if address != expectedIPAMAllocationSpec.Addresses[i] {
				t.Logf("Expected IPAM allocation: %+v\nActual IPAM allocation: %+v", expectedIPAMAllocationSpec, ipamAllocation.Spec)
				return false
			}
		}
		return true
	})
}

func checkIPAMAllocationIsGone(t *testing.T, ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string) bool {
	return utils.WaitFor(10*time.Second, 5*time.Minute, func() bool {
		ipamAllocation := &kubermaticv1.IPAMAllocation{}
		err := seedClient.Get(ctx, types.NamespacedName{Name: ipamAllocationName, Namespace: cluster.Status.NamespaceName}, ipamAllocation)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true
			}
			t.Logf("Error getting IPAM allocation for cluster %s (namespace %s): %v", cluster.Name, cluster.Status.NamespaceName, err)
			return false
		}
		return false
	})
}
