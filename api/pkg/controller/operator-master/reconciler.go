package operatormaster

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/kubermatic"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
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

	identifier, err := cache.MetaNamespaceKeyFunc(config)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to determine string key for KubermaticConfiguration: %v", err)
	}

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

	cfgReconciler := configData{
		Reconciler: *r,
		ctx:        ctx,
		config:     config,
		log:        logger,
	}

	return cfgReconciler.Reconcile()
}

type configData struct {
	Reconciler

	ctx    context.Context
	config *operatorv1alpha1.KubermaticConfiguration
	log    *zap.SugaredLogger
	ns     string
}

func (r *configData) Reconcile() error {
	r.applyDefaults()
	r.ns = r.config.Spec.Namespace

	r.log.Debug("Reconciling Kubermatic installation")
	if err := r.reconcileResources(); err != nil {
		return fmt.Errorf("failed to reconcile: %v", err)
	}

	return nil
}

func (r *configData) applyDefaults() {
	if r.config.Spec.Namespace == "" {
		r.config.Spec.Namespace = r.config.Namespace
		r.log.Debugf("Defaulting field namespace to %s", r.config.Spec.Namespace)
	}

	if r.config.Spec.ExposeStrategy == "" {
		r.config.Spec.ExposeStrategy = "NodePort"
		r.log.Debugf("Defaulting field exposeStrategy to %s", r.config.Spec.ExposeStrategy)
	}

	auth := r.config.Spec.Auth

	if auth.TokenIssuer == "" {
		auth.TokenIssuer = fmt.Sprintf("https://%s/dex", r.config.Spec.Domain)
		r.log.Debugf("Defaulting field auth.tokenIssuer to %s", auth.TokenIssuer)
	}

	if auth.ClientID == "" {
		auth.ClientID = "kubermatic"
		r.log.Debugf("Defaulting field auth.clientID to %s", auth.ClientID)
	}

	if auth.IssuerClientID == "" {
		auth.IssuerClientID = fmt.Sprintf("%sIssuer", auth.ClientID)
		r.log.Debugf("Defaulting field auth.issuerClientID to %s", auth.IssuerClientID)
	}

	if auth.IssuerRedirectURL == "" {
		auth.IssuerRedirectURL = fmt.Sprintf("https://%s/api/v1/kubeconfig", r.config.Spec.Domain)
		r.log.Debugf("Defaulting field auth.issuerRedirectURL to %s", auth.IssuerRedirectURL)
	}

	r.config.Spec.Auth = auth

	r.applyImageDefaults(&r.config.Spec.API.Image, "quay.io/kubermatic/api", "", corev1.PullIfNotPresent, "api.image")
	r.applyImageDefaults(&r.config.Spec.UI.Image, "quay.io/kubermatic/ui-v2", "v1.3.0", corev1.PullIfNotPresent, "ui.image")
	r.applyImageDefaults(&r.config.Spec.MasterController.Image, "quay.io/kubermatic/api", "", corev1.PullIfNotPresent, "masterController.image")
	r.applyImageDefaults(&r.config.Spec.SeedController.Image, "quay.io/kubermatic/api", "", corev1.PullIfNotPresent, "seedController.image")
	r.applyImageDefaults(&r.config.Spec.SeedController.Addons.Kubernetes.Image, "quay.io/kubermatic/addons", "v0.2.19", corev1.PullIfNotPresent, "seedController.addons.kubernetes.image")
	r.applyImageDefaults(&r.config.Spec.SeedController.Addons.Openshift.Image, "quay.io/kubermatic/openshift-addons", "v0.9", corev1.PullIfNotPresent, "seedController.addons.openshift.image")
}

func (r *configData) applyImageDefaults(img *operatorv1alpha1.DockerImage, repo string, tag string, pullPolicy corev1.PullPolicy, key string) {
	if img.Repository == "" && repo != "" {
		img.Repository = repo
		r.log.Debugf("Defaulting Docker repository for %s.repository to %s", key, repo)
	}

	if img.Tag == "" && tag != "" {
		img.Tag = tag
		r.log.Debugf("Defaulting Docker repository for %s.tag to %s", key, tag)
	}

	if img.PullPolicy == "" && pullPolicy != "" {
		img.PullPolicy = pullPolicy
		r.log.Debugf("Defaulting Docker repository for %s.pullPolicy to %s", key, pullPolicy)
	}
}

// applyDefaultFields is generating a new ObjectModifier that wraps an
// ObjectCreator and takes care of applying the default labels and
// annotations from this operator. These are then used to establish
// a weak ownership.
func (r *configData) applyDefaultFields() reconciling.ObjectModifier {
	return func(create reconciling.ObjectCreator) reconciling.ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			if o, ok := obj.(metav1.Object); ok {
				annotations := o.GetAnnotations()
				if annotations == nil {
					annotations = make(map[string]string)
				}

				identifier, err := cache.MetaNamespaceKeyFunc(r.config)
				if err != nil {
					return obj, fmt.Errorf("failed to determine KubermaticConfiguration string key: %v", err)
				}

				annotations[ConfigurationOwnerAnnotation] = identifier
				o.SetAnnotations(annotations)

				labels := o.GetLabels()
				if labels == nil {
					labels = make(map[string]string)
				}
				labels[ManagedByLabel] = ControllerName
				o.SetLabels(labels)
			}

			return obj, nil
		}
	}
}

func (r *configData) reconcileResources() error {
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

	if err := r.reconcilePodDisruptionBudgets(); err != nil {
		return err
	}

	if err := r.reconcileServices(); err != nil {
		return err
	}

	if err := r.reconcileIngresses(); err != nil {
		return err
	}

	return nil
}

func (r *configData) reconcileNamespaces() error {
	r.log.Debug("Reconciling Namespaces")

	creators := []reconciling.NamedNamespaceCreatorGetter{
		kubermatic.NamespaceCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileNamespaces(r.ctx, creators, "", r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile Namespaces: %v", err)
	}

	return nil
}

func (r *configData) reconcileConfigMaps() error {
	r.log.Debug("Reconciling ConfigMaps")

	creators := []reconciling.NamedConfigMapCreatorGetter{
		kubermatic.UIConfigConfigMapCreator(r.ns, r.config),
		kubermatic.BackupContainersConfigMapCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileConfigMaps(r.ctx, creators, r.ns, r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}

	return nil
}

func (r *configData) reconcileSecrets() error {
	r.log.Debug("Reconciling Secrets")

	creators := []reconciling.NamedSecretCreatorGetter{
		kubermatic.DockercfgSecretCreator(r.ns, r.config),
		kubermatic.KubeconfigSecretCreator(r.ns, r.config),
		kubermatic.DexCASecretCreator(r.ns, r.config),
	}

	if r.config.Spec.Datacenters != "" {
		creators = append(creators, kubermatic.DatacentersSecretCreator(r.ns, r.config))
	}

	if len(r.config.Spec.MasterFiles) > 0 {
		creators = append(creators, kubermatic.MasterFilesSecretCreator(r.ns, r.config))
	}

	if r.config.Spec.UI.Presets != "" {
		creators = append(creators, kubermatic.PresetsSecretCreator(r.ns, r.config))
	}

	if err := reconciling.ReconcileSecrets(r.ctx, creators, r.ns, r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	return nil
}

func (r *configData) reconcileServiceAccounts() error {
	r.log.Debug("Reconciling ServiceAccounts")

	creators := []reconciling.NamedServiceAccountCreatorGetter{
		kubermatic.ServiceAccountCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileServiceAccounts(r.ctx, creators, r.ns, r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %v", err)
	}

	return nil
}

func (r *configData) reconcileClusterRoleBindings() error {
	r.log.Debug("Reconciling ClusterRoleBindings")

	creators := []reconciling.NamedClusterRoleBindingCreatorGetter{
		kubermatic.ClusterRoleBindingCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileClusterRoleBindings(r.ctx, creators, "", r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %v", err)
	}

	return nil
}

func (r *configData) reconcileDeployments() error {
	r.log.Debug("Reconciling Deployments")

	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubermatic.APIDeploymentCreator(r.ns, r.config),
		kubermatic.UIDeploymentCreator(r.ns, r.config),
		kubermatic.MasterControllerManagerDeploymentCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileDeployments(r.ctx, creators, r.ns, r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	return nil
}

func (r *configData) reconcilePodDisruptionBudgets() error {
	r.log.Debug("Reconciling PodDisruptionBudgets")

	creators := []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		kubermatic.APIPDBCreator(r.ns, r.config),
		kubermatic.UIPDBCreator(r.ns, r.config),
		kubermatic.MasterControllerManagerPDBCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcilePodDisruptionBudgets(r.ctx, creators, r.ns, r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %v", err)
	}

	return nil
}

func (r *configData) reconcileServices() error {
	r.log.Debug("Reconciling Services")

	creators := []reconciling.NamedServiceCreatorGetter{
		kubermatic.APIServiceCreator(r.ns, r.config),
		kubermatic.UIServiceCreator(r.ns, r.config),
		kubermatic.MasterControllerManagerServiceCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileServices(r.ctx, creators, r.ns, r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile Services: %v", err)
	}

	return nil
}

func (r *configData) reconcileIngresses() error {
	r.log.Debug("Reconciling Ingresses")

	creators := []reconciling.NamedIngressCreatorGetter{
		kubermatic.IngressCreator(r.ns, r.config),
	}

	if err := reconciling.ReconcileIngresses(r.ctx, creators, r.ns, r.Client, r.applyDefaultFields()); err != nil {
		return fmt.Errorf("failed to reconcile Ingresses: %v", err)
	}

	return nil
}
