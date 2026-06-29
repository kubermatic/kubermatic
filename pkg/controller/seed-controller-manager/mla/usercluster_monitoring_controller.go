/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

const (
	UserClusterMonitoringControllerName = "kkp-mla-user-cluster-monitoring-controller"

	nodeExporterAppName   = "node-exporter"
	nodeExporterNamespace = "kube-system"

	kubeStateMetricsAppName   = "kube-state-metrics"
	kubeStateMetricsNamespace = "kube-system"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

// AddUserClusterMonitoring registers the user-cluster monitoring controller. It runs
// independently of the seed-level MLA stack (Grafana/Cortex/Loki) and is gated only on
// the UserClusterMLA feature flag in the caller.
func AddUserClusterMonitoring(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	userClusterConnectionProvider UserClusterClientProvider,
) error {
	return newUserClusterMonitoringReconciler(mgr, log, numWorkers, workerName, versions, userClusterConnectionProvider)
}

type userClusterMonitoringReconciler struct {
	ctrlruntimeclient.Client

	log                           *zap.SugaredLogger
	workerName                    string
	recorder                      events.EventRecorder
	versions                      kubermatic.Versions
	userClusterConnectionProvider UserClusterClientProvider
}

func newUserClusterMonitoringReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	userClusterConnectionProvider UserClusterClientProvider,
) error {
	subname := "user-cluster-monitoring"
	reconciler := &userClusterMonitoringReconciler{
		Client:                        mgr.GetClient(),
		log:                           log.Named(subname),
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorder(controllerName(subname)),
		versions:                      versions,
		userClusterConnectionProvider: userClusterConnectionProvider,
	}

	clusterPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			cluster := e.Object.(*kubermaticv1.Cluster)
			return cluster.Spec.MLA != nil && cluster.Spec.MLA.MonitoringEnabled
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldCluster := e.ObjectOld.(*kubermaticv1.Cluster)
			newCluster := e.ObjectNew.(*kubermaticv1.Cluster)
			oldEnabled := oldCluster.Spec.MLA != nil && oldCluster.Spec.MLA.MonitoringEnabled
			newEnabled := newCluster.Spec.MLA != nil && newCluster.Spec.MLA.MonitoringEnabled
			return oldEnabled != newEnabled
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	// When a legacy monitoring Addon is deleted, enqueue the owning cluster — but only if
	// monitoring is actually enabled on that cluster, to avoid spurious reconciles.
	addonPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			name := e.Object.GetName()
			return name == nodeExporterAppName || name == kubeStateMetricsAppName
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
	enqueueClusterIfMonitoringEnabled := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o ctrlruntimeclient.Object) []reconcile.Request {
		cluster, err := kubernetes.ClusterFromNamespace(ctx, mgr.GetClient(), o.GetNamespace())
		if err != nil || cluster == nil {
			return nil
		}
		if cluster.Spec.MLA == nil || !cluster.Spec.MLA.MonitoringEnabled {
			return nil
		}
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName(subname)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(clusterPredicate, workerlabel.Predicate(workerName))).
		Watches(&kubermaticv1.Addon{}, enqueueClusterIfMonitoringEnabled, builder.WithPredicates(addonPredicate)).
		Build(reconciler)

	return err
}

func (r *userClusterMonitoringReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if cluster.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	result, err := controllerutil.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMLAControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return *result, err
}

func (r *userClusterMonitoringReconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	monitoringEnabled := cluster.Spec.MLA != nil && cluster.Spec.MLA.MonitoringEnabled

	if !monitoringEnabled {
		return nil, r.ensureApplicationInstallationsRemoved(ctx, log, cluster)
	}

	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		log.Debug("Application controller not healthy, requeuing")
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cluster client: %w", err)
	}

	for _, app := range []struct {
		appName   string
		namespace string
	}{
		{nodeExporterAppName, nodeExporterNamespace},
		{kubeStateMetricsAppName, kubeStateMetricsNamespace},
	} {
		// If the user previously installed this app as an Addon, leave their setup untouched.
		if managed, err := r.isAddonManaged(ctx, cluster, app.appName); err != nil {
			return nil, err
		} else if managed {
			log.Debugf("%s is managed by an Addon CR, skipping ApplicationInstallation", app.appName)
			continue
		}

		version, initialValues, err := r.parseAppDefVersionAndDefaultValues(ctx, log, app.appName)
		if err != nil {
			return nil, err
		}
		if version == "" {
			continue
		}

		reconcilers := []reconciling.NamedApplicationInstallationReconcilerFactory{
			monitoringAppInstallationReconciler(app.appName, app.namespace, version, initialValues),
		}
		if err := reconciling.ReconcileApplicationInstallations(ctx, reconcilers, app.namespace, userClusterClient); err != nil {
			return nil, fmt.Errorf("failed to reconcile %s ApplicationInstallation: %w", app.appName, err)
		}
	}

	return nil, nil
}

// parseAppDefVersionAndDefaultValues fetches the ApplicationDefinition for appName and returns
// the version to use (empty string if the AppDef is missing or has no versions) and its parsed
// default values. Mirrors the pattern used by the CNI controller.
func (r *userClusterMonitoringReconciler) parseAppDefVersionAndDefaultValues(ctx context.Context, log *zap.SugaredLogger, appName string) (string, map[string]any, error) {
	appDef := &appskubermaticv1.ApplicationDefinition{}
	if err := r.Get(ctx, types.NamespacedName{Name: appName}, appDef); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debugf("ApplicationDefinition %s not found, skipping", appName)
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("failed to get ApplicationDefinition %s: %w", appName, err)
	}

	version := selectLatestAppVersion(appDef)
	if version == "" {
		log.Warnf("ApplicationDefinition %s has no versions, skipping", appName)
		return "", nil, nil
	}

	values, err := appDef.Spec.GetParsedDefaultValues()
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse default values for ApplicationDefinition %s: %w", appName, err)
	}

	return version, values, nil
}

// isAddonManaged returns true if a user-installed Addon CR for appName already exists in the
// cluster namespace. When it does, we leave the existing addon untouched and skip deploying
// an ApplicationInstallation for the same app to avoid running duplicate workloads.
func (r *userClusterMonitoringReconciler) isAddonManaged(ctx context.Context, cluster *kubermaticv1.Cluster, appName string) (bool, error) {
	addon := &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	err := r.Get(ctx, types.NamespacedName{Name: addon.Name, Namespace: addon.Namespace}, addon)
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}

func (r *userClusterMonitoringReconciler) ensureApplicationInstallationsRemoved(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		return nil
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	for _, app := range []struct {
		name      string
		namespace string
	}{
		{nodeExporterAppName, nodeExporterNamespace},
		{kubeStateMetricsAppName, kubeStateMetricsNamespace},
	} {
		appInstall := &appskubermaticv1.ApplicationInstallation{}
		err := userClusterClient.Get(ctx, types.NamespacedName{Name: app.name, Namespace: app.namespace}, appInstall)
		if err != nil {
			if err := ctrlruntimeclient.IgnoreNotFound(err); err != nil {
				return err
			}
			continue
		}

		if appInstall.Labels[appskubermaticv1.ApplicationManagedByLabel] != appskubermaticv1.ApplicationManagedByKKPValue {
			continue
		}

		log.Debugf("Deleting %s ApplicationInstallation as monitoring is disabled", app.name)
		if err := userClusterClient.Delete(ctx, appInstall); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete %s ApplicationInstallation: %w", app.name, err)
		}
	}

	return nil
}

// monitoringAppInstallationReconciler returns a reconciler factory for a monitoring ApplicationInstallation.
// On the first install (existing values empty) it seeds ValuesBlock from initialValues; on subsequent
// reconciliations it leaves the user's values untouched. This mirrors the CNI controller pattern.
func monitoringAppInstallationReconciler(appName, namespace, version string, initialValues map[string]any) reconciling.NamedApplicationInstallationReconcilerFactory {
	return func() (string, reconciling.ApplicationInstallationReconciler) {
		return appName, func(app *appskubermaticv1.ApplicationInstallation) (*appskubermaticv1.ApplicationInstallation, error) {
			kuberneteshelper.EnsureLabels(app, map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
			})
			app.Spec.ApplicationRef = appskubermaticv1.ApplicationRef{
				Name:    appName,
				Version: version,
			}
			app.Spec.Namespace = &appskubermaticv1.AppNamespaceSpec{
				Name:   namespace,
				Create: false,
			}

			// Unmarshal existing values; if empty (first install) seed from AppDef defaults.
			values, err := app.Spec.GetParsedValues()
			if err != nil {
				return app, fmt.Errorf("failed to unmarshal existing values for %s: %w", appName, err)
			}
			if len(values) == 0 {
				values = initialValues
			}

			rawValues, err := yaml.Marshal(values)
			if err != nil {
				return app, fmt.Errorf("failed to marshal values for %s: %w", appName, err)
			}
			app.Spec.ValuesBlock = string(rawValues)
			// Clear the deprecated Values field so ValuesBlock is the sole source of truth.
			app.Spec.Values = runtime.RawExtension{Raw: []byte("{}")}

			return app, nil
		}
	}
}

func selectLatestAppVersion(appDef *appskubermaticv1.ApplicationDefinition) string {
	if appDef.Spec.DefaultVersion != "" {
		return appDef.Spec.DefaultVersion
	}

	selected := ""
	for _, v := range appDef.Spec.Versions {
		if selected == "" {
			selected = v.Version
			continue
		}

		current, err := semver.NewSemver(v.Version)
		if err != nil {
			continue
		}
		best, err := semver.NewSemver(selected)
		if err != nil {
			continue
		}
		if current.GreaterThan(best) {
			selected = v.Version
		}
	}

	return selected
}
