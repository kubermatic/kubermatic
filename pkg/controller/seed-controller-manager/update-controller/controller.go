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

package updatecontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version"
	clusterversion "k8c.io/kubermatic/v2/pkg/version/cluster"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-update-controller"

	ClusterConditionUpToDate    = "UpToDate"
	ClusterConditionProgressing = "Progressing"
	ClusterConditionOldNodes    = "OldNodes"
)

type controlPlaneChecker func(context.Context, ctrlruntimeclient.Client, *zap.SugaredLogger, *kubermaticv1.Cluster) (*controlPlaneStatus, error)

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName   string
	configGetter provider.KubermaticConfigurationGetter
	recorder     record.EventRecorder
	log          *zap.SugaredLogger
	versions     kubermatic.Versions

	// cpChecker is here to make unit testing easier
	cpChecker controlPlaneChecker
}

// Add creates a new update controller.
func Add(mgr manager.Manager, numWorkers int, workerName string, configGetter provider.KubermaticConfigurationGetter, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client:       mgr.GetClient(),
		workerName:   workerName,
		configGetter: configGetter,
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		log:          log,
		versions:     versions,
		cpChecker:    getCurrentControlPlaneVersions,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		// watch Cluster objects to react to spec.version changing
		For(&kubermaticv1.Cluster{}).
		// Watch Deployments in cluster namespaces to react to the control plane change over time
		Watches(&appsv1.Deployment{}, controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())).
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

	if cluster.DeletionTimestamp != nil {
		log.Debug("Cluster is in deletion, no further reconciling.")
		return reconcile.Result{}, nil
	}

	// Clusters need to be reconciled regardless of their NamespaceName,
	// as the kubernetes controller will wait for the target version to
	// be determined before beginning its reconciling (i.e. before creating
	// the cluster namespace).

	// Add a wrapping here so we can emit an event on error
	result, err := controllerutil.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionUpdateControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return nil, r.reconcile(ctx, log, cluster)
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

func (r *Reconciler) setClusterCondition(ctx context.Context, cluster *kubermaticv1.Cluster, reason, message string) error {
	return controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		controllerutil.SetClusterCondition(
			c,
			r.versions,
			kubermaticv1.ClusterConditionUpdateProgress,
			corev1.ConditionTrue,
			reason,
			message,
		)
	})
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	// if the cluster status has no version information yet, set the initial status
	if cluster.Status.Versions.ControlPlane == "" || cluster.Status.Versions.Apiserver == "" || cluster.Status.Versions.ControllerManager == "" || cluster.Status.Versions.Scheduler == "" {
		if err := setInitialClusterVersions(ctx, r, cluster); err != nil {
			return fmt.Errorf("failed to set initial cluster status: %w", err)
		}

		log.Info("Set initial cluster version")

		// setting the status above will trigger a reconciliation anyway
		return nil
	}

	// Before making any further decisions, find out how the control plane is currently running.
	cpStatus, err := r.cpChecker(ctx, r, log, cluster)
	if err != nil {
		return fmt.Errorf("failed to determine version status for control plane: %w", err)
	}

	spec := normalize(&cluster.Spec.Version)

	log.With(
		"spec", spec,
		"apiserverCurrently", cpStatus.apiserver,
		"controllerManagerCurrently", cpStatus.controllerManager,
		"schedulerCurrently", cpStatus.scheduler,
		"apiserverGoal", cluster.Status.Versions.Apiserver,
		"controllerManagerGoal", cluster.Status.Versions.ControllerManager,
		"schedulerGoal", cluster.Status.Versions.Scheduler,
		"oldestNode", cluster.Status.Versions.OldestNodeVersion,
	).Debug("Cluster version overview")

	// Store the currently active apiserver version, as this is what most of the reconciling
	// of other components like cloud-controller-managers, the user-ccm etc. is based on.
	if cpStatus.apiserver != nil && !cluster.Status.Versions.ControlPlane.Equal(cpStatus.apiserver) {
		if err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Versions.ControlPlane = *cpStatus.apiserver
		}); err != nil {
			return fmt.Errorf("failed to update controller-manager version status: %w", err)
		}

		log.Infow("Cluster apiserver has been updated", "version", *cpStatus.apiserver)
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "ControlPlaneVersionChanged", "Cluster control plane has reached version %s.", *cpStatus.apiserver)
	}

	// Now that we are aware of the state of the world, we can check if we already reached
	// the final state (spec.version).
	if cpStatus.Equal(spec) {
		// Cluster is already at the desired version; the control plane might still be rolling out
		// or in need of reconciling, but for this controller there is no further work to be done.
		log.Debugw("Cluster control plane has reached the spec'ed version.", "spec", spec)

		return r.setClusterCondition(ctx, cluster, ClusterConditionUpToDate, "No update in progress, cluster has reached its desired version.")
	}

	// We have not yet reached the desired state; before taking actions towards that goal,
	// we must wait and ensure that the control plane is at least healthy. This gets really important
	// because changing the apiserver version will not just change the apiserver Deployment, but
	// also potentially etcd, etc.
	if !cluster.Status.ExtendedHealth.AllHealthy() {
		// Cluster not healthy yet. Nothing to do. Changes to the health will trigger another reconciliation.
		log.Debug("Cluster control plane has not reached the spec'ed version, but is also not yet healthy.")

		return r.setClusterCondition(ctx, cluster, ClusterConditionProgressing, "Update in progress, control plane is not yet healthy.")
	}

	// Cluster is healthy but has not yet reached the spec'ed version. However maybe it didn't
	// even reach the status version yet (for example if the KKP kubernetes controller is defunct
	// and hasn't reconciled yet); we have to wait for the control plane to match what we configured
	// in the cluster status before proceeding.
	// Do this as 3 distinct checks to provide nice looking log messages.
	if !cpStatus.apiserver.Equal(&cluster.Status.Versions.Apiserver) {
		log.Debugw("Cluster control plane is healthy but apiserver is out-of-sync.", "running", cpStatus.apiserver, "desired", cluster.Status.Versions.Apiserver)
		return r.setClusterCondition(ctx, cluster, ClusterConditionProgressing, "Update in progress, control plane is healthy but apiserver is out-of-sync.")
	}

	if !cpStatus.controllerManager.Equal(&cluster.Status.Versions.ControllerManager) {
		log.Debugw("Cluster control plane is healthy but controller-manager is out-of-sync.", "running", cpStatus.controllerManager, "desired", cluster.Status.Versions.ControllerManager)
		return r.setClusterCondition(ctx, cluster, ClusterConditionProgressing, "Update in progress, control plane is healthy but controller-manager is out-of-sync.")
	}

	if !cpStatus.scheduler.Equal(&cluster.Status.Versions.Scheduler) {
		log.Debugw("Cluster control plane is healthy but scheduler is out-of-sync.", "running", cpStatus.scheduler, "desired", cluster.Status.Versions.Scheduler)
		return r.setClusterCondition(ctx, cluster, ClusterConditionProgressing, "Update in progress, control plane is healthy but scheduler is out-of-sync.")
	}

	// Cluster is healthy, all Pods match what we intend to deploy as per the cluster status and
	// the nodes are up-to-date, but overall we're still outdated and haven't reached the spec'ed version.
	// The first important step is to make sure that scheduler/controller-manager both match the
	// apiserver's version (which basically completes the update from one release to another).
	// This is so that when the control plane is on apiserver:20, ctrlmgr:19,
	// scheduler:19, we do not update the apiserver to v21 before the other two components have
	// caught up.
	// Scheduler and controller-manager can be bumped and reconciled at the same time.
	versions := cluster.Status.Versions
	updated := false

	if !versions.Apiserver.Equal(&versions.ControllerManager) {
		if err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Versions.ControllerManager = versions.Apiserver
		}); err != nil {
			return fmt.Errorf("failed to update controller-manager version status: %w", err)
		}

		log.Infow("Updating controller-manager to match apiserver", "apiserver", versions.Apiserver, "controllerManager", versions.ControllerManager)
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "ControllerManagerUpdated", "Kubernetes controller-manager was updated to version %s.", versions.Apiserver)
		updated = true
	}

	if !versions.Apiserver.Equal(&versions.Scheduler) {
		if err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.Versions.Scheduler = versions.Apiserver
		}); err != nil {
			return fmt.Errorf("failed to update scheduler version status: %w", err)
		}

		log.Infow("Updating scheduler to match apiserver", "apiserver", versions.Apiserver, "scheduler", versions.Scheduler)
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "SchedulerUpdated", "Kubernetes scheduler was updated to version %s.", versions.Apiserver)
		updated = true
	}

	// updating the status above will trigger a reconciliation, which will update the cluster condition
	if updated {
		return nil
	}

	// This controller does not update nodes, as nodes can and will be updated independently (for example, the
	// update rules configured in the KubermaticConfiguration make a distinction between control-plane upgrades
	// and node upgrades). However we must still ensure that this controller doesn't advance the control-plane
	// to a point where it is not compatible with the nodes anymore.
	// Kubelets can be 2 versions behind the control-plane.
	if cpStatus.nodes != nil {
		a := int(cpStatus.nodes.Semver().Minor())
		b := int(cluster.Status.Versions.ControlPlane.Semver().Minor())

		distance := a - b
		if distance < 0 {
			distance = -distance
		}

		if distance >= 2 {
			log.Debugw("Cluster control plane is healthy but cluster still has old nodes.", "controlPlane", cluster.Status.Versions.ControlPlane, "oldestNode", cpStatus.nodes)
			return r.setClusterCondition(ctx, cluster, ClusterConditionOldNodes, fmt.Sprintf("Update in progress, control plane (v%s) is healthy but cluster still has old nodes (v%s).", cluster.Status.Versions.ControlPlane.String(), cpStatus.nodes.String()))
		}

		// Distance is at most 1 release, so the control plane is free to be updated at any time.
	}

	// At this point we know that the entire control plane is healthy, that scheduler/ctrlmgr versions
	// are equal to the apiserver, but have still not reached the spec'ed version. It's now time to
	// update the apiserver to the next minor release. The next minor will be the latest patch release
	// that is configured for the minor and is not newer than the spec'ed version.
	config, err := r.configGetter(ctx)
	if err != nil {
		return fmt.Errorf("failed to load KubermaticConfiguration: %w", err)
	}

	newVersion, err := getNextApiserverVersion(ctx, config, cluster)
	if err != nil {
		return fmt.Errorf("failed to determine update path: %w", err)
	}

	// Set this new target version as the next step on our upgrading journey. This will trigger a
	// reconciliation for us and also make the KKP kubernetes controller roll out the new apiserver.
	if err := controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.Versions.Apiserver = *newVersion
	}); err != nil {
		return fmt.Errorf("failed to update apiserver version: %w", err)
	}

	log.Infow("Updating apiserver", "from", versions.Apiserver, "to", newVersion.String(), "spec", spec)
	r.recorder.Eventf(cluster, corev1.EventTypeNormal, "ApiserverUpdated", "Kubernetes apiserver was updated to version %s.", newVersion.String())

	return nil
}

// setInitialClusterVersions assumes that the cluster was never up and running and sets
// the desired versions in the status to be equal to the version from the spec. The
// status about currently running components is left empty and filled in during later
// reconciliations.
func setInitialClusterVersions(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	return controllerutil.UpdateClusterStatus(ctx, client, cluster, func(cluster *kubermaticv1.Cluster) {
		spec := normalize(&cluster.Spec.Version)

		if cluster.Status.Versions.Apiserver == "" {
			cluster.Status.Versions.Apiserver = *spec
		}

		if cluster.Status.Versions.ControlPlane == "" {
			cluster.Status.Versions.ControlPlane = cluster.Status.Versions.Apiserver
		}

		if cluster.Status.Versions.ControllerManager == "" {
			cluster.Status.Versions.ControllerManager = *spec
		}

		if cluster.Status.Versions.Scheduler == "" {
			cluster.Status.Versions.Scheduler = *spec
		}
	})
}

type controlPlaneStatus struct {
	apiserver         *semver.Semver
	controllerManager *semver.Semver
	scheduler         *semver.Semver
	nodes             *semver.Semver
}

func (s *controlPlaneStatus) Equal(b *semver.Semver) bool {
	return s.apiserver.Equal(b) && s.controllerManager.Equal(b) && s.scheduler.Equal(b)
}

func getCurrentControlPlaneVersions(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*controlPlaneStatus, error) {
	result := &controlPlaneStatus{
		// we rely on the node-version-controller (uccm) to update this field in the cluster status for us,
		// so that we do not have to connect to the usercluster
		nodes: cluster.Status.Versions.OldestNodeVersion,
	}

	// If no namespace is given yet, there is no point in trying to find the
	// current ReplicaSets and deduce their versions.
	if cluster.Status.NamespaceName == "" {
		return result, nil
	}

	tasks := []struct {
		name    string
		updater func(*semver.Semver)
	}{
		{
			name: resources.ApiserverDeploymentName,
			updater: func(v *semver.Semver) {
				result.apiserver = v
			},
		},
		{
			name: resources.ControllerManagerDeploymentName,
			updater: func(v *semver.Semver) {
				result.controllerManager = v
			},
		},
		{
			name: resources.SchedulerDeploymentName,
			updater: func(v *semver.Semver) {
				result.scheduler = v
			},
		},
	}

	for _, task := range tasks {
		key := types.NamespacedName{
			Namespace: cluster.Status.NamespaceName,
			Name:      task.name,
		}

		replicaSets, err := getReplicaSetsForDeployment(ctx, client, log, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get ReplicaSets for %s: %w", key.Name, err)
		}

		task.updater(getOldestAvailableVersion(log, replicaSets))
	}

	return result, nil
}

// getOldestAvailableVersion finds the ReplicaSet with at least one replica and with the lowest version and returns that version.
func getOldestAvailableVersion(log *zap.SugaredLogger, replicaSets []appsv1.ReplicaSet) *semver.Semver {
	var oldest *semver.Semver
	for _, rs := range replicaSets {
		// ignore ReplicaSets with no active pods (note, do not consider the "Ready" status,
		// as even pods that are not Ready anymore during shutdown can still be active and
		// do things).
		if rs.Status.Replicas == 0 {
			continue
		}

		versionLabel := rs.GetLabels()[resources.VersionLabel]
		if versionLabel == "" {
			log.Warnw("ReplicaSet has no version label, this should not happen", "replicaset", rs.Name)
			continue
		}

		parsedVersion, err := semver.NewSemver(versionLabel)
		if versionLabel == "" {
			log.Warnw("ReplicaSet has invalid version label", "replicaset", rs.Name, "label", versionLabel, zap.Error(err))
			continue
		}

		if oldest == nil || oldest.GreaterThan(parsedVersion) {
			oldest = parsedVersion
		}
	}

	return normalize(oldest)
}

func getReplicaSetsForDeployment(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, deploymentName types.NamespacedName) ([]appsv1.ReplicaSet, error) {
	rsList := &appsv1.ReplicaSetList{}
	if err := client.List(ctx, rsList, ctrlruntimeclient.InNamespace(deploymentName.Namespace)); err != nil {
		return nil, err
	}

	if len(rsList.Items) == 0 {
		return nil, nil
	}

	ownerNames := sets.New(deploymentName.Name)
	result := []appsv1.ReplicaSet{}
	for i, rs := range rsList.Items {
		if hasOwnerRefToAny(&rs, "Deployment", ownerNames) {
			result = append(result, rsList.Items[i])
		}
	}

	return result, nil
}

func hasOwnerRefToAny(obj ctrlruntimeclient.Object, ownerKind string, ownerNames sets.Set[string]) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind == ownerKind && ownerNames.Has(ref.Name) {
			return true
		}
	}

	return false
}

func getNextApiserverVersion(ctx context.Context, config *kubermaticv1.KubermaticConfiguration, cluster *kubermaticv1.Cluster) (*semver.Semver, error) {
	updateConditions := clusterversion.GetVersionConditions(&cluster.Spec)
	updateManager := version.NewFromConfiguration(config)
	currentVersion := cluster.Status.Versions.Apiserver.Semver()
	targetVersion := cluster.Spec.Version.Semver()

	// The returned versions will consist only of direct update paths, i.e. even if
	// enough versions and updates are configured to go from 1.20 to 1.23, only
	// 1 level of updates is applied, so effectively only 1.20 and 1.21 versions
	// will be returned.
	versions, err := updateManager.GetPossibleUpdates(currentVersion.String(), kubermaticv1.ProviderType(cluster.Spec.Cloud.ProviderName), updateConditions...)
	if err != nil {
		return nil, err
	}

	var newTarget *semver.Semver

	if currentVersion.Minor() == targetVersion.Minor() {
		// When jumping from x.y.0 to x.y.7, we only care if this specific
		// target version is a possible update, i.e. we would not want to
		// update to x.y.5, even if that is the only supported x.y update.
		for _, version := range versions {
			if version.Version.Equal(targetVersion) {
				newTarget = semver.NewSemverOrDie(version.Version.String())
			}
		}
	} else {
		// When jumping to the next or further minor releases, we are looking
		// for the highest patch release of the next minor.
		for _, version := range versions {
			// This version is already more recent than ultimately desired for the cluster, skip it.
			if version.Version.GreaterThan(targetVersion) {
				continue
			}

			// Just in case, this should never happen if the version manager works properly.
			if version.Version.LessThan(currentVersion) {
				continue
			}

			if newTarget == nil || version.Version.GreaterThan(newTarget.Semver()) {
				newTarget = semver.NewSemverOrDie(version.Version.String())
			}
		}
	}

	// The webhook should reject KubermaticConfigurations that do not offer
	// at least one minor version for each Kubernetes release.
	if newTarget == nil {
		return nil, fmt.Errorf("no update available from %s to %s", currentVersion, cluster.Spec.Version)
	}

	return normalize(newTarget), nil
}

// normalize ensures that the string representation for semvers is always consistent.
func normalize(s *semver.Semver) *semver.Semver {
	if s == nil {
		return nil
	}

	return semver.NewSemverOrDie(s.String())
}
