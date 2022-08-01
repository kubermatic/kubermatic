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

package clustercredentialscontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-cluster-credentials-controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName string
	recorder   record.EventRecorder
	log        *zap.SugaredLogger
	versions   kubermatic.Versions
}

// Add creates a new cluster-credentials controller.
func Add(
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	log *zap.SugaredLogger,
	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName: workerName,
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		log:        log,
		versions:   versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}

	return nil
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

	// do not migrate a cluster in deletion
	if cluster.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionNone,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Failed to reconcile cluster", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	oldCluster := cluster.DeepCopy()

	// add the cleanup finalizer first (the pkg/clusterdeletion takes care of cleaning up)
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer) {
		if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer); err != nil {
			return nil, fmt.Errorf("failed to add finalizer: %w", err)
		}

		return &reconcile.Result{Requeue: true}, nil
	}

	// make sure cluster credentials are placed in a dedicated Secret in the KKP namespace
	if err := kubernetesprovider.CreateOrUpdateCredentialSecretForCluster(ctx, r, cluster); err != nil {
		return nil, fmt.Errorf("failed to migrate Cluster credentials: %w", err)
	}

	// if the function above performed some magic, we need to persist the change and requeue
	if !equality.Semantic.DeepEqual(oldCluster.Spec.Cloud, cluster.Spec.Cloud) {
		if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return nil, fmt.Errorf("failed to patch cluster with credentials secret: %w", err)
		}

		return &reconcile.Result{Requeue: true}, nil
	}

	// Now that a Secret was (possibly) created in the KKP namespace, duplicate it into
	// the cluster namespace so that Deployments like the kube-apiserver can reference it.

	// We need a cluster namespace to mirror the Secret.
	if cluster.Status.NamespaceName == "" {
		return nil, nil
	}

	reference, err := resources.GetCredentialsReference(cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to determine cluster credentials: %w", err)
	}

	// Clusters using BYO provider do not have a credential secret.
	if reference == nil {
		return nil, nil
	}

	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: reference.Name, Namespace: reference.Namespace}, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials secret: %w", err)
	}

	creators := []reconciling.NamedSecretCreatorGetter{
		secretCreator(secret),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, cluster.Status.NamespaceName, r); err != nil {
		return nil, fmt.Errorf("failed to ensure credentials secret: %w", err)
	}

	return nil, nil
}

func secretCreator(original *corev1.Secret) reconciling.NamedSecretCreatorGetter {
	return func() (name string, create reconciling.SecretCreator) {
		return resources.ClusterCloudCredentialsSecretName, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Data = original.Data

			return existing, nil
		}
	}
}
