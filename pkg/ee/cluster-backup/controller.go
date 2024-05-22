//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2023 Kubermatic GmbH

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

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/cluster-backup/resources/user-cluster"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "cluster-backup-controller"

	clusterBackupComponentLabelKey   = "component"
	clusterBackupComponentLabelValue = "velero"

	// componentsInstalledLabel is a label that is put on Clusters where the cluster-backup
	// components are installed. This is only used for cleanups to prevent this controller
	// from endlessly cleaning up the same cluster over and over again.
	// This should not be a finalizer because we do not want to block cluster deletion just
	// because a Velero CRD is installed in the usercluster.
	componentsInstalledLabel = "k8c.io/cluster-backup-installed"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type reconciler struct {
	ctrlruntimeclient.Client

	seedGetter                    provider.SeedGetter
	workerName                    string
	recorder                      record.EventRecorder
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

func Add(mgr manager.Manager, numWorkers int, workerName string, userClusterConnectionProvider UserClusterClientProvider, seedGetter provider.SeedGetter, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &reconciler{
		Client:                        mgr.GetClient(),
		seedGetter:                    seedGetter,
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
	}

	clusterIsAlive := predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		// Only watch clusters that are in a state where they can be reconciled.
		// Pause-flag is checked by the ReconcileWrapper.
		return cluster.DeletionTimestamp == nil && cluster.Status.ExtendedHealth.ControlPlaneHealthy()
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
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Resource is marked for deletion.
	if cluster.DeletionTimestamp != nil {
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
			return r.reconcile(ctx, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, err
	}

	return *result, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	log := r.log.With("cluster", cluster)
	log.Debug("Reconciling")

	if !cluster.Spec.IsClusterBackupEnabled() {
		if err := r.removeUserClusterComponents(ctx, cluster); err != nil {
			return nil, fmt.Errorf("failed to remove backup components: %w", err)
		}
		return nil, nil
	}

	cbsl := &kubermaticv1.ClusterBackupStorageLocation{}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.Spec.BackupConfig.BackupStorageLocation.Name}, cbsl); err != nil {
		log.Debug("ClusterBackupStorageLocation not found")
		return nil, nil
	}

	if !inSameProject(cluster, cbsl) {
		return nil, fmt.Errorf("unable to use ClusterBackupStorageLocation %q: cluster and CBSL must belong to the same project", cbsl.Name)
	}
	if err := r.ensureUserClusterResources(ctx, cluster, cbsl); err != nil {
		return nil, fmt.Errorf("failed to ensure user-cluster resources: %w", err)
	}
	return nil, nil
}

func addManagedByLabel(create reconciling.ObjectReconciler) reconciling.ObjectReconciler {
	return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
		created, err := create(existing)
		if err != nil {
			return nil, err
		}

		kubernetes.EnsureLabels(created, map[string]string{
			appskubermaticv1.ApplicationManagedByLabel: ControllerName,
		})

		return created, nil
	}
}

func (r *reconciler) ensureUserClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster, cbsl *kubermaticv1.ClusterBackupStorageLocation) error {
	// mark the cluster so we can clean it up when necessary
	if _, ok := cluster.Labels[componentsInstalledLabel]; !ok {
		err := r.patchCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			kubernetes.EnsureLabels(c, map[string]string{componentsInstalledLabel: "yes"})
		})
		if err != nil {
			return fmt.Errorf("failed to mark cluster: %w", err)
		}
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}
	nsReconcilers := []reconciling.NamedNamespaceReconcilerFactory{
		userclusterresources.NamespaceReconciler(),
	}
	if err := reconciling.ReconcileNamespaces(ctx, nsReconcilers, "", userClusterClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero Namespace: %w", err)
	}

	saReconcilers := []reconciling.NamedServiceAccountReconcilerFactory{
		userclusterresources.ServiceAccountReconciler(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, saReconcilers, resources.ClusterBackupNamespaceName, userClusterClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero ServiceAccount: %w", err)
	}

	clusterRoleBindingReconciler := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userclusterresources.ClusterRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingReconciler, "", userClusterClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero ClusterRoleBinding: %w", err)
	}

	secretReconcilers := []reconciling.NamedSecretReconcilerFactory{
		userclusterresources.SecretReconciler(ctx, r.Client, cluster, cbsl),
	}

	// Create kubeconfig secret in the user cluster namespace.
	if err := reconciling.ReconcileSecrets(ctx, secretReconcilers, resources.ClusterBackupNamespaceName, userClusterClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile cluster backup kubeconfig Secret: %w", err)
	}

	deploymentReconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		userclusterresources.DeploymentReconciler(),
	}
	if err := reconciling.ReconcileDeployments(ctx, deploymentReconcilers, resources.ClusterBackupNamespaceName, userClusterClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile the cluster backup Deployment: %w", err)
	}
	dsReconcilers := []reconciling.NamedDaemonSetReconcilerFactory{
		userclusterresources.DaemonSetReconciler(),
	}

	if err := reconciling.ReconcileDaemonSets(ctx, dsReconcilers, resources.ClusterBackupNamespaceName, userClusterClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero node-agent DaemonSet: %w", err)
	}

	clusterBackupCRDs, err := userclusterresources.CRDs()
	if err != nil {
		return fmt.Errorf("failed to load cluster backup CRDs: %w", err)
	}

	creators := []kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory{}
	for i := range clusterBackupCRDs {
		creators = append(creators, userclusterresources.CRDReconciler(clusterBackupCRDs[i]))
	}

	if err = kkpreconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", userClusterClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero CRDs: %w", err)
	}

	bslReconcilers := []kkpreconciling.NamedBackupStorageLocationReconcilerFactory{
		userclusterresources.BSLReconciler(ctx, cluster, cbsl),
	}

	if err := kkpreconciling.ReconcileBackupStorageLocations(ctx, bslReconcilers, resources.ClusterBackupNamespaceName, userClusterClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero BSL: %w", err)
	}

	return nil
}

func (r *reconciler) removeUserClusterComponents(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if _, ok := cluster.Labels[componentsInstalledLabel]; !ok {
		return nil
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	if err := r.removeUserClusterResources(ctx, userClusterClient); err != nil {
		return fmt.Errorf("failed to remove user-cluster resources: %w", err)
	}

	if err := r.removeCRDs(ctx, userClusterClient); err != nil {
		return fmt.Errorf("failed to remove CRDs: %w", err)
	}

	err = r.patchCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		delete(c.Labels, componentsInstalledLabel)
	})
	if err != nil {
		return fmt.Errorf("failed to unmark cluster: %w", err)
	}

	return nil
}

func (r *reconciler) removeUserClusterResources(ctx context.Context, userClusterClient ctrlruntimeclient.Client) error {
	// remove resources created in ./pkg/ee/cluster-backup/resources/user-cluster
	userClusterResources := []ctrlruntimeclient.Object{
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userclusterresources.DaemonSetName,
				Namespace: resources.ClusterBackupNamespaceName,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userclusterresources.DeploymentName,
				Namespace: resources.ClusterBackupNamespaceName,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.ClusterBackupServiceAccountName,
				Namespace: resources.ClusterBackupNamespaceName,
			},
		},
		&velerov1.BackupStorageLocation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userclusterresources.DefaultBSLName,
				Namespace: resources.ClusterBackupNamespaceName,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      userclusterresources.CloudCredentialsSecretName,
				Namespace: resources.ClusterBackupNamespaceName,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: userclusterresources.ClusterRoleBindingName,
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.ClusterBackupNamespaceName,
			},
		},
	}

	for _, resource := range userClusterResources {
		if err := removeManagedObject(ctx, userClusterClient, resource); err != nil {
			return err
		}
	}
	return nil
}

func removeManagedObject(ctx context.Context, client ctrlruntimeclient.Client, resource ctrlruntimeclient.Object) error {
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(resource), resource); err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil
		}
		return fmt.Errorf("failed to list user-cluster resources: %w", err)
	}

	if isManagedBackupResource(resource) {
		if err := client.Delete(ctx, resource); err != nil && !(apierrors.IsNotFound(err) || meta.IsNoMatchError(err)) {
			return fmt.Errorf("failed to delete user-cluster resource: %w", err)
		}
	}

	return nil
}

func (r *reconciler) removeCRDs(ctx context.Context, userClusterClient ctrlruntimeclient.Client) error {
	crdList := &apiextensionsv1.CustomResourceDefinitionList{}

	listOpts := &ctrlruntimeclient.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			clusterBackupComponentLabelKey:             clusterBackupComponentLabelValue,
			appskubermaticv1.ApplicationManagedByLabel: ControllerName,
		}),
	}
	if err := userClusterClient.List(ctx, crdList, listOpts); err != nil {
		return fmt.Errorf("failed to list CRDs: %w", err)
	}
	for _, crd := range crdList.Items {
		if err := userClusterClient.Delete(ctx, &crd); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete CRD: %w", err)
		}
	}
	return nil
}

func inSameProject(cluster *kubermaticv1.Cluster, cbsl *kubermaticv1.ClusterBackupStorageLocation) bool {
	return cluster.Labels[kubermaticv1.ProjectIDLabelKey] == cbsl.Labels[kubermaticv1.ProjectIDLabelKey]
}

func isManagedBackupResource(resource ctrlruntimeclient.Object) bool {
	labels := resource.GetLabels()
	return labels[appskubermaticv1.ApplicationManagedByLabel] == ControllerName
}

func (r *reconciler) patchCluster(ctx context.Context, cluster *kubermaticv1.Cluster, patch kubermaticv1helper.ClusterPatchFunc) error {
	// modify it
	original := cluster.DeepCopy()
	patch(cluster)

	// update the status
	return r.Client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original))
}
