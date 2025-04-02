/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package externalcluster

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aks"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/eks"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gke"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-external-cluster-controller"
)

// Reconciler is a controller which is responsible for managing clusters.
type Reconciler struct {
	ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	recorder record.EventRecorder
}

// Add creates a cluster controller.
func Add(ctx context.Context, mgr manager.Manager, log *zap.SugaredLogger) error {
	reconciler := &Reconciler{
		log:      log.Named(ControllerName),
		Client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor(ControllerName),
	}

	// Watch for changes to ExternalCluster except KubeOne and generic clusters.
	skipKubeOneClusters := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		externalCluster, ok := object.(*kubermaticv1.ExternalCluster)
		return ok && externalCluster.Spec.CloudSpec.ProviderName != kubermaticv1.ExternalClusterKubeOneProvider && externalCluster.Spec.CloudSpec.ProviderName != kubermaticv1.ExternalClusterBringYourOwnProvider
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		For(
			&kubermaticv1.ExternalCluster{},
			builder.WithPredicates(
				skipKubeOneClusters,
				predicate.GenerationChangedPredicate{},
			),
		).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	paused, err := kuberneteshelper.ExternalClusterPausedChecker(ctx, request.Name, r)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check external cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	log := r.log.With("externalcluster", request)
	log.Debug("Processing...")

	cluster := &kubermaticv1.ExternalCluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Could not find external cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	result, err := r.reconcile(ctx, log.With("provider", cluster.Spec.CloudSpec.ProviderName), cluster)
	if err != nil {
		switch {
		case isHTTPError(err, http.StatusNotFound):
			r.recorder.Event(cluster, corev1.EventTypeWarning, "ResourceNotFound", err.Error())
			err = nil
		case isHTTPError(err, http.StatusForbidden):
			r.recorder.Event(cluster, corev1.EventTypeWarning, "AuthorizationFailed", err.Error())
			err = nil
		default:
			r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		}
	}

	return result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.ExternalCluster) (reconcile.Result, error) {
	// handling deletion
	if !cluster.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, cluster); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion of external cluster: %w", err)
		}
		return reconcile.Result{}, nil
	}

	cloud := cluster.Spec.CloudSpec
	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, r)

	if cloud.ProviderName == kubermaticv1.ExternalClusterBringYourOwnProvider {
		if cluster.Spec.KubeconfigReference != nil {
			if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, kubermaticv1.ExternalClusterKubeconfigCleanupFinalizer); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to add kubeconfig secret finalizer: %w", err)
			}
		}
		return reconcile.Result{}, nil
	}

	if cloud.GKE != nil {
		log.Debug("Reconciling GKE cluster")
		if cloud.GKE.CredentialsReference != nil {
			if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to add credential secret finalizer: %w", err)
			}
		}
		condition, err := gke.GetClusterStatus(ctx, secretKeySelector, cloud.GKE)
		if err != nil {
			if isHTTPError(err, http.StatusNotFound) {
				condition := &kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: err.Error(),
				}
				return reconcile.Result{}, r.updateStatus(ctx, *condition, cluster)
			} else {
				return reconcile.Result{}, err
			}
		}
		if condition.Phase == kubermaticv1.ExternalClusterPhaseProvisioning {
			if cluster.Status.Condition.Phase != condition.Phase {
				if err := r.updateStatus(ctx, *condition, cluster); err != nil {
					return reconcile.Result{}, err
				}
			}
			// repeat after some time to get/store kubeconfig
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
		if err := r.updateStatus(ctx, *condition, cluster); err != nil {
			return reconcile.Result{}, err
		}
		if condition.Phase == kubermaticv1.ExternalClusterPhaseRunning || condition.Phase == kubermaticv1.ExternalClusterPhaseReconciling {
			if err := r.ensureGKEKubeconfig(ctx, cluster); err != nil {
				condition := &kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: err.Error(),
				}
				if err := r.updateStatus(ctx, *condition, cluster); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, err
			}
		}
		if err := r.updateStatus(ctx, *condition, cluster); err != nil {
			return reconcile.Result{}, err
		}
		// updating status every 5 minutes
		return reconcile.Result{RequeueAfter: time.Minute * 5}, nil
	}

	if cloud.EKS != nil {
		log.Debug("Reconciling EKS cluster")
		if cloud.EKS.CredentialsReference != nil {
			if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to add credential secret finalizer: %w", err)
			}
		}
		condition, err := eks.GetClusterStatus(ctx, secretKeySelector, cloud.EKS)
		if err != nil {
			if isHTTPError(err, http.StatusNotFound) {
				condition := &kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: err.Error(),
				}
				return reconcile.Result{}, r.updateStatus(ctx, *condition, cluster)
			} else {
				return reconcile.Result{}, err
			}
		}
		if condition.Phase == kubermaticv1.ExternalClusterPhaseProvisioning {
			if cluster.Status.Condition.Phase != condition.Phase {
				if err := r.updateStatus(ctx, *condition, cluster); err != nil {
					return reconcile.Result{}, err
				}
			}
			// repeat after some time to get/store kubeconfig
			return reconcile.Result{RequeueAfter: time.Second * 10}, nil
		}
		if err := r.updateStatus(ctx, *condition, cluster); err != nil {
			return reconcile.Result{}, err
		}
		if condition.Phase == kubermaticv1.ExternalClusterPhaseRunning || condition.Phase == kubermaticv1.ExternalClusterPhaseReconciling {
			if err := r.ensureEKSKubeconfig(ctx, cluster); err != nil {
				condition := &kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: err.Error(),
				}
				if err := r.updateStatus(ctx, *condition, cluster); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, err
			}
		}
		// updating status every 5 minutes
		return reconcile.Result{RequeueAfter: time.Minute * 5}, nil
	}

	if cloud.AKS != nil {
		log.Debug("Reconciling AKS cluster")
		if cloud.AKS.CredentialsReference != nil {
			if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to add credential secret finalizer: %w", err)
			}
		}
		condition, err := aks.GetClusterStatus(ctx, secretKeySelector, cloud.AKS)
		if err != nil {
			if isHTTPError(err, http.StatusNotFound) {
				condition := &kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: err.Error(),
				}
				return reconcile.Result{}, r.updateStatus(ctx, *condition, cluster)
			}

			return reconcile.Result{}, err
		}
		if condition.Phase == kubermaticv1.ExternalClusterPhaseProvisioning {
			if cluster.Status.Condition.Phase != condition.Phase {
				if err := r.updateStatus(ctx, *condition, cluster); err != nil {
					return reconcile.Result{}, err
				}
			}
			// repeat after some time to get/store kubeconfig
			return reconcile.Result{RequeueAfter: time.Second * 10}, nil
		}
		if err := r.updateStatus(ctx, *condition, cluster); err != nil {
			return reconcile.Result{}, err
		}
		if condition.Phase == kubermaticv1.ExternalClusterPhaseRunning || condition.Phase == kubermaticv1.ExternalClusterPhaseReconciling {
			if err := r.ensureAKSKubeconfig(ctx, cluster); err != nil {
				condition := &kubermaticv1.ExternalClusterCondition{
					Phase:   kubermaticv1.ExternalClusterPhaseError,
					Message: err.Error(),
				}
				if err := r.updateStatus(ctx, *condition, cluster); err != nil {
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, err
			}
		}
		// reconcile to update kubeconfig for cases like starting a stopped cluster
		return reconcile.Result{RequeueAfter: time.Minute * 2}, nil
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) handleDeletion(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	if kuberneteshelper.HasFinalizer(cluster, kubermaticv1.ExternalClusterKubeconfigCleanupFinalizer) {
		if err := r.cleanUpKubeconfigSecret(ctx, cluster); err != nil {
			return err
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer) {
		if err := r.cleanUpCredentialsSecret(ctx, cluster); err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) cleanUpKubeconfigSecret(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	if err := r.deleteSecret(ctx, cluster.GetKubeconfigSecretName()); err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r, cluster, kubermaticv1.ExternalClusterKubeconfigCleanupFinalizer)
}

func (r *Reconciler) cleanUpCredentialsSecret(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	if err := r.deleteSecret(ctx, cluster.GetCredentialsSecretName()); err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer)
}

func (r *Reconciler) deleteSecret(ctx context.Context, secretName string) error {
	if secretName == "" {
		return nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: secretName, Namespace: resources.KubermaticNamespace}
	err := r.Get(ctx, name, secret)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get Secret %q: %w", name.String(), err)
	}

	if err := r.Delete(ctx, secret); err != nil {
		return fmt.Errorf("failed to delete Secret %q: %w", name.String(), err)
	}

	// We successfully deleted the secret
	return nil
}

func (r *Reconciler) updateStatus(ctx context.Context, condition kubermaticv1.ExternalClusterCondition, cluster *kubermaticv1.ExternalCluster) error {
	oldCluster := cluster.DeepCopy()
	cluster.Status.Condition = condition
	if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to patch cluster with new phase %q: %w", condition, err)
	}
	return nil
}

func (r *Reconciler) ensureGKEKubeconfig(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	cloud := cluster.Spec.CloudSpec
	cred, err := resources.GetGKECredentials(ctx, r, cluster)
	if err != nil {
		return err
	}
	config, err := gke.GetClusterConfig(ctx, cred.ServiceAccount, cloud.GKE.Name, cloud.GKE.Zone)
	if err != nil {
		return err
	}

	return r.ensureKubeconfigSecret(ctx, config, cluster)
}

func (r *Reconciler) ensureEKSKubeconfig(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	cloud := cluster.Spec.CloudSpec
	cred, err := resources.GetEKSCredentials(ctx, r, cluster)
	if err != nil {
		return err
	}
	config, err := eks.GetClusterConfig(ctx, cred.AccessKeyID, cred.SecretAccessKey, cloud.EKS.Name, cloud.EKS.Region)
	if err != nil {
		return err
	}

	return r.ensureKubeconfigSecret(ctx, config, cluster)
}

func (r *Reconciler) ensureAKSKubeconfig(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	cloud := cluster.Spec.CloudSpec
	cred, err := resources.GetAKSCredentials(ctx, r, cluster)
	if err != nil {
		return err
	}
	config, err := aks.GetClusterConfig(ctx, cred, cloud.AKS.Name, cloud.AKS.ResourceGroup)
	if err != nil {
		return err
	}

	return r.ensureKubeconfigSecret(ctx, config, cluster)
}

func (r *Reconciler) ensureKubeconfigSecret(ctx context.Context, config *api.Config, cluster *kubermaticv1.ExternalCluster) error {
	if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, kubermaticv1.ExternalClusterKubeconfigCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	kubeconfig, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	secretData := map[string][]byte{
		resources.ExternalClusterKubeconfig: kubeconfig,
	}

	creators := []reconciling.NamedSecretReconcilerFactory{
		kubeconfigSecretReconcilerFactory(cluster, secretData),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, resources.KubermaticNamespace, r); err != nil {
		return fmt.Errorf("failed to ensure Secret: %w", err)
	}

	cluster.Spec.KubeconfigReference = &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      cluster.GetKubeconfigSecretName(),
			Namespace: resources.KubermaticNamespace,
		},
	}

	return r.Update(ctx, cluster)
}

func kubeconfigSecretReconcilerFactory(cluster *kubermaticv1.ExternalCluster, secretData map[string][]byte) reconciling.NamedSecretReconcilerFactory {
	return func() (name string, create reconciling.SecretReconciler) {
		return cluster.GetKubeconfigSecretName(), func(existing *corev1.Secret) (*corev1.Secret, error) {
			if existing.Labels == nil {
				existing.Labels = map[string]string{}
			}

			existing.Labels[kubermaticv1.ProjectIDLabelKey] = cluster.Labels[kubermaticv1.ProjectIDLabelKey]
			existing.Data = secretData

			return existing, nil
		}
	}
}

func isHTTPError(err error, status int) bool {
	var httpError utilerrors.HTTPError
	if errors.As(err, &httpError) {
		if httpError.StatusCode() == status {
			return true
		}
	}
	return false
}
