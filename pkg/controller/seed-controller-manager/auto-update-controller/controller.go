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

package autoupdatecontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-auto-update-controller"
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

// Add creates a new auto-update controller.
func Add(
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	configGetter provider.KubermaticConfigurationGetter,
	userClusterConnectionProvider *client.Provider,
	log *zap.SugaredLogger,
	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName:                    workerName,
		configGetter:                  configGetter,
		userClusterConnectionProvider: userClusterConnectionProvider,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		log:                           log,
		versions:                      versions,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// auto-updates should not be applied during cluster deletion
	if cluster.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionUpdateControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
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

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
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

	if err := r.controlPlaneUpgrade(ctx, log, cluster, updateManager); err != nil {
		return nil, fmt.Errorf("failed to update the controlplane: %w", err)
	}

	// nodeUpdate works based on the Cluster.Status.Versions.ControlPlane field, so it properly waits
	// for the control plane to be upgraded before updating the nodes.
	if err := r.nodeUpdate(ctx, log, cluster, updateManager); err != nil {
		return nil, fmt.Errorf("failed to update the controlplane: %w", err)
	}

	return nil, nil
}

func (r *Reconciler) nodeUpdate(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, updateManager *version.Manager) error {
	c, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get usercluster client: %w", err)
	}

	machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
	// Kubermatic only creates MachineDeployments in the kube-system namespace, everything else is essentially unsupported
	if err := c.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		return fmt.Errorf("failed to list MachineDeployments: %w", err)
	}

	for _, md := range machineDeployments.Items {
		targetVersion, err := updateManager.AutomaticNodeUpdate(md.Spec.Template.Spec.Versions.Kubelet, cluster.Status.Versions.ControlPlane.String())
		if err != nil {
			return fmt.Errorf("failed to get automatic update for machinedeployment %s/%s that has version %q: %w", md.Namespace, md.Name, md.Spec.Template.Spec.Versions.Kubelet, err)
		}
		if targetVersion == nil {
			continue
		}

		target := targetVersion.Version.String()
		old := md.Spec.Template.Spec.Versions.Kubelet

		if old != target {
			oldMD := md.DeepCopy()
			identifier := fmt.Sprintf("%s/%s", md.Namespace, md.Name)

			log.Infow("Applying automatic update to MachineDeployment", "machinedeployment", identifier, "from", old, "to", target)

			md.Spec.Template.Spec.Versions.Kubelet = target
			if err := c.Patch(ctx, &md, ctrlruntimeclient.MergeFrom(oldMD)); err != nil {
				return fmt.Errorf("failed to update MachineDeployment: %w", err)
			}

			r.recorder.Eventf(cluster, corev1.EventTypeNormal, "AutoUpdateMachineDeployment", "Triggered automatic update of MachineDeployment %s to version %q", identifier, target)
		}
	}

	return nil
}

func (r *Reconciler) controlPlaneUpgrade(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, updateManager *version.Manager) error {
	update, err := updateManager.AutomaticControlplaneUpdate(cluster.Spec.Version.String())
	if err != nil {
		return fmt.Errorf("failed to get automatic update for cluster for version %s: %w", cluster.Spec.Version.String(), err)
	}
	if update == nil {
		return nil
	}
	oldCluster := cluster.DeepCopy()

	sver, err := semver.NewSemver(update.Version.String())
	if err != nil {
		return fmt.Errorf("failed to parse version %q: %w", update.Version.String(), err)
	}

	log.Infow("Applying automatic control-plane upgrade", "from", oldCluster.Spec.Version, "to", cluster.Spec.Version)

	// Set the new target version; this in turn will trigger the incremental update controller
	// to begin rolling out the necessary changes and over time we will converge to the version
	// set here.
	cluster.Spec.Version = *sver
	if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	log.Infow("Applied automatic cluster upgrade", "from", oldCluster.Spec.Version, "to", cluster.Spec.Version)
	r.recorder.Eventf(cluster, corev1.EventTypeNormal, "AutoUpdateApplied", "Cluster was automatically updated from v%s to v%s.", oldCluster.Spec.Version, cluster.Spec.Version)

	err = util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		// Invalidating the health to prevent automatic updates directly on the next processing.
		c.Status.ExtendedHealth.Apiserver = kubermaticv1.HealthStatusDown
		c.Status.ExtendedHealth.Controller = kubermaticv1.HealthStatusDown
		c.Status.ExtendedHealth.Scheduler = kubermaticv1.HealthStatusDown
	})
	if err != nil {
		return fmt.Errorf("failed to update cluster status: %w", err)
	}

	return nil
}
