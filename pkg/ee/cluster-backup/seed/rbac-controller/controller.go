//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2024 Kubermatic GmbH

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

package rbaccontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/usercluster"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "cluster-backup-rbac-controller"
	finalizer      = "kubermatic.k8c.io/cleanup-cluster-backup-seed-rbac"
	kkpNamespace   = resources.KubermaticNamespace
)

type reconciler struct {
	seedClient ctrlruntimeclient.Client
	log        *zap.SugaredLogger
}

func Add(
	seedMgr manager.Manager,
	log *zap.SugaredLogger,
) error {
	reconciler := &reconciler{
		seedClient: seedMgr.GetClient(),
		log:        log,
	}

	_, err := builder.ControllerManagedBy(seedMgr).
		Named(ControllerName).
		For(&kubermaticv1.Cluster{}).
		Build(reconciler)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Cannot do anything with unfinished clusters.
	if cluster.Status.NamespaceName == "" {
		return reconcile.Result{}, nil
	}

	// Cleanup when the cluster is going away or the feature was disabled again.
	if cluster.DeletionTimestamp != nil || !cluster.Spec.IsClusterBackupEnabled() {
		return reconcile.Result{}, r.cleanup(ctx, cluster)
	}

	err := r.reconcile(ctx, cluster)

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if err := kuberneteshelper.TryAddFinalizer(ctx, r.seedClient, cluster, finalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	cbsl := &kubermaticv1.ClusterBackupStorageLocation{}
	key := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: cluster.Spec.BackupConfig.BackupStorageLocation.Name}
	if err := r.seedClient.Get(ctx, key, cbsl); err != nil {
		return fmt.Errorf("failed to get ClusterBackupStorageLocation %v: %w", key, err)
	}

	roleReconcilers := []reconciling.NamedRoleReconcilerFactory{
		roleReconciler(cluster, cbsl),
	}
	if err := reconciling.ReconcileRoles(ctx, roleReconcilers, kkpNamespace, r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile Roles: %w", err)
	}

	roleBindingReconcilers := []reconciling.NamedRoleBindingReconcilerFactory{
		roleBindingReconciler(cluster),
	}
	if err := reconciling.ReconcileRoleBindings(ctx, roleBindingReconcilers, kkpNamespace, r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile RoleBindings: %w", err)
	}

	return nil
}

func (r *reconciler) cleanup(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if !kuberneteshelper.HasFinalizer(cluster, finalizer) {
		return nil
	}

	if err := removeResource(ctx, r.seedClient, &rbacv1.RoleBinding{}, types.NamespacedName{Name: roleBindingName(cluster), Namespace: kkpNamespace}); err != nil {
		return fmt.Errorf("failed to delete old RoleBinding: %w", err)
	}

	if err := removeResource(ctx, r.seedClient, &rbacv1.Role{}, types.NamespacedName{Name: roleName(cluster), Namespace: kkpNamespace}); err != nil {
		return fmt.Errorf("failed to delete old Role: %w", err)
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.seedClient, cluster, finalizer)
}

func removeResource(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, key types.NamespacedName) error {
	// Get the resource first to make use of the cache and skip unnecessary delete calls.
	if err := client.Get(ctx, key, obj); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return err
	} else if err != nil {
		return nil
	}

	return client.Delete(ctx, obj)
}

func roleName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("kubermatic:manage-cluster-backups:%s", cluster.Name)
}

func roleBindingName(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("kubermatic:manage-cluster-backups:%s", cluster.Name)
}

func roleReconciler(cluster *kubermaticv1.Cluster, cbsl *kubermaticv1.ClusterBackupStorageLocation) reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return roleName(cluster), func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups:     []string{kubermaticv1.GroupName},
					Resources:     []string{"clusterbackupstoragelocations"},
					ResourceNames: []string{cbsl.Name},
					Verbs:         []string{"get", "list", "watch"},
				},
			}

			// This might be overly cautious. Most other places in the codebase require Credential
			// not to be nil, but here we want to be more cautious.
			if cred := cbsl.Spec.Credential; cred != nil {
				r.Rules = append(r.Rules, rbacv1.PolicyRule{
					APIGroups:     []string{""},
					Resources:     []string{"secrets"},
					ResourceNames: []string{cred.Name},
					Verbs:         []string{"get", "list", "watch"},
				})
			}

			return r, nil
		}
	}
}

func roleBindingReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return roleBindingName(cluster), func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     roleName(cluster),
			}

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      usercluster.ServiceAccountName,
					Namespace: cluster.Status.NamespaceName,
				},
			}

			return rb, nil
		}
	}
}
