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

package initialapplicationinstallationcontroller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-openapi/errors"
	"go.uber.org/zap"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-initial-application-installation-controller"
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

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, predicateutil.ByAnnotation(apiv1.InitialApplicationInstallationsRequestAnnotation, "", false)); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}

	return nil
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
		kubermaticv1.ClusterConditionApplicationInstallationControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)
	if err != nil {
		r.log.Errorw("Failed to reconcile cluster", "cluster", cluster.Name, zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// there is no annotation anymore
	request := cluster.Annotations[apiv1.InitialApplicationInstallationsRequestAnnotation]
	if request == "" {
		return nil, nil
	}

	// Ensure that cluster is in a state when creating ApplicationInstallation is permissible
	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		r.log.Debug("Application controller not healthy")
		return nil, nil
	}

	applications, err := r.parseApplications(cluster, request)
	if err != nil {
		if removeErr := r.removeAnnotation(ctx, cluster); removeErr != nil {
			return nil, fmt.Errorf("failed to remove invalid (%v) initial ApplicationInstallation annotation: %w", err, removeErr)
		}

		return nil, err
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cluster client: %w", err)
	}

	var errs []error
	for _, app := range applications {
		if err := r.createInitialApplicationInstallations(ctx, userClusterClient, app, cluster); err != nil {
			errs = append(errs, err)
			r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ApplicationInstallationFailed", "Failed to create ApplicationInstallion %s", app.Name)
		}
	}

	if len(errs) > 0 {
		return nil, errors.CompositeValidationError(errs...)
	}

	if err := r.removeAnnotation(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to remove initial ApplicationInstallation annotation: %w", err)
	}

	return nil, nil
}

func (r *Reconciler) createInitialApplicationInstallations(ctx context.Context, client ctrlruntimeclient.Client, application apiv1.Application, cluster *kubermaticv1.Cluster) error {
	applicationInstallation := appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   metav1.NamespaceSystem,
			Annotations: application.Annotations,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: appskubermaticv1.NamespaceSpec{
				Name:        application.Spec.Namespace.Name,
				Create:      application.Spec.Namespace.Create,
				Labels:      application.Spec.Namespace.Labels,
				Annotations: application.Spec.Namespace.Annotations,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    application.Spec.ApplicationRef.Name,
				Version: application.Spec.ApplicationRef.Version,
			},
			Values: runtime.RawExtension{Raw: application.Spec.Values},
		},
	}

	err := client.Create(ctx, &applicationInstallation)
	if err != nil {
		// If the application already exists, we just ignore the error and move forward.
		if apierrors.IsAlreadyExists(err) {
			return nil
		}

		return err
	}

	r.recorder.Eventf(cluster, corev1.EventTypeNormal, "ApplicationInstallationCreated", "Initial ApplicationInstallion %s has been created", applicationInstallation.Name)

	return nil
}

func (r *Reconciler) parseApplications(cluster *kubermaticv1.Cluster, request string) ([]apiv1.Application, error) {
	var applications []apiv1.Application
	if err := json.Unmarshal([]byte(request), &applications); err != nil {
		return nil, fmt.Errorf("cannot unmarshal initial Applications request: %w", err)
	}
	return applications, nil
}

func (r *Reconciler) removeAnnotation(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	oldCluster := cluster.DeepCopy()
	delete(cluster.Annotations, apiv1.InitialApplicationInstallationsRequestAnnotation)
	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}
