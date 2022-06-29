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
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

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

	seedClientGetter provider.SeedClientGetter
	seedsGetter      provider.SeedsGetter
	log              *zap.SugaredLogger
	recorder         record.EventRecorder
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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

	// validate that GroupProjectBinding references an existing project and set an owner reference

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
	}); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to set project owner reference: %w", err)
	}

	log := r.log.With("GroupProjectBinding", binding.Name)

	// reconcile master cluster first
	if err := r.reconcile(ctx, r.Client, log, binding); err != nil {
		r.recorder.Event(binding, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		r.log.Errorw("Reconciling master failed", zap.Error(err))
	}

	seeds, err := r.seedsGetter()
	if err != nil {
		return reconcile.Result{}, err
	}

	for _, seed := range seeds {
		// don't try to reconcile Seed that has an invalid kubeconfig
		if seed.Status.HasConditionValue(kubermaticv1.SeedConditionKubeconfigValid, corev1.ConditionTrue) {
			seedClient, err := r.seedClientGetter(seed)
			if err != nil {
				r.log.Warnw("Getting seed client failed", "seed", seed.Name, zap.Error(err))
				continue
			}

			log := log.With("seed", seed.Name)

			if err := r.reconcile(ctx, seedClient, log, binding); err != nil {
				r.recorder.Event(binding, corev1.EventTypeWarning, "ReconcilingError", err.Error())
				r.log.Warnw("Reconciling seed failed", "seed", seed.Name, zap.Error(err))
			}
		} else {
			r.log.Debugw("Skipped reconciling non-ready seed", "seed", seed.Name)
		}
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, client ctrlruntimeclient.Client, log *zap.SugaredLogger, binding *kubermaticv1.GroupProjectBinding) error {
	clusterRoles, err := getTargetClusterRoles(ctx, client, binding)
	if err != nil {
		return fmt.Errorf("failed to get target ClusterRoles: %w", err)
	}

	log.Debugw("Found ClusterRoles matching role label", "count", len(clusterRoles))

	if err := pruneClusterRoleBindings(ctx, client, log, binding, clusterRoles); err != nil {
		return fmt.Errorf("failed to prune ClusterRoleBindings: %w", err)
	}

	clusterRoleBindingCreators := []reconciling.NamedClusterRoleBindingCreatorGetter{}

	for _, clusterRole := range clusterRoles {
		clusterRoleBindingCreators = append(clusterRoleBindingCreators, clusterRoleBindingCreator(*binding, clusterRole))
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingCreators, "", client); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBindings: %w", err)
	}

	// reconcile RoleBindings next. Roles are spread out across several namespaces, so we need to reconcile by namespace.

	rolesMap, err := getTargetRoles(ctx, client, binding)
	if err != nil {
		return fmt.Errorf("failed to get target Roles: %w", err)
	}

	log.Debugw("Found namespaces with Roles matching conditions", "GroupProjectBinding", binding.Name, "count", len(rolesMap))

	if err := pruneRoleBindings(ctx, client, log, binding); err != nil {
		return fmt.Errorf("failed to prune Roles: %w", err)
	}

	for ns, roles := range rolesMap {
		roleBindingCreators := []reconciling.NamedRoleBindingCreatorGetter{}
		for _, role := range roles {
			roleBindingCreators = append(roleBindingCreators, roleBindingCreator(*binding, role))
		}

		if err := reconciling.ReconcileRoleBindings(ctx, roleBindingCreators, ns, client); err != nil {
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
