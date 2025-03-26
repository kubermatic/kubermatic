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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func pruneClusterRoleBindings(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, binding *kubermaticv1.GroupProjectBinding, clusterRoles []rbacv1.ClusterRole) error {
	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}

	if err := client.List(ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels{
		kubermaticv1.AuthZGroupProjectBindingLabel: binding.Name,
	}); err != nil {
		return err
	}

	pruneList := []rbacv1.ClusterRoleBinding{}

	for _, clusterRoleBinding := range clusterRoleBindingList.Items {
		// looks like GroupProjectBinding.Spec.Role has been changed, so we need to prune this ClusterRoleBinding.
		if label, ok := clusterRoleBinding.Labels[kubermaticv1.AuthZRoleLabel]; !ok || label != binding.Spec.Role {
			pruneList = append(pruneList, clusterRoleBinding)
			continue
		}

		// make sure a ClusterRole with that name still exists; if not, the resource referenced by the ClusterRole
		// was likely deleted and we should clean up this ClusterRoleBinding to prevent namesquatting in the future
		// (create a `Cluster` resource named "xyz", delete the cluster, wait for someone else to create a cluster
		// under the same name, gain access to it via the still existing ClusterRoleBinding).
		roleInScope := false
		for _, role := range clusterRoles {
			if clusterRoleBinding.RoleRef.Kind == "ClusterRole" && clusterRoleBinding.RoleRef.Name == role.Name {
				roleInScope = true
			}
		}

		if !roleInScope {
			pruneList = append(pruneList, clusterRoleBinding)
		}
	}

	for _, prunedBinding := range pruneList {
		log.Debugw("found ClusterRoleBinding to prune", "GroupProjectBinding", binding.Name, "ClusterRoleBinding", prunedBinding.Name)
		if err := client.Delete(ctx, &prunedBinding); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func pruneRoleBindings(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, binding *kubermaticv1.GroupProjectBinding) error {
	roleBindingList := &rbacv1.RoleBindingList{}

	if err := client.List(ctx, roleBindingList, ctrlruntimeclient.MatchingLabels{
		kubermaticv1.AuthZGroupProjectBindingLabel: binding.Name,
	}); err != nil {
		return err
	}

	pruneList := []rbacv1.RoleBinding{}

	for _, roleBinding := range roleBindingList.Items {
		// looks like GroupProjectBinding.Spec.Role has been changed, so we need to prune this RoleBinding.
		if label, ok := roleBinding.Labels[kubermaticv1.AuthZRoleLabel]; !ok || label != binding.Spec.Role {
			pruneList = append(pruneList, roleBinding)
			continue
		}
	}

	for _, prunedBinding := range pruneList {
		log.Debugw("found RoleBinding to prune", "GroupProjectBinding", binding.Name, "RoleBinding", prunedBinding.Name)
		if err := client.Delete(ctx, &prunedBinding); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}
