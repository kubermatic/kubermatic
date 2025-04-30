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
	"flag"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestIPAM(t *testing.T) {
	ctx := context.Background()
	log := log.NewFromOptions(logOptions).Sugar()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("failed to get credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	log.Info("Creating first user cluster...")
	cluster1, userClient1, cleanupUserCluster1, err := createUserCluster(ctx, t, log, seedClient, "ipam1")
	if err != nil {
		t.Fatalf("failed to create user cluster: %v", err)
	}

	log.Info("Creating first IPAM Pool (for metallb addon)...")
	ipamPool1, err := createNewIPAMPool(ctx, seedClient, "metallb", map[string]kubermaticv1.IPAMPoolDatacenterSettings{
		credentials.KKPDatacenter: {
			Type:            "range",
			PoolCIDR:        "192.168.1.0/27",
			AllocationRange: 8,
			ExcludeRanges:   []string{"192.168.1.3", "192.168.1.5-192.168.1.5", "192.168.1.8-192.168.1.10"},
		},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Info("Checking IPAM Pool 1 allocation on first cluster...")
	if err := checkAllocation(ctx, log, seedClient, userClient1, cluster1, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        credentials.KKPDatacenter,
		Addresses: []string{"192.168.1.0-192.168.1.2", "192.168.1.4-192.168.1.4", "192.168.1.6-192.168.1.7", "192.168.1.11-192.168.1.12"},
	}); err != nil {
		t.Fatal(err)
	}

	log.Info("Creating second IPAM Pool...")
	ipamPool2, err := createNewIPAMPool(ctx, seedClient, "", map[string]kubermaticv1.IPAMPoolDatacenterSettings{
		credentials.KKPDatacenter: {
			Type:             "prefix",
			PoolCIDR:         "192.169.1.0/27",
			AllocationPrefix: 29,
			ExcludePrefixes:  []kubermaticv1.SubnetCIDR{"192.169.1.0/29", "192.169.1.24/29"},
		},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Info("Checking IPAM Pool 2 allocation on first cluster...")
	if err := checkAllocation(ctx, log, seedClient, userClient1, cluster1, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   credentials.KKPDatacenter,
		CIDR: "192.169.1.8/29",
	}); err != nil {
		t.Fatal(err)
	}

	log.Info("Creating second user cluster...")
	cluster2, userClient2, cleanupUserCluster2, err := createUserCluster(ctx, t, log, seedClient, "ipam2")
	if err != nil {
		t.Fatalf("failed to create user cluster: %v", err)
	}

	log.Info("Checking IPAM Pool 1 allocation on second cluster...")
	if err := checkAllocation(ctx, log, seedClient, userClient2, cluster2, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        credentials.KKPDatacenter,
		Addresses: []string{"192.168.1.13-192.168.1.20"},
	}); err != nil {
		t.Fatal(err)
	}

	log.Info("Checking IPAM Pool 2 allocation on second cluster...")
	if err := checkAllocation(ctx, log, seedClient, userClient2, cluster2, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   credentials.KKPDatacenter,
		CIDR: "192.169.1.16/29",
	}); err != nil {
		t.Fatal(err)
	}

	log.Info("Deleting first cluster...")
	cleanupUserCluster1()

	log.Info("Checking that the first cluster allocations were gone...")
	if err := checkAllocationIsGone(ctx, log, seedClient, userClient1, cluster1, ipamPool1.Name); err != nil {
		t.Fatal(err)
	}
	if err := checkAllocationIsGone(ctx, log, seedClient, userClient1, cluster1, ipamPool2.Name); err != nil {
		t.Fatal(err)
	}

	log.Info("Creating third IPAM Pool...")
	ipamPool3Datacenters := map[string]kubermaticv1.IPAMPoolDatacenterSettings{
		credentials.KKPDatacenter: {
			Type:             "prefix",
			PoolCIDR:         "193.169.1.0/28",
			AllocationPrefix: 29,
		},
		credentials.KKPDatacenter + "-dummy": {
			Type:            "range",
			PoolCIDR:        "194.170.1.0/28",
			AllocationRange: 8,
		},
	}
	ipamPool3, err := createNewIPAMPool(ctx, seedClient, "", ipamPool3Datacenters)
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Info("Checking IPAM Pool 3 allocation on second cluster...")
	if err := checkAllocation(ctx, log, seedClient, userClient2, cluster2, ipamPool3.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   credentials.KKPDatacenter,
		CIDR: "193.169.1.0/29",
	}); err != nil {
		t.Fatal(err)
	}

	log.Info("Creating third user cluster...")
	cluster3, userClient3, cleanupUserCluster3, err := createUserCluster(ctx, t, log, seedClient, "ipam3")
	if err != nil {
		t.Fatalf("failed to create user cluster: %v", err)
	}

	log.Info("Checking IPAM Pool 3 allocation on third cluster...")
	if err := checkAllocation(ctx, log, seedClient, userClient3, cluster3, ipamPool3.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   credentials.KKPDatacenter,
		CIDR: "193.169.1.8/29",
	}); err != nil {
		t.Fatal(err)
	}

	log.Info("Removing cluster datacenter spec from IPAM pool 3...")
	delete(ipamPool3Datacenters, credentials.KKPDatacenter)
	err = updateIPAMPool(ctx, seedClient, ipamPool3.Name, ipamPool3Datacenters)
	if err != nil {
		t.Fatal(err.Error())
	}

	if err := checkAllocationIsGone(ctx, log, seedClient, userClient2, cluster2, ipamPool3.Name); err != nil {
		t.Fatal(err)
	}
	if err := checkAllocationIsGone(ctx, log, seedClient, userClient3, cluster3, ipamPool3.Name); err != nil {
		t.Fatal(err)
	}

	log.Info("Checking IPAM Pool 1 allocation on third cluster...")
	if err := checkAllocation(ctx, log, seedClient, userClient3, cluster3, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        credentials.KKPDatacenter,
		Addresses: []string{"192.168.1.0-192.168.1.2", "192.168.1.4-192.168.1.4", "192.168.1.6-192.168.1.7", "192.168.1.11-192.168.1.12"},
	}); err != nil {
		t.Fatal(err)
	}

	log.Info("Checking IPAM Pool 2 allocation on third cluster...")
	if err := checkAllocation(ctx, log, seedClient, userClient3, cluster3, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   credentials.KKPDatacenter,
		CIDR: "192.169.1.8/29",
	}); err != nil {
		t.Fatal(err)
	}

	log.Info("Deleting IPAM Pool 1...")
	if err := seedClient.Delete(ctx, ipamPool1); err != nil {
		t.Fatalf("Failed to delete IPAM Pool 1: %v", err)
	}

	if err := checkAllocationIsGone(ctx, log, seedClient, userClient2, cluster2, ipamPool1.Name); err != nil {
		t.Fatal(err)
	}
	if err := checkAllocationIsGone(ctx, log, seedClient, userClient3, cluster3, ipamPool1.Name); err != nil {
		t.Fatal(err)
	}

	log.Info("Deleting second cluster...")
	cleanupUserCluster2()

	if err := checkAllocationIsGone(ctx, log, seedClient, userClient2, cluster2, ipamPool2.Name); err != nil {
		t.Fatal(err)
	}

	log.Info("Deleting third cluster...")
	cleanupUserCluster3()

	if err := checkAllocationIsGone(ctx, log, seedClient, userClient3, cluster3, ipamPool2.Name); err != nil {
		t.Fatal(err)
	}
}

func createUserCluster(
	ctx context.Context,
	t *testing.T,
	log *zap.SugaredLogger,
	seedClient ctrlruntimeclient.Client,
	name string,
) (*kubermaticv1.Cluster, ctrlruntimeclient.Client, func(), error) {
	testJig := jig.NewAWSCluster(seedClient, log, credentials, 1, nil)
	testJig.ProjectJig.WithHumanReadableName("IPAM test")
	testJig.ClusterJig.
		WithTestName(name).
		WithAddons(jig.Addon{
			Name: "metallb",
			Labels: map[string]string{
				"addons.kubermatic.io/ensure": "true",
			},
		})

	cleanup := func() {
		testJig.Cleanup(ctx, t, true)
	}

	_, cluster, err := testJig.Setup(ctx, jig.WaitForReadyPods)
	if err != nil {
		return nil, nil, cleanup, fmt.Errorf("failed to setup test cluster: %w", err)
	}

	clusterClient, err := testJig.ClusterJig.ClusterClient(ctx)
	if err != nil {
		return nil, nil, cleanup, fmt.Errorf("failed to get user cluster client: %w", err)
	}

	return cluster, clusterClient, cleanup, err
}

func createNewIPAMPool(ctx context.Context, seedClient ctrlruntimeclient.Client, ipamPoolName string, datacenters map[string]kubermaticv1.IPAMPoolDatacenterSettings) (*kubermaticv1.IPAMPool, error) {
	if ipamPoolName == "" {
		ipamPoolName = rand.String(10)
	}

	if err := seedClient.Create(ctx, &kubermaticv1.IPAMPool{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ipamPoolName,
			Labels: map[string]string{},
		},
		Spec: kubermaticv1.IPAMPoolSpec{
			Datacenters: datacenters,
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

func updateIPAMPool(ctx context.Context, seedClient ctrlruntimeclient.Client, ipamPoolName string, datacenters map[string]kubermaticv1.IPAMPoolDatacenterSettings) error {
	ipamPool := &kubermaticv1.IPAMPool{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: ipamPoolName}, ipamPool); err != nil {
		return fmt.Errorf("failed to get IPAM pool: %w", err)
	}

	newIPAMPool := ipamPool.DeepCopy()
	newIPAMPool.Spec.Datacenters = datacenters

	if err := seedClient.Patch(ctx, newIPAMPool, ctrlruntimeclient.MergeFrom(ipamPool)); err != nil {
		return fmt.Errorf("failed to update IPAM pool: %w", err)
	}

	return nil
}

func checkAllocation(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string, expectedIPAMAllocationSpec kubermaticv1.IPAMAllocationSpec) error {
	if !checkIPAMAllocation(ctx, log, seedClient, userClient, cluster, ipamAllocationName, expectedIPAMAllocationSpec) {
		return fmt.Errorf("IPAM Allocation %s wasn't created on cluster %s", ipamAllocationName, cluster.Name)
	}

	if ipamAllocationName == "metallb" {
		ipamAllocation := &kubermaticv1.IPAMAllocation{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: ipamAllocationName, Namespace: cluster.Status.NamespaceName}, ipamAllocation); err != nil {
			return err
		}

		if !checkMetallbIPAddressPool(ctx, log, userClient, cluster, ipamAllocation) {
			return fmt.Errorf("metallb IP address pool for IPAM Allocation %s was not created properly on cluster %s", ipamAllocationName, cluster.Name)
		}
	}

	return nil
}

func checkIPAMAllocation(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string, expectedIPAMAllocationSpec kubermaticv1.IPAMAllocationSpec) bool {
	return wait.PollLog(ctx, log, 10*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		ipamAllocation := &kubermaticv1.IPAMAllocation{}
		if err := seedClient.Get(ctx, types.NamespacedName{Name: ipamAllocationName, Namespace: cluster.Status.NamespaceName}, ipamAllocation); err != nil {
			return fmt.Errorf("error getting IPAM allocation for cluster %s: %w", cluster.Name, err), nil
		}

		if ipamAllocation.Spec.Type != expectedIPAMAllocationSpec.Type ||
			ipamAllocation.Spec.CIDR != expectedIPAMAllocationSpec.CIDR ||
			ipamAllocation.Spec.DC != expectedIPAMAllocationSpec.DC ||
			len(ipamAllocation.Spec.Addresses) != len(expectedIPAMAllocationSpec.Addresses) {
			return fmt.Errorf("expected IPAM allocation: %+v\nActual IPAM allocation: %+v", expectedIPAMAllocationSpec, ipamAllocation.Spec), nil
		}

		for i, address := range ipamAllocation.Spec.Addresses {
			if address != expectedIPAMAllocationSpec.Addresses[i] {
				return fmt.Errorf("expected IPAM allocation: %+v\nActual IPAM allocation: %+v", expectedIPAMAllocationSpec, ipamAllocation.Spec), nil
			}
		}

		return nil, nil
	}) == nil
}

type ipAddressPoolV1Beta1 struct {
	Spec struct {
		Addresses []string `json:"addresses"`
	} `json:"spec"`
}

func checkMetallbIPAddressPool(ctx context.Context, log *zap.SugaredLogger, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocation *kubermaticv1.IPAMAllocation) bool {
	return wait.PollLog(ctx, log, 20*time.Second, 10*time.Minute, func(ctx context.Context) (error, error) {
		// We use an unstructured object instead of using metallb's v1beta1 directly
		// because Cilium has its own fork of metallb which does not contain v1beta1.
		metallbIPAddressPool := &unstructured.Unstructured{}
		metallbIPAddressPool.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "metallb.io",
			Version: "v1beta1",
			Kind:    "IPAddressPool",
		})
		if err := userClient.Get(ctx, types.NamespacedName{Name: "kkp-managed-pool", Namespace: "metallb-system"}, metallbIPAddressPool); err != nil {
			return fmt.Errorf("error getting metallb IPAddressPool in user cluster %s: %w", cluster.Name, err), nil
		}

		pool := &ipAddressPoolV1Beta1{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(metallbIPAddressPool.Object, pool); err != nil {
			return fmt.Errorf("failed to decode metallb IPAddressPool: %w", err), nil
		}

		switch ipamAllocation.Spec.Type {
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			if len(pool.Spec.Addresses) != 1 {
				return fmt.Errorf("metallb ip address pool: no single address for IPAM allocation type \"prefix\""), nil
			}
			if pool.Spec.Addresses[0] != string(ipamAllocation.Spec.CIDR) {
				return fmt.Errorf("metallb ip address pool: not expected CIDR for IPAM allocation type \"prefix\": \"%s\" (expected \"%s\")", pool.Spec.Addresses[0], ipamAllocation.Spec.CIDR), nil
			}
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			if len(pool.Spec.Addresses) != len(ipamAllocation.Spec.Addresses) {
				return fmt.Errorf("metallb ip address pool: not expected number of addresses for IPAM allocation type \"range\": %d (expected %d)", len(pool.Spec.Addresses), len(ipamAllocation.Spec.Addresses)), nil
			}
			for i, address := range pool.Spec.Addresses {
				if address != ipamAllocation.Spec.Addresses[i] {
					return fmt.Errorf("metallb ip address pool: not expected address range: \"%s\" (expected \"%s\")", address, ipamAllocation.Spec.Addresses[i]), nil
				}
			}
		}

		return nil, nil
	}) == nil
}

func checkAllocationIsGone(ctx context.Context, log *zap.SugaredLogger, seedClient, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string) error {
	if !checkIPAMAllocationIsGone(ctx, log, seedClient, userClient, cluster, ipamAllocationName) {
		return fmt.Errorf("IPAM Allocation %s in cluster %s is still persisted", ipamAllocationName, cluster.Name)
	}

	return nil
}

func checkIPAMAllocationIsGone(ctx context.Context, log *zap.SugaredLogger, seedClient, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string) bool {
	return wait.PollLog(ctx, log, 10*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		ipamAllocation := &kubermaticv1.IPAMAllocation{}
		err := seedClient.Get(ctx, types.NamespacedName{Name: ipamAllocationName, Namespace: cluster.Status.NamespaceName}, ipamAllocation)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}

			return fmt.Errorf("error getting IPAM allocation for cluster %s: %w", cluster.Name, err), nil
		}

		return fmt.Errorf("IPAM allocation is not gone for cluster %s", cluster.Name), nil
	}) == nil
}
