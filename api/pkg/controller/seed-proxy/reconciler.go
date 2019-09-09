package seedproxy

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for proxying requests to
// seed clusters. It also takes care of creating a nice ConfigMap for
// Grafana's provisioning mechanism.
type Reconciler struct {
	ctrlruntimeclient.Client

	ctx context.Context
	log *zap.SugaredLogger

	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter
	seedClientGetter     provider.SeedClientGetter

	recorder record.EventRecorder
}

// Reconcile acts upon requests and will restore the state of resources
// for the given seed cluster context (the request's name). Will return
// an error if any API operation failed, otherwise will return an empty
// dummy Result struct.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("seed", request.NamespacedName.String())
	log.Debugw("reconciling seed")

	seeds, err := r.seedsGetter()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get seeds: %v", err)
	}

	log.Debug("garbage-collecting orphaned resources...")
	if err := r.garbageCollect(seeds, log); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to garbage collect: %v", err)
	}

	seed, found := seeds[request.Name]
	if !found {
		return reconcile.Result{}, fmt.Errorf("didn't find seed %q", request.Name)
	}

	// both of these namespaces must exist or else we cannot establish
	// the service account token sharing
	if !r.namespaceExists(r, MasterTargetNamespace, log) {
		log.Debug("skipping because target master namespace does not exist", "namespace", MasterTargetNamespace)
		return reconcile.Result{}, nil
	}

	client, err := r.seedClientGetter(seed)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get seed client: %v", err)
	}

	if !r.namespaceExists(client, SeedServiceAccountNamespace, log) {
		log.Debug("skipping because seed namespace does not exist", "namespace", SeedServiceAccountNamespace)
		return reconcile.Result{}, nil
	}

	log.Debug("reconciling seed cluster...")
	if err := r.reconcileSeedProxy(seed, client, log); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile: %v", err)
	}

	if r.namespaceExists(r, MasterGrafanaNamespace, log) {
		log.Debug("reconciling Grafana provisioning...")
		if err := r.ensureMasterGrafanaProvisioning(seeds); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to reconcile Grafana: %v", err)
		}
	} else {
		log.Debugw("skipping Grafana setup because namespace does not exist in master", "namespace", MasterGrafanaNamespace)
	}

	log.Debug("successfully reconciled seed")
	return reconcile.Result{}, nil
}

// garbageCollect finds secrets referencing non-existing seeds and deletes
// those. It relies on the owner references on all other master-cluster
// resources to let the apiserver remove them automatically.
func (r *Reconciler) garbageCollect(seeds map[string]*kubermaticv1.Seed, log *zap.SugaredLogger) error {
	list := &corev1.SecretList{}
	options := &ctrlruntimeclient.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			ManagedByLabel: ControllerName,
		}),
	}

	if err := r.List(r.ctx, options, list); err != nil {
		return fmt.Errorf("failed to list Secrets: %v", err)
	}

	for _, item := range list.Items {
		seed := item.Labels[InstanceLabel]

		if _, exists := seeds[seed]; !exists {
			log.Debugw("deleting orphaned Secret referencing non-existing seed", "secret", item, "seed", seed)
			if err := r.Delete(r.ctx, &item); err != nil {
				return fmt.Errorf("failed to delete Secret: %v", err)
			}
		}
	}

	return nil
}

func (r *Reconciler) reconcileSeedProxy(seed *kubermaticv1.Seed, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	cfg, err := r.seedKubeconfigGetter(seed)
	if err != nil {
		return err
	}

	log.Debug("reconciling ServiceAccounts...")
	if err := r.ensureSeedServiceAccounts(client, log); err != nil {
		return fmt.Errorf("failed to ensure ServiceAccount: %v", err)
	}

	if r.namespaceExists(client, SeedPrometheusNamespace, log) {
		log.Debug("reconciling roles...")
		if err := r.ensureSeedRoles(client, log); err != nil {
			return fmt.Errorf("failed to ensure Role: %v", err)
		}

		log.Debug("reconciling role bindings...")
		if err := r.ensureSeedRoleBindings(client, log); err != nil {
			return fmt.Errorf("failed to ensure RoleBinding: %v", err)
		}
	} else {
		log.Debugw("skipping RBAC setup because Prometheus namespace does not exist in master", "namespace", SeedPrometheusNamespace)
	}

	log.Debug("fetching ServiceAccount details from seed cluster...")
	serviceAccountSecret, err := r.fetchServiceAccountSecret(client, log)
	if err != nil {
		return fmt.Errorf("failed to fetch ServiceAccount: %v", err)
	}

	if err := r.reconcileMaster(seed.Name, cfg, serviceAccountSecret, log); err != nil {
		return fmt.Errorf("failed to reconcile master: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureSeedServiceAccounts(client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	creators := []reconciling.NamedServiceAccountCreatorGetter{
		seedServiceAccountCreator(),
	}

	if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, SeedServiceAccountNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", SeedServiceAccountNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureSeedRoles(client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	creators := []reconciling.NamedRoleCreatorGetter{
		seedMonitoringRoleCreator(),
	}

	if err := reconciling.ReconcileRoles(r.ctx, creators, SeedMonitoringNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", SeedMonitoringNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureSeedRoleBindings(client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	creators := []reconciling.NamedRoleBindingCreatorGetter{
		seedMonitoringRoleBindingCreator(),
	}

	if err := reconciling.ReconcileRoleBindings(r.ctx, creators, SeedMonitoringNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in the namespace %s: %v", SeedMonitoringNamespace, err)
	}

	return nil
}

func (r *Reconciler) fetchServiceAccountSecret(client ctrlruntimeclient.Client, log *zap.SugaredLogger) (*corev1.Secret, error) {
	sa := &corev1.ServiceAccount{}
	name := types.NamespacedName{
		Namespace: SeedServiceAccountNamespace,
		Name:      SeedServiceAccountName,
	}

	if err := client.Get(r.ctx, name, sa); err != nil {
		return nil, fmt.Errorf("could not find ServiceAccount '%s'", name)
	}

	if len(sa.Secrets) == 0 {
		return nil, fmt.Errorf("no Secret associated with ServiceAccount '%s'", name)
	}

	secret := &corev1.Secret{}
	name = types.NamespacedName{
		Namespace: SeedServiceAccountNamespace,
		Name:      sa.Secrets[0].Name,
	}

	if err := r.Get(r.ctx, name, secret); err != nil {
		return nil, fmt.Errorf("could not find Secret '%s'", name)
	}

	return secret, nil
}

func (r *Reconciler) reconcileMaster(seedName string, kubeconfig *rest.Config, credentials *corev1.Secret, log *zap.SugaredLogger) error {
	log.Debug("reconciling Secrets...")
	secret, err := r.ensureMasterSecrets(seedName, kubeconfig, credentials)
	if err != nil {
		return fmt.Errorf("failed to ensure Secrets: %v", err)
	}

	log.Debug("reconciling Deployments...")
	if err := r.ensureMasterDeployments(seedName, secret); err != nil {
		return fmt.Errorf("failed to ensure Deployments: %v", err)
	}

	log.Debug("reconciling Services...")
	if err := r.ensureMasterServices(seedName, secret); err != nil {
		return fmt.Errorf("failed to ensure Services: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureMasterSecrets(seedName string, kubeconfig *rest.Config, credentials *corev1.Secret) (*corev1.Secret, error) {
	creators := []reconciling.NamedSecretCreatorGetter{
		masterSecretCreator(seedName, kubeconfig, credentials),
	}

	if err := reconciling.ReconcileSecrets(r.ctx, creators, MasterTargetNamespace, r.Client); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets in the namespace %s: %v", MasterTargetNamespace, err)
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{
		Namespace: MasterTargetNamespace,
		Name:      secretName(seedName),
	}

	if err := r.Get(r.ctx, name, secret); err != nil {
		return nil, fmt.Errorf("could not find Secret '%s'", name)
	}

	return secret, nil
}

func (r *Reconciler) ensureMasterDeployments(seedName string, secret *corev1.Secret) error {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		masterDeploymentCreator(seedName, secret),
	}

	if err := reconciling.ReconcileDeployments(r.ctx, creators, MasterTargetNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in the namespace %s: %v", MasterTargetNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureMasterServices(seedName string, secret *corev1.Secret) error {
	creators := []reconciling.NamedServiceCreatorGetter{
		masterServiceCreator(seedName, secret),
	}

	if err := reconciling.ReconcileServices(r.ctx, creators, MasterTargetNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services in the namespace %s: %v", MasterTargetNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureMasterGrafanaProvisioning(seeds map[string]*kubermaticv1.Seed) error {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		r.masterGrafanaConfigmapCreator(seeds),
	}

	if err := reconciling.ReconcileConfigMaps(r.ctx, creators, MasterGrafanaNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in the namespace %s: %v", MasterGrafanaNamespace, err)
	}

	return nil
}

func (r *Reconciler) namespaceExists(client ctrlruntimeclient.Client, namespace string, log *zap.SugaredLogger) bool {
	ns := corev1.Namespace{}
	if err := client.Get(r.ctx, types.NamespacedName{Name: namespace}, &ns); err != nil {
		if !kerrors.IsNotFound(err) {
			log.Errorw("failed to check for namespace", "namespace", namespace, zap.Error(err))
		}

		return false
	}

	return true
}
