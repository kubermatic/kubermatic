/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package update

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_update_controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName                    string
	configGetter                  provider.KubermaticConfigurationGetter
	recorder                      record.EventRecorder
	userClusterConnectionProvider *client.Provider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

// Add creates a new update controller
func Add(mgr manager.Manager, numWorkers int, workerName string, configGetter provider.KubermaticConfigurationGetter,
	userClusterConnectionProvider *client.Provider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName:                    workerName,
		configGetter:                  configGetter,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionUpdateControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)
	if err != nil {
		r.log.Errorw("Failed to reconcile cluster", "namespace", request.NamespacedName.String(), zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if !cluster.Status.ExtendedHealth.AllHealthy() {
		// Cluster not healthy yet. Nothing to do.
		// If it gets healthy we'll get notified by the event. No need to requeue
		return nil, nil
	}

	config, err := r.configGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load KubermaticConfiguration: %w", err)
	}

	updateManager := version.NewFromConfiguration(config)

	// NodeUpdate may need the controlplane to be updated first
	updated, err := r.controlPlaneUpgrade(ctx, cluster, updateManager, v1.KubernetesClusterType)
	if err != nil {
		return nil, fmt.Errorf("failed to update the controlplane: %w", err)
	}
	// Give the controller time to do the update
	// TODO: This is not really safe. We should add a `Version` to the status
	// that gets incremented when the controller does this. Combined with a
	// `SeedResourcesUpToDate` condition, that should do the trick
	if updated {
		return &reconcile.Result{RequeueAfter: time.Minute}, nil
	}

	if err := r.nodeUpdate(ctx, cluster, updateManager, v1.KubernetesClusterType); err != nil {
		return nil, fmt.Errorf("failed to update machineDeployments: %v", err)
	}

	return nil, nil
}

func (r *Reconciler) nodeUpdate(ctx context.Context, cluster *kubermaticv1.Cluster, updateManager *version.Manager, clusterType string) error {
	c, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get usercluster client: %v", err)
	}

	machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
	// Kubermatic only creates MachineDeployments in the kube-system namespace, everything else is essentially unsupported
	if err := c.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace("kube-system")); err != nil {
		return fmt.Errorf("failed to list MachineDeployments: %v", err)
	}

	for _, md := range machineDeployments.Items {
		targetVersion, err := updateManager.AutomaticNodeUpdate(md.Spec.Template.Spec.Versions.Kubelet, clusterType, cluster.Spec.Version.String())
		if err != nil {
			return fmt.Errorf("failed to get automatic update for machinedeployment %s/%s that has version %q: %v", md.Namespace, md.Name, md.Spec.Template.Spec.Versions.Kubelet, err)
		}
		if targetVersion == nil {
			continue
		}
		md.Spec.Template.Spec.Versions.Kubelet = targetVersion.Version.String()
		// DeepCopy it so we don't get a NPD when we return an error
		if err := c.Update(ctx, md.DeepCopy()); err != nil {
			return fmt.Errorf("failed to update MachineDeployment %s/%s to %q: %v", md.Namespace, md.Name, md.Spec.Template.Spec.Versions.Kubelet, err)
		}
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "AutoUpdateMachineDeployment", "Triggered automatic update of MachineDeployment %s/%s to version %q", md.Namespace, md.Name, targetVersion.Version.String())
	}

	return nil
}

func (r *Reconciler) controlPlaneUpgrade(ctx context.Context, cluster *kubermaticv1.Cluster, updateManager *version.Manager, clusterType string) (upgraded bool, err error) {
	update, err := updateManager.AutomaticControlplaneUpdate(cluster.Spec.Version.String(), clusterType)
	if err != nil {
		return false, fmt.Errorf("failed to get automatic update for cluster for version %s: %v", cluster.Spec.Version.String(), err)
	}
	if update == nil {
		return false, nil
	}
	oldCluster := cluster.DeepCopy()

	cluster.Spec.Version = *semver.NewSemverOrDie(update.Version.String())
	// Invalidating the health to prevent automatic updates directly on the next processing.
	cluster.Status.ExtendedHealth.Apiserver = kubermaticv1.HealthStatusDown
	cluster.Status.ExtendedHealth.Controller = kubermaticv1.HealthStatusDown
	cluster.Status.ExtendedHealth.Scheduler = kubermaticv1.HealthStatusDown
	if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return false, fmt.Errorf("failed to update cluster: %v", err)
	}
	return true, nil
}
