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

package velerocontroller

import (
	"context"
	"fmt"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	userclusterresources "k8c.io/kubermatic/v2/pkg/ee/cluster-backup/user-cluster/velero-controller/resources"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
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
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "cluster-backup-velero-controller"

	clusterBackupComponentLabelKey   = "component"
	clusterBackupComponentLabelValue = "velero"

	// componentsInstalledLabel is a label that is put on Clusters where the cluster-backup
	// components are installed. This is only used for cleanups to prevent this controller
	// from endlessly cleaning up the same cluster over and over again.
	// This should not be a finalizer because we do not want to block cluster deletion just
	// because a Velero CRD is installed in the usercluster.
	componentsInstalledLabel = "k8c.io/cluster-backup-installed"
)

type reconciler struct {
	seedClient ctrlruntimeclient.Client
	userClient ctrlruntimeclient.Client

	recorder          events.EventRecorder
	log               *zap.SugaredLogger
	versions          kubermatic.Versions
	overwriteRegistry string
}

func Add(
	seedMgr, userMgr manager.Manager,
	log *zap.SugaredLogger,
	clusterName string,
	versions kubermatic.Versions,
	overwriteRegistry string,
) error {
	reconciler := &reconciler{
		seedClient:        seedMgr.GetClient(),
		userClient:        userMgr.GetClient(),
		recorder:          seedMgr.GetEventRecorder(ControllerName),
		log:               log,
		versions:          versions,
		overwriteRegistry: overwriteRegistry,
	}

	_, err := builder.ControllerManagedBy(userMgr).
		Named(ControllerName).
		WatchesRawSource(source.Kind(
			seedMgr.GetCache(),
			&kubermaticv1.Cluster{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, c *kubermaticv1.Cluster) []reconcile.Request {
				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Name: clusterName,
					},
				}}
			}),
			predicate.TypedByName[*kubermaticv1.Cluster](clusterName),
			predicate.TypedFactory(func(cluster *kubermaticv1.Cluster) bool {
				// Only watch clusters that are in a state where they can be reconciled.
				// Pause-flag is checked by the ReconcileWrapper.
				return cluster.DeletionTimestamp == nil && cluster.Status.ExtendedHealth.ControlPlaneHealthy()
			}),
		)).
		Build(reconciler)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
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
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r.seedClient,
		"",
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
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
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
	key := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.Spec.BackupConfig.BackupStorageLocation.Name}
	if err := r.seedClient.Get(ctx, key, cbsl); err != nil {
		return nil, fmt.Errorf("failed to get ClusterBackupStorageLocation %v: %w", key, err)
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

	key := types.NamespacedName{Name: cbsl.Spec.Credential.Name, Namespace: resources.KubermaticNamespace}
	credentials := &corev1.Secret{}
	if err := r.seedClient.Get(ctx, key, credentials); err != nil {
		return fmt.Errorf("failed to get backup destination credentials secret: %w", err)
	}

	data := resources.NewTemplateDataBuilder().
		WithCluster(cluster).
		WithOverwriteRegistry(r.overwriteRegistry).
		Build()

	nsReconcilers := []reconciling.NamedNamespaceReconcilerFactory{
		userclusterresources.NamespaceReconciler(),
	}
	if err := reconciling.ReconcileNamespaces(ctx, nsReconcilers, "", r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero Namespace: %w", err)
	}

	cmReconcilers := []reconciling.NamedConfigMapReconcilerFactory{
		userclusterresources.CustomizationConfigMapReconciler(data.RewriteImage),
	}
	if err := reconciling.ReconcileConfigMaps(ctx, cmReconcilers, resources.ClusterBackupNamespaceName, r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero ConfigMaps: %w", err)
	}

	saReconcilers := []reconciling.NamedServiceAccountReconcilerFactory{
		userclusterresources.ServiceAccountReconciler(),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, saReconcilers, resources.ClusterBackupNamespaceName, r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero ServiceAccount: %w", err)
	}

	clusterRoleBindingReconciler := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		userclusterresources.ClusterRoleBindingReconciler(),
	}
	if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingReconciler, "", r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero ClusterRoleBinding: %w", err)
	}

	secretReconcilers := []reconciling.NamedSecretReconcilerFactory{
		userclusterresources.SecretReconciler(credentials),
	}
	if err := reconciling.ReconcileSecrets(ctx, secretReconcilers, resources.ClusterBackupNamespaceName, r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero cloud credentials: %w", err)
	}

	deploymentReconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		userclusterresources.DeploymentReconciler(data),
	}
	if err := reconciling.ReconcileDeployments(ctx, deploymentReconcilers, resources.ClusterBackupNamespaceName, r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero Deployment: %w", err)
	}

	dsReconcilers := []reconciling.NamedDaemonSetReconcilerFactory{
		userclusterresources.DaemonSetReconciler(data),
	}
	if err := reconciling.ReconcileDaemonSets(ctx, dsReconcilers, resources.ClusterBackupNamespaceName, r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero node-agent DaemonSet: %w", err)
	}

	clusterBackupCRDs, err := userclusterresources.CRDs()
	if err != nil {
		return fmt.Errorf("failed to load Velero CRDs: %w", err)
	}

	creators := []kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory{}
	for i := range clusterBackupCRDs {
		creators = append(creators, userclusterresources.CRDReconciler(clusterBackupCRDs[i]))
	}
	if err = kkpreconciling.ReconcileCustomResourceDefinitions(ctx, creators, "", r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero CRDs: %w", err)
	}

	bslReconcilers := []kkpreconciling.NamedBackupStorageLocationReconcilerFactory{
		userclusterresources.BSLReconciler(cluster, cbsl),
	}
	if err := kkpreconciling.ReconcileBackupStorageLocations(ctx, bslReconcilers, resources.ClusterBackupNamespaceName, r.userClient, addManagedByLabel); err != nil {
		return fmt.Errorf("failed to reconcile Velero BSL: %w", err)
	}

	return nil
}

func (r *reconciler) removeUserClusterComponents(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if _, ok := cluster.Labels[componentsInstalledLabel]; !ok {
		return nil
	}

	if err := r.removeUserClusterResources(ctx); err != nil {
		return fmt.Errorf("failed to remove user-cluster resources: %w", err)
	}

	if err := r.removeCRDs(ctx); err != nil {
		return fmt.Errorf("failed to remove CRDs: %w", err)
	}

	err := r.patchCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		delete(c.Labels, componentsInstalledLabel)
	})
	if err != nil {
		return fmt.Errorf("failed to unmark cluster: %w", err)
	}

	return nil
}

func (r *reconciler) removeUserClusterResources(ctx context.Context) error {
	// remove resources created in ./resources
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
		if err := r.removeManagedObject(ctx, resource); err != nil {
			return err
		}
	}
	return nil
}

func (r *reconciler) removeManagedObject(ctx context.Context, resource ctrlruntimeclient.Object) error {
	if err := r.userClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(resource), resource); err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil
		}
		return fmt.Errorf("failed to list user-cluster resources: %w", err)
	}

	if isManagedBackupResource(resource) {
		if err := r.userClient.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
			return fmt.Errorf("failed to delete user-cluster resource: %w", err)
		}
	}

	return nil
}

func (r *reconciler) removeCRDs(ctx context.Context) error {
	crdList := &apiextensionsv1.CustomResourceDefinitionList{}

	listOpts := &ctrlruntimeclient.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			clusterBackupComponentLabelKey:             clusterBackupComponentLabelValue,
			appskubermaticv1.ApplicationManagedByLabel: ControllerName,
		}),
	}
	if err := r.userClient.List(ctx, crdList, listOpts); err != nil {
		return fmt.Errorf("failed to list CRDs: %w", err)
	}
	for _, crd := range crdList.Items {
		if err := r.userClient.Delete(ctx, &crd); err != nil && !apierrors.IsNotFound(err) {
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

func (r *reconciler) patchCluster(ctx context.Context, cluster *kubermaticv1.Cluster, patch util.ClusterPatchFunc) error {
	// modify it
	original := cluster.DeepCopy()
	patch(cluster)

	// update the status
	return r.seedClient.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(original))
}
