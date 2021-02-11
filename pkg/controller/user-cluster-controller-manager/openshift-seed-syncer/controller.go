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

package openshiftseedsyncer

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	openshiftresources "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/openshift/resources"
	userclusteropenshiftresources "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/openshift"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "openshift_seed_syncer"
	// ConsoleAdminPasswordSecretName is the OAuth bootstrap secret which we
	// use to authenticate the console against the oauth service.
	ConsoleAdminPasswordSecretName = "openshift-bootstrap-password"
)

func Add(
	log *zap.SugaredLogger,
	mgr manager.Manager,
	seedMgr manager.Manager,
	externalClusterAddress string,
	clusterNamespace string,
) error {
	r := &reconciler{
		log:                    log.Named(controllerName),
		userClusterClient:      mgr.GetClient(),
		seedClient:             seedMgr.GetClient(),
		externalClusterAddress: externalClusterAddress,
		clusterName:            strings.ReplaceAll(clusterNamespace, "cluster-", ""),
		clusterNamespace:       clusterNamespace,
	}
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	seedSecretSource := &source.Kind{Type: &corev1.Secret{}}
	if err := seedSecretSource.InjectCache(seedMgr.GetCache()); err != nil {
		return fmt.Errorf("failed to inject seed cache into watch: %v", err)
	}
	if err := c.Watch(seedSecretSource, controllerutil.EnqueueConst("")); err != nil {
		return fmt.Errorf("failed to watch secrets in seed: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, controllerutil.EnqueueConst("")); err != nil {
		return fmt.Errorf("failed to watch secrets in usercluster: %v", err)
	}

	oauthClientConfigKind := &unstructured.Unstructured{}
	oauthClientConfigKind.SetAPIVersion("oauth.openshift.io/v1")
	oauthClientConfigKind.SetKind("OAuthClient")
	if err := c.Watch(&source.Kind{Type: oauthClientConfigKind}, controllerutil.EnqueueConst("")); err != nil {
		return fmt.Errorf("failed to watch OauthClients in usercluster: %v", err)
	}

	return nil
}

type reconciler struct {
	log                    *zap.SugaredLogger
	userClusterClient      ctrlruntimeclient.Client
	seedClient             ctrlruntimeclient.Client
	externalClusterAddress string
	clusterName            string
	clusterNamespace       string
}

func (r *reconciler) getCACert(ctx context.Context) (*x509.Certificate, error) {
	pair, err := resources.GetClusterRootCA(ctx, r.clusterNamespace, r.seedClient)
	if err != nil {
		return nil, err
	}
	return pair.Cert, nil
}

func (r *reconciler) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {
	result, err := r.reconcile(ctx)
	if err != nil {
		r.log.Errorw("Reconciliation failed", zap.Error(err))
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *reconciler) reconcile(ctx context.Context) (*reconcile.Result, error) {
	serviceAccount := &corev1.ServiceAccount{}
	if err := r.userClusterClient.Get(ctx, types.NamespacedName{Namespace: "kube-system", Name: userclusteropenshiftresources.TokenOwnerServiceAccountName}, serviceAccount); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("error trying to get ServiceAccount: %v", err)
		}
		// The ServiceAccount is created by another controller and may not exist yet
		r.log.Debug("Got a NotFound error when trying to get ServiceAccount, retrying later")
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if len(serviceAccount.Secrets) < 1 {
		r.log.Debug("ServiceAccount has no token associated, retrying later")
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	tokenSecret := &corev1.Secret{}
	tokenSecretName := types.NamespacedName{
		Namespace: metav1.NamespaceSystem,
		Name:      serviceAccount.Secrets[0].Name,
	}
	if err := r.userClusterClient.Get(ctx, tokenSecretName, tokenSecret); err != nil {
		return nil, fmt.Errorf("failed to get token secret from user cluster: %v", err)
	}
	if len(tokenSecret.Data["token"]) == 0 {
		r.log.Debug("tokenSecret is empty, retrying later")
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	token := string(tokenSecret.Data["token"])

	caCert, err := r.getCACert(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get caCert: %v", err)
	}

	secretCreators := []reconciling.NamedSecretCreatorGetter{
		seedAdminKubeconfigSecretCreatorGetter(caCert, r.externalClusterAddress, r.clusterName, token),
		oauthBootstrapSecretCreatorGetter(r.userClusterClient, r.seedClient, r.clusterNamespace),
		consoleOAuthSecretCreatorGetter(r.userClusterClient),
	}
	if err := reconciling.ReconcileSecrets(
		ctx,
		secretCreators,
		r.clusterNamespace,
		r.seedClient,
	); err != nil {
		return nil, fmt.Errorf("failed to reconcile secrets: %v", err)
	}

	return nil, nil
}

func seedAdminKubeconfigSecretCreatorGetter(
	caCert *x509.Certificate,
	apiServerAddress string,
	clusterName string,
	token string,
) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.AdminKubeconfigSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			config := resources.GetBaseKubeconfig(caCert, apiServerAddress, clusterName)
			config.AuthInfos = map[string]*clientcmdapi.AuthInfo{
				resources.KubeconfigDefaultContextKey: {
					Token: token,
				},
			}
			b, err := clientcmd.Write(*config)
			if err != nil {
				return nil, fmt.Errorf("failed to write kubeconfig: %v", err)
			}

			s.Data = map[string][]byte{
				resources.KubeconfigSecretKey: b,
				"token":                       []byte(token),
			}
			return s, nil
		}
	}
}

func oauthBootstrapSecretCreatorGetter(userClusterClient, seedClient ctrlruntimeclient.Client, seedNamespace string) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		ctx := context.Background()
		name := userclusteropenshiftresources.OAuthBootstrapSecretName
		return ConsoleAdminPasswordSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			userClusterOAuthSecretName := types.NamespacedName{
				Namespace: metav1.NamespaceSystem,
				Name:      name,
			}
			userClusterOAuthSecret := &corev1.Secret{}
			if err := userClusterClient.Get(ctx, userClusterOAuthSecretName, userClusterOAuthSecret); err != nil {
				return nil, fmt.Errorf("failed to get the %s/%s secret from the usercluster: %v", userClusterOAuthSecretName.Namespace, userClusterOAuthSecretName.Name, err)
			}
			encyptedValue, exists := userClusterOAuthSecret.Data[userclusteropenshiftresources.OAuthBootstrapEncryptedkeyName]
			if !exists {
				return nil, fmt.Errorf("usercluster secret has no %s key", userclusteropenshiftresources.OAuthBootstrapEncryptedkeyName)
			}
			secretKey, err := userclusteropenshiftresources.GetOAuthEncryptionKey(ctx, seedClient, seedNamespace)
			if err != nil {
				return nil, fmt.Errorf("failed to get the oauth encryption key: %v", err)
			}
			rawValue, err := userclusteropenshiftresources.AESDecrypt(encyptedValue, secretKey)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt: %v", err)
			}

			if s.Data == nil {
				s.Data = map[string][]byte{}
			}
			s.Data[name] = rawValue

			return s, nil
		}
	}
}

func consoleOAuthSecretCreatorGetter(userClusterClient ctrlruntimeclient.Client) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return openshiftresources.ConsoleOAuthSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			oAuthClient := &unstructured.Unstructured{}
			oAuthClient.SetAPIVersion("oauth.openshift.io/v1")
			oAuthClient.SetKind("OAuthClient")
			oAuthClientName := types.NamespacedName{
				Name: userclusteropenshiftresources.ConsoleOAuthClientName,
			}
			if err := userClusterClient.Get(context.Background(), oAuthClientName, oAuthClient); err != nil {
				return nil, fmt.Errorf("failed to get OAuthClient: %v", err)
			}
			if _, ok := oAuthClient.Object["secret"]; !ok {
				return nil, errors.New("OAuthClient has no `secret key`")
			}
			stringVal, ok := oAuthClient.Object["secret"].(string)
			if !ok {
				return nil, fmt.Errorf("`secret field of OAuthClient was not a string but a %t`", oAuthClient.Object["secret"])
			}

			if s.Data == nil {
				s.Data = map[string][]byte{}
			}
			s.Data["clientSecret"] = []byte(stringVal)

			return s, nil
		}
	}
}
