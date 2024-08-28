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
	"strings"

	"go.uber.org/zap"
	"golang.org/x/mod/semver"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	ControllerName      = "kkp-default-application-controller"
	AnnotationTrueValue = "true"
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
		For(&kubermaticv1.Cluster{}, builder.WithPredicates()).
		// Watch changes for OSPs and then enqueue all the clusters where OSM is enabled.
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
	// for initial applications and initial application controller will not reconcile the cluster.
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
		if val, ok := applicationDefinition.Annotations[appskubermaticv1.ApplicationTargetDatacenterAnnotation]; ok {
			// Split the list and check if the cluster's datacenter is included.
			datacenters := strings.Split(val, ",")
			if !slices.Contains(datacenters, cluster.Spec.Cloud.DatacenterName) {
				// Skip this ApplicationDefinition if the cluster's datacenter is not in the list
				continue
			}
		}
		if applicationDefinition.Annotations[appskubermaticv1.ApplicationEnforcedAnnotation] == AnnotationTrueValue || (applicationDefinition.Annotations[appskubermaticv1.ApplicationDefaultAnnotation] == AnnotationTrueValue && !ignoreDefaultApplications) {
			applications = append(applications, r.generateApplicationInstallation(applicationDefinition))
		}
	}

	// We don't want to fail the reconciliation if one application fails so we collect all the errors and return them as a single error.
	var errors []error
	for _, application := range applications {
		// Using reconciler framework here doesn't help since the namespaces are different for the application installations.
		err := r.ensureApplicationInstallation(ctx, application, cluster)
		if err != nil {
			errors = append(errors, err)
		}
	}
	return nil, kerrors.NewAggregate(errors)
}

func (r *Reconciler) ensureApplicationInstallation(ctx context.Context, application appskubermaticv1.ApplicationInstallation, cluster *kubermaticv1.Cluster) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get usercluster client: %w", err)
	}

	existingApplication := &appskubermaticv1.ApplicationInstallation{}
	if err := userClusterClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: application.Name, Namespace: application.Namespace}, existingApplication); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get application installation: %w", err)
		}
		// Create the application installation
		err := userClusterClient.Create(ctx, &application)
		if err != nil {
			return fmt.Errorf("failed to create application installation: %w", err)
		}
	}

	// If the application is not enforced then we can't update it.
	if application.Annotations[appskubermaticv1.ApplicationEnforcedAnnotation] != AnnotationTrueValue {
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
		return fmt.Errorf("failed to update application installation: %w", err)
	}
	return nil
}

func (r *Reconciler) generateApplicationInstallation(application appskubermaticv1.ApplicationDefinition) appskubermaticv1.ApplicationInstallation {
	appVersion := application.Spec.DefaultVersion
	if appVersion == "" {
		// Iterate through all the verions and find the latest one by semver comparison
		for _, version := range application.Spec.Versions {
			if semver.Compare(version.Version, appVersion) > 0 {
				appVersion = version.Version
			}
		}
	}

	err := convertDefaultValuesToDefaultValuesBlock(&application)
	if err != nil {
		// This is a non-critical error and we can still continue by using the `values` field instead of the `valuesBlock` field.
		r.log.Debugf("Failed to convert default values to default values block: %v", err)
	}

	// Drop apps.kubermatic.k8c.io/target-datacenter annotation. Datacenter is a concept used in master/seed components of KKP and user clusters shouldn't be aware of it.
	delete(application.Annotations, appskubermaticv1.ApplicationTargetDatacenterAnnotation)

	app := appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   application.Name,
			Annotations: application.Annotations,
			Labels:      application.Labels,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.AppNamespaceSpec{
				Name: application.Name,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    application.Name,
				Version: appVersion,
			},
			ValuesBlock: application.Spec.DefaultValuesBlock,
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
		datacenters := []string{}

		// Check if the application definition is enforced
		if a.GetAnnotations()[appskubermaticv1.ApplicationEnforcedAnnotation] != AnnotationTrueValue {
			return requests
		}

		// Check if the application is scoped to a datacenter
		if a.GetAnnotations()[appskubermaticv1.ApplicationTargetDatacenterAnnotation] != "" {
			// Get the datacenters the application is scoped to
			datacenters = strings.Split(a.GetAnnotations()[appskubermaticv1.ApplicationTargetDatacenterAnnotation], ",")
		}

		// List all clusters
		clusters := &kubermaticv1.ClusterList{}
		if err := client.List(ctx, clusters); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list clusters: %w", err))
		}

		for _, cluster := range clusters.Items {
			if len(datacenters) == 0 || slices.Contains(datacenters, cluster.Spec.Cloud.DatacenterName) {
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
			if e.Object.GetDeletionTimestamp() != nil {
				return false
			}

			if e.Object.GetAnnotations()[appskubermaticv1.ApplicationEnforcedAnnotation] == AnnotationTrueValue {
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectNew.GetDeletionTimestamp() != nil {
				return false
			}
			if e.ObjectNew.GetAnnotations()[appskubermaticv1.ApplicationEnforcedAnnotation] == AnnotationTrueValue {
				return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			if e.Object.GetDeletionTimestamp() != nil {
				return false
			}

			if e.Object.GetAnnotations()[appskubermaticv1.ApplicationEnforcedAnnotation] == AnnotationTrueValue {
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
