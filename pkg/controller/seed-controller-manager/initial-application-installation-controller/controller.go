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
	"time"

	"go.uber.org/zap"

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
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
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	recorder                      events.EventRecorder
	seedGetter                    provider.SeedGetter
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

func Add(ctx context.Context, mgr manager.Manager, numWorkers int, workerName string, seedGetter provider.SeedGetter, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorder(ControllerName),
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
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(predicateutil.ByAnnotation(kubermaticv1.InitialApplicationInstallationsRequestAnnotation, "", false))).
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
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionApplicationInstallationControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
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

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// there is no annotation anymore
	request := cluster.Annotations[kubermaticv1.InitialApplicationInstallationsRequestAnnotation]
	if request == "" {
		return nil, nil
	}

	// Ensure that cluster is in a state when creating ApplicationInstallation is permissible
	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		r.log.Debug("Application controller not healthy")
		return nil, nil
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

	applications, err := r.parseApplications(request)
	if err != nil {
		if removeErr := r.removeAnnotation(ctx, cluster); removeErr != nil {
			return nil, fmt.Errorf("failed to remove invalid (%w) initial ApplicationInstallation annotation: %w", err, removeErr)
		}

		return nil, err
	}

	var errs []error
	for _, app := range applications {
		if err := r.createInitialApplicationInstallation(ctx, userClusterClient, app, cluster); err != nil {
			errs = append(errs, err)
			r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ApplicationInstallationFailed", "Reconciling", "Failed to create ApplicationInstallation %s", app.Name)
		}
	}

	if len(errs) > 0 {
		return nil, utilerrors.NewAggregate(errs)
	}

	if err := r.removeAnnotation(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to remove initial ApplicationInstallation annotation: %w", err)
	}

	return nil, nil
}

func (r *Reconciler) createInitialApplicationInstallation(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	application apiv1.Application,
	cluster *kubermaticv1.Cluster,
) error {
	namespace := application.Namespace
	if namespace == "" {
		namespace = application.Spec.Namespace.Name
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	// Before creating an application, make sure that the namespace exists.
	err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(ns), ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the namespace does not exist, create it
			err = client.Create(ctx, ns)
			if err != nil {
				return fmt.Errorf("failed to create namespace: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}

	applicationDefinition := appskubermaticv1.ApplicationDefinition{}

	err = r.Get(ctx, types.NamespacedName{Name: application.Spec.ApplicationRef.Name}, &applicationDefinition)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to get ApplicationDefinition %s: %w", application.Spec.ApplicationRef.Name, err)
	}

	// Create the application installation resource.
	applicationInstallation := appskubermaticv1.ApplicationInstallation{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   namespace,
			Annotations: application.Annotations,
			Labels:      applicationDefinition.Labels,
		},
		Spec: appskubermaticv1.ApplicationInstallationSpec{
			Namespace: &appskubermaticv1.AppNamespaceSpec{
				Name:        application.Spec.Namespace.Name,
				Create:      application.Spec.Namespace.Create,
				Labels:      application.Spec.Namespace.Labels,
				Annotations: application.Spec.Namespace.Annotations,
			},
			ApplicationRef: appskubermaticv1.ApplicationRef{
				Name:    application.Spec.ApplicationRef.Name,
				Version: application.Spec.ApplicationRef.Version,
			},
			Values:      runtime.RawExtension{Raw: application.Spec.Values},
			ValuesBlock: application.Spec.ValuesBlock,
		},
	}

	// At this point, the namespace exists, so we can create the application installation.
	err = client.Create(ctx, &applicationInstallation)
	if err != nil {
		// If the application already exists, we just ignore the error and move forward.
		return ctrlruntimeclient.IgnoreAlreadyExists(err)
	}

	r.recorder.Eventf(cluster, nil, corev1.EventTypeNormal, "ApplicationInstallationCreated", "Reconciling", "Initial ApplicationInstallation %s has been created", applicationInstallation.Name)

	return nil
}

func (r *Reconciler) parseApplications(request string) ([]apiv1.Application, error) {
	var applications []apiv1.Application
	if err := json.Unmarshal([]byte(request), &applications); err != nil {
		return nil, fmt.Errorf("cannot unmarshal initial Applications request: %w", err)
	}
	return applications, nil
}

func (r *Reconciler) removeAnnotation(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	oldCluster := cluster.DeepCopy()
	delete(cluster.Annotations, kubermaticv1.InitialApplicationInstallationsRequestAnnotation)
	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}
