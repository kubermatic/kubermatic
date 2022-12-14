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

package kubevirtvmievictioncontroller

import (
	"context"
	"fmt"

	"github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"go.uber.org/zap"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v2/pkg/resources"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	kubevirtv1 "kubevirt.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName   = "kubevirt-vmi-eviction-controller"
	machineNamespace = "kube-system"
)

type reconciler struct {
	log             *zap.SugaredLogger
	infraClient     ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker, clusterName string) error {

	seedClient := seedMgr.GetClient()
	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to get cluster %q: %w", clusterName, err)
	}

	infraClient, err := getInfraClientForCluster(ctx, seedClient, cluster)
	if err != nil {
		return fmt.Errorf("failed creating kubevirt infra client: %w", err)
	}

	r := &reconciler{
		log:             log.Named(controllerName),
		infraClient:     infraClient,
		userClient:      userMgr.GetClient(),
		clusterIsPaused: clusterIsPaused,
	}

	c, err := controller.New(controllerName, userMgr, controller.Options{Reconciler: r})

	namespacePredicate := predicate.ByNamespace(cluster.Status.NamespaceName)

	if err = c.Watch(&source.Kind{Type: &kubevirtv1.VirtualMachineInstance{}}, &handler.EnqueueRequestForObject{}, namespacePredicate); err != nil {
		return fmt.Errorf("failed creating watch for kubevirt VirtualMachineInstance: %w", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("kubevirt-vmi-eviction", request)
	log.Debug("Processing")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	vmi := &kubevirtv1.VirtualMachineInstance{}
	if err = r.infraClient.Get(ctx, request.NamespacedName, vmi); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("VirtualMachineInstance not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed getting VirtualMachineInstance: %w", err)
	}

	err = r.reconcile(ctx, log, vmi)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, vmi *kubevirtv1.VirtualMachineInstance) error {

	// If `status.evacuationNodeName` is set, trigger graceful delete of the Machine linked to this VMI
	if vmi.Status.EvacuationNodeName != "" {
		namepacedMachineName := types.NamespacedName{Name: vmi.Name, Namespace: machineNamespace}
		machine := &v1alpha1.Machine{}
		if err := r.userClient.Get(ctx, namepacedMachineName, machine); err != nil {
			if apierrors.IsNotFound(err) {
				log.Debugf("Machine %q already gone. Nothing to do here.")
				return nil
			} else {
				return fmt.Errorf("failed getting Machine %q: %w", vmi.Name, err)
			}
		}

		if err := r.userClient.Delete(ctx, machine); err != nil {
			return fmt.Errorf("failed deleting Machine %q: %w", vmi.Name, err)
		}
	}

	return nil
}

func getInfraClientForCluster(ctx context.Context, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (*kubevirt.Client, error) {
	data := resources.NewTemplateDataBuilder().WithCluster(cluster).Build()

	credentials, err := resources.GetCredentials(data)
	if err != nil {
		return nil, fmt.Errorf("failed getting credentials: %w", err)
	}
	infraKubeconfig := credentials.Kubevirt.KubeConfig
	return kubevirt.NewClient(infraKubeconfig, kubevirt.ClientOptions{})
}
