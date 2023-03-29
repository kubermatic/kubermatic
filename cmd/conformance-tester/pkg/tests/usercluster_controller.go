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

package tests

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	ctypes "k8c.io/kubermatic/v3/cmd/conformance-tester/pkg/types"
	rbacusercluster "k8c.io/kubermatic/v3/pkg/controller/user-cluster-controller-manager/rbac"
	"k8c.io/kubermatic/v3/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUserclusterControllerRBAC(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster, userClusterClient, seedClusterClient ctrlruntimeclient.Client) error {
	if !opts.Tests.Has(ctypes.UserClusterRBACTests) {
		log.Info("User cluster RBAC tests disabled, skipping.")
		return nil
	}

	log.Info("Testing user cluster RBAC controller")
	clusterNamespace := fmt.Sprintf("cluster-%s", cluster.Name)

	// check if usercluster-controller was deployed on seed cluster
	deployment := &appsv1.Deployment{}
	if err := seedClusterClient.Get(ctx, types.NamespacedName{Namespace: clusterNamespace, Name: resources.UserClusterControllerDeploymentName}, deployment); err != nil {
		return fmt.Errorf("failed to get Deployment %s: %w", resources.UserClusterControllerDeploymentName, err)
	}

	if deployment.Status.AvailableReplicas == 0 {
		return fmt.Errorf("%s deployment is not ready", resources.UserClusterControllerDeploymentName)
	}

	// check user cluster resources: ClusterRoles and ClusterRoleBindings
	for _, resourceName := range rbacResourceNames() {
		log.Infof("Getting ClusterRole: %s", resourceName)
		clusterRole := &rbacv1.ClusterRole{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Name: resourceName}, clusterRole); err != nil {
			return fmt.Errorf("failed to get ClusterRole %s: %w", clusterRole, err)
		}

		defaultClusterRole, err := rbacusercluster.CreateClusterRole(resourceName, &rbacv1.ClusterRole{})
		if err != nil {
			return fmt.Errorf("failed to generate default ClusterRole %s: %w", resourceName, err)
		}

		if !equality.Semantic.DeepEqual(clusterRole.Rules, defaultClusterRole.Rules) {
			return fmt.Errorf("incorrect ClusterRole Rules were returned, got %v, want %v", clusterRole.Rules, defaultClusterRole.Rules)
		}

		log.Infof("Getting ClusterRoleBinding %s", resourceName)
		clusterRoleBinding := &rbacv1.ClusterRoleBinding{}
		if err := userClusterClient.Get(ctx, types.NamespacedName{Name: resourceName}, clusterRoleBinding); err != nil {
			return fmt.Errorf("failed to get ClusterRoleBinding %s: %w", resourceName, err)
		}

		defaultClusterRoleBinding, err := rbacusercluster.CreateClusterRoleBinding(resourceName, &rbacv1.ClusterRoleBinding{})
		if err != nil {
			return fmt.Errorf("failed to generate default ClusterRoleBinding %s: %w", resourceName, err)
		}

		if !equality.Semantic.DeepEqual(clusterRoleBinding.RoleRef, defaultClusterRoleBinding.RoleRef) {
			return fmt.Errorf("incorrect ClusterRoleBinding RoleRef were returned, got %v, want %v", clusterRoleBinding.RoleRef, defaultClusterRoleBinding.RoleRef)
		}
		if !equality.Semantic.DeepEqual(clusterRoleBinding.Subjects, defaultClusterRoleBinding.Subjects) {
			return fmt.Errorf("incorrect ClusterRoleBinding Subjects were returned, got %v, want %v", clusterRoleBinding.Subjects, defaultClusterRoleBinding.Subjects)
		}
	}

	return nil
}

func TestUserClusterSeccompProfiles(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client) error {
	if !opts.Tests.Has(ctypes.UserClusterSeccompTests) {
		log.Info("User cluster Seccomp tests disabled, skipping.")
		return nil
	}

	pods := &corev1.PodList{}

	errors := []string{}

	// get all Pods running on the cluster
	if err := userClusterClient.List(ctx, pods, &ctrlruntimeclient.ListOptions{Namespace: ""}); err != nil {
		return fmt.Errorf("failed to list Pods in user cluster: %w", err)
	}

	for _, pod := range pods.Items {
		// we only check for a couple of namespaces
		if pod.Namespace != "kube-system" && pod.Namespace != "mla-system" && pod.Namespace != "gatekeeper-system" && pod.Namespace != "kubernetes-dashboard" {
			continue
		}

		var privilegedContainers int
		for _, container := range pod.Spec.Containers {
			if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
				privilegedContainers++
			}
		}

		// all containers in the Pod are running as privileged, we can skip the Pod; privileged mode disables any seccomp profile
		if len(pod.Spec.Containers) == privilegedContainers {
			continue
		}

		// no security context means no seccomp profile
		if pod.Spec.SecurityContext == nil {
			errors = append(
				errors,
				fmt.Sprintf("expected security context on Pod %s/%s, got none", pod.Namespace, pod.Name),
			)
			continue
		}

		// no seccomp profile means no profile is applied to the containers
		if pod.Spec.SecurityContext.SeccompProfile == nil {
			errors = append(
				errors,
				fmt.Sprintf("expected seccomp profile on Pod %s/%s, got none", pod.Namespace, pod.Name),
			)
			continue
		}

		// the 'unconfined' profile disables any seccomp filtering
		if pod.Spec.SecurityContext.SeccompProfile.Type == corev1.SeccompProfileTypeUnconfined {
			errors = append(
				errors,
				fmt.Sprintf(
					"seccomp profile of Pod %s/%s is '%s', should be '%s' or '%s'", pod.Namespace, pod.Name,
					corev1.SeccompProfileTypeUnconfined, corev1.SeccompProfileTypeRuntimeDefault, corev1.SeccompProfileTypeLocalhost,
				),
			)
			continue
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf(strings.Join(errors, "\n"))
}

func TestUserClusterNoK8sGcrImages(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client) error {
	if !opts.Tests.Has(ctypes.UserClusterSeccompTests) {
		log.Info("User cluster Seccomp tests disabled, skipping.")
		return nil
	}

	pods := &corev1.PodList{}

	errors := []string{}

	// get all Pods running on the cluster
	if err := userClusterClient.List(ctx, pods, &ctrlruntimeclient.ListOptions{Namespace: ""}); err != nil {
		return fmt.Errorf("failed to list Pods in user cluster: %w", err)
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if strings.HasPrefix(container.Image, resources.RegistryK8SGCR) {
				errors = append(
					errors,
					fmt.Sprintf("Container %s in Pod %s/%s has image from k8s.gcr.io and should be using registry.k8s.io instead", container.Name, pod.Namespace, pod.Name),
				)
			}
			if strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", resources.RegistryGCR)) ||
				strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", resources.RegistryEUGCR)) ||
				strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", resources.RegistryUSGCR)) {
				errors = append(
					errors,
					fmt.Sprintf("Container %s in Pod %s/%s has image from gcr.io/k8s-* and should be using registry.k8s.io instead", container.Name, pod.Namespace, pod.Name),
				)
			}
		}

		for _, initContainer := range pod.Spec.InitContainers {
			if strings.HasPrefix(initContainer.Image, resources.RegistryK8SGCR) {
				errors = append(
					errors,
					fmt.Sprintf("InitContainer %s in Pod %s/%s has image from k8s.gcr.io and should be using registry.k8s.io instead", initContainer.Name, pod.Namespace, pod.Name),
				)
			}
			if strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", resources.RegistryGCR)) ||
				strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", resources.RegistryEUGCR)) ||
				strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", resources.RegistryUSGCR)) {
				errors = append(
					errors,
					fmt.Sprintf("Container %s in Pod %s/%s has image from gcr.io/k8s-* and should be using registry.k8s.io instead", initContainer.Name, pod.Namespace, pod.Name),
				)
			}
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf(strings.Join(errors, "\n"))
}

func rbacResourceNames() []string {
	return []string{rbacusercluster.ResourceOwnerName, rbacusercluster.ResourceEditorName, rbacusercluster.ResourceViewerName}
}
