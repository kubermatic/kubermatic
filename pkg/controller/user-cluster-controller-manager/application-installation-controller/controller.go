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

package applicationinstallationcontroller

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"dario.cat/mergo"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/apis/equality"
	"k8c.io/kubermatic/v2/pkg/applications"
	applicationtemplates "k8c.io/kubermatic/v2/pkg/applications/providers/template"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "kkp-app-installation-controller"

	// Event raised when an applicationInstallation that has been installed references an applicationDefinition that does not exist anymore.
	applicationDefinitionRemovedEvent = "ApplicationDefinitionRemoved"

	// Event raised when an applicationInstallation that has been installed references an applicationVersion that does not exist anymore.
	applicationVersionRemovedEvent = "ApplicationVersionRemoved"

	// Event raised when applicationInstallation reference an applicalicationDefinition  that is being deleted.
	applicationDefinitionDeletingEvent = "ApplicationDefinitionDeleting"

	// Event raised when the reconciliation of an applicationInstallation failed.
	applicationInstallationReconcileFailedEvent = "ApplicationInstallationReconcileFailed"

	// maxRetries is the maximum number of retries on installation or upgrade failure.
	maxRetries = 5

	// initialRequeueDuration is the time interval which is used until a node object to schedule workloads is registered in the cluster.
	initialRequeueDuration = 10 * time.Second
)

type reconciler struct {
	log                  *zap.SugaredLogger
	seedClient           ctrlruntimeclient.Client
	userClient           ctrlruntimeclient.Client
	userRecorder         record.EventRecorder
	clusterIsPaused      userclustercontrollermanager.IsPausedChecker
	appInstaller         applications.ApplicationInstaller
	seedClusterNamespace string
	overwriteRegistry    string
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker, seedClusterNamespace, overwriteRegistry string, appInstaller applications.ApplicationInstaller) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:                  log,
		seedClient:           seedMgr.GetClient(),
		userClient:           userMgr.GetClient(),
		userRecorder:         userMgr.GetEventRecorderFor(controllerName),
		clusterIsPaused:      clusterIsPaused,
		appInstaller:         appInstaller,
		seedClusterNamespace: seedClusterNamespace,
		overwriteRegistry:    overwriteRegistry,
	}

	_, err := builder.ControllerManagedBy(userMgr).
		Named(controllerName).
		// update of the status with conditions or HelmInfo triggers an update event. To avoid reconciling in loop, we filter
		// update event on generation. We also allow update events if annotations have changed so that the user can force a
		// reconciliation without changing the spec.
		For(&appskubermaticv1.ApplicationInstallation{}, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}))).
		// We also watch ApplicationDefinition because it contains information about how to install the application. Moreover
		// if KKP admin deletes ApplicationDefinition, the related application must also be deleted.
		WatchesRawSource(source.Kind(
			seedMgr.GetCache(),
			&appskubermaticv1.ApplicationDefinition{},
			handler.TypedEnqueueRequestsFromMapFunc(enqueueAppInstallationForAppDef(r.userClient)),
		)).
		Build(r)

	return err
}

// Reconcile ApplicationInstallation (i.e. install / update or uninstall application into the user-cluster).
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("applicationinstallation", request)
	log.Debug("Processing")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	nodesAvailable, err := util.NodesAvailable(ctx, r.userClient)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check if nodes are available: %w", err)
	}
	if !nodesAvailable {
		r.log.Debug("waiting for nodes to join the cluster to be able to install applications")
		return reconcile.Result{RequeueAfter: initialRequeueDuration}, nil
	}

	appInstallation := &appskubermaticv1.ApplicationInstallation{}

	if err := r.userClient.Get(ctx, request.NamespacedName, appInstallation); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("applicationInstallation not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get applicationInstallation: %w", err)
	}

	err = r.reconcile(ctx, log, appInstallation)
	if err != nil {
		r.userRecorder.Event(appInstallation, corev1.EventTypeWarning, applicationInstallationReconcileFailedEvent, err.Error())
		return reconcile.Result{}, err
	}

	log.Debug("Processed")
	return reconcile.Result{RequeueAfter: appInstallation.Spec.ReconciliationInterval.Duration}, nil
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, appInstallation *appskubermaticv1.ApplicationInstallation) error {
	// handling deletion
	if !appInstallation.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, appInstallation); err != nil {
			return fmt.Errorf("handling deletion of application installation: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.userClient, appInstallation, appskubermaticv1.ApplicationInstallationCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	appHasBeenInstalled := appInstallation.Status.ApplicationVersion != nil

	// get applicationDefinition. If it can not be found, there are 2 cases:
	//   1) KKP admin has removed the applicationDefinition, and we have to remove the corresponding ApplicationInstallation(s)
	//   2) User made a mistake, or applicationDefinition has not been synced yet on this seed. So we just notify the user.
	applicationDef := &appskubermaticv1.ApplicationDefinition{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Name: appInstallation.Spec.ApplicationRef.Name}, applicationDef); err != nil {
		if apierrors.IsNotFound(err) {
			if appHasBeenInstalled {
				r.traceWarning(appInstallation, log, applicationDefinitionRemovedEvent, fmt.Sprintf("ApplicationDefinition '%s' has been deleted, removing applicationInstallation", applicationDef.Name))
				return r.userClient.Delete(ctx, appInstallation)
			} else {
				return fmt.Errorf("ApplicationDefinition '%s' does not exist. can not install application", applicationDef.Name)
			}
		}
		return err
	}

	if !applicationDef.DeletionTimestamp.IsZero() {
		r.traceWarning(appInstallation, log, applicationDefinitionDeletingEvent, fmt.Sprintf("ApplicationDefinition '%s' is being deleted,  removing applicationInstallation", applicationDef.Name))
		return r.userClient.Delete(ctx, appInstallation)
	}

	// get applicationVersion. If it can not be found, there are 2 cases:
	//   1) KKP admin has removed the applicationVersion, and we have to remove the corresponding ApplicationInstallation(s)
	//   2) User made a mistake, or applicationDefinition has not been synced yet on this seed. So we just notify the user.
	appVersion := &appskubermaticv1.ApplicationVersion{}
	if err := r.getApplicationVersion(appInstallation, applicationDef, appVersion); err != nil {
		if appHasBeenInstalled {
			r.traceWarning(appInstallation, log, applicationVersionRemovedEvent, fmt.Sprintf("applicationVersion: '%s' has been deleted. removing Application", appInstallation.Spec.ApplicationRef.Version))
			return r.userClient.Delete(ctx, appInstallation)
		} else {
			return fmt.Errorf("applicationVersion: '%s' does not exist. can not install application", appInstallation.Spec.ApplicationRef.Version)
		}
	}

	if !equality.Semantic.DeepEqual(appVersion, appInstallation.Status.ApplicationVersion) || appInstallation.Status.Method != applicationDef.Spec.Method {
		oldAppInstallation := appInstallation.DeepCopy()
		appInstallation.Status.ApplicationVersion = appVersion
		appInstallation.Status.Method = applicationDef.Spec.Method

		if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
			return fmt.Errorf("failed to update status with applicationVersion: %w", err)
		}
	}

	// for addons migrated to ee default-application-catalog we need to purge resources before re-installing them via helm
	if err := handleAddonCleanup(ctx, appInstallation.Name, r.seedClusterNamespace, r.seedClient, r.log); err != nil {
		return err
	}

	if r.overwriteRegistry != "" {
		err := r.useOverwriteRegistry(ctx, applicationDef, appInstallation)
		if err != nil {
			return fmt.Errorf("failed to overwrite the registry in application installation %w", err)
		}
	}

	// install application into the user-cluster
	if err := r.handleInstallation(ctx, log, applicationDef, appInstallation); err != nil {
		return fmt.Errorf("handling installation of application installation: %w", err)
	}

	return nil
}

func (r *reconciler) useOverwriteRegistry(ctx context.Context, appDefinition *appskubermaticv1.ApplicationDefinition, appInstallation *appskubermaticv1.ApplicationInstallation) error {
	if IsSystemApplication(appDefinition) {
		return r.updateValuesBlock(ctx, appDefinition, appInstallation)
	}
	return nil
}

// IsSystemApplication checks if the ApplicationDefinition is system application.
func IsSystemApplication(appDefinition *appskubermaticv1.ApplicationDefinition) bool {
	labels := appDefinition.GetLabels()
	return strings.EqualFold(labels["apps.kubermatic.k8c.io/managed-by"], "kkp")
}

// getApplicationVersion finds the applicationVersion defined by appInstallation into the applicationDef and updates the struct appVersion with it.
// An error is returned if the applicationVersion is not found.
func (r *reconciler) getApplicationVersion(appInstallation *appskubermaticv1.ApplicationInstallation, applicationDef *appskubermaticv1.ApplicationDefinition, appVersion *appskubermaticv1.ApplicationVersion) error {
	desiredVersion := appInstallation.Spec.ApplicationRef.Version
	for _, version := range applicationDef.Spec.Versions {
		if version.Version == desiredVersion {
			version.DeepCopyInto(appVersion)
			return nil
		}
	}
	return fmt.Errorf("application version '%s' does not exist in applicationDefinition %s", desiredVersion, applicationDef.Name)
}

// handleInstallation installs or updates the application in the user cluster.
func (r *reconciler) handleInstallation(ctx context.Context, log *zap.SugaredLogger, appDefinition *appskubermaticv1.ApplicationDefinition, appInstallation *appskubermaticv1.ApplicationInstallation) error {
	if err := r.resetFailuresIfSpecHasChanged(ctx, appInstallation); err != nil {
		return err
	}

	// Install or upgrade application only if max number of retries is not exceeded.
	if appInstallation.Status.Failures > maxRetries && hasLimitedRetries(appDefinition, appInstallation) {
		oldAppInstallation := appInstallation.DeepCopy()
		appInstallation.SetCondition(appskubermaticv1.Ready, corev1.ConditionFalse, "InstallationFailedRetriesExceeded", "Max number of retries was exceeded. Last error: "+oldAppInstallation.Status.Conditions[appskubermaticv1.Ready].Message)

		if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
		log.Infow("Max number of retries was exceeded. Do not reconcile application", "failures", appInstallation.Status.Failures, "maxRetries", maxRetries)
		return nil
	}

	// Because some upstream tools are not completely idempotent, we need a check to make sure a release is not stuck.
	// This should be run before we make any changes to the status field, so we can use it in our analysis
	stuck, err := r.appInstaller.IsStuck(ctx, log, r.seedClient, r.userClient, appInstallation)
	if err != nil {
		return fmt.Errorf("failed to check if the previous release is stuck: %w", err)
	}
	if stuck {
		log.Infof("Release for ApplicationInstallation seems to be stuck, attempting rollback now")
		if err := r.appInstaller.Rollback(ctx, log, r.seedClient, r.userClient, appInstallation); err != nil {
			return fmt.Errorf("failed to rollback release: %w", err)
		}
		log.Infof("Release for ApplicationInstallation has been rolled back successfully")
	}

	downloadDest, err := os.MkdirTemp(r.appInstaller.GetAppCache(), appInstallation.Namespace+"-"+appInstallation.Name)
	if err != nil {
		return fmt.Errorf("failed to create temporary directory where application source will be downloaded: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(downloadDest); err != nil {
			log.Error("failed to remove temporary directory where application source has been downloaded: %s", err)
		}
	}()

	// Download application sources.
	oldAppInstallation := appInstallation.DeepCopy()
	appSourcePath, downloadErr := r.appInstaller.DownloadSource(ctx, log, r.seedClient, appInstallation, downloadDest)
	if downloadErr != nil {
		appInstallation.SetCondition(appskubermaticv1.ManifestsRetrieved, corev1.ConditionFalse, "DownloadSourceFailed", downloadErr.Error())
		if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
		return downloadErr
	}
	appInstallation.SetCondition(appskubermaticv1.ManifestsRetrieved, corev1.ConditionTrue, "DownloadSourceSuccessful", "application's source successfully downloaded")
	appInstallation.SetCondition(appskubermaticv1.Ready, corev1.ConditionUnknown, "InstallationInProgress", "application is installing or upgrading")
	if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Install or upgrade application.
	oldAppInstallation = appInstallation.DeepCopy()
	statusUpdater, installErr := r.appInstaller.Apply(ctx, log, r.seedClient, r.userClient, appDefinition, appInstallation, appSourcePath)

	statusUpdater(&appInstallation.Status)
	appInstallation.SetReadyCondition(installErr, hasLimitedRetries(appDefinition, appInstallation))

	// we set condition in every case and condition update the LastHeartbeatTime. So patch will not be empty.
	if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return installErr
}

func hasLimitedRetries(appDefinition *appskubermaticv1.ApplicationDefinition, appInstallation *appskubermaticv1.ApplicationInstallation) bool {
	// todo VGR factorize code with pkg/applications/providers/template/helm.go::getDeployOpts
	// Read atomic from applicationInstallation.
	if appInstallation.Spec.DeployOptions != nil && appInstallation.Spec.DeployOptions.Helm != nil {
		return appInstallation.Spec.DeployOptions.Helm.Atomic
	}
	// Fallback to atomic defined in ApplicationDefinition.
	if appDefinition.Spec.DefaultDeployOptions != nil && appDefinition.Spec.DefaultDeployOptions.Helm != nil {
		return appDefinition.Spec.DefaultDeployOptions.Helm.Atomic
	}
	//  Fallback to default
	return false
}

// resetFailuresIfSpecHasChanged set Status.Failures to 0 if the spec has changed. Returns an error if status can not be updated.
func (r reconciler) resetFailuresIfSpecHasChanged(ctx context.Context, appInstallation *appskubermaticv1.ApplicationInstallation) error {
	oldAppInstallation := appInstallation.DeepCopy()
	if appInstallation.Status.Conditions[appskubermaticv1.Ready].ObservedGeneration != appInstallation.Generation {
		appInstallation.Status.Failures = 0
		if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
	}
	return nil
}

// handleDeletion uninstalls the application in the user cluster.
func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, appInstallation *appskubermaticv1.ApplicationInstallation) error {
	if kuberneteshelper.HasFinalizer(appInstallation, appskubermaticv1.ApplicationInstallationCleanupFinalizer) {
		statusUpdater, uninstallErr := r.appInstaller.Delete(ctx, log, r.seedClient, r.userClient, appInstallation)
		oldAppInstallation := appInstallation.DeepCopy()
		if uninstallErr != nil {
			appInstallation.SetCondition(appskubermaticv1.Ready, corev1.ConditionFalse, "UninstallFailed", uninstallErr.Error())
		}
		statusUpdater(&appInstallation.Status)

		if !equality.Semantic.DeepEqual(oldAppInstallation.Status, appInstallation.Status) { // avoid to send empty patch
			if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
				return fmt.Errorf("failed to update status: %w", err)
			}
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.userClient, appInstallation, appskubermaticv1.ApplicationInstallationCleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove application installation finalizer %s: %w", appInstallation.Name, err)
		}
	}
	return nil
}

// traceWarning logs the message in warning mode and raise a k8s event on appInstallation with the eventReason and the message.
func (r *reconciler) traceWarning(appInstallation *appskubermaticv1.ApplicationInstallation, log *zap.SugaredLogger, eventReason, message string) {
	log.Warn(message)
	r.userRecorder.Event(appInstallation, corev1.EventTypeWarning, eventReason, message)
}

// enqueueAppInstallationForAppDef fan-out updates from applicationDefinition to the ApplicationInstallation that reference
// this applicationDefinition.
func enqueueAppInstallationForAppDef(userClient ctrlruntimeclient.Client) func(context.Context, *appskubermaticv1.ApplicationDefinition) []reconcile.Request {
	return func(ctx context.Context, applicationDefinition *appskubermaticv1.ApplicationDefinition) []reconcile.Request {
		appList := &appskubermaticv1.ApplicationInstallationList{}
		if err := userClient.List(ctx, appList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list applicationInstallation: %w", err))
			return []reconcile.Request{}
		}

		var res []reconcile.Request
		for _, appInstallation := range appList.Items {
			if appInstallation.Spec.ApplicationRef.Name == applicationDefinition.GetName() {
				res = append(res, reconcile.Request{NamespacedName: types.NamespacedName{Name: appInstallation.Name, Namespace: appInstallation.Namespace}})
			}
		}
		return res
	}
}

func handleAddonCleanup(ctx context.Context, applicationName string, seedClusterNamespace string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	return applicationtemplates.HandleAddonCleanup(ctx, applicationName, seedClusterNamespace, seedClient, log)
}

// updateValuesBlock updates the valuesBlock of an ApplicationInstallation in-place
func (r *reconciler) updateValuesBlock(ctx context.Context, appDefinition *appskubermaticv1.ApplicationDefinition, appInstallation *appskubermaticv1.ApplicationInstallation) error {
	appName := appDefinition.Name
	getOverrideValues, exists := SystemAppsValuesGenerators[appName]
	if !exists {
		return nil
	}

	values, err := appInstallation.Spec.GetParsedValues()
	if err != nil {
		return fmt.Errorf("failed to unmarshal CNI values: %w", err)
	}

	// If (and only if) existing values is empty, use the initial values
	if len(values) == 0 {
		initialValues, err := appDefinition.Spec.GetParsedDefaultValues()
		if err != nil {
			return fmt.Errorf("failed to unmarshal ApplicationDefinition default values: %w", err)
		}
		values = initialValues
	}

	// Generate the Helm values
	overrideValues := getOverrideValues(appInstallation, r.overwriteRegistry)

	if err := mergo.Merge(&values, overrideValues, mergo.WithOverride); err != nil {
		return fmt.Errorf("failed to merge application values: %w", err)
	}

	rawValues, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal Helm values for %s: %w", appName, err)
	}

	// Update the valuesBlock field in-place
	appInstallation.Spec.ValuesBlock = string(rawValues)
	// Clear the deprecated .spec.values field to avoid conflicts
	appInstallation.Spec.Values = runtime.RawExtension{
		Raw: []byte("{}"),
	}

	return r.userClient.Update(ctx, appInstallation)
}
