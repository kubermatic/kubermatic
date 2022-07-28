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
	metallbv1beta1 "go.universe.tf/metallb/api/v1beta1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	logOptions = log.NewDefaultOptions()
)

func init() {
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestIPAM(t *testing.T) {
	ctx := context.Background()
	log := log.NewFromOptions(logOptions).Sugar()

	seedClient, _, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	clusterClientProvider, err := clusterclient.NewExternal(seedClient)
	if err != nil {
		t.Fatalf("failed to get user cluster client provider: %v", err)
	}

	// prepare jig, create projecta nd the first cluster
	testJig1 := jig.NewBYOCluster(seedClient, log)
	testJig1.ProjectJig.WithHumanReadableName("IPAM test")
	testJig1.ClusterJig.WithAddons("metallb")

	_, cluster1, err := testJig1.Setup(ctx, jig.WaitForReadyPods)
	if err != nil {
		t.Fatalf("failed to setup first test cluster: %v", err)
	}
	defer testJig1.Cleanup(ctx, t, false)

	// get the user client
	userClient1, err := clusterClientProvider.GetClient(ctx, cluster1)
	if err != nil {
		t.Fatalf("failed to get user cluster client: %v", err)
	}
	utilruntime.Must(metallbv1beta1.AddToScheme(userClient1.Scheme()))

	log.Info("Creating first IPAM Pool...")
	ipamPool1, err := createNewIPAMPool(ctx, seedClient, "192.168.1.0/28", "range", 8)
	if err != nil {
		t.Fatal(err.Error())
	}

	// TODO: test block, should be deleted {
	addon := &kubermaticv1.Addon{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: "metallb", Namespace: cluster1.Status.NamespaceName}, addon); err != nil {
		t.Fatal(err.Error())
	}
	log.Infof("MetalLB addon: %+v", addon)
	//}

	log.Info("Checking IPAM Pool 1 allocation on first cluster...")
	if !checkIPAMAllocation(ctx, log, seedClient, userClient1, cluster1, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        jig.DatacenterName(),
		Addresses: []string{"192.168.1.0-192.168.1.7"},
	}) {
		t.Fatal("IPAM Allocation 1 wasn't created on cluster 1")
	}

	log.Info("Creating second IPAM Pool...")
	ipamPool2, err := createNewIPAMPool(ctx, seedClient, "192.168.1.0/27", "prefix", 28)
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Info("Checking IPAM Pool 2 allocation on first cluster...")
	if !checkIPAMAllocation(ctx, log, seedClient, userClient1, cluster1, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   jig.DatacenterName(),
		CIDR: "192.168.1.0/28",
	}) {
		t.Fatal("IPAM Allocation 2 wasn't created on cluster 1")
	}

	log.Info("Creating second cluster...")
	testJig2 := jig.NewBYOCluster(seedClient, log)
	testJig2.ProjectJig = testJig1.ProjectJig // stay in the same project
	testJig2.ClusterJig.WithAddons("metallb")

	_, cluster2, err := testJig2.Setup(ctx, jig.WaitForReadyPods)
	if err != nil {
		t.Fatalf("failed to setup second test cluster: %v", err)
	}
	defer testJig2.Cleanup(ctx, t, false)

	// get the user client
	userClient2, err := clusterClientProvider.GetClient(ctx, cluster2)
	if err != nil {
		t.Fatalf("failed to get user cluster client: %v", err)
	}
	utilruntime.Must(metallbv1beta1.AddToScheme(userClient2.Scheme()))

	log.Info("Checking IPAM Pool 1 allocation on second cluster...")
	if !checkIPAMAllocation(ctx, log, seedClient, userClient2, cluster2, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        jig.DatacenterName(),
		Addresses: []string{"192.168.1.8-192.168.1.15"},
	}) {
		t.Fatal("IPAM Allocation 1 wasn't created on cluster 2")
	}

	log.Info("Checking IPAM Pool 2 allocation on second cluster...")
	if !checkIPAMAllocation(ctx, log, seedClient, userClient2, cluster2, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   jig.DatacenterName(),
		CIDR: "192.168.1.16/28",
	}) {
		t.Fatal("IPAM Allocation 2 wasn't created on cluster 2")
	}

	log.Info("Deleting first cluster...")
	if err := testJig1.ClusterJig.Delete(ctx, true); err != nil {
		t.Fatalf("Failed to delete first cluster: %v", err)
	}

	log.Info("Checking that the first cluster allocations were gone...")
	if !checkIPAMAllocationIsGone(ctx, log, seedClient, userClient1, cluster1, ipamPool1.Name) ||
		!checkIPAMAllocationIsGone(ctx, log, seedClient, userClient1, cluster1, ipamPool2.Name) {
		t.Fatal("IPAM Allocations in first cluster are still persisted")
	}

	log.Info("Creating third IPAM Pool...")
	ipamPool3, err := createNewIPAMPool(ctx, seedClient, "193.169.1.0/28", "prefix", 29)
	if err != nil {
		t.Fatal(err.Error())
	}

	log.Info("Checking IPAM Pool 3 allocation on second cluster...")
	if !checkIPAMAllocation(ctx, log, seedClient, userClient2, cluster2, ipamPool3.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   jig.DatacenterName(),
		CIDR: "193.169.1.0/29",
	}) {
		t.Fatalf("IPAM Allocation 3 wasn't created on cluster 2")
	}

	log.Info("Creating third cluster...")
	testJig3 := jig.NewBYOCluster(seedClient, log)
	testJig3.ProjectJig = testJig1.ProjectJig // stay in the same project
	testJig3.ClusterJig.WithAddons("metallb")

	_, cluster3, err := testJig3.Setup(ctx, jig.WaitForReadyPods)
	if err != nil {
		t.Fatalf("failed to setup second test cluster: %v", err)
	}
	defer testJig3.Cleanup(ctx, t, false)

	// get the user client
	userClient3, err := clusterClientProvider.GetClient(ctx, cluster3)
	if err != nil {
		t.Fatalf("failed to get user cluster client: %v", err)
	}
	utilruntime.Must(metallbv1beta1.AddToScheme(userClient3.Scheme()))

	log.Info("Checking IPAM Pool 3 allocation on third cluster...")
	if !checkIPAMAllocation(ctx, log, seedClient, userClient3, cluster3, ipamPool3.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   jig.DatacenterName(),
		CIDR: "193.169.1.8/29",
	}) {
		t.Fatal("IPAM Allocation 3 wasn't created on cluster 3")
	}

	log.Info("Checking IPAM Pool 1 allocation on third cluster...")
	if !checkIPAMAllocation(ctx, log, seedClient, userClient3, cluster3, ipamPool1.Name, kubermaticv1.IPAMAllocationSpec{
		Type:      "range",
		DC:        jig.DatacenterName(),
		Addresses: []string{"192.168.1.0-192.168.1.7"},
	}) {
		t.Fatal("IPAM Allocation 1 wasn't created on cluster 3")
	}

	log.Info("Checking IPAM Pool 2 allocation on third cluster...")
	if !checkIPAMAllocation(ctx, log, seedClient, userClient3, cluster3, ipamPool2.Name, kubermaticv1.IPAMAllocationSpec{
		Type: "prefix",
		DC:   jig.DatacenterName(),
		CIDR: "192.168.1.0/28",
	}) {
		t.Fatal("IPAM Allocation 2 wasn't created on cluster 3")
	}

	log.Info("Deleting IPAM Pool 1...")
	if err := seedClient.Delete(ctx, ipamPool1); err != nil {
		t.Fatalf("Failed to delete IPAM Pool 1: %v", err)
	}

	if !checkIPAMAllocationIsGone(ctx, log, seedClient, userClient2, cluster2, ipamPool1.Name) ||
		!checkIPAMAllocationIsGone(ctx, log, seedClient, userClient3, cluster3, ipamPool1.Name) {
		t.Fatal("Some IPAM Allocation is still persisted after IPAM Pool 1 was deleted")
	}

	// no cloud provider resources were involved, so we do not need to wait for cleanup
	if err := testJig2.ClusterJig.Delete(ctx, false); err != nil {
		t.Fatalf("Failed to delete second cluster: %v", err)
	}

	if err := testJig3.ClusterJig.Delete(ctx, false); err != nil {
		t.Fatalf("Failed to delete third cluster: %v", err)
	}

	if !checkIPAMAllocationIsGone(ctx, log, seedClient, userClient2, cluster2, ipamPool2.Name) ||
		!checkIPAMAllocationIsGone(ctx, log, seedClient, userClient2, cluster2, ipamPool3.Name) ||
		!checkIPAMAllocationIsGone(ctx, log, seedClient, userClient3, cluster3, ipamPool2.Name) ||
		!checkIPAMAllocationIsGone(ctx, log, seedClient, userClient3, cluster3, ipamPool3.Name) {
		t.Fatal("Some IPAM Allocation is still persisted")
	}
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
				jig.DatacenterName(): {
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

func checkIPAMAllocation(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string, expectedIPAMAllocationSpec kubermaticv1.IPAMAllocationSpec) bool {
	return wait.PollLog(log, 10*time.Second, 5*time.Minute, func() (error, error) {
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

		if !checkMetallbIPAddressPool(ctx, log, userClient, cluster, ipamAllocation) {
			return fmt.Errorf("metallb IP address pool was not created properly"), nil
		}

		return nil, nil
	}) == nil
}

func checkMetallbIPAddressPool(ctx context.Context, log *zap.SugaredLogger, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocation *kubermaticv1.IPAMAllocation) bool {
	return wait.PollLog(log, 10*time.Second, 5*time.Minute, func() (error, error) {
		dep := &appsv1.Deployment{}
		if err := userClient.Get(ctx, types.NamespacedName{Name: "dashboard-metrics-scraper", Namespace: "kubernetes-dashboard"}, dep); err != nil {
			log.Info(fmt.Errorf("!!! error getting dashboard-metrics-scraper Deployment in user cluster %s: %w", cluster.Name, err))
		}

		dep2 := &appsv1.Deployment{}
		if err := userClient.Get(ctx, types.NamespacedName{Name: "coredns", Namespace: "kube-system"}, dep2); err != nil {
			log.Info(fmt.Errorf("!!! error getting coredns Deployment in user cluster %s: %w", cluster.Name, err))
		}

		dep3 := &appsv1.Deployment{}
		if err := userClient.Get(ctx, types.NamespacedName{Name: "controller", Namespace: "metallb-system"}, dep3); err != nil {
			log.Info(fmt.Errorf("!!! error getting metallb Deployment in user cluster %s: %w", cluster.Name, err))
		}
		log.Infof("MetalLB controller deployment: %+v", dep3)

		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := userClient.Get(ctx, types.NamespacedName{Name: "ipaddresspools.metallb.io"}, crd); err != nil {
			log.Info(fmt.Errorf("!!! error getting metallb IPAddressPool CRD in user cluster %s: %w", cluster.Name, err))
		}
		log.Infof("MetalLB IPAddressPool CRD: %+v", crd)

		metallbIPAddressPool := &metallbv1beta1.IPAddressPool{}
		if err := userClient.Get(ctx, types.NamespacedName{Name: ipamAllocation.Name, Namespace: "metallb-system"}, metallbIPAddressPool); err != nil {
			return fmt.Errorf("error getting metallb IPAddressPool in user cluster %s: %w", cluster.Name, err), nil
		}

		switch ipamAllocation.Spec.Type {
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			if len(metallbIPAddressPool.Spec.Addresses) != 1 {
				return fmt.Errorf("metallb ip address pool: no single address for IPAM allocation type \"prefix\""), nil
			}
			if metallbIPAddressPool.Spec.Addresses[0] != string(ipamAllocation.Spec.CIDR) {
				return fmt.Errorf("metallb ip address pool: not expected CIDR for IPAM allocation type \"prefix\": \"%s\" (expected \"%s\")", metallbIPAddressPool.Spec.Addresses[0], ipamAllocation.Spec.CIDR), nil
			}
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			if len(metallbIPAddressPool.Spec.Addresses) != len(ipamAllocation.Spec.Addresses) {
				return fmt.Errorf("metallb ip address pool: not expected number of addresses for IPAM allocation type \"range\": %d (expected %d)", len(metallbIPAddressPool.Spec.Addresses), len(ipamAllocation.Spec.Addresses)), nil
			}
			for i, address := range metallbIPAddressPool.Spec.Addresses {
				if address != ipamAllocation.Spec.Addresses[i] {
					return fmt.Errorf("metallb ip address pool: not expected address range: \"%s\" (expected \"%s\")", address, ipamAllocation.Spec.Addresses[i]), nil
				}
			}
		}

		return nil, nil
	}) == nil
}

func checkIPAMAllocationIsGone(ctx context.Context, log *zap.SugaredLogger, seedClient, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string) bool {
	return wait.PollLog(log, 10*time.Second, 5*time.Minute, func() (error, error) {
		ipamAllocation := &kubermaticv1.IPAMAllocation{}
		err := seedClient.Get(ctx, types.NamespacedName{Name: ipamAllocationName, Namespace: cluster.Status.NamespaceName}, ipamAllocation)
		if err != nil {
			if apierrors.IsNotFound(err) {
				if !checkMetallbIPAddressPoolIsGone(ctx, log, userClient, cluster, ipamAllocationName) {
					return fmt.Errorf("metallb IP address pool is still persisted"), nil
				}
				return nil, nil
			}

			return fmt.Errorf("error getting IPAM allocation for cluster %s: %w", cluster.Name, err), nil
		}

		return fmt.Errorf("IPAM allocation is not gone for cluster %s", cluster.Name), nil
	}) == nil
}

func checkMetallbIPAddressPoolIsGone(ctx context.Context, log *zap.SugaredLogger, userClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, ipamAllocationName string) bool {
	return wait.PollLog(log, 10*time.Second, 5*time.Minute, func() (error, error) {
		metallbIPAddressPool := &metallbv1beta1.IPAddressPool{}
		err := userClient.Get(ctx, types.NamespacedName{Name: ipamAllocationName, Namespace: "metallb-system"}, metallbIPAddressPool)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}

			return fmt.Errorf("error getting metallb IP address pool for cluster %s: %w", cluster.Name, err), nil
		}

		return fmt.Errorf("metallb IP address pool is not gone for cluster %s", cluster.Name), nil
	}) == nil
}
