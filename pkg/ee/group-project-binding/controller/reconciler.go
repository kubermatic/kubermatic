//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

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

package controller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	log      *zap.SugaredLogger
	recorder record.EventRecorder

	setOwnerRef bool
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	binding := &kubermaticv1.GroupProjectBinding{}
	if err := r.Get(ctx, request.NamespacedName, binding); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get GroupProjectBinding: %w", err)
	}

	if binding.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	if r.setOwnerRef {
		// validate that GroupProjectBinding references an existing project and set an owner reference and project-id label

		project := &kubermaticv1.Project{}
		if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: binding.Spec.ProjectID}, project); err != nil {
			if apierrors.IsNotFound(err) {
				r.recorder.Event(binding, corev1.EventTypeWarning, "ProjectNotFound", err.Error())
			}
			return reconcile.Result{}, err
		}

		if err := updateGroupProjectBinding(ctx, r.Client, binding, func(binding *kubermaticv1.GroupProjectBinding) {
			kuberneteshelper.EnsureOwnerReference(binding, metav1.OwnerReference{
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
				Kind:       kubermaticv1.ProjectKindName,
				Name:       project.Name,
				UID:        project.UID,
			})
			kuberneteshelper.EnsureLabels(binding, map[string]string{
				kubermaticv1.ProjectIDLabelKey: project.Name,
			})
		}); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to set project owner reference: %w", err)
		}
	}

	if err := r.reconcile(ctx, r.Client, log, binding); err != nil {
		r.recorder.Event(binding, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, binding *kubermaticv1.GroupProjectBinding) error {
	clusterRoles, err := getTargetClusterRoles(ctx, client, binding)
	if err != nil {
		return fmt.Errorf("failed to get target ClusterRoles: %w", err)
	}

	log.Debugw("found ClusterRoles matching role label", "count", len(clusterRoles))

	if err := pruneClusterRoleBindings(ctx, client, log, binding, clusterRoles); err != nil {
		return fmt.Errorf("failed to prune ClusterRoleBindings: %w", err)
	}

	clusterRoleBindingReconcilers := []reconciling.NamedClusterRoleBindingReconcilerFactory{}

	for _, clusterRole := range clusterRoles {
		clusterRoleBindingReconcilers = append(clusterRoleBindingReconcilers, clusterRoleBindingReconciler(*binding, clusterRole))
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingReconcilers, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	// reconcile RoleBindings next. Roles are spread out across several namespaces, so we need to reconcile by namespace.

	rolesMap, err := getTargetRoles(ctx, client, binding)
	if err != nil {
		return fmt.Errorf("failed to get target Roles: %w", err)
	}

	log.Debugw("found namespaces with Roles matching conditions", "GroupProjectBinding", binding.Name, "count", len(rolesMap))

	if err := pruneRoleBindings(ctx, client, log, binding); err != nil {
		return fmt.Errorf("failed to prune Roles: %w", err)
	}

	for ns, roles := range rolesMap {
		roleBindingReconcilers := []reconciling.NamedRoleBindingReconcilerFactory{}
		for _, role := range roles {
			roleBindingReconcilers = append(roleBindingReconcilers, roleBindingReconciler(*binding, role))
		}

		if err := reconciling.ReconcileRoleBindings(ctx, roleBindingReconcilers, ns, client); err != nil {
			return fmt.Errorf("failed to reconcile RoleBindings for namespace '%s': %w", ns, err)
		}
	}

	return nil
}

// getTargetClusterRoles returns a list of ClusterRoles that match the authz.kubermatic.io/role label for the specific role and project
// that the GroupProjectBinding was created for.
func getTargetClusterRoles(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.GroupProjectBinding) ([]rbacv1.ClusterRole, error) {
	var (
		clusterRoles []rbacv1.ClusterRole
	)

	clusterRoleList := &rbacv1.ClusterRoleList{}

	// find those ClusterRoles created for a specific role in a specific project.
	if err := client.List(ctx, clusterRoleList, ctrlruntimeclient.MatchingLabels{
		kubermaticv1.AuthZRoleLabel: fmt.Sprintf("%s-%s", binding.Spec.Role, binding.Spec.ProjectID),
	}); err != nil {
		return nil, err
	}

	clusterRoles = append(clusterRoles, clusterRoleList.Items...)

	// find those ClusterRoles created for a specific role globally.
	if err := client.List(ctx, clusterRoleList, ctrlruntimeclient.MatchingLabels{
		kubermaticv1.AuthZRoleLabel: binding.Spec.Role,
	}); err != nil {
		return nil, err
	}

	clusterRoles = append(clusterRoles, clusterRoleList.Items...)

	return clusterRoles, nil
}

func getTargetRoles(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.GroupProjectBinding) (map[string][]rbacv1.Role, error) {
	roleMap := make(map[string][]rbacv1.Role)
	roleList := &rbacv1.RoleList{}

	// find those Roles created for a specific role in a specific project.
	if err := client.List(ctx, roleList, ctrlruntimeclient.MatchingLabels{
		kubermaticv1.AuthZRoleLabel: fmt.Sprintf("%s-%s", binding.Spec.Role, binding.Spec.ProjectID),
	}); err != nil {
		return nil, err
	}

	for _, role := range roleList.Items {
		if roleMap[role.Namespace] == nil {
			roleMap[role.Namespace] = []rbacv1.Role{}
		}
		roleMap[role.Namespace] = append(roleMap[role.Namespace], role)
	}

	// find those Roles created for a specific role globally.
	if err := client.List(ctx, roleList, ctrlruntimeclient.MatchingLabels{
		kubermaticv1.AuthZRoleLabel: binding.Spec.Role,
	}); err != nil {
		return nil, err
	}

	for _, role := range roleList.Items {
		if roleMap[role.Namespace] == nil {
			roleMap[role.Namespace] = []rbacv1.Role{}
		}
		roleMap[role.Namespace] = append(roleMap[role.Namespace], role)
	}

	return roleMap, nil
}
