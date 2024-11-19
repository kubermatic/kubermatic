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

package defaultapplicationcontroller

import (
	"context"
	"fmt"
	"slices"
	"time"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
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
	ControllerName = "kkp-default-application-controller"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName                    string
	recorder                      record.EventRecorder
	seedGetter                    provider.SeedGetter
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

func Add(ctx context.Context, mgr manager.Manager, numWorkers int, workerName string, seedGetter provider.SeedGetter, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		seedGetter:                    seedGetter,
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		// Watch for clusters
		For(&kubermaticv1.Cluster{}).
		// Watch changes for ApplicationDefinitions that have been enforced.
		Watches(&appskubermaticv1.ApplicationDefinition{}, enqueueClusters(reconciler.Client, log), builder.WithPredicates(withEventFilter())).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		// Cluster is queued for deletion; no action required
		r.log.Debugw("Cluster is queued for deletion; no action required", "cluster", cluster.Name)
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionDefaultApplicationInstallationControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
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

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	ignoreDefaultApplications := false

	// If the cluster has the initial application installations request annotation, we don't want to install the default applications as they will be
	// installed, if required, by the initial application installations controller.
	if cluster.Annotations[kubermaticv1.InitialApplicationInstallationsRequestAnnotation] != "" {
		ignoreDefaultApplications = true
	}

	_, exists := cluster.Status.Conditions[kubermaticv1.ClusterConditionApplicationInstallationControllerReconcilingSuccess]
	// We don't care about the state of the condition here since it `exists` is enough information to know that the initial application installations controller has already reconciled this cluster.
	if exists {
		ignoreDefaultApplications = true
	}

	// Default applications are already created.
	if cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionDefaultApplicationInstallationsControllerCreatedSuccessfully, corev1.ConditionTrue) {
		ignoreDefaultApplications = true
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cluster client: %w", err)
	}

	cniReady, err := util.IsCNIApplicationReady(ctx, userClusterClient, cluster)
	if err != nil {
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, fmt.Errorf("failed to check if CNI application is ready: %w", err)
	}
	if !cniReady {
		r.log.Debug("CNI application is not ready yet")
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Ensure that cluster is in a state when creating ApplicationInstallation is permissible
	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		r.log.Debug("Application controller not healthy")
		return nil, nil
	}

	// List all ApplicationDefinitions
	applicationDefinitions := &appskubermaticv1.ApplicationDefinitionList{}
	if err := r.List(ctx, applicationDefinitions); err != nil {
		return nil, fmt.Errorf("failed to list ApplicationDefinitions: %w", err)
	}

	// Collect all applications that need to be installed/updated.
	applications := []appskubermaticv1.ApplicationInstallation{}
	for _, applicationDefinition := range applicationDefinitions.Items {
		if applicationDefinition.DeletionTimestamp != nil {
			continue
		}

		// Check if the ApplicationDefinition is targeted to the current cluster's datacenter.
		if applicationDefinition.Spec.Selector.Datacenters != nil {
			if !slices.Contains(applicationDefinition.Spec.Selector.Datacenters, cluster.Spec.Cloud.DatacenterName) {
				continue
			}
		}

		if applicationDefinition.Spec.Enforced || (applicationDefinition.Spec.Default && !ignoreDefaultApplications) {
			applications = append(applications, r.generateApplicationInstallation(applicationDefinition))
		}
	}

	// We don't want to fail the reconciliation if one application fails so we collect all the errors and return them as a single error.
	var errors []error
	for _, application := range applications {
		// Using reconciler framework here doesn't help since the namespaces are different for the application installations.
		err := r.ensureApplicationInstallation(ctx, userClusterClient, application, cluster)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) == 0 && !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionDefaultApplicationInstallationsControllerCreatedSuccessfully, corev1.ConditionTrue) {
		if err := kubermaticv1helper.UpdateClusterStatus(ctx, r.Client, cluster, func(c *kubermaticv1.Cluster) {
			kubermaticv1helper.SetClusterCondition(
				cluster,
				r.versions,
				kubermaticv1.ClusterConditionDefaultApplicationInstallationsControllerCreatedSuccessfully,
				corev1.ConditionTrue,
				"",
				"",
			)
		}); err != nil {
			return &reconcile.Result{}, err
		}
	}

	return nil, kerrors.NewAggregate(errors)
}

func (r *Reconciler) ensureApplicationInstallation(ctx context.Context, userClusterClient ctrlruntimeclient.Client, application appskubermaticv1.ApplicationInstallation, cluster *kubermaticv1.Cluster) error {
	// First ensure that the namespace exists
	namespace := &corev1.Namespace{}
	if err := userClusterClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: application.Namespace}, namespace); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get namespace: %w", err)
		}

		// Create the namespace if it doesn't exist
		namespace.Name = application.Namespace
		err := userClusterClient.Create(ctx, namespace)
		if err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}

	existingApplication := &appskubermaticv1.ApplicationInstallation{}
	if err := userClusterClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: application.Name, Namespace: application.Namespace}, existingApplication); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get application installation: %w", err)
		}

		// Create the application
		err := userClusterClient.Create(ctx, &application)
		if err != nil {
			return fmt.Errorf("failed to create application installation: %w", err)
		}
		return nil
	}

	// If the application is not enforced then we can't update it.
	if application.Annotations[appskubermaticv1.ApplicationEnforcedAnnotation] != "true" {
		return nil
	}

	// Before comparison delete the kubectl.kubernetes.io/last-applied-configuration annotation.
	// This annotation is automatically generated by kubectl when applying a resource
	// and causes unnecessary diffs
	delete(existingApplication.Annotations, corev1.LastAppliedConfigAnnotation)

	// Application installation already exists, update it if needed
	if equality.Semantic.DeepEqual(existingApplication.Spec, application.Spec) &&
		equality.Semantic.DeepEqual(existingApplication.Labels, application.Labels) &&
		equality.Semantic.DeepEqual(existingApplication.Annotations, application.Annotations) {
		return nil
	}

	// Required to update the object.
	application.ResourceVersion = existingApplication.ResourceVersion
	application.UID = existingApplication.UID

	if err := userClusterClient.Update(ctx, &application); err != nil {
		return fmt.Errorf("failed to update application installation %q: %w", application.Name, err)
	}
	return nil
}

func (r *Reconciler) generateApplicationInstallation(application appskubermaticv1.ApplicationDefinition) appskubermaticv1.ApplicationInstallation {
	appVersion := application.Spec.DefaultVersion
	if appVersion == "" {
		// Iterate through all the versions and find the latest one by semver comparison
		for _, version := range application.Spec.Versions {
			if appVersion == "" {
				appVersion = version.Version
			}

			currentVersion, err := semver.NewSemver(version.Version)
			if err != nil {
				// We can't do much here. The webhooks and kubebuilder validation markers already impose semver usage so this error should never happen.
				continue
			}

			selectedVersion, err := semver.NewSemver(appVersion)
			if err != nil {
				// We can't do much here. The webhooks and kubebuilder validation markers already impose semver usage so this error should never happen.
				continue
			}

			if currentVersion.GreaterThan(selectedVersion) {
				appVersion = version.Version
			}
		}
	}
	err := convertDefaultValuesToDefaultValuesBlock(&application)
	if err != nil {
		// This is a non-critical error and we can still continue by using the `values` field instead of the `valuesBlock` field.
		r.log.Debugf("Failed to convert default values to default values block: %v", err)
	}

	delete(application.Annotations, corev1.LastAppliedConfigAnnotation)

	annotations := application.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if application.Spec.Enforced {
		annotations[appskubermaticv1.ApplicationEnforcedAnnotation] = "true"
	}
	if application.Spec.Default {
		annotations[appskubermaticv1.ApplicationDefaultedAnnotation] = "true"
	}

	app := appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   application.Name,
			Annotations: annotations,
			Labels:      application.Labels,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.AppNamespaceSpec{
				Name: application.Name,
				// This ensures that the namespace is created in the user cluster, if it doesn't already exist.
				Create: true,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    application.Name,
				Version: appVersion,
			},
			ValuesBlock: application.Spec.DefaultValuesBlock,
			Values:      runtime.RawExtension{Raw: []byte("{}")},
		},
	}

	// We already tried conversion and it failed. This should never happen but we have to work around it anyways.
	// Both DefaultValues and DefaultValuesBlock can not be set at the same time, our webhooks should prevent this.
	if len(app.Spec.ValuesBlock) == 0 && application.Spec.DefaultValues != nil {
		app.Spec.Values = *application.Spec.DefaultValues
	}

	return app
}

func enqueueClusters(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request
		application := a.(*appskubermaticv1.ApplicationDefinition)

		// Check if the application definition is enforced
		if !application.Spec.Enforced {
			return requests
		}

		// List all clusters
		clusters := &kubermaticv1.ClusterList{}
		if err := client.List(ctx, clusters); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list clusters: %w", err))
		}

		for _, cluster := range clusters.Items {
			if len(application.Spec.Selector.Datacenters) == 0 || slices.Contains(application.Spec.Selector.Datacenters, cluster.Spec.Cloud.DatacenterName) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: cluster.Name,
					},
				})
			}
		}
		return requests
	})
}

func withEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			obj := e.Object.(*appskubermaticv1.ApplicationDefinition)
			if obj.GetDeletionTimestamp() != nil {
				return false
			}

			if obj.Spec.Enforced {
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*appskubermaticv1.ApplicationDefinition)
			newObj := e.ObjectNew.(*appskubermaticv1.ApplicationDefinition)

			if newObj.GetDeletionTimestamp() != nil {
				return false
			}

			if newObj.Spec.Enforced != oldObj.Spec.Enforced && newObj.Spec.Enforced {
				return true
			}

			if newObj.Spec.Enforced {
				return newObj.GetGeneration() != oldObj.GetGeneration()
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			obj := e.Object.(*appskubermaticv1.ApplicationDefinition)
			if obj.GetDeletionTimestamp() != nil {
				return false
			}

			if obj.Spec.Enforced {
				return true
			}
			return false
		},
	}
}

func convertDefaultValuesToDefaultValuesBlock(app *appskubermaticv1.ApplicationDefinition) error {
	if len(app.Spec.DefaultValuesBlock) > 0 {
		return nil
	}

	if app.Spec.DefaultValues != nil {
		oldDefVals, err := yaml.JSONToYAML(app.Spec.DefaultValues.Raw)
		if err != nil {
			return err
		}
		app.Spec.DefaultValuesBlock = string(oldDefVals)
		app.Spec.DefaultValues = nil
	}
	return nil
}
