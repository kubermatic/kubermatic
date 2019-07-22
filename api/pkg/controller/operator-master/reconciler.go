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
	// logger.Debug("Garbage-collecting orphaned resources...")
	// if err := r.garbageCollect(ctx); err != nil {
	// 	return reconcile.Result{}, fmt.Errorf("failed to garbage collect: %v", err)
	// }

	cfgReconciler := configReconciler{
		Reconciler: *r,
		ctx:        ctx,
		config:     config,
		log:        logger,
	}

	logger.Debug("Reconciling Kubermatic installation")
	if err := cfgReconciler.reconcile(); err != nil {
		return fmt.Errorf("failed to reconcile: %v", err)
	}

	return nil
}

/*
func (r *Reconciler) garbageCollect(ctx context.Context) error {
	list := &corev1.SecretList{}
	options := &ctrlruntimeclient.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			ManagedByLabel: ControllerName,
		}),
	}

	if err := r.List(ctx, options, list); err != nil {
		return fmt.Errorf("failed to list Secrets: %v", err)
	}

	for _, item := range list.Items {
		meta := item.GetObjectMeta()
		seed := meta.GetLabels()[InstanceLabel]

		if r.unknownSeed(seed) {
			logger.Debug("Deleting orphaned Secret %s/%s referencing non-existing seed '%s'...", item.Namespace, item.Name, seed)
			if err := r.Delete(ctx, &item); err != nil {
				return fmt.Errorf("failed to delete Secret: %v", err)
			}
		}
	}

	return nil
}
*/

type configReconciler struct {
	Reconciler

	ctx    context.Context
	config *operatorv1alpha1.KubermaticConfiguration
	log    *zap.SugaredLogger
}

func (r *configReconciler) reconcile() error {
	r.applyDefaults()

	r.log.Debug("Reconciling Kubermatic installation")
	if err := r.reconcileResources(); err != nil {
		return fmt.Errorf("failed to reconcile: %v", err)
	}

	// logger.Debug("Reconciling Grafana provisioning...")
	// if err := r.reconcileMasterGrafanaProvisioning(ctx); err != nil {
	// 	return reconcile.Result{}, fmt.Errorf("failed to reconcile Grafana: %v", err)
	// }

	return nil
}

func (r *configReconciler) applyDefaults() {
	if r.config.Spec.Namespace == "" {
		r.config.Spec.Namespace = r.config.Namespace
	}
}

func (r *configReconciler) reconcileResources() error {
	if err := r.reconcileNamespaces(); err != nil {
		return fmt.Errorf("failed to reconcile namespaces: %v", err)
	}

	// logger.Debug("Reconciling secrets...")
	// secret, err := r.reconcileSecrets()
	// if err != nil {
	// 	return fmt.Errorf("failed to reconcile secrets: %v", err)
	// }

	// logger.Debug("Reconciling deployments...")
	// if err := r.reconcileMasterDeployments(ctx, contextName, secret); err != nil {
	// 	return fmt.Errorf("failed to reconcile deployments: %v", err)
	// }

	// logger.Debug("Reconciling services...")
	// if err := r.reconcileMasterServices(ctx, contextName, secret); err != nil {
	// 	return fmt.Errorf("failed to reconcile services: %v", err)
	// }

	return nil
}

func (r *configReconciler) reconcileNamespaces() error {
	r.log.Debug("Reconciling namespaces...")

	creators := []reconciling.NamedNamespaceCreatorGetter{
		namespaceCreator(r.config),
	}

	if err := reconciling.ReconcileNamespaces(r.ctx, creators, r.config.Spec.Namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Namespaces: %v", err)
	}

	return nil
}

/*
func (r *configReconciler) reconcileSecrets() (*corev1.Secret, error) {
	creators := []reconciling.NamedSecretCreatorGetter{
		masterSecretCreator(contextName, credentials),
	}

	if err := reconciling.ReconcileSecrets(ctx, creators, MasterTargetNamespace, r.Client); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets in the namespace %s: %v", MasterTargetNamespace, err)
	}

	secret := &corev1.Secret{}
	name := types.NamespacedName{
		Namespace: MasterTargetNamespace,
		Name:      secretName(contextName),
	}

	if err := r.Get(ctx, name, secret); err != nil {
		return nil, fmt.Errorf("could not find Secret '%s'", name)
	}

	return secret, nil
}

func (r *configReconciler) reconcileMasterDeployments(ctx context.Context, contextName string, secret *corev1.Secret) error {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		masterDeploymentCreator(contextName, secret),
	}

	if err := reconciling.ReconcileDeployments(ctx, creators, MasterTargetNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments in the namespace %s: %v", MasterTargetNamespace, err)
	}

	return nil
}

func (r *configReconciler) reconcileMasterServices(ctx context.Context, contextName string, secret *corev1.Secret) error {
	creators := []reconciling.NamedServiceCreatorGetter{
		masterServiceCreator(contextName, secret),
	}

	if err := reconciling.ReconcileServices(ctx, creators, MasterTargetNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services in the namespace %s: %v", MasterTargetNamespace, err)
	}

	return nil
}

func (r *configReconciler) reconcileMasterGrafanaProvisioning(ctx context.Context) error {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		masterGrafanaConfigmapCreator(r.seeds, r.kubeconfig),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, creators, MasterGrafanaNamespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps in the namespace %s: %v", MasterGrafanaNamespace, err)
	}

	return nil
}

func (r *configReconciler) unknownSeed(seed string) bool {
	_, ok := r.kubeconfig.Contexts[seed]

	return !ok
}
*/
