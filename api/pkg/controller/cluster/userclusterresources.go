package cluster

import (
	"fmt"

	"github.com/go-test/deep"
	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/controllermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/ipamcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/kubestatemetrics"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/userclustercontrollermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	admissionv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (cc *Controller) userClusterEnsureInitializerConfiguration(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	creators := []resources.InitializerConfigurationCreator{
		ipamcontroller.MachineIPAMInitializerConfiguration,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *admissionv1alpha1.InitializerConfiguration
		initializerConfiguration, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build InitializerConfiguration: %v", err)
		}

		if existing, err = client.AdmissionregistrationV1alpha1().InitializerConfigurations().Get(initializerConfiguration.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = client.AdmissionregistrationV1alpha1().InitializerConfigurations().Create(initializerConfiguration); err != nil {
				return fmt.Errorf("failed to create InitializerConfiguration %s %v", initializerConfiguration.Name, err)
			}
			glog.V(4).Infof("Created InitializerConfiguration %s inside user-cluster %s", initializerConfiguration.Name, c.Name)
			continue
		}

		initializerConfiguration, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build InitializerConfiguration: %v", err)
		}

		if diff := deep.Equal(initializerConfiguration, existing); diff == nil {
			continue
		}

		if _, err = client.AdmissionregistrationV1alpha1().InitializerConfigurations().Update(initializerConfiguration); err != nil {
			return fmt.Errorf("failed to update InitializerConfiguration %s: %v", initializerConfiguration.Name, err)
		}
		glog.V(4).Infof("Updated InitializerConfiguration %s inside user-cluster %s", initializerConfiguration.Name, c.Name)
	}

	return nil
}

func (cc *Controller) userClusterEnsureRoles(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	creators := []resources.RoleCreator{
		machinecontroller.Role,
		machinecontroller.KubeSystemRole,
		machinecontroller.KubePublicRole,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *rbacv1.Role
		role, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build Role: %v", err)
		}

		if existing, err = client.RbacV1().Roles(role.Namespace).Get(role.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = client.RbacV1().Roles(role.Namespace).Create(role); err != nil {
				return fmt.Errorf("failed to create Role %s in namespace %s: %v", role.Name, role.Namespace, err)
			}
			glog.V(4).Infof("Created Role %s inside user-cluster %s", role.Name, c.Name)
			continue
		}

		role, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Role: %v", err)
		}

		if equality.Semantic.DeepEqual(role, existing) {
			continue
		}

		if _, err = client.RbacV1().Roles(role.Namespace).Update(role); err != nil {
			return fmt.Errorf("failed to update Role %s in namespace %s: %v", role.Name, role.Namespace, err)
		}
		glog.V(4).Infof("Updated Role %s inside user-cluster %s", role.Name, c.Name)
	}

	return nil
}

func (cc *Controller) userClusterEnsureRoleBindings(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	creators := []resources.RoleBindingCreator{
		machinecontroller.DefaultRoleBinding,
		machinecontroller.KubeSystemRoleBinding,
		machinecontroller.KubePublicRoleBinding,
		controllermanager.SystemBootstrapSignerRoleBinding,
		controllermanager.PublicBootstrapSignerRoleBinding,
	}

	for _, create := range creators {
		var existing *rbacv1.RoleBinding
		rb, err := create(nil, nil)
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding: %v", err)
		}

		if existing, err = client.RbacV1().RoleBindings(rb.Namespace).Get(rb.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = client.RbacV1().RoleBindings(rb.Namespace).Create(rb); err != nil {
				return fmt.Errorf("failed to create RoleBinding %s in namespace %s: %v", rb.Name, rb.Namespace, err)
			}
			glog.V(4).Infof("Created RoleBinding %s inside user-cluster %s", rb.Name, c.Name)
			continue
		}

		rb, err = create(nil, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding: %v", err)
		}

		if equality.Semantic.DeepEqual(rb, existing) {
			continue
		}

		if _, err = client.RbacV1().RoleBindings(rb.Namespace).Update(rb); err != nil {
			return fmt.Errorf("failed to update RoleBinding %s in namespace %s: %v", rb.Name, rb.Namespace, err)
		}
		glog.V(4).Infof("Updated RoleBinding %s inside user-cluster %s", rb.Name, c.Name)
	}

	return nil
}

// GetUserClusterRoleBindingCreators returns a list of ClusterRoleBindingCreators which should be used to - guess what - create user cluster role bindings.
func GetUserClusterRoleBindingCreators(c *kubermaticv1.Cluster) []resources.ClusterRoleBindingCreator {
	creators := []resources.ClusterRoleBindingCreator{
		machinecontroller.ClusterRoleBinding,
		machinecontroller.NodeBootstrapperClusterRoleBinding,
		machinecontroller.NodeSignerClusterRoleBinding,
		controllermanager.AdminClusterRoleBinding,
		userclustercontrollermanager.AdminClusterRoleBinding,
		kubestatemetrics.ClusterRoleBinding,
		vpnsidecar.DnatControllerClusterRoleBinding,
	}

	if len(c.Spec.MachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.ClusterRoleBinding)
	}

	return creators
}

func (cc *Controller) userClusterEnsureClusterRoleBindings(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	creators := GetUserClusterRoleBindingCreators(c)

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *rbacv1.ClusterRoleBinding
		crb, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
		}

		if existing, err = client.RbacV1().ClusterRoleBindings().Get(crb.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = client.RbacV1().ClusterRoleBindings().Create(crb); err != nil {
				return fmt.Errorf("failed to create ClusterRoleBinding %s: %v", crb.Name, err)
			}
			glog.V(4).Infof("Created ClusterRoleBinding %s inside user-cluster %s", crb.Name, c.Name)
			continue
		}

		crb, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
		}

		if equality.Semantic.DeepEqual(crb, existing) {
			continue
		}

		if _, err = client.RbacV1().ClusterRoleBindings().Update(crb); err != nil {
			return fmt.Errorf("failed to update ClusterRoleBinding %s: %v", crb.Name, err)
		}
		glog.V(4).Infof("Updated ClusterRoleBinding %s inside user-cluster %s", crb.Name, c.Name)
	}

	return nil
}
