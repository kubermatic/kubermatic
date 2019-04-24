package seedproxy

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
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

	kubeconfig  *clientcmdapi.Config
	datacenters map[string]provider.DatacenterMeta

	recorder record.EventRecorder
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	glog.V(4).Info("Garbage-collecting orphaned resources...")
	if err := r.garbageCollect(ctx); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to garbage collect: %v", err)
	}

	glog.V(4).Infof("Reconciling seed cluster %s...", request.Name)
	if err := r.reconcileContext(ctx, request.Name); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile: %v", err)
	}

	glog.V(4).Info("Reconciling Grafana provisioning...")
	if err := r.reconcileGrafana(ctx); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile Grafana: %v", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) garbageCollect(ctx context.Context) error {
	options := &ctrlruntimeclient.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			ManagedByLabel: ControllerName,
		}),
	}

	if err := r.garbageCollectServices(ctx, options); err != nil {
		return fmt.Errorf("failed to garbage collect Services: %v", err)
	}

	if err := r.garbageCollectDeployments(ctx, options); err != nil {
		return fmt.Errorf("failed to garbage collect Deployments: %v", err)
	}

	if err := r.garbageCollectConfigMaps(ctx, options); err != nil {
		return fmt.Errorf("failed to garbage collect ConfigMaps: %v", err)
	}

	if err := r.garbageCollectSecrets(ctx, options); err != nil {
		return fmt.Errorf("failed to garbage collect Secrets: %v", err)
	}

	return nil
}

func (r *Reconciler) garbageCollectServices(ctx context.Context, opt *ctrlruntimeclient.ListOptions) error {
	list := &corev1.ServiceList{}

	if err := r.List(ctx, opt, list); err != nil {
		return fmt.Errorf("failed to list Services: %v", err)
	}

	for _, item := range list.Items {
		meta := item.GetObjectMeta()
		seed := meta.GetLabels()[InstanceLabel]

		if r.unknownSeed(seed) {
			glog.V(4).Infof("Deleting orphaned Service %s/%s referencing non-existing seed '%s'...", item.Namespace, item.Name, seed)
			if err := r.Delete(ctx, &item); err != nil {
				return fmt.Errorf("failed to delete Service: %v", err)
			}
		}
	}

	return nil
}

func (r *Reconciler) garbageCollectSecrets(ctx context.Context, opt *ctrlruntimeclient.ListOptions) error {
	list := &corev1.SecretList{}

	if err := r.List(ctx, opt, list); err != nil {
		return fmt.Errorf("failed to list Secrets: %v", err)
	}

	for _, item := range list.Items {
		meta := item.GetObjectMeta()
		seed := meta.GetLabels()[InstanceLabel]

		if r.unknownSeed(seed) {
			glog.V(4).Infof("Deleting orphaned Secret %s/%s referencing non-existing seed '%s'...", item.Namespace, item.Name, seed)
			if err := r.Delete(ctx, &item); err != nil {
				return fmt.Errorf("failed to delete Secret: %v", err)
			}
		}
	}

	return nil
}

func (r *Reconciler) garbageCollectConfigMaps(ctx context.Context, opt *ctrlruntimeclient.ListOptions) error {
	// We have one ConfigMap for all seeds, so if there is at least one seed left,
	// rely on the reconciling to trim unused elements from the ConfigMap.
	if len(r.kubeconfig.Contexts) > 0 {
		return nil
	}

	list := &corev1.ConfigMapList{}

	if err := r.List(ctx, opt, list); err != nil {
		return fmt.Errorf("failed to list ConfigMaps: %v", err)
	}

	for _, item := range list.Items {
		glog.V(4).Infof("Deleting orphaned ConfigMap %s/%s...", item.Namespace, item.Name)
		if err := r.Delete(ctx, &item); err != nil {
			return fmt.Errorf("failed to delete ConfigMap: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) garbageCollectDeployments(ctx context.Context, opt *ctrlruntimeclient.ListOptions) error {
	list := &appsv1.DeploymentList{}

	if err := r.List(ctx, opt, list); err != nil {
		return fmt.Errorf("failed to list Deployments: %v", err)
	}

	for _, item := range list.Items {
		meta := item.GetObjectMeta()
		seed := meta.GetLabels()[InstanceLabel]

		if r.unknownSeed(seed) {
			glog.V(4).Infof("Deleting orphaned Deployment %s/%s referencing non-existing seed '%s'...", item.Namespace, item.Name, seed)
			if err := r.Delete(ctx, &item); err != nil {
				return fmt.Errorf("failed to delete Deployment: %v", err)
			}
		}
	}

	return nil
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

	glog.V(4).Info("Reconciling seed cluster...")
	if err := r.reconcileSeed(ctx, client); err != nil {
		return fmt.Errorf("failed to reconcile seed: %v", err)
	}

	glog.V(4).Info("Fetching service account details from seed cluster...")
	serviceAccountSecret, err := r.fetchServiceAccountSecret(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to fetch service account: %v", err)
	}

	if err := r.reconcileMaster(ctx, contextName, serviceAccountSecret); err != nil {
		return fmt.Errorf("failed to reconcile master: %v", err)
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

func (r *Reconciler) reconcileSeed(ctx context.Context, client ctrlruntimeclient.Client) error {
	glog.V(4).Info("Reconciling service accounts...")
	if err := r.ensureSeedServiceAccounts(ctx, client); err != nil {
		return fmt.Errorf("failed to ensure service account: %v", err)
	}

	glog.V(4).Info("Reconciling roles...")
	if err := r.ensureSeedRoles(ctx, client); err != nil {
		return fmt.Errorf("failed to ensure role: %v", err)
	}

	glog.V(4).Info("Reconciling role bindings...")
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

func (r *Reconciler) reconcileMaster(ctx context.Context, contextName string, credentials *corev1.Secret) error {
	glog.V(4).Info("Reconciling secrets...")
	if err := r.ensureMasterSecrets(ctx, contextName, credentials); err != nil {
		return fmt.Errorf("failed to ensure secrets: %v", err)
	}

	glog.V(4).Info("Reconciling deployments...")
	if err := r.ensureMasterDeployments(ctx, contextName); err != nil {
		return fmt.Errorf("failed to ensure deployments: %v", err)
	}

	glog.V(4).Info("Reconciling services...")
	if err := r.ensureMasterServices(ctx, contextName); err != nil {
		return fmt.Errorf("failed to ensure services: %v", err)
	}

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

func (r *Reconciler) reconcileGrafana(ctx context.Context) error {
	if err := r.ensureMasterGrafanaProvisioning(ctx); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) ensureMasterGrafanaProvisioning(ctx context.Context) error {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		masterGrafanaConfigmapCreator(r.datacenters, r.kubeconfig),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, MasterGrafanaNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in the namespace %s: %v", MasterGrafanaNamespace, err)
	}

	return nil
}

func (r *Reconciler) unknownSeed(seed string) bool {
	_, ok := r.kubeconfig.Contexts[seed]

	return !ok
}
