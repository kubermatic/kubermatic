package openshiftseedsyncer

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	userclusteropenshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/resources/resources/openshift"
	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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
)

func Add(
	log *zap.SugaredLogger,
	mgr manager.Manager,
	seedMgr manager.Manager,
	externalClusterAddress string,
	clusterNamespace string,
) error {
	r := &reconciler{
		ctx:                    context.Background(),
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
	seedSecretSource.InjectCache(seedMgr.GetCache())
	if err := c.Watch(seedSecretSource, controllerutil.EnqueueConst("")); err != nil {
		return fmt.Errorf("failed to watch secrets in seed: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, controllerutil.EnqueueConst("")); err != nil {
		return fmt.Errorf("failed to watch secrets in usercluster: %v", err)
	}

	return nil
}

type reconciler struct {
	ctx                    context.Context
	log                    *zap.SugaredLogger
	userClusterClient      ctrlruntimeclient.Client
	seedClient             ctrlruntimeclient.Client
	externalClusterAddress string
	clusterName            string
	clusterNamespace       string
}

func (r *reconciler) getCACert() (*x509.Certificate, error) {
	pair, err := resources.GetClusterRootCA(r.ctx, r.clusterNamespace, r.seedClient)
	if err != nil {
		return nil, err
	}
	return pair.Cert, nil
}

func (r *reconciler) Reconcile(_ reconcile.Request) (reconcile.Result, error) {
	result, err := r.reconcile()
	if err != nil {
		r.log.Errorw("Reconciliation failed", zap.Error(err))
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *reconciler) reconcile() (*reconcile.Result, error) {
	serviceAccount := &corev1.ServiceAccount{}
	if err := r.userClusterClient.Get(r.ctx, types.NamespacedName{Namespace: "kube-system", Name: userclusteropenshiftresources.TokenOwnerServiceAccountName}, serviceAccount); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("error trying to get serviceAccount: %v", err)
		}
		// The ServiceAccount is created by another controller and may not exist yet
		r.log.Debug("Gota NotFound when trying to get ServiceAccount, retrying later")
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if len(serviceAccount.Secrets) < 1 {
		r.log.Debug("ServiceAccount has no Token associated, retrying later")
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	tokenSecret := &corev1.Secret{}
	tokenSecretName := types.NamespacedName{
		Namespace: "kube-system",
		Name:      serviceAccount.Secrets[0].Name,
	}
	if err := r.userClusterClient.Get(r.ctx, tokenSecretName, tokenSecret); err != nil {
		return nil, fmt.Errorf("failed to get token secret from user cluster: %v", err)
	}
	if len(tokenSecret.Data["token"]) == 0 {
		r.log.Debug("tokenSecret is empty, retrying later")
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	token := string(tokenSecret.Data["token"])

	caCert, err := r.getCACert()
	if err != nil {
		return nil, fmt.Errorf("failed to get caCert: %v", err)
	}

	secretCreators := []reconciling.NamedSecretCreatorGetter{
		seedAdminKubeconfigSecretCreatorGetter(caCert, r.externalClusterAddress, r.clusterName, token),
	}
	if err := reconciling.ReconcileSecrets(
		r.ctx,
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
			}
			return s, nil
		}
	}
}
