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

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/apis/equality"
	"k8c.io/kubermatic/v2/pkg/applications"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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
)

type reconciler struct {
	log             *zap.SugaredLogger
	seedClient      ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	userRecorder    record.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
	appInstaller    applications.ApplicationInstaller
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker, appInstaller applications.ApplicationInstaller) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		seedClient:      seedMgr.GetClient(),
		userClient:      userMgr.GetClient(),
		userRecorder:    userMgr.GetEventRecorderFor(controllerName),
		clusterIsPaused: clusterIsPaused,
		appInstaller:    appInstaller,
	}

	c, err := controller.New(controllerName, userMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller %s: %w", controllerName, err)
	}

	// update of the status with conditions or HelmInfo triggers an update event. To avoid reconciling in loop, we filter
	// update event on generation. We also allow update events if annotations have changed so that the user can force a
	// reconciliation without changing the spec.
	if err = c.Watch(&source.Kind{Type: &appskubermaticv1.ApplicationInstallation{}}, &handler.EnqueueRequestForObject{}, predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{})); err != nil {
		return fmt.Errorf("failed to create watch for ApplicationInstallation: %w", err)
	}

	// We also watch ApplicationDefinition because it contains information about how to install the application. Moreover
	// if KKP admin deletes ApplicationDefinition, the related application must also be deleted.
	appDefInformer, err := seedMgr.GetCache().GetInformer(ctx, &appskubermaticv1.ApplicationDefinition{})
	if err != nil {
		return fmt.Errorf("failed to get informer for applicationDefinition: %w", err)
	}

	if err = c.Watch(&source.Informer{Informer: appDefInformer}, handler.EnqueueRequestsFromMapFunc(enqueueAppInstallationForAppDef(ctx, r.userClient))); err != nil {
		return fmt.Errorf("failed to watch applicationDefinition: %w", err)
	}

	return nil
}

// Reconcile ApplicationInstallation (ie install / update or uninstall applicationinto the user-cluster).
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
		log.Errorw("ReconcilingError", zap.Error(err))
		r.userRecorder.Event(appInstallation, corev1.EventTypeWarning, applicationInstallationReconcileFailedEvent, err.Error())
	}

	log.Debug("Processed")
	return reconcile.Result{RequeueAfter: appInstallation.Spec.ReconciliationInterval.Duration}, err
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

	// install application into the user-cluster
	if err := r.handleInstallation(ctx, log, applicationDef, appInstallation); err != nil {
		return fmt.Errorf("handling installation of application installation: %w", err)
	}

	return nil
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
	downloadDest, err := os.MkdirTemp(r.appInstaller.GetAppCache(), appInstallation.Namespace+"-"+appInstallation.Name)
	if err != nil {
		return fmt.Errorf("failed to create temporary directory where application source will be downloaded: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(downloadDest); err != nil {
			log.Error("failed to remove temporary directory where application source has been downloaded: %s", err)
		}
	}()

	oldAppInstallation := appInstallation.DeepCopy()
	appSourcePath, downloadErr := r.appInstaller.DonwloadSource(ctx, log, r.seedClient, appInstallation, downloadDest)
	if downloadErr != nil {
		appInstallation.SetCondition(appskubermaticv1.ManifestsRetrieved, corev1.ConditionFalse, "DownaloadSourceFailed", downloadErr.Error())
		if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
			return fmt.Errorf("failed to update status: %w", err)
		}
		return downloadErr
	}
	appInstallation.SetCondition(appskubermaticv1.ManifestsRetrieved, corev1.ConditionTrue, "DownaloadSourceSuccessful", "application's source successfully downloaded")
	if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	oldAppInstallation = appInstallation.DeepCopy()
	statusUpdater, installErr := r.appInstaller.Apply(ctx, log, r.seedClient, r.userClient, appDefinition, appInstallation, appSourcePath)

	if installErr != nil {
		appInstallation.SetCondition(appskubermaticv1.Ready, corev1.ConditionFalse, "InstallationFailed", installErr.Error())
	} else {
		appInstallation.SetCondition(appskubermaticv1.Ready, corev1.ConditionTrue, "InstallationSuccessful", "application successfully installed or upgraded")
	}
	statusUpdater(&appInstallation.Status)

	// we set condition in every case and condition update the LastHeartbeatTime. So patch will not be empty.
	if err := r.userClient.Status().Patch(ctx, appInstallation, ctrlruntimeclient.MergeFrom(oldAppInstallation)); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return installErr
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
func enqueueAppInstallationForAppDef(ctx context.Context, userClient ctrlruntimeclient.Client) func(object ctrlruntimeclient.Object) []reconcile.Request {
	return func(applicationDefinition ctrlruntimeclient.Object) []reconcile.Request {
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
