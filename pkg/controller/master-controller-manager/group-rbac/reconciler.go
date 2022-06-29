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

	// reconcile master cluster first
	if err := r.reconcile(ctx, r.Client, binding); err != nil {
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
			if err := r.reconcile(ctx, seedClient, binding); err != nil {
				r.recorder.Event(binding, corev1.EventTypeWarning, "ReconcilingError", err.Error())
				r.log.Warnw("Reconciling seed failed", "seed", seed.Name, zap.Error(err))
			}
		} else {
			r.log.Debugw("Skipped reconciling non-ready seed", "seed", seed.Name)
		}
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.GroupProjectBinding) error {
	clusterRoles, err := getTargetClusterRoles(ctx, client, binding)
	if err != nil {
		return err
	}

	r.log.Debugw("found ClusterRoles matching role label", "count", len(clusterRoles))

	clusterRoleBindingCreators := []reconciling.NamedClusterRoleBindingCreatorGetter{}

	for _, clusterRole := range clusterRoles {
		clusterRoleBindingCreators = append(clusterRoleBindingCreators, clusterRoleBindingCreator(binding, &clusterRole))
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingCreators, "", client); err != nil {
		return err
	}

	return nil
}

// getTargetClusterRoles returns a list of ClusterRoles that match the authz.kubermatic.io/role label for the specific role and project
// that the GroupProjectBinding was created for.
func getTargetClusterRoles(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.GroupProjectBinding) ([]rbacv1.ClusterRole, error) {
	var clusterRoles rbacv1.ClusterRoleList

	if err := client.List(ctx, &clusterRoles, ctrlruntimeclient.MatchingLabels{
		kubermaticv1.AuthZRoleLabel: fmt.Sprintf("%s-%s", binding.Spec.Role, binding.Spec.ProjectID),
	}); err != nil {
		return nil, err
	}

	return clusterRoles.Items, nil
}
