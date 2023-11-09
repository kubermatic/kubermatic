//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

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

package clusterbackup

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	seedclusterresources "k8c.io/kubermatic/v2/pkg/ee/cluster-backup/resources/seed-cluster"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/cluster-backup/resources/user-cluster"

	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	clusterutil "k8c.io/kubermatic/v2/pkg/util/cluster"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "cluster-backup-controller"
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
	overwriteRegistry             string
	versions                      kubermatic.Versions
	clusterName                   string
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

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &kubermaticv1.Cluster{}), &handler.EnqueueRequestForObject{}, workerlabel.Predicates(workerName), predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		// Only watch clusters that are in a state where they can be reconciled.
		return !cluster.Spec.Pause && cluster.DeletionTimestamp == nil && cluster.Status.NamespaceName != ""
	})); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}
	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Resource is marked for deletion.
	if cluster.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	backupConfig, err := clusterutil.FetchClusterBackupConfigWithSeedClient(ctx, r.Client, cluster, r.log)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to fetch cluster backup config: %w", err)
	}

	// clusterbackup is disabled. Nothing to do.
	if !backupConfig.Enabled {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionClusterBackupControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster, backupConfig)
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

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, backupConfig *resources.ClusterBackupConfig) (*reconcile.Result, error) {
	if err := r.ensureClusterBackupUserClusterResources(ctx, cluster, backupConfig); err != nil {
		return nil, fmt.Errorf("failed to ensure Cluster Backup user-cluster resources: %w", err)
	}

	if err := r.ensureClusterBackupSeedClusterResources(ctx, cluster, backupConfig); err != nil {
		return nil, fmt.Errorf("failed to ensure Cluster Backup Seed cluster resources: %w", err)
	}
	return nil, nil
}

func (r *reconciler) ensureClusterBackupUserClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster, backupConfig *resources.ClusterBackupConfig) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}
	nsReconcilers := []reconciling.NamedNamespaceReconcilerFactory{
		userclusterresources.NamespaceReconciler(),
	}
	if err := reconciling.ReconcileNamespaces(ctx, nsReconcilers, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile velero namespace: %w", err)
	}

	saReconcilers := []reconciling.NamedServiceAccountReconcilerFactory{
		userclusterresources.ServiceAccountReconciler(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, saReconcilers, resources.ClusterBackupNamespaceName, userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile velero service account: %w", err)
	}

	clusterRoleBindingReconciler := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userclusterresources.ClusterRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingReconciler, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile velero cluster role binding: %w", err)
	}

	clusterBackupCRDs, err := userclusterresources.CRDs()
	if err != nil {
		return fmt.Errorf("failed to load Cluster Backup CRDs: %w", err)
	}
	creators := []kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory{}
	for i := range clusterBackupCRDs {
		creators = append(creators, userclusterresources.CRDReconciler(clusterBackupCRDs[i]))
	}

	err = kkpreconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", userClusterClient)

	if err != nil {
		return fmt.Errorf("failed to reconcile velero CustomResourceDefinitions: %w", err)
	}

	if err := userclusterresources.EnsureVeleroBSL(ctx, userClusterClient, backupConfig, r.clusterName); err != nil {
		return fmt.Errorf("failed to ensure velero BSL: %w", err)
	}

	return nil
}

func (r *reconciler) ensureClusterBackupSeedClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster, backupConfig *resources.ClusterBackupConfig) error {
	namespace := cluster.Status.NamespaceName

	clusterData := newClusterData(ctx, cluster, backupConfig)
	secretReconcilers := []reconciling.NamedSecretReconcilerFactory{
		seedclusterresources.SecretReconciler(ctx, r.Client, clusterData),
	}

	// Create kubeconfig secret in the user cluster namespace.
	if err := reconciling.ReconcileSecrets(ctx, secretReconcilers, namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Cluster Backup kubeconfig secret: %w", err)
	}

	deploymentReconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		seedclusterresources.DeploymentReconciler(clusterData),
	}
	if err := reconciling.ReconcileDeployments(ctx, deploymentReconcilers, namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the Cluster Backup deployment: %w", err)
	}
	return nil
}

func newClusterData(ctx context.Context, cluster *kubermaticv1.Cluster, backupConfig *resources.ClusterBackupConfig) *resources.TemplateData {
	return resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithCluster(cluster).
		WithClusterBackupConfig(backupConfig).
		Build()
}
