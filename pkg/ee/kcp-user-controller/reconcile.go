//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kcpusercontroller

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	"github.com/kcp-dev/logicalcluster/v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	masterClient ctrlruntimeclient.Client
	scheme       *runtime.Scheme
	configGetter provider.KubermaticConfigurationGetter
}

// Reconcile reconciles a User.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("user", request.Name)
	log.Debug("Reconciling")

	config, err := r.configGetter(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get current KubermaticConfiguration: %w", err)
	}

	// feature is not enabled
	if !config.Spec.FeatureGates[features.KCPUserManagement] {
		return reconcile.Result{}, nil
	}

	ca, err := r.getCA(ctx, config)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get kcp client-auth CA: %w", err)
	}

	user := &kubermaticv1.User{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, user); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get User: %w", err)
	}

	err = r.reconcile(ctx, ca, config, user)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(user, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) getCA(ctx context.Context, config *kubermaticv1.KubermaticConfiguration) (*triple.KeyPair, error) {
	secretName := config.Spec.KCP.ClientCASecretName
	if secretName == "" {
		return nil, errors.New("no clientCASecretName configured")
	}

	key := types.NamespacedName{Namespace: config.Namespace, Name: secretName}

	secret := &corev1.Secret{}
	if err := r.masterClient.Get(ctx, key, secret); err != nil {
		return nil, err
	}

	certPEM, exists := secret.Data[resources.CACertSecretKey]
	if !exists {
		return nil, fmt.Errorf("Secret does not contain %q key", resources.CACertSecretKey)
	}

	keyPEM, exists := secret.Data[resources.CAKeySecretKey]
	if !exists {
		return nil, fmt.Errorf("Secret does not contain %q key", resources.CAKeySecretKey)
	}

	// validate that the key matches the cert
	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return nil, fmt.Errorf("CA or private key are invalid: %w", err)
	}

	return triple.ParseRSAKeyPair(certPEM, keyPEM)
}

func (r *reconciler) reconcile(ctx context.Context, ca *triple.KeyPair, config *kubermaticv1.KubermaticConfiguration, user *kubermaticv1.User) error {
	if user.DeletionTimestamp != nil {
		return nil // the owner reference takes care of removing the Secret
	}

	secretName := fmt.Sprintf("user-%s-kcp", user.Name)

	// Ensure a Secret with a copy of the CA (for convenience), a client cert
	// and a preconfigured kubeconfig
	secretCreatorGetters := []reconciling.NamedSecretCreatorGetter{
		r.userSecretCreatorGetter(ctx, secretName, ca, config, user),
	}

	err := reconciling.ReconcileSecrets(ctx, secretCreatorGetters, config.Namespace, r.masterClient)
	if err != nil {
		return fmt.Errorf("failed to ensure Secret: %w", err)
	}

	return nil
}

func (r *reconciler) userSecretCreatorGetter(ctx context.Context, name string, ca *triple.KeyPair, config *kubermaticv1.KubermaticConfiguration, user *kubermaticv1.User) reconciling.NamedSecretCreatorGetter {
	caGetter := func() (*triple.KeyPair, error) {
		return ca, nil
	}

	// use the shared helper to maintain the client-cert and then enrich the Secret
	// later with the kubeconfig
	clientCertCreatorGetter := certificates.GetClientCertificateCreator(name, user.Name, nil, ClientCertSecretKey, ClientCertKeySecretKey, caGetter)
	_, clientCertCreator := clientCertCreatorGetter()

	return func() (string, reconciling.SecretCreator) {
		return name, func(secret *corev1.Secret) (*corev1.Secret, error) {
			secret, err := clientCertCreator(secret)
			if err != nil {
				return nil, err
			}

			kubeconfig := createKubeconfig(
				secret.Data[resources.CACertSecretKey],
				secret.Data[ClientCertSecretKey],
				secret.Data[ClientCertKeySecretKey],
				r.homeWorkspaceURL(config, user),
				user.Name,
			)

			kubeconfigEncoded, err := clientcmd.Write(*kubeconfig)
			if err != nil {
				return nil, err
			}

			secret.Data[KubeconfigSecretKey] = kubeconfigEncoded

			// try to set an owner reference
			err = controllerutil.SetControllerReference(user, secret, r.scheme)
			if err != nil {
				var cerr *controllerutil.AlreadyOwnedError // do not use errors.Is() on this error type
				if !errors.As(err, &cerr) {
					return nil, fmt.Errorf("failed to set owner reference: %w", err)
				}
			}

			return secret, nil
		}
	}
}

func (r *reconciler) homeWorkspaceURL(config *kubermaticv1.KubermaticConfiguration, user *kubermaticv1.User) string {
	homePrefix := logicalcluster.New(config.Spec.KCP.HomeRootPrefix)
	workspaceName := getHomeLogicalClusterName(homePrefix, user.Name)

	return config.Spec.KCP.PublicURL + workspaceName.Path()
}

func createKubeconfig(ca, clientCert, clientKey []byte, server, userName string) *clientcmdapi.Config {
	config := clientcmdapi.NewConfig()
	config.Clusters["home"] = &clientcmdapi.Cluster{
		Server:                   server,
		CertificateAuthorityData: ca,
	}
	config.CurrentContext = "default-context"

	authInfoName := fmt.Sprintf("as-%s", userName)
	config.Contexts["default-context"] = &clientcmdapi.Context{
		Cluster:   "home",
		Namespace: "default",
		AuthInfo:  authInfoName,
	}
	config.AuthInfos[authInfoName] = &clientcmdapi.AuthInfo{
		ClientCertificateData: clientCert,
		ClientKeyData:         clientKey,
	}

	return config
}
