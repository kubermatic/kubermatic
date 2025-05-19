//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

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

package kyverno

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	admissioncontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/admission-controller"

	backgroundcontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/background-controller"
	cleanupcontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/cleanup-controller"
	commonresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/common"
	reportscontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/reports-controller"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster"
	userclusteradmissioncontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/admission-controller"
	userclusterbackgroundcontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/background-controller"
	userclustercleanupcontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/cleanup-controller"
	userclustercommonresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/common"
	userclusterreportscontrollerresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/user-cluster/reports-controller"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName    = "kkp-kyverno-controller"
	CleanupFinalizer  = "kubermatic.k8c.io/cleanup-kyverno"
	healthCheckPeriod = 5 * time.Second
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type reconciler struct {
	ctrlruntimeclient.Client

	workerName                    string
	recorder                      record.EventRecorder
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

func Add(mgr manager.Manager, numWorkers int, workerName string, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &reconciler{
		Client:                        mgr.GetClient(),
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
	}

	clusterIsAlive := predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		// Only watch clusters that are in a state where they can be reconciled.
		return !cluster.Spec.Pause && cluster.Status.NamespaceName != ""
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(workerlabel.Predicate(workerName), clusterIsAlive)).
		Build(reconciler)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		if kuberneteshelper.HasFinalizer(cluster, CleanupFinalizer) {
			log.Debug("Cleaning up Kyverno resources")
			return reconcile.Result{}, r.handleKyvernoCleanup(ctx, cluster)
		}
		return reconcile.Result{}, nil
	}

	// Kyverno was disabled after it was enabled. Clean up resources.
	if kuberneteshelper.HasFinalizer(cluster, CleanupFinalizer) && !cluster.Spec.IsKyvernoEnabled() {
		log.Debug("Cleaning up Kyverno resources")
		return reconcile.Result{}, r.handleKyvernoCleanup(ctx, cluster)
	}

	// Kyverno is disabled. Nothing to do.
	if !cluster.Spec.IsKyvernoEnabled() {
		return reconcile.Result{}, nil
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because it has no namespace yet")
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		log.Debugf("API server is not running, trying again in %v", healthCheckPeriod)
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionKyvernoControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return *result, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Add finalizer to the cluster
	if !kuberneteshelper.HasFinalizer(cluster, CleanupFinalizer) {
		if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, CleanupFinalizer); err != nil {
			return nil, fmt.Errorf("failed to set %q finalizer: %w", CleanupFinalizer, err)
		}
	}

	// Install CRDs in user cluster
	if err := r.ensureUserClusterResources(ctx, cluster); err != nil {
		return nil, err
	}

	// Install resources in seed user cluster namespace
	if err := r.ensureSeedClusterNamespaceResources(ctx, cluster); err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *reconciler) ensureUserClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	// // Create/update Namespace for all controllers
	// namespaceCreators := []reconciling.NamedNamespaceReconcilerFactory{
	// 	commonresources.KyvernoNamespaceConfigMapReconciler(cluster),
	// }

	// if err := reconciling.ReconcileNamespaces(ctx, namespaceCreators, "", userClusterClient); err != nil {
	// 	return fmt.Errorf("failed to reconcile Namespaces: %w", err)
	// }

	// Create/update ConfigMaps for all controllers
	configMapCreators := []reconciling.NamedConfigMapReconcilerFactory{
		commonresources.KyvernoConfigMapReconciler(cluster),
		commonresources.KyvernoMetricsConfigMapReconciler(cluster),
	}

	if err := reconciling.ReconcileConfigMaps(ctx, configMapCreators, cluster.Status.NamespaceName, userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %w", err)
	}

	crds, err := userclusterresources.KyvernoCRDs()
	if err != nil {
		return fmt.Errorf("failed to get Kyverno CRDs: %w", err)
	}

	crdReconcilers := make([]kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory, 0, len(crds))
	for _, crd := range crds {
		crdReconcilers = append(crdReconcilers, userclusterresources.KyvernoCRDReconciler(crd))
	}

	if err := kkpreconciling.ReconcileCustomResourceDefinitions(ctx, crdReconcilers, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile Kyverno CRDs: %w", err)
	}

	// Create/update ServiceAccounts for all controllers
	serviceAccountCreators := []reconciling.NamedServiceAccountReconcilerFactory{
		backgroundcontrollerresources.ServiceAccountReconciler(cluster),
		reportscontrollerresources.ServiceAccountReconciler(cluster),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, serviceAccountCreators, cluster.Status.NamespaceName, userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %w", err)
	}

	// Reconcile ClusterRoles for all controllers
	admissionControllerClusterRoleCreators := []reconciling.NamedClusterRoleReconcilerFactory{
		userclusteradmissioncontrollerresources.ClusterRoleReconciler(),
		userclusteradmissioncontrollerresources.CoreClusterRoleReconciler(),
	}

	if err := reconciling.ReconcileClusterRoles(ctx, admissionControllerClusterRoleCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile admission controller ClusterRoles: %w", err)
	}

	backgroundControllerClusterRoleCreators := []reconciling.NamedClusterRoleReconcilerFactory{
		userclusterbackgroundcontrollerresources.BackgroundClusterRoleReconciler(),
		userclusterbackgroundcontrollerresources.BackgroundCoreClusterRoleReconciler(),
	}

	if err := reconciling.ReconcileClusterRoles(ctx, backgroundControllerClusterRoleCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile background controller ClusterRoles: %w", err)
	}

	cleanupControllerClusterRoleCreators := []reconciling.NamedClusterRoleReconcilerFactory{
		userclustercleanupcontrollerresources.ClusterRoleReconciler(),
		userclustercleanupcontrollerresources.CoreClusterRoleReconciler(),
	}

	if err := reconciling.ReconcileClusterRoles(ctx, cleanupControllerClusterRoleCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile cleanup controller ClusterRoles: %w", err)
	}

	reportsControllerClusterRoleCreators := []reconciling.NamedClusterRoleReconcilerFactory{
		userclusterreportscontrollerresources.ClusterRoleReconciler(),
		userclusterreportscontrollerresources.CoreClusterRoleReconciler(),
	}

	if err := reconciling.ReconcileClusterRoles(ctx, reportsControllerClusterRoleCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile reports controller ClusterRoles: %w", err)
	}

	commonClusterRoleCreators := []reconciling.NamedClusterRoleReconcilerFactory{
		userclustercommonresources.AdminPoliciesClusterRoleReconciler(),
		userclustercommonresources.ViewPoliciesClusterRoleReconciler(),
		userclustercommonresources.AdminPolicyReportsClusterRoleReconciler(),
		userclustercommonresources.ViewPolicyReportsClusterRoleReconciler(),
		userclustercommonresources.AdminReportsClusterRoleReconciler(),
		userclustercommonresources.ViewReportsClusterRoleReconciler(),
		userclustercommonresources.AdminUpdateRequestsClusterRoleReconciler(),
		userclustercommonresources.ViewUpdateRequestsClusterRoleReconciler(),
	}

	if err := reconciling.ReconcileClusterRoles(ctx, commonClusterRoleCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile common ClusterRoles: %w", err)
	}

	// Reconcile ClusterRoleBindings for all controllers
	admissionControllerClusterRoleBindingCreators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userclusteradmissioncontrollerresources.ClusterRoleBindingReconciler(),
		userclusteradmissioncontrollerresources.ViewClusterRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, admissionControllerClusterRoleBindingCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile admission controller ClusterRoleBindings: %w", err)
	}

	backgroundControllerClusterRoleBindingCreators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userclusterbackgroundcontrollerresources.BackgroundClusterRoleBindingReconciler(),
		userclusterbackgroundcontrollerresources.BackgroundViewClusterRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, backgroundControllerClusterRoleBindingCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile background controller ClusterRoleBindings: %w", err)
	}

	cleanupControllerClusterRoleBindingCreators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userclustercleanupcontrollerresources.ClusterRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, cleanupControllerClusterRoleBindingCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile cleanup controller ClusterRoleBindings: %w", err)
	}

	reportsControllerClusterRoleBindingCreators := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userclusterreportscontrollerresources.ClusterRoleBindingReconciler(),
		userclusterreportscontrollerresources.ViewClusterRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, reportsControllerClusterRoleBindingCreators, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile reports controller ClusterRoleBindings: %w", err)
	}

	// ---------------------------------------------------------------------
	// Admission WebhookConfigurations (Mutating + Validating)
	// ---------------------------------------------------------------------

	// Fetch cluster root CA to embed into webhook configurations
	// caKeyPair, err := resources.GetClusterRootCA(ctx, cluster.Status.NamespaceName, r.Client)
	// if err != nil {
	// 	return fmt.Errorf("failed to get cluster root CA: %w", err)
	// }

	// mutatingCreators := []reconciling.NamedMutatingWebhookConfigurationReconcilerFactory{
	// 	userclusterresources.MutatingWebhookConfigurationReconciler(caKeyPair.Cert, cluster.Status.NamespaceName),
	// }
	// if err := reconciling.ReconcileMutatingWebhookConfigurations(ctx, mutatingCreators, "", userClusterClient); err != nil {
	// 	return fmt.Errorf("failed to reconcile Kyverno mutating webhook configuration: %w", err)
	// }

	// validatingCreators := []reconciling.NamedValidatingWebhookConfigurationReconcilerFactory{
	// 	userclusterresources.ValidatingWebhookConfigurationReconciler(caKeyPair.Cert, cluster.Status.NamespaceName),
	// }
	// if err := reconciling.ReconcileValidatingWebhookConfigurations(ctx, validatingCreators, "", userClusterClient); err != nil {
	// 	return fmt.Errorf("failed to reconcile Kyverno validating webhook configuration: %w", err)
	// }

	return nil
}

func (r *reconciler) ensureSeedClusterNamespaceResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Create/update ServiceAccounts for all controllers
	serviceAccountCreators := []reconciling.NamedServiceAccountReconcilerFactory{
		admissioncontrollerresources.ServiceAccountReconciler(cluster),
		backgroundcontrollerresources.ServiceAccountReconciler(cluster),
		// reportscontrollerresources.ServiceAccountReconciler(cluster),
		cleanupcontrollerresources.ServiceAccountReconciler(cluster),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, serviceAccountCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAccounts: %w", err)
	}

	// // Build minimal TemplateData to satisfy the tlsServingCertReconcilerData interface.
	// td := resources.NewTemplateDataBuilder().
	// 	WithContext(ctx).
	// 	WithClient(r.Client).
	// 	WithCluster(cluster).
	// 	Build()

	// secretCreators := []reconciling.NamedSecretReconcilerFactory{
	// 	admissioncontrollerresources.TLSServingCAReconciler(td),
	// 	admissioncontrollerresources.TLSServingCertificateReconciler(td),
	// }

	// if err := reconciling.ReconcileSecrets(ctx, secretCreators, cluster.Status.NamespaceName, r.Client); err != nil {
	// 	return fmt.Errorf("failed to reconcile Secrets: %w", err)
	// }

	// // Create/update ConfigMaps for all controllers
	// configMapCreators := []reconciling.NamedConfigMapReconcilerFactory{
	// 	commonresources.KyvernoConfigMapReconciler(cluster),
	// 	commonresources.KyvernoMetricsConfigMapReconciler(cluster),
	// }

	// if err := reconciling.ReconcileConfigMaps(ctx, configMapCreators, cluster.Status.NamespaceName, r.Client); err != nil {
	// 	return fmt.Errorf("failed to reconcile ConfigMaps: %w", err)
	// }

	// Create/update Roles and RoleBindings for all controllers
	roleCreators := []reconciling.NamedRoleReconcilerFactory{
		admissioncontrollerresources.RoleReconciler(cluster),
		backgroundcontrollerresources.RoleReconciler(cluster),
		// reportscontrollerresources.RoleReconciler(cluster),
		cleanupcontrollerresources.RoleReconciler(cluster),
	}

	if err := reconciling.ReconcileRoles(ctx, roleCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %w", err)
	}

	roleBindingCreators := []reconciling.NamedRoleBindingReconcilerFactory{
		admissioncontrollerresources.RoleBindingReconciler(cluster),
		backgroundcontrollerresources.RoleBindingReconciler(cluster),
		// reportscontrollerresources.RoleBindingReconciler(cluster),
		cleanupcontrollerresources.RoleBindingReconciler(cluster),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, roleBindingCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings: %w", err)
	}

	// Create/update Services for all controllers
	serviceCreators := []reconciling.NamedServiceReconcilerFactory{
		admissioncontrollerresources.ServiceReconciler(cluster),
		admissioncontrollerresources.MetricsServiceReconciler(cluster),
		backgroundcontrollerresources.MetricsServiceReconciler(cluster),
		reportscontrollerresources.MetricsServiceReconciler(cluster),
		cleanupcontrollerresources.ServiceReconciler(cluster),
		cleanupcontrollerresources.MetricsServiceReconciler(cluster),
	}

	if err := reconciling.ReconcileServices(ctx, serviceCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Services: %w", err)
	}

	// Create/update Deployments for all controllers
	deploymentCreators := []reconciling.NamedDeploymentReconcilerFactory{
		admissioncontrollerresources.DeploymentReconciler(cluster),
		backgroundcontrollerresources.DeploymentReconciler(cluster),
		reportscontrollerresources.DeploymentReconciler(cluster),
		cleanupcontrollerresources.DeploymentReconciler(cluster),
	}

	if err := reconciling.ReconcileDeployments(ctx, deploymentCreators, cluster.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %w", err)
	}

	return nil
}
