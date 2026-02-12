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

	"k8c.io/kubermatic/sdk/v2/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-cluster-credentials-controller"
)

type reconciler struct {
	ctrlruntimeclient.Client

	workerName string
	recorder   events.EventRecorder
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
	kkpNamespace string,
) error {
	reconciler := &reconciler{
		Client: mgr.GetClient(),

		workerName: workerName,
		recorder:   mgr.GetEventRecorder(ControllerName),
		log:        log,
		versions:   versions,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}).
		Watches(
			&corev1.Secret{},
			newCredentialSecretHandler(mgr.GetClient()),
			builder.WithPredicates(predicate.ByNamespace(kkpNamespace)),
		).
		Build(reconciler)

	return err
}

func newCredentialSecretHandler(client ctrlruntimeclient.Client) handler.TypedEventHandler[ctrlruntimeclient.Object, reconcile.Request] {
	return handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, secret ctrlruntimeclient.Object) []reconcile.Request {
		clusters := &kubermaticv1.ClusterList{}
		if err := client.List(ctx, clusters); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %w", err))
			return nil
		}

		requests := []reconcile.Request{}
		for _, cluster := range clusters.Items {
			ref, err := resources.GetCredentialsReference(&cluster)
			if err != nil {
				// ignore "errors" here silently, all this function wants is to know which clusters use the
				// given secret, not which cluster are valid.
				continue
			}

			if ref == nil {
				continue
			}

			if ref.Name == secret.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&cluster),
				})
			}
		}

		return requests
	})
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionNone,
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

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
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
		log.Info("Moving cloud credentials into Secret…")
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

	creators := []reconciling.NamedSecretReconcilerFactory{
		secretReconciler(secret),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, cluster.Status.NamespaceName, r); err != nil {
		return nil, fmt.Errorf("failed to ensure credentials secret: %w", err)
	}

	return nil, nil
}

func secretReconciler(original *corev1.Secret) reconciling.NamedSecretReconcilerFactory {
	return func() (name string, create reconciling.SecretReconciler) {
		return resources.ClusterCloudCredentialsSecretName, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Data = original.Data

			return existing, nil
		}
	}
}
