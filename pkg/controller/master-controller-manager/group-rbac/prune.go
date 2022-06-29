/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grouprbac

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

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
