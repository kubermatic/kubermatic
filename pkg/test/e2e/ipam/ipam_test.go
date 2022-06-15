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
	defer masterClient.CleanupCluster(t, project.ID, datacenter, cluster1.Name)

	t.Log("creating first IPAM Pool...")
	ipamPool1, err := createNewIPAMPool(ctx, seedClient, "192.168.1.0/28", "range", 8)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("checking IPAM Pool 1 allocation on first cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster1, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        datacenter,
		Addresses: []string{"192.168.1.0-192.168.1.7"},
	}) {
		t.Fatalf("IPAM Allocation 1 wasn't created on cluster 1")
	}

	t.Log("creating second cluster...")
	cluster2, err := createNewCluster(ctx, masterClient, seedClient, project.ID)
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer masterClient.CleanupCluster(t, project.ID, datacenter, cluster2.Name)

	t.Log("checking IPAM Pool 1 allocation on second cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster2, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        datacenter,
		Addresses: []string{"192.168.1.8-192.168.1.15"},
	}) {
		t.Fatalf("IPAM Allocation 1 wasn't created on cluster 2")
	}

	t.Log("creating second IPAM Pool...")
	ipamPool2, err := createNewIPAMPool(ctx, seedClient, "192.168.1.0/27", "prefix", 28)
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("checking IPAM Pool 2 allocation on first cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster1, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   datacenter,
		CIDR: "192.168.1.0/28",
	}) {
		t.Fatalf("IPAM Allocation 2 wasn't created on cluster 1")
	}

	t.Log("checking IPAM Pool 2 allocation on second cluster...")
	if !checkIPAMAllocation(t, ctx, seedClient, cluster2, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   datacenter,
		CIDR: "192.168.1.16/28",
	}) {
		t.Fatalf("IPAM Allocation 2 wasn't created on cluster 2")
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
				datacenter: {
					Type:             allocationType,
					PoolCIDR:         poolCIDR,
					AllocationRange:  uint32(allocationValue),
					AllocationPrefix: uint8(allocationValue),
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
	return utils.WaitFor(1*time.Second, 5*time.Second, func() bool {
		ipamAllocation := &kubermaticv1.IPAMAllocation{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: ipamAllocationName, Namespace: cluster.Status.NamespaceName}, ipamAllocation); err != nil {
			t.Logf("Error getting IPAM allocation for cluster: %v", err)
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
