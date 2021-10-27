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
	"encoding/base64"
	"fmt"
	"time"

	"go.uber.org/zap"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "external_cluster_controller"
)

// Reconciler is a controller which is responsible for managing clusters
type Reconciler struct {
	ctrlruntimeclient.Client
	log *zap.SugaredLogger
}

// Add creates a cluster controller.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger) error {
	reconciler := &Reconciler{
		log:    log.Named(ControllerName),
		Client: mgr.GetClient(),
	}
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}
	// Watch for changes to ExternalCluster
	err = c.Watch(&source.Kind{Type: &kubermaticv1.ExternalCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil

}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	resourceName := request.Name
	log := r.log.With("request", request)
	log.Debug("Processing")

	icl := &kubermaticv1.ExternalCluster{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: metav1.NamespaceAll, Name: resourceName}, icl); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Could not find imported cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if icl.DeletionTimestamp != nil {
		if kuberneteshelper.HasFinalizer(icl, kubermaticapiv1.ExternalClusterKubeconfigCleanupFinalizer) {
			if err := r.cleanUpKubeconfigSecret(ctx, icl); err != nil {
				log.Errorf("Could not delete kubeconfig secret, %v", err)
				return reconcile.Result{}, err
			}
		}
		if kuberneteshelper.HasFinalizer(icl, kubermaticapiv1.CredentialsSecretsCleanupFinalizer) {
			if err := r.cleanUpCredentialsSecret(ctx, icl); err != nil {
				log.Errorf("Could not delete credentials secret, %v", err)
				return reconcile.Result{}, err
			}
		}
	}

	return r.reconcile(ctx, icl)
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (reconcile.Result, error) {
	cloud := cluster.Spec.CloudSpec
	if cloud == nil {
		return reconcile.Result{}, nil
	}
	if cloud.GKE != nil {
		r.log.Debugf("reconcile GKE cluster")
		err := r.createOrUpdateGKEKubeconfig(ctx, cluster)
		if err != nil {
			r.log.Errorf("failed to create or update kubeconfig secret %v", err)
			return reconcile.Result{}, err
		}
		// the kubeconfig token is valid 1h, it will update token every 30min
		return reconcile.Result{RequeueAfter: time.Minute * 30}, nil
	}
	if cloud.EKS != nil {
		r.log.Debugf("reconcile EKS cluster")
		err := r.createOrUpdateEKSKubeconfig(ctx, cluster)
		if err != nil {
			r.log.Errorf("failed to create or update kubeconfig secret %v", err)
			return reconcile.Result{}, err
		}
		// the kubeconfig token is valid 1h, it will update token every 30min
		return reconcile.Result{RequeueAfter: time.Minute * 30}, nil
	}
	return reconcile.Result{}, nil
}

func (r *Reconciler) cleanUpKubeconfigSecret(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	if err := r.deleteSecret(ctx, cluster.GetKubeconfigSecretName()); err != nil {
		return err
	}

	oldCluster := cluster.DeepCopy()
	kuberneteshelper.RemoveFinalizer(cluster, kubermaticapiv1.ExternalClusterKubeconfigCleanupFinalizer)
	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (r *Reconciler) cleanUpCredentialsSecret(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	if err := r.deleteSecret(ctx, cluster.GetCredentialsSecretName()); err != nil {
		return err
	}

	oldCluster := cluster.DeepCopy()
	kuberneteshelper.RemoveFinalizer(cluster, kubermaticapiv1.CredentialsSecretsCleanupFinalizer)
	return r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (r *Reconciler) deleteSecret(ctx context.Context, secretName string) error {
	if secretName == "" {
		return nil
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{Name: secretName, Namespace: resources.KubermaticNamespace}
	err := r.Get(ctx, name, secret)
	// Its already gone
	if kerrors.IsNotFound(err) {
		return nil
	}

	// Something failed while loading the secret
	if err != nil {
		return fmt.Errorf("failed to get Secret %q: %v", name.String(), err)
	}

	if err := r.Delete(ctx, secret); err != nil {
		return fmt.Errorf("failed to delete Secret %q: %v", name.String(), err)
	}

	// We successfully deleted the secret
	return nil
}

func (r *Reconciler) getGKECredentials(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (string, error) {
	cloud := cluster.Spec.CloudSpec
	if cloud == nil {
		return "", fmt.Errorf("cloud struct is empty")
	}
	if cloud.GKE == nil {
		return "", fmt.Errorf("gke cloud struct is empty")
	}
	if cloud.GKE.CredentialsReference == nil {
		return "", fmt.Errorf("no credentials provided")
	}
	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetCredentialsSecretName()}
	if err := r.Get(ctx, namespacedName, secret); err != nil {
		return "", fmt.Errorf("failed to get secret %q: %v", namespacedName.String(), err)
	}

	if _, ok := secret.Data[resources.GCPServiceAccount]; !ok {
		return "", fmt.Errorf("secret %q has no key %q", namespacedName.String(), resources.GCPServiceAccount)
	}
	return string(secret.Data[resources.GCPServiceAccount]), nil
}

func (r *Reconciler) getEKSCredentials(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (string, string, error) {
	cloud := cluster.Spec.CloudSpec
	if cloud == nil {
		return "", "", fmt.Errorf("cloud struct is empty")
	}
	if cloud.EKS == nil {
		return "", "", fmt.Errorf("eks cloud struct is empty")
	}
	if cloud.EKS.CredentialsReference == nil {
		return "", "", fmt.Errorf("no credentials provided")
	}
	secret := &corev1.Secret{}
	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.GetCredentialsSecretName()}
	if err := r.Get(ctx, namespacedName, secret); err != nil {
		return "", "", fmt.Errorf("failed to get secret %q: %v", namespacedName.String(), err)
	}
	accessKeyID, ok := secret.Data[resources.AWSAccessKeyID]
	if !ok {
		return "", "", fmt.Errorf("secret %q has no AWS access key ID %q", namespacedName.String(), resources.AWSAccessKeyID)
	}
	secretAccessKey, ok := secret.Data[resources.AWSSecretAccessKey]
	if !ok {
		return "", "", fmt.Errorf("secret %q has no AWS secret access key %q", namespacedName.String(), resources.AWSSecretAccessKey)
	}
	return string(accessKeyID), string(secretAccessKey), nil
}

func createKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, name, projectID string, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
			Labels:    map[string]string{kubermaticv1.ProjectIDLabelKey: projectID},
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	}
	if err := client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig secret: %v", err)
	}
	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
		},
	}, nil
}

func (r *Reconciler) createOrUpdateGKEKubeconfig(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	cloud := cluster.Spec.CloudSpec
	sa, err := r.getGKECredentials(ctx, cluster)
	if err != nil {
		return err
	}
	config, err := gcp.GetGKECLusterConfig(ctx, sa, cloud.GKE.Name, cloud.GKE.Zone)
	if err != nil {
		return err
	}
	err = r.updateKubeconfigSecret(ctx, config, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) createOrUpdateEKSKubeconfig(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	cloud := cluster.Spec.CloudSpec
	accessKeyID, secretAccessKey, err := r.getEKSCredentials(ctx, cluster)
	if err != nil {
		return err
	}
	config, err := aws.GetEKSCLusterConfig(ctx, accessKeyID, secretAccessKey, cloud.EKS.Name, cloud.EKS.Region)
	if err != nil {
		return err
	}
	err = r.updateKubeconfigSecret(ctx, config, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reconciler) updateKubeconfigSecret(ctx context.Context, config *api.Config, cluster *kubermaticv1.ExternalCluster) error {

	kubeconfigSecretName := cluster.GetKubeconfigSecretName()
	kubeconfig, err := clientcmd.Write(*config)
	if err != nil {
		return err
	}

	projectID := ""
	if cluster.Labels != nil {
		projectID = cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	}

	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: kubeconfigSecretName}

	existingSecret := &corev1.Secret{}
	if err := r.Get(ctx, namespacedName, existingSecret); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to probe for secret %v: %v", namespacedName, err)
	}

	// update if already exists
	if existingSecret.Name != "" {
		existingSecret.Data = map[string][]byte{
			resources.ExternalClusterKubeconfig: []byte(base64.StdEncoding.EncodeToString(kubeconfig)),
		}
		r.log.Debugf("update kubeconfig for cluster %s", cluster.Name)

		return r.Update(ctx, existingSecret)
	}

	keyRef, err := createKubeconfigSecret(ctx, r.Client, kubeconfigSecretName, projectID, map[string][]byte{
		resources.ExternalClusterKubeconfig: []byte(base64.StdEncoding.EncodeToString(kubeconfig)),
	})
	if err != nil {
		return err
	}
	cluster.Spec.KubeconfigReference = keyRef
	kuberneteshelper.AddFinalizer(cluster, kubermaticapiv1.ExternalClusterKubeconfigCleanupFinalizer)

	return r.Update(ctx, cluster)
}
