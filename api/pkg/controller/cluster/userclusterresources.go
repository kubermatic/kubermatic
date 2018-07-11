package cluster

import (
	"fmt"

	"github.com/go-test/deep"
	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/controllermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroler"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
			continue
		}

		role, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Role: %v", err)
		}

		if diff := deep.Equal(role, existing); diff == nil {
			continue
		}

		if _, err = client.RbacV1().Roles(role.Namespace).Update(role); err != nil {
			return fmt.Errorf("failed to update Role %s in namespace %s: %v", role.Name, role.Namespace, err)
		}
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
			continue
		}

		rb, err = create(nil, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding: %v", err)
		}

		if diff := deep.Equal(rb, existing); diff == nil {
			continue
		}

		if _, err = client.RbacV1().RoleBindings(rb.Namespace).Update(rb); err != nil {
			return fmt.Errorf("failed to update RoleBinding %s in namespace %s: %v", rb.Name, rb.Namespace, err)
		}
	}

	return nil
}

func (cc *Controller) userClusterEnsureClusterRoles(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	creators := []resources.ClusterRoleCreator{
		machinecontroller.ClusterRole,
	}
	glog.V(2).Infof("userClusterEnsureClusterRoles entry %v", creators[0])

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *rbacv1.ClusterRole
		cRole, err := create(data, nil)
		glog.V(4).Infof("user-cluster ensureClusterRoles created %v, %v", cRole, err)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole: %v", err)
		}

		if existing, err = client.RbacV1().ClusterRoles().Get(cRole.Name, metav1.GetOptions{}); err != nil {
			glog.V(4).Infof("ensureClusterRoles existing: %v, %v", existing, err)
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = client.RbacV1().ClusterRoles().Create(cRole); err != nil {
				glog.V(4).Infof("user-cluster ensureClusterRoles Create (did not exist): %v", err)
				return fmt.Errorf("failed to create ClusterRole %s: %v", cRole.Name, err)
			}
			continue
		}

		cRole, err = create(data, existing.DeepCopy())
		glog.V(4).Infof("user-cluster ensureClusterRoles Create (did exist): existing:%v; created:%v", existing, cRole, err)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole: %v", err)
		}

		if diff := deep.Equal(cRole, existing); diff == nil {
			continue
		}

		if _, err = client.RbacV1().ClusterRoles().Update(cRole); err != nil {
			return fmt.Errorf("failed to update ClusterRole %s: %v", cRole.Name, err)
		}
	}

	return nil
}

func (cc *Controller) userClusterEnsureClusterRoleBindings(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetClient(c)
	if err != nil {
		return err
	}

	creators := []resources.ClusterRoleBindingCreator{
		machinecontroller.ClusterRoleBinding,
		machinecontroller.NodeBootstrapperClusterRoleBinding,
		machinecontroller.NodeSignerClusterRoleBinding,
		controllermanager.AdminClusterRoleBinding,
	}

	for _, create := range creators {
		var existing *rbacv1.ClusterRoleBinding
		crb, err := create(nil, nil)
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
			continue
		}

		crb, err = create(nil, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
		}

		if diff := deep.Equal(crb, existing); diff == nil {
			continue
		}

		if _, err = client.RbacV1().ClusterRoleBindings().Update(crb); err != nil {
			return fmt.Errorf("failed to update ClusterRoleBinding %s: %v", crb.Name, err)
		}
	}

	return nil
}
