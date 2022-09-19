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

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aks"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/eks"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gke"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
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
	ControllerName = "kkp-external-cluster-controller"
)

// Reconciler is a controller which is responsible for managing clusters.
type Reconciler struct {
	ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	recorder record.EventRecorder
}

// Add creates a cluster controller.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger) error {
	reconciler := &Reconciler{
		log:      log.Named(ControllerName),
		Client:   mgr.GetClient(),
		recorder: mgr.GetEventRecorderFor(ControllerName),
	}
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	// Watch for changes to ExternalCluster except KubeOne and generic clusters.
	skipKubeOneClusters := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		externalCluster, ok := object.(*kubermaticv1.ExternalCluster)
		return ok && externalCluster.Spec.CloudSpec.ProviderName != kubermaticv1.ExternalClusterKubeOneProvider && externalCluster.Spec.CloudSpec.ProviderName != kubermaticv1.ExternalClusterBringYourOwnProvider
	})

	return c.Watch(&source.Kind{Type: &kubermaticv1.ExternalCluster{}}, &handler.EnqueueRequestForObject{}, skipKubeOneClusters, predicate.GenerationChangedPredicate{})
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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
		case isHttpError(err, http.StatusNotFound):
			r.recorder.Event(cluster, corev1.EventTypeWarning, "ResourceNotFound", err.Error())
			err = nil
		case isHttpError(err, http.StatusForbidden):
			r.recorder.Event(cluster, corev1.EventTypeWarning, "AuthorizationFailed", err.Error())
			err = nil
		default:
			log.Errorw("Reconciling failed", zap.Error(err))
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
	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, r.Client)

	if cloud.ProviderName == kubermaticv1.ExternalClusterBringYourOwnProvider {
		if cluster.Spec.KubeconfigReference != nil {
			if err := kuberneteshelper.TryAddFinalizer(ctx, r.Client, cluster, kubermaticv1.ExternalClusterKubeconfigCleanupFinalizer); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to add kubeconfig secret finalizer: %w", err)
			}
		}
		return reconcile.Result{}, nil
	}

	if cloud.GKE != nil {
		log.Debug("Reconciling GKE cluster")
		if cloud.GKE.CredentialsReference != nil {
			if err := kuberneteshelper.TryAddFinalizer(ctx, r.Client, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to add credential secret finalizer: %w", err)
			}
		}
		status, err := gke.GetClusterStatus(ctx, secretKeySelector, cloud.GKE)
		if err != nil {
			return reconcile.Result{}, err
		}
		if status.State == apiv2.ProvisioningExternalClusterState {
			// repeat after some time to get/store kubeconfig
			return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
		}
		if status.State == apiv2.RunningExternalClusterState || status.State == apiv2.ReconcilingExternalClusterState {
			// the kubeconfig token is valid 1h, it will update token every 30min
			return reconcile.Result{RequeueAfter: time.Minute * 30}, r.ensureGKEKubeconfig(ctx, cluster)
		}
	}

	if cloud.EKS != nil {
		log.Debug("Reconciling EKS cluster")
		if cloud.EKS.CredentialsReference != nil {
			if err := kuberneteshelper.TryAddFinalizer(ctx, r.Client, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to add credential secret finalizer: %w", err)
			}
		}
		status, err := eks.GetClusterStatus(ctx, secretKeySelector, cloud.EKS)
		if err != nil {
			return reconcile.Result{}, err
		}
		if status.State == apiv2.ProvisioningExternalClusterState {
			// repeat after some time to get/store kubeconfig
			return reconcile.Result{RequeueAfter: time.Second * 10}, nil
		}
		if status.State == apiv2.RunningExternalClusterState || status.State == apiv2.ReconcilingExternalClusterState {
			// the kubeconfig token is valid 14min, it will update token every 10min
			return reconcile.Result{RequeueAfter: time.Minute * 10}, r.ensureEKSKubeconfig(ctx, cluster)
		}
	}

	if cloud.AKS != nil {
		log.Debug("Reconciling AKS cluster")
		if cloud.AKS.CredentialsReference != nil {
			if err := kuberneteshelper.TryAddFinalizer(ctx, r.Client, cluster, kubermaticv1.CredentialsSecretsCleanupFinalizer); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to add credential secret finalizer: %w", err)
			}
		}
		status, err := aks.GetClusterStatus(ctx, secretKeySelector, cloud.AKS)
		if err != nil {
			return reconcile.Result{}, err
		}
		if status.State == apiv2.ProvisioningExternalClusterState {
			// repeat after some time to get/store kubeconfig
			return reconcile.Result{RequeueAfter: time.Second * 10}, nil
		}
		if status.State == apiv2.RunningExternalClusterState || status.State == apiv2.ReconcilingExternalClusterState {
			// reconcile to update kubeconfig for cases like starting a stopped cluster
			return reconcile.Result{RequeueAfter: time.Minute * 2}, r.ensureAKSKubeconfig(ctx, cluster)
		}
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

func (r *Reconciler) ensureGKEKubeconfig(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	cloud := cluster.Spec.CloudSpec
	cred, err := resources.GetGKECredentials(ctx, r.Client, cluster)
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
	cred, err := resources.GetEKSCredentials(ctx, r.Client, cluster)
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
	cred, err := resources.GetAKSCredentials(ctx, r.Client, cluster)
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

	creators := []reconciling.NamedSecretCreatorGetter{
		kubeconfigSecretCreatorGetter(cluster, secretData),
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

func kubeconfigSecretCreatorGetter(cluster *kubermaticv1.ExternalCluster, secretData map[string][]byte) reconciling.NamedSecretCreatorGetter {
	return func() (name string, create reconciling.SecretCreator) {
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

func isHttpError(err error, status int) bool {
	var httpError utilerrors.HTTPError
	if errors.As(err, &httpError) {
		if httpError.StatusCode() == status {
			return true
		}
	}
	return false
}
