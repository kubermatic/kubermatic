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

package kubevirtvmieviction

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/zap"
	kubevirtv1 "kubevirt.io/api/core/v1"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName   = "kkp-kubevirt-vmi-eviction"
	machineNamespace = "kube-system"
)

func init() {
	if err := kubevirtv1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalf("failed to add kubevirtv1 to scheme: %v", err)
	}
	if err := clusterv1alpha1.AddToScheme(scheme.Scheme); err != nil {
		log.Fatalf("failed to add kubevirtv1 to scheme: %v", err)
	}
}

type reconciler struct {
	log             *zap.SugaredLogger
	userClient      ctrlruntimeclient.Client
	infraClient     ctrlruntimeclient.Client
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(ctx context.Context, log *zap.SugaredLogger, userMgr, kubevirtInfraMgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker, clusterName string) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		userClient:      userMgr.GetClient(),
		infraClient:     kubevirtInfraMgr.GetClient(),
		clusterIsPaused: clusterIsPaused,
	}

	_, err := builder.ControllerManagedBy(userMgr).
		Named(controllerName).
		WatchesRawSource(source.Kind(
			kubevirtInfraMgr.GetCache(),
			&kubevirtv1.VirtualMachineInstance{},
			&handler.TypedEnqueueRequestForObject[*kubevirtv1.VirtualMachineInstance]{},
			kubermaticpred.TypedFactory(func(vmi *kubevirtv1.VirtualMachineInstance) bool {
				return vmi.Status.EvacuationNodeName != ""
			}),
		)).
		Build(r)

	return err
}

// Reconcile:
// - watches the VMI status.evacuationNodeName in the KubeVirt infra cluster.
// - deletes the corresponding Machine in the user cluster.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("kvvmieviction", request)

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		log.Info("Cluster paused. no reconcile")
		return reconcile.Result{}, nil
	}

	vmi := &kubevirtv1.VirtualMachineInstance{}
	if err := r.infraClient.Get(ctx, request.NamespacedName, vmi); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("VirtualMachineInstance not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed getting VirtualMachineInstance: %w", err)
	}

	err = r.deleteMachine(ctx, log, vmi)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed deleting Machine: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) deleteMachine(ctx context.Context, log *zap.SugaredLogger, vmi *kubevirtv1.VirtualMachineInstance) error {
	machine := &clusterv1alpha1.Machine{}

	// No need to check on Status.EvictionNodeName as it's already filtered out by the Predicate
	if err := r.userClient.Get(ctx, types.NamespacedName{Name: vmi.Name, Namespace: machineNamespace}, machine); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debugf("Machine %s already gone. Nothing to do here.", vmi.Name)
			return nil
		} else {
			return fmt.Errorf("failed getting Machine %s: %w", vmi.Name, err)
		}
	}

	if err := r.userClient.Delete(ctx, machine); err != nil {
		return fmt.Errorf("failed deleting Machine %q: %w", vmi.Name, err)
	}
	log.Infof("Machine deleted.")

	return nil
}
