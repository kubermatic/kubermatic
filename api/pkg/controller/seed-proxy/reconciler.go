package seedproxy

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
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
	ctx context.Context
	ctrlruntimeclient.Client

	seedsGetter          provider.SeedsGetter
	seedKubeconfigGetter provider.SeedKubeconfigGetter

	recorder record.EventRecorder
}

// Reconcile acts upon requests and will restore the state of resources
// for the given seed cluster context (the request's name). Will return
// an error if any API operation failed, otherwise will return an empty
// dummy Result struct.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	glog.V(4).Infof("Reconciling seed %q", request.NamespacedName.String())
	seeds, err := r.seedsGetter()
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get seeds: %v", err)
	}

	glog.V(4).Info("Garbage-collecting orphaned resources...")
	if err := r.garbageCollect(seeds); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to garbage collect: %v", err)
	}

	glog.V(4).Infof("Reconciling seed cluster %s...", request.Name)
	seed, found := seeds[request.Name]
	if !found {
		return reconcile.Result{}, fmt.Errorf("didn't find seed %q", request.Name)
	}
	if err := r.reconcileSeed(seed); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile: %v", err)
	}

	glog.V(4).Info("Reconciling Grafana provisioning...")
	if err := r.ensureMasterGrafanaProvisioning(seeds); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile Grafana: %v", err)
	}

	glog.V(4).Infof("Successfully reconciled seed %q", request.Name)
	return reconcile.Result{}, nil
}

// garbageCollect finds secrets referencing non-existing seeds and deletes
// those. It relies on the owner references on all other master-cluster
// resources to let the apiserver remove them automatically.
func (r *Reconciler) garbageCollect(seeds map[string]*kubermaticv1.Seed) error {
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
			glog.V(4).Infof("Deleting orphaned Secret %s/%s referencing non-existing seed '%s'...", item.Namespace, item.Name, seed)
			if err := r.Delete(r.ctx, &item); err != nil {
				return fmt.Errorf("failed to delete Secret: %v", err)
			}
		}
	}

	return nil
}

func (r *Reconciler) reconcileSeed(seed *kubermaticv1.Seed) error {
	cfg, err := r.seedKubeconfigGetter(seed)
	if err != nil {
		return err
	}
	name := seed.Name

	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
	if err != nil {
		return fmt.Errorf("failed to create client for seed: %v", err)
	}

	glog.V(4).Infof("Reconciling seed cluster %q", name)

	glog.V(4).Info("Reconciling service accounts...")
	if err := r.ensureSeedServiceAccounts(client); err != nil {
		return fmt.Errorf("failed to ensure service account: %v", err)
	}

	glog.V(4).Info("Reconciling roles...")
	if err := r.ensureSeedRoles(client); err != nil {
		return fmt.Errorf("failed to ensure role: %v", err)
	}

	glog.V(4).Info("Reconciling role bindings...")
	if err := r.ensureSeedRoleBindings(client); err != nil {
		return fmt.Errorf("failed to ensure role binding: %v", err)
	}

	glog.V(4).Info("Fetching service account details from seed cluster...")
	serviceAccountSecret, err := r.fetchServiceAccountSecret(client)
	if err != nil {
		return fmt.Errorf("failed to fetch service account: %v", err)
	}

	if err := r.reconcileMaster(name, cfg, serviceAccountSecret); err != nil {
		return fmt.Errorf("failed to reconcile master: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureSeedServiceAccounts(client ctrlruntimeclient.Client) error {
	creators := []reconciling.NamedServiceAccountCreatorGetter{
		seedServiceAccountCreator(),
	}

	if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, SeedServiceAccountNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts in the namespace %s: %v", SeedServiceAccountNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureSeedRoles(client ctrlruntimeclient.Client) error {
	creators := []reconciling.NamedRoleCreatorGetter{
		seedMonitoringRoleCreator(),
	}

	if err := reconciling.ReconcileRoles(r.ctx, creators, SeedMonitoringNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile Roles in the namespace %s: %v", SeedMonitoringNamespace, err)
	}

	return nil
}

func (r *Reconciler) ensureSeedRoleBindings(client ctrlruntimeclient.Client) error {
	creators := []reconciling.NamedRoleBindingCreatorGetter{
		seedMonitoringRoleBindingCreator(),
	}

	if err := reconciling.ReconcileRoleBindings(r.ctx, creators, SeedMonitoringNamespace, client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings in the namespace %s: %v", SeedMonitoringNamespace, err)
	}

	return nil
}

func (r *Reconciler) fetchServiceAccountSecret(client ctrlruntimeclient.Client) (*corev1.Secret, error) {
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

func (r *Reconciler) reconcileMaster(seedName string, kubeconfig *rest.Config, credentials *corev1.Secret) error {
	glog.V(4).Info("Reconciling secrets...")
	secret, err := r.ensureMasterSecrets(seedName, kubeconfig, credentials)
	if err != nil {
		return fmt.Errorf("failed to ensure secrets: %v", err)
	}

	glog.V(4).Info("Reconciling deployments...")
	if err := r.ensureMasterDeployments(seedName, secret); err != nil {
		return fmt.Errorf("failed to ensure deployments: %v", err)
	}

	glog.V(4).Info("Reconciling services...")
	if err := r.ensureMasterServices(seedName, secret); err != nil {
		return fmt.Errorf("failed to ensure services: %v", err)
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
