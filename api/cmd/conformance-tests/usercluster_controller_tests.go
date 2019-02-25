package main

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac-user-cluster"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (r *testRunner) testUserclusterControllerRBAC(log *logrus.Entry, cluster *kubermaticv1.Cluster, kubeClient, seedKubeClient kubernetes.Interface) error {
	log.Info("Testing user cluster RBAC controller")

	clusterNamespace := fmt.Sprintf("cluster-%s", cluster.Name)

	// check if usercluster-controller was deployed on seed cluster
	deployment, err := seedKubeClient.ExtensionsV1beta1().Deployments(clusterNamespace).Get(resources.UserClusterControllerDeploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get Deployment: %s, error: %v", resources.UserClusterControllerDeploymentName, err)
	}
	if deployment.Status.AvailableReplicas == 0 {
		return fmt.Errorf("%s deployment is not ready", resources.UserClusterControllerDeploymentName)
	}

	// check user cluster resources: ClusterRoles and ClusterRoleBindings
	for _, resourceName := range rbacResourceNames() {
		log.Info("Getting a Cluster Role: ", resourceName)
		clusterRole, err := kubeClient.RbacV1().ClusterRoles().Get(resourceName, metav1.GetOptions{})
		if err != nil {
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
		clusterRoleBinding, err := kubeClient.RbacV1().ClusterRoleBindings().Get(resourceName, metav1.GetOptions{})
		if err != nil {
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

func rbacResourceNames() []string {
	return []string{rbacusercluster.ResourceOwnerName, rbacusercluster.ResourceEditorName, rbacusercluster.ResourceViewerName}
}
