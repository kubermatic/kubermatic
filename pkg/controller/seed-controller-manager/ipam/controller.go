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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_ipam_controller"
)

// Reconciler stores all components required for the IPAM controller.
type Reconciler struct {
	ctrlruntimeclient.Client

	workerName   string
	log          *zap.SugaredLogger
	configGetter provider.KubermaticConfigurationGetter
	recorder     record.EventRecorder
	versions     kubermatic.Versions
}

// Add creates a new IPAM controller
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	configGetter provider.KubermaticConfigurationGetter,
	versions kubermatic.Versions,
) error {
	log = log.Named(ControllerName)

	reconciler := &Reconciler{
		Client:       mgr.GetClient(),
		workerName:   workerName,
		log:          log,
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		configGetter: configGetter,
		versions:     versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: 1, // this should be 1 to avoid race condition
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.IPAMPool{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	ipamPool := &kubermaticv1.IPAMPool{}
	if err := r.Get(ctx, request.NamespacedName, ipamPool); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Skipping because the IPAM pool is already gone")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get IPAM pool: %w", err)
	}

	result, err := r.reconcile(ctx, ipamPool)
	if err != nil {
		log.Errorw("Failed to reconcile IPAM pool", zap.Error(err))
		r.recorder.Event(ipamPool, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return result, err
}

func (r *Reconciler) reconcile(ctx context.Context, ipamPool *kubermaticv1.IPAMPool) (reconcile.Result, error) {
	dcIPAMPoolUsageMap, err := r.compileCurrentAllocationsForPool(ctx, ipamPool)
	if err != nil {
		return reconcile.Result{}, err
	}

	newClustersAllocations, err := r.generateNewAllocationsForPool(ctx, ipamPool, dcIPAMPoolUsageMap)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Apply the new clusters allocations
	for _, newClusterAllocation := range newClustersAllocations {
		// TODO
		fmt.Println(newClusterAllocation)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) compileCurrentAllocationsForPool(ctx context.Context, ipamPool *kubermaticv1.IPAMPool) (datacenterIPAMPoolUsageMap, error) {
	dcIPAMPoolUsageMap := newDatacenterIPAMPoolUsageMap()

	// List all IPAM allocations
	ipamAllocationList := &kubermaticv1.IPAMAllocationList{}
	err := r.Client.List(ctx, ipamAllocationList)
	if err != nil {
		return nil, fmt.Errorf("failed to list IPAM allocations: %w", err)
	}

	// Iterate current IPAM allocations to build a map of used IPs (for range allocation type)
	// or used subnets (for prefix allocation type) per datacenter pool
	for _, ipamAllocation := range ipamAllocationList.Items {
		dcIPAMPoolCfg, isDCConfigured := ipamPool.Spec.Datacenters[ipamAllocation.Spec.DC]
		if !isDCConfigured || ipamAllocation.Name != ipamPool.Name {
			// IPAM Pool + Datacenter is not configured in the IPAM pool spec, so we can skip it
			continue
		}

		switch ipamAllocation.Spec.Type {
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			currentAllocatedIPs, err := getUsedIPsFromAddressRanges(ipamAllocation.Spec.Addresses)
			if err != nil {
				return nil, err
			}
			// check if the current allocation is compatible with the IPAMPool being applied
			err = checkRangeAllocation(currentAllocatedIPs, string(dcIPAMPoolCfg.PoolCIDR), int(dcIPAMPoolCfg.AllocationRange))
			if err != nil {
				return nil, err
			}
			for _, ip := range currentAllocatedIPs {
				dcIPAMPoolUsageMap.setUsed(ipamAllocation.Spec.DC, ip)
			}
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			// check if the current allocation is compatible with the IPAMPool being applied
			err := checkPrefixAllocation(string(ipamAllocation.Spec.CIDR), string(dcIPAMPoolCfg.PoolCIDR), int(dcIPAMPoolCfg.AllocationPrefix))
			if err != nil {
				return nil, err
			}
			dcIPAMPoolUsageMap.setUsed(ipamAllocation.Spec.DC, string(ipamAllocation.Spec.CIDR))
		}
	}

	return dcIPAMPoolUsageMap, nil
}

func (r *Reconciler) generateNewAllocationsForPool(ctx context.Context, ipamPool *kubermaticv1.IPAMPool, dcIPAMPoolUsageMap datacenterIPAMPoolUsageMap) ([]kubermaticv1.IPAMAllocation, error) {
	newClustersAllocations := []kubermaticv1.IPAMAllocation{}

	// List and loop clusters for the new allocations
	clusterList := &kubermaticv1.ClusterList{}
	err := r.Client.List(ctx, clusterList)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	for _, cluster := range clusterList.Items {
		dc := cluster.Spec.Cloud.DatacenterName
		dcIPAMPoolCfg, isDCConfigured := ipamPool.Spec.Datacenters[dc]
		if !isDCConfigured {
			// Cluster datacenter is not configured in the IPAM pool spec, so nothing to do for it
			continue
		}

		allocationNamespace := fmt.Sprintf("cluster-%s", cluster.Name)

		ipamAllocation := &kubermaticv1.IPAMAllocation{}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: allocationNamespace, Name: ipamPool.Name}, ipamAllocation)
		if err == nil {
			// skip because pool is already allocated for cluster
			continue
		} else if !apierrors.IsNotFound(err) {
			return nil, err
		}

		newClustersAllocation := kubermaticv1.IPAMAllocation{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: allocationNamespace,
				Name:      ipamPool.Name,
			},
			Spec: kubermaticv1.IPAMAllocationSpec{
				Type: dcIPAMPoolCfg.Type,
				DC:   dc,
			},
		}

		switch dcIPAMPoolCfg.Type {
		case kubermaticv1.IPAMPoolAllocationTypeRange:
			addresses, err := findFirstFreeRangesOfPool(dc, string(dcIPAMPoolCfg.PoolCIDR), int(dcIPAMPoolCfg.AllocationRange), dcIPAMPoolUsageMap)
			if err != nil {
				return nil, err
			}
			newClustersAllocation.Spec.Addresses = addresses
		case kubermaticv1.IPAMPoolAllocationTypePrefix:
			subnetCIDR, err := findFirstFreeSubnetOfPool(dc, string(dcIPAMPoolCfg.PoolCIDR), int(dcIPAMPoolCfg.AllocationPrefix), dcIPAMPoolUsageMap)
			if err != nil {
				return nil, err
			}
			newClustersAllocation.Spec.CIDR = kubermaticv1.SubnetCIDR(subnetCIDR)
		}

		newClustersAllocations = append(newClustersAllocations, newClustersAllocation)
	}

	return newClustersAllocations, nil
}
