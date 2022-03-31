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

	"go.uber.org/zap"

	appkubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "kkp-app-installation-controller"
)

type reconciler struct {
	log             *zap.SugaredLogger
	seedClient      ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	userRecorder    record.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		seedClient:      seedMgr.GetClient(),
		userClient:      userMgr.GetClient(),
		userRecorder:    userMgr.GetEventRecorderFor(controllerName),
		clusterIsPaused: clusterIsPaused,
	}

	c, err := controller.New(controllerName, userMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller %s: %w", controllerName, err)
	}

	if err = c.Watch(&source.Kind{Type: &appkubermaticv1.ApplicationInstallation{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch for ApplicationInstallation: %w", err)
	}

	// We also watch ApplicationDefinition  because contains information about how to install the application. Morover
	// if KKP admin delete ApplicationDefinition, the related application must be also deleted.
	appDefInformer, err := seedMgr.GetCache().GetInformer(ctx, &appkubermaticv1.ApplicationDefinition{})
	if err != nil {
		return fmt.Errorf("failed to get informer for applicationDefinition: %w", err)
	}

	if err = c.Watch(&source.Informer{Informer: appDefInformer}, handler.EnqueueRequestsFromMapFunc(enqueueAppInstallationForAppDef(ctx, log, r.userClient))); err != nil {
		return fmt.Errorf("failed to watch applicationDefinition: %w", err)
	}

	return nil
}

// Reconcile ApplicationInstallation (ie install / update or uninstall applicationinto user cluster).
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	appInstallation := &appkubermaticv1.ApplicationInstallation{}

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
		r.userRecorder.Event(appInstallation, corev1.EventTypeWarning, "ApplicationInstallationReconcileFailed", err.Error())
	}

	log.Debug("Processed")
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, appInstallation *appkubermaticv1.ApplicationInstallation) error {
	// to implements logic in next PR
	return nil
}

// enqueueAppInstallationForAppDef fan-out updates from applicationDefinition to the ApplicationInstallation that reference
// this applicationDefinition.
func enqueueAppInstallationForAppDef(ctx context.Context, log *zap.SugaredLogger, userClient ctrlruntimeclient.Client) func(object ctrlruntimeclient.Object) []reconcile.Request {
	return func(applicationDefinition ctrlruntimeclient.Object) []reconcile.Request {
		appList := &appkubermaticv1.ApplicationInstallationList{}
		if err := userClient.List(ctx, appList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list applicationInstallation: %w", err))
			return []reconcile.Request{}
		}

		var res []reconcile.Request
		for _, appInstallation := range appList.Items {
			if appInstallation.Spec.ApplicationRef.Name == applicationDefinition.GetName() {
				res = append(res, reconcile.Request{NamespacedName: types.NamespacedName{Name: appInstallation.Name}})
			}
		}

		if res != nil {
			log.Debugf("ApplicationDefinition '%s' has changed. Enqueue ApplicationInstallations %v", applicationDefinition.GetName(), res)
		}
		return res
	}
}
