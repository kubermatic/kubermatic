package operatormaster

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler (re)stores all components required for proxying requests to
// seed clusters. It also takes care of creating a nice ConfigMap for
// Grafana's provisioning mechanism.
type Reconciler struct {
	ctrlruntimeclient.Client

	clientConfig *clientcmdapi.Config
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	workerName   string
}

// Reconcile acts upon requests and will restore the state of resources
// for the given namespace. Will return an error if any API operation
// failed, otherwise will return an empty dummy Result struct.
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// find the requested configuration
	config, err := r.fetchKubermaticConfiguration(ctx, request.NamespacedName)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("invalid reconcile request: %v", err)
	}

	// silently ignore other worker names
	if labels := config.GetLabels(); labels[WorkerNameLabel] != r.workerName {
		return reconcile.Result{}, nil
	}

	identifier := joinNamespaceName(config.GetNamespace(), config.GetName())
	logger := r.log.With("config", identifier)

	return reconcile.Result{}, r.reconcile(ctx, config, logger)
}

func (r *Reconciler) fetchKubermaticConfiguration(ctx context.Context, identifier types.NamespacedName) (*operatorv1alpha1.KubermaticConfiguration, error) {
	cfg := &operatorv1alpha1.KubermaticConfiguration{}

	if err := r.Get(ctx, identifier, cfg); err != nil {
		return nil, fmt.Errorf("could not find KubermaticConfiguration %s", identifier)
	}

	return cfg, nil
}

func (r *Reconciler) reconcile(ctx context.Context, config *operatorv1alpha1.KubermaticConfiguration, logger *zap.SugaredLogger) error {
	// TODO: Implement garbage collection to remove orphaned resources.

	cfgReconciler := configReconciler{
		Reconciler: *r,
		ctx:        ctx,
		config:     config,
		log:        logger,
	}

	return cfgReconciler.Reconcile()
}

type configReconciler struct {
	Reconciler

	ctx    context.Context
	config *operatorv1alpha1.KubermaticConfiguration
	log    *zap.SugaredLogger
	ns     string
}

func (r *configReconciler) Reconcile() error {
	r.applyDefaults()
	r.ns = r.config.Spec.Namespace

	r.log.Debug("Reconciling Kubermatic installation")
	if err := r.reconcileResources(); err != nil {
		return fmt.Errorf("failed to reconcile: %v", err)
	}

	return nil
}

func (r *configReconciler) applyDefaults() {
	if r.config.Spec.Namespace == "" {
		r.config.Spec.Namespace = r.config.Namespace
	}
}

func (r *configReconciler) reconcileResources() error {
	if err := r.reconcileNamespaces(); err != nil {
		return err
	}

	if err := r.reconcileServiceAccounts(); err != nil {
		return err
	}

	if err := r.reconcileClusterRoleBindings(); err != nil {
		return err
	}

	if err := r.reconcileSecrets(); err != nil {
		return err
	}

	if err := r.reconcileConfigMaps(); err != nil {
		return err
	}

	if err := r.reconcileDeployments(); err != nil {
		return err
	}

	return nil
}

func (r *configReconciler) reconcileNamespaces() error {
	r.log.Debug("Reconciling Namespaces")

	creators := []reconciling.NamedNamespaceCreatorGetter{
		namespaceCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileNamespaces(r.ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Namespaces: %v", err)
	}

	return nil
}

func (r *configReconciler) reconcileConfigMaps() error {
	r.log.Debug("Reconciling ConfigMaps")

	creators := []reconciling.NamedConfigMapCreatorGetter{
		uiConfigConfigMapCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileConfigMaps(r.ctx, creators, r.ns, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}

	return nil
}

func (r *configReconciler) reconcileSecrets() error {
	r.log.Debug("Reconciling Secrets")

	creators := []reconciling.NamedSecretCreatorGetter{
		dockercfgSecretCreator(r.ns, r.config),
		kubeconfigSecretCreator(r.ns, r.config),
		dexCASecretCreator(r.ns, r.config),
	}

	if len(r.config.Spec.MasterFiles) > 0 {
		creators = append(creators, masterFilesSecretCreator(r.ns, r.config))
	}

	if r.config.Spec.UI.Presets != "" {
		creators = append(creators, presetsSecretCreator(r.ns, r.config))
	}

	if err := reconciling.ReconcileSecrets(r.ctx, creators, r.ns, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	return nil
}

func (r *configReconciler) reconcileServiceAccounts() error {
	r.log.Debug("Reconciling ServiceAccounts")

	creators := []reconciling.NamedServiceAccountCreatorGetter{
		serviceAccountCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, r.ns, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %v", err)
	}

	return nil
}

func (r *configReconciler) reconcileClusterRoleBindings() error {
	r.log.Debug("Reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		clusterRoleBindingCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileClusterRoleBindings(r.ctx, creators, "", r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %v", err)
	}

	return nil
}

func (r *configReconciler) reconcileDeployments() error {
	r.log.Debug("Reconciling Deployments")

	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubermaticAPIDeploymentCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileDeployments(r.ctx, creators, r.ns, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	return nil
}
