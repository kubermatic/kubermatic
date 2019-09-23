package usersshkeys

import (
	"context"
	"fmt"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	corev1 "k8s.io/api/core/v1"

	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_usersshkeys_controller"
)

// Reconciler is a controller which is responsible for managing clusters
type Reconciler struct {
	ctrlruntimeclient.Client
	sshKeyProvdier *kubernetesprovider.SSHKeyProvider
	log            *zap.SugaredLogger
	workerName     string
	recorder       record.EventRecorder
}

func Add(
	mgr manager.Manager,
	sshProvdier *kubernetesprovider.SSHKeyProvider,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {

	reconciler := &Reconciler{
		sshKeyProvdier: sshProvdier,
		log:            log,
		workerName:     workerName,
		Client:         mgr.GetClient(),
		recorder:       mgr.GetRecorder(ControllerName),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.UserSSHKey{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	userSSHKey := &kubermaticv1.UserSSHKey{}
	if err := r.Get(ctx, request.NamespacedName, userSSHKey); err != nil {
		if kubeapierrors.IsNotFound(err) {
			log.Debug("Could not find user ssh key")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err := r.reconcileUserSSHKeys(ctx, userSSHKey)
	if err != nil {
		log.Errorw("Failed to reconcile user ssh keys", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcileUserSSHKeys(ctx context.Context, userSSHKey *kubermaticv1.UserSSHKey) error {
	clusterList := &kubermaticv1.ClusterList{}
	if err := r.List(ctx, &ctrlruntimeclient.ListOptions{}, clusterList); err != nil {
		return err
	}

	clustersNames := buildClustersMapFromSecret(userSSHKey.Spec.Clusters)
	for _, cluster := range clusterList.Items {
		namespace := fmt.Sprintf("cluster-%v", cluster.Name)
		if _, found := clustersNames[cluster.Name]; !found {
			return r.deleteUserSSHKey(ctx, cluster, namespace, userSSHKey.Name)
		}

		return r.updateUserSSHKeysSecrets(ctx, userSSHKey, cluster.Name, namespace)
	}

	return nil
}

func (r *Reconciler) updateUserSSHKeysSecrets(ctx context.Context, userSSHKey *kubermaticv1.UserSSHKey, clusterName, namespace string) error {
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      fmt.Sprintf("%v-%v", resources.UserSSHKeys, clusterName),
	}

	secret := &corev1.Secret{}
	if err := r.Get(ctx, key, secret); err != nil {
		if kubeapierrors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					userSSHKey.Name: []byte(userSSHKey.Spec.PublicKey),
				},
			}
			return r.Create(ctx, secret)
		}
		return err
	}

	if _, found := secret.Data[userSSHKey.Name]; !found {
		secret.Data[userSSHKey.Name] = []byte(userSSHKey.Spec.PublicKey)
		return r.Update(ctx, secret)
	}

	return nil
}

func (r *Reconciler) deleteUserSSHKey(ctx context.Context, cluster kubermaticv1.Cluster, namespace, keyName string) error {
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      fmt.Sprintf("%v-%v", resources.UserSSHKeys, cluster.Name),
	}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, key, secret); err != nil {
		if kubeapierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	delete(secret.Data, keyName)
	return r.Update(ctx, secret)
}

func buildClustersMapFromSecret(clusters []string) (mappedClusters map[string]struct{}) {
	for _, cluster := range clusters {
		mappedClusters[cluster] = struct{}{}
	}

	return mappedClusters
}
