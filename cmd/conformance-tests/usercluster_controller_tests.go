/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	rbacusercluster "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/rbac"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *testRunner) testUserclusterControllerRBAC(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, userClusterClient, seedClusterClient ctrlruntimeclient.Client) error {
	log.Info("Testing user cluster RBAC controller")
	clusterNamespace := fmt.Sprintf("cluster-%s", cluster.Name)

	// check if usercluster-controller was deployed on seed cluster
	deployment := &appsv1.Deployment{}
	if err := seedClusterClient.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: resources.UserClusterControllerDeploymentName}, deployment); err != nil {
		return fmt.Errorf("failed to get Deployment: %s, error: %v", resources.UserClusterControllerDeploymentName, err)
	}

	if deployment.Status.AvailableReplicas == 0 {
		return fmt.Errorf("%s deployment is not ready", resources.UserClusterControllerDeploymentName)
	}

	// check user cluster resources: ClusterRoles and ClusterRoleBindings
	for _, resourceName := range rbacResourceNames() {
		log.Info("Getting a Cluster Role: ", resourceName)
		clusterRole := &rbacv1.ClusterRole{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Name: resourceName}, clusterRole); err != nil {
			return fmt.Errorf("failed to get Cluster Role: %s, error: %v", clusterRole, err)
		}

		defaultClusterRole, err := rbacusercluster.GenerateRBACClusterRole(resourceName)
		if err != nil {
			return fmt.Errorf("failed to generate default Cluster Role: %s, error: %v", resourceName, err)
		}

		if !equality.Semantic.DeepEqual(clusterRole.Rules, defaultClusterRole.Rules) {
			return fmt.Errorf("incorrect Cluster Role Rules were returned, got: %v, want: %v", clusterRole.Rules, defaultClusterRole.Rules)
		}

		log.Info("Getting a Cluster Role Binding: ", resourceName)
		clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Name: resourceName}, clusterRoleBinding); err != nil {
			return fmt.Errorf("failed to get Cluster Role Binding: %s, error: %v", resourceName, err)
		}

		defaultClusterRoleBinding, err := rbacusercluster.GenerateRBACClusterRoleBinding(resourceName)
		if err != nil {
			return fmt.Errorf("failed to generate default Cluster Role Binding: %s, error: %v", resourceName, err)
		}

		if !equality.Semantic.DeepEqual(clusterRoleBinding.RoleRef, defaultClusterRoleBinding.RoleRef) {
			return fmt.Errorf("incorrect Cluster Role Binding RoleRef were returned, got: %v, want: %v", clusterRoleBinding.RoleRef, defaultClusterRoleBinding.RoleRef)
		}
		if !equality.Semantic.DeepEqual(clusterRoleBinding.Subjects, defaultClusterRoleBinding.Subjects) {
			return fmt.Errorf("incorrect Cluster Role Binding Subjects were returned, got: %v, want: %v", clusterRoleBinding.Subjects, defaultClusterRoleBinding.Subjects)
		}
	}

	return nil
}

func (r *testRunner) testUserClusterSeccompProfiles(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client) error {
	pods := &corev1.PodList{}

	// get all Pods running on the cluster
	if err := userClusterClient.List(ctx, pods, &ctrlruntimeclient.ListOptions{Namespace: ""}); err != nil {
		return fmt.Errorf("failed to list Pods in user cluster: %v", err)
	}

	for _, pod := range pods.Items {
		// no security context means no seccomp profile
		if pod.Spec.SecurityContext == nil {
			return fmt.Errorf("expected security context on Pod %s/%s, got none", pod.Namespace, pod.Name)
		}

		// no seccomp profile means no profile is applied to the containers
		if pod.Spec.SecurityContext.SeccompProfile == nil {
			return fmt.Errorf("expected seccomp profile on Pod %s/%s, got none", pod.Namespace, pod.Name)
		}

		// the 'unconfined' profile disables any seccomp filtering
		if pod.Spec.SecurityContext.SeccompProfile.Type == corev1.SeccompProfileTypeUnconfined {
			return fmt.Errorf(
				"seccomp profile of Pod %s/%s is '%s', should be '%s' or '%s'", pod.Namespace, pod.Name,
				corev1.SeccompProfileTypeUnconfined, corev1.SeccompProfileTypeRuntimeDefault, corev1.SeccompProfileTypeLocalhost,
			)
		}
	}

	return nil
}

func rbacResourceNames() []string {
	return []string{rbacusercluster.ResourceOwnerName, rbacusercluster.ResourceEditorName, rbacusercluster.ResourceViewerName}
}
