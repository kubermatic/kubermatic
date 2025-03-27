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
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	rbacusercluster "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/resources"

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

	errorMsgs := []string{}

	// get all Pods running on the cluster
	if err := userClusterClient.List(ctx, pods, &ctrlruntimeclient.ListOptions{Namespace: ""}); err != nil {
		return fmt.Errorf("failed to list Pods in user cluster: %w", err)
	}

	for _, pod := range pods.Items {
		// we only check for a couple of namespaces
		if pod.Namespace != "kube-system" && pod.Namespace != "mla-system" && pod.Namespace != "gatekeeper-system" && pod.Namespace != "kubernetes-dashboard" {
			continue
		}
		// TODO remove this
		// https://github.com/kubermatic/kubermatic/pull/12752#discussion_r1367133164
		if pod.Labels["k8s-app"] == "hubble-generate-certs" {
			continue
		}

		err := validatePodSecurityContext(&pod)
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("Pod %s/%s invalid: %v", pod.Namespace, pod.Name, err))
		}
	}

	if len(errorMsgs) == 0 {
		return nil
	}

	return errors.New(strings.Join(errorMsgs, "\n"))
}

const (
	// legacyRegistryK8SGCR defines the kubernetes specific docker registry at google.
	legacyRegistryK8SGCR = "k8s.gcr.io"
	// legacyRegistryEUGCR defines the docker registry at google EU.
	legacyRegistryEUGCR = "eu.gcr.io"
	// legacyRegistryUSGCR defines the docker registry at google US.
	legacyRegistryUSGCR = "us.gcr.io"
	// legacyRegistryGCR defines the kubernetes docker registry at google.
	legacyRegistryGCR = "gcr.io"
)

func TestUserClusterNoK8sGcrImages(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client) error {
	if !opts.Tests.Has(ctypes.UserClusterSeccompTests) {
		log.Info("User cluster Seccomp tests disabled, skipping.")
		return nil
	}

	pods := &corev1.PodList{}

	errorMsgs := []string{}

	// get all Pods running on the cluster
	if err := userClusterClient.List(ctx, pods, &ctrlruntimeclient.ListOptions{Namespace: ""}); err != nil {
		return fmt.Errorf("failed to list Pods in user cluster: %w", err)
	}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if strings.HasPrefix(container.Image, legacyRegistryK8SGCR) {
				errorMsgs = append(
					errorMsgs,
					fmt.Sprintf("Container %s in Pod %s/%s has image from k8s.gcr.io and should be using registry.k8s.io instead", container.Name, pod.Namespace, pod.Name),
				)
			}
			if strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", legacyRegistryGCR)) ||
				strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", legacyRegistryEUGCR)) ||
				strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", legacyRegistryUSGCR)) {
				errorMsgs = append(
					errorMsgs,
					fmt.Sprintf("Container %s in Pod %s/%s has image from gcr.io/k8s-* and should be using registry.k8s.io instead", container.Name, pod.Namespace, pod.Name),
				)
			}
		}

		for _, initContainer := range pod.Spec.InitContainers {
			if strings.HasPrefix(initContainer.Image, legacyRegistryK8SGCR) {
				errorMsgs = append(
					errorMsgs,
					fmt.Sprintf("InitContainer %s in Pod %s/%s has image from k8s.gcr.io and should be using registry.k8s.io instead", initContainer.Name, pod.Namespace, pod.Name),
				)
			}
			if strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", legacyRegistryGCR)) ||
				strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", legacyRegistryEUGCR)) ||
				strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", legacyRegistryUSGCR)) {
				errorMsgs = append(
					errorMsgs,
					fmt.Sprintf("Container %s in Pod %s/%s has image from gcr.io/k8s-* and should be using registry.k8s.io instead", initContainer.Name, pod.Namespace, pod.Name),
				)
			}
		}
	}

	if len(errorMsgs) == 0 {
		return nil
	}

	return errors.New(strings.Join(errorMsgs, "\n"))
}

func rbacResourceNames() []string {
	return []string{rbacusercluster.ResourceOwnerName, rbacusercluster.ResourceEditorName, rbacusercluster.ResourceViewerName}
}
