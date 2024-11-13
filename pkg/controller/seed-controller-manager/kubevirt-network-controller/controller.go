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
	"errors"
	"fmt"

	kubeovnv1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	"go.uber.org/zap"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v2/pkg/resources"
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
	ControllerName                = "kubevirt-network-controller"
	WorkloadSubnetLabel           = "k8c.io/kubevirt-workload-subnet"
	NetworkPolicyPodSelectorLabel = "cluster.x-k8s.io/cluster-name"
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

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
			cluster := o.(*kubermaticv1.Cluster)
			return cluster.Spec.Cloud.ProviderName == string(kubermaticv1.KubevirtCloudProvider)
		}))).
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

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return reconcile.Result{}, fmt.Errorf("couldn't find datacenter %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}

	// Check if the datacenter is kubevirt and if kubevirt is configured with NamespacedMode
	if datacenter.Spec.Kubevirt == nil || !datacenter.Spec.Kubevirt.NamespacedMode.Enabled {
		log.Debug("Skipping reconciliation as the datacenter is not kubevirt or kubevirt is not configured with NamespacedMode")
		return reconcile.Result{}, nil
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
			return r.reconcile(ctx, log, cluster, datacenter.Spec.Kubevirt)
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

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, dc *kubermaticv1.DatacenterSpecKubevirt) (*reconcile.Result, error) {
	var subnets kubeovnv1.SubnetList
	gateways := make([]string, 0, len(subnets.Items))
	cidrs := make([]string, 0, len(subnets.Items))
	if err := r.List(ctx, &subnets, ctrlruntimeclient.HasLabels{WorkloadSubnetLabel}); err != nil {
		return &reconcile.Result{}, err
	}
	for _, subnet := range subnets.Items {
		gateways = append(gateways, subnet.Spec.Gateway)
		cidrs = append(cidrs, subnet.Spec.CIDRBlock)
	}

	kubeVirtInfraClient, err := r.setupKubeVirtInfraClient(ctx, cluster)
	if err != nil {
		return &reconcile.Result{}, err
	}

	if err := reconcileNamespacedClusterIsolationNetworkPolicy(ctx, kubeVirtInfraClient, cluster, cidrs, gateways, dc.NamespacedMode.Namespace); err != nil {
		return &reconcile.Result{}, err
	}

	return nil, nil
}

func (r *Reconciler) setupKubeVirtInfraClient(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubevirt.Client, error) {
	kubeconfig, err := r.getKubeVirtInfraKConfig(ctx, cluster)
	if err != nil {
		return nil, err
	}

	kubeVirtInfraClient, err := kubevirt.NewClient(kubeconfig, kubevirt.ClientOptions{})
	if err != nil {
		return nil, err
	}

	return kubeVirtInfraClient, nil
}

func (r *Reconciler) getKubeVirtInfraKConfig(ctx context.Context, cluster *kubermaticv1.Cluster) (string, error) {
	if cluster.Spec.Cloud.Kubevirt.Kubeconfig != "" {
		return cluster.Spec.Cloud.Kubevirt.Kubeconfig, nil
	}

	if cluster.Spec.Cloud.Kubevirt.CredentialsReference == nil {
		return "", errors.New("no credentials provided")
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, r.Client)
	kubeconfig, err := secretKeySelectorFunc(cluster.Spec.Cloud.Kubevirt.CredentialsReference, resources.KubeVirtKubeconfig)
	if err != nil {
		return "", err
	}

	return kubeconfig, nil
}
