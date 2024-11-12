/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package kubevirtnetworkcontroller

import (
	"context"

	kubeovnv1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	"go.uber.org/zap"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName      = "kubevirt-network-controller"
	WorkloadSubnetLabel = "k8c.io/kubevirt-workload-subnet"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter

	log        *zap.SugaredLogger
	versions   kubermatic.Versions
	workerName string
	recorder   record.EventRecorder
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,

	numWorkers int,
	workerName string,

	seedGetter provider.SeedGetter,
	configGetter provider.KubermaticConfigurationGetter,

	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		log:          log.Named(ControllerName),
		Client:       mgr.GetClient(),
		seedGetter:   seedGetter,
		configGetter: configGetter,
		workerName:   workerName,
		recorder:     mgr.GetEventRecorderFor(ControllerName),
	}
	enqueueCluster := controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())
	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}).
		Watches(&kubeovnv1.Subnet{}, enqueueCluster, builder.WithPredicates(predicateutil.ByLabelExists(WorkloadSubnetLabel))).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	seed, err := r.seedGetter()
	if err != nil {
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionKubeVirtNetworkControllerSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster, seed)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, seed *kubermaticv1.Seed) (*reconcile.Result, error) {
	var subnets kubeovnv1.SubnetList
	gateways := make(map[string]string, len(subnets.Items))
	cidrs := make(map[string]string, len(subnets.Items))
	if err := r.List(ctx, &subnets, ctrlruntimeclient.MatchingLabels{
		WorkloadSubnetLabel: cluster.Name,
	}); err != nil {
		return &reconcile.Result{}, err
	}
	for _, subnet := range subnets.Items {
		gateways[subnet.Name] = subnet.Spec.Gateway
		cidrs[subnet.Name] = subnet.Spec.CIDRBlock
	}

	// TODO use gateways and cidrs

	if err := reconcileNamespacedClusterIsolationNetworkPolicy(ctx, r.Client, cluster, cidrs, gateways, cluster.Status.NamespaceName); err != nil {
		return &reconcile.Result{}, err
	}

	return nil, nil
}
