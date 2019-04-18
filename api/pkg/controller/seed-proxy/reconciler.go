package seedproxy

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler stores all components required for monitoring
type Reconciler struct {
	ctrlruntimeclient.Client

	// userClusterConnProvider userClusterConnectionProvider
	kubeconfig  *clientcmdapi.Config
	datacenters map[string]provider.DatacenterMeta

	recorder record.EventRecorder
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	glog.V(4).Info("Reconciling seed proxies...")

	for name := range r.kubeconfig.Contexts {
		if err := r.reconcileContext(ctx, name); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to reconcile seed %s: %v", name, err)
		}
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileContext(ctx context.Context, contextName string) error {
	var context *clientcmdapi.Context

	for name, c := range r.kubeconfig.Contexts {
		if name == contextName {
			context = c
			break
		}
	}

	if context == nil {
		return errors.New("could not find context in kubeconfig")
	}

	client, err := r.seedClusterClient(contextName)
	if err != nil {
		return fmt.Errorf("failed to create client for seed: %v", err)
	}

	if err := r.ensureSeed(ctx, client); err != nil {
		return fmt.Errorf("failed to ensure seed %s: %v", contextName, err)
	}

	glog.V(6).Info("Fetching service account details from seed cluster...")
	serviceAccountSecret, err := r.fetchServiceAccountSecret(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to ensure seed %s: %v", contextName, err)
	}

	if err := r.ensureMaster(ctx, contextName, serviceAccountSecret); err != nil {
		return fmt.Errorf("failed to ensure master: %v", err)
	}

	return nil
}

func (r *Reconciler) seedClusterClient(contextName string) (ctrlruntimeclient.Client, error) {
	clientConfig := clientcmd.NewNonInteractiveClientConfig(
		*r.kubeconfig,
		contextName,
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
		nil,
	)

	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
}

func (r *Reconciler) ensureSeed(ctx context.Context, client ctrlruntimeclient.Client) error {
	glog.V(6).Info("Reconciling service accounts in seed cluster...")
	if err := r.ensureSeedServiceAccounts(ctx, client); err != nil {
		return fmt.Errorf("failed to ensure service account: %v", err)
	}

	glog.V(6).Info("Reconciling roles in seed cluster...")
	if err := r.ensureSeedRoles(ctx, client); err != nil {
		return fmt.Errorf("failed to ensure role: %v", err)
	}

	glog.V(6).Info("Reconciling role bindings in seed cluster...")
	if err := r.ensureSeedRoleBindings(ctx, client); err != nil {
		return fmt.Errorf("failed to ensure role binding: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureSeedServiceAccounts(ctx context.Context, client ctrlruntimeclient.Client) error {
	creators := []reconciling.NamedServiceAccountCreatorGetter{
		seedServiceAccountCreator(),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, creators, ServiceAccountNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", ServiceAccountNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureSeedRoles(ctx context.Context, client ctrlruntimeclient.Client) error {
	creators := []reconciling.NamedRoleCreatorGetter{
		seedPrometheusRoleCreator(),
	}

	if err := reconciling.ReconcileRoles(ctx, creators, SeedPrometheusNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", SeedPrometheusNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureSeedRoleBindings(ctx context.Context, client ctrlruntimeclient.Client) error {
	creators := []reconciling.NamedRoleBindingCreatorGetter{
		seedPrometheusRoleBindingCreator(),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, creators, SeedPrometheusNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in the namespace %s: %v", SeedPrometheusNamespace, err)
	}

	return nil
}

func (r *Reconciler) fetchServiceAccountSecret(ctx context.Context, client ctrlruntimeclient.Client) (*corev1.Secret, error) {
	sa := &corev1.ServiceAccount{}
	name := types.NamespacedName{
		Namespace: ServiceAccountNamespace,
		Name:      ServiceAccountName,
	}

	if err := r.Get(ctx, name, sa); err != nil {
		return nil, fmt.Errorf("could not find ServiceAccount '%s'", name)
	}

	if len(sa.Secrets) == 0 {
		return nil, fmt.Errorf("no Secret associated with ServiceAccount '%s'", name)
	}

	secret := &corev1.Secret{}
	name = types.NamespacedName{
		Namespace: ServiceAccountNamespace,
		Name:      sa.Secrets[0].Name,
	}

	if err := r.Get(ctx, name, secret); err != nil {
		return nil, fmt.Errorf("could not find Secret '%s'", name)
	}

	return secret, nil
}

func (r *Reconciler) ensureMaster(ctx context.Context, contextName string, credentials *corev1.Secret) error {
	glog.V(6).Info("Reconciling secrets in master cluster...")
	if err := r.ensureMasterSecrets(ctx, contextName, credentials); err != nil {
		return fmt.Errorf("failed to ensure secrets: %v", err)
	}

	glog.V(6).Info("Reconciling deployments in master cluster...")
	if err := r.ensureMasterDeployments(ctx, contextName); err != nil {
		return fmt.Errorf("failed to ensure deployments: %v", err)
	}

	glog.V(6).Info("Reconciling services in master cluster...")
	if err := r.ensureMasterServices(ctx, contextName); err != nil {
		return fmt.Errorf("failed to ensure services: %v", err)
	}

	// glog.V(6).Info("Reconciling roles in seed cluster...")
	// if err := r.ensureSeedRoles(ctx, client); err != nil {
	// 	return fmt.Errorf("failed to ensure role: %v", err)
	// }

	// glog.V(6).Info("Reconciling role bindings in seed cluster...")
	// if err := r.ensureSeedRoleBindings(ctx, client); err != nil {
	// 	return fmt.Errorf("failed to ensure role binding: %v", err)
	// }

	return nil
}

func (r *Reconciler) ensureMasterSecrets(ctx context.Context, contextName string, credentials *corev1.Secret) error {
	creators := []reconciling.NamedSecretCreatorGetter{
		masterSecretCreator(contextName, credentials),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, KubermaticNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets in the namespace %s: %v", KubermaticNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureMasterDeployments(ctx context.Context, contextName string) error {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		masterDeploymentCreator(contextName),
	}

	if err := reconciling.ReconcileDeployments(ctx, creators, KubermaticNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in the namespace %s: %v", KubermaticNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureMasterServices(ctx context.Context, contextName string) error {
	creators := []reconciling.NamedServiceCreatorGetter{
		masterServiceCreator(contextName),
	}

	if err := reconciling.ReconcileServices(ctx, creators, KubermaticNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services in the namespace %s: %v", KubermaticNamespace, err)
	}

	return nil
}

// func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
// 	glog.V(4).Infof("Reconciling cluster %s", cluster.Name)

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	data, err := r.getClusterTemplateData(context.Background(), r.Client, cluster)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// check that all service accounts are created
// 	if err := r.ensureServiceAccounts(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all roles are created
// 	if err := r.ensureRoles(ctx, cluster); err != nil {
// 		return nil, err
// 	}

// 	// check that all role bindings are created
// 	if err := r.ensureRoleBindings(ctx, cluster); err != nil {
// 		return nil, err
// 	}

// 	// check that all secrets are created
// 	if err := r.ensureSecrets(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all ConfigMaps are available
// 	if err := r.ensureConfigMaps(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all Deployments are available
// 	if err := r.ensureDeployments(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all StatefulSets are created
// 	if err := r.ensureStatefulSets(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all VerticalPodAutoscaler's are created
// 	if err := r.ensureVerticalPodAutoscalers(ctx, cluster); err != nil {
// 		return nil, err
// 	}

// 	// check that all Services's are created
// 	if err := r.ensureServices(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	return &reconcile.Result{}, nil
// }
