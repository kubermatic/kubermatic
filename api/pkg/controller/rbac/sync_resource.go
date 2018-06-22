package rbac

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
)

// syncProjectResource generates RBAC Role and Binding for a resource that belongs to a project.
// in order to support multiple cluster this code doesn't retrieve the project from the kube-api server
// instead it assumes that all required information is stored in OwnerReferences
//
// note:
// the project resources live only on master cluster and there are times
// that this code will be run on a seed cluster
func (c *Controller) syncProjectResource(item *projectResourceQueueItem) error {
	projectName := ""
	for _, owner := range item.metaObject.GetOwnerReferences() {
		if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
			len(owner.Name) > 0 && len(owner.UID) > 0 {
			projectName = owner.Name
			break
		}
	}
	if len(projectName) == 0 {
		return nil
		//
		// TODO: uncomment this when existing object are migrated to projects
		//       see: https://github.com/kubermatic/kubermatic/issues/1219
		//
		//return fmt.Errorf("unable to find owing project for the object name = %s, gvr = %s", item.metaObject.GetName(), item.gvr.String())
		//
		//  END of TODO
		//
	}

	if err := c.ensureClusterRBACRoleForNamedResource(projectName, item.gvr.Resource, item.kind, item.metaObject, item.clusterProvider.kubeClient, item.clusterProvider.rbacClusterRoleLister); err != nil {
		err = fmt.Errorf("failed to sync cluster RBAC Role for %s resource for %s cluster provider", item.gvr.String(), item.clusterProvider.providerName)
		return err
	}
	err := c.ensureClusterRBACRoleBindingForNamedResource(projectName, item.kind, item.metaObject, item.clusterProvider.kubeClient, item.clusterProvider.rbacClusterRoleBindingLister)
	if err != nil {
		err = fmt.Errorf("failed to sync cluster RBAC Role Binding for %s resource for %s cluster provider", item.gvr.String(), item.clusterProvider.providerName)
	}
	return err
}

func (c *Controller) ensureClusterRBACRoleForNamedResource(projectName string, objectResource string, objectKind string, object metav1.Object, kubeClient kubernetes.Interface, rbacClusterRoleLister rbaclister.ClusterRoleLister) error {
	for _, groupPrefix := range allGroupsPrefixes {
		generatedRole, err := generateClusterRBACRoleNamedResource(
			objectKind,
			generateActualGroupNameFor(projectName, groupPrefix),
			objectResource,
			kubermaticv1.SchemeGroupVersion.Group,
			object.GetName(),
			metav1.OwnerReference{
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
				Kind:       objectKind,
				UID:        object.GetUID(),
				Name:       object.GetName(),
			},
		)
		if err != nil {
			return err
		}
		sharedExistingRole, err := rbacClusterRoleLister.Get(generatedRole.Name)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return err
			}
		}

		// make sure that existing rbac role has appropriate rules/policies
		if sharedExistingRole != nil {
			if equality.Semantic.DeepEqual(sharedExistingRole.Rules, generatedRole.Rules) {
				continue
			}
			existingRole := sharedExistingRole.DeepCopy()
			existingRole.Rules = generatedRole.Rules
			if _, err = kubeClient.RbacV1().ClusterRoles().Update(existingRole); err != nil {
				return err
			}
			continue
		}

		if _, err = kubeClient.RbacV1().ClusterRoles().Create(generatedRole); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) ensureClusterRBACRoleBindingForNamedResource(projectName string, objectKind string, object metav1.Object, kubeClient kubernetes.Interface, rbacClusterRoleBindingLister rbaclister.ClusterRoleBindingLister) error {
	for _, groupPrefix := range allGroupsPrefixes {
		generatedRoleBinding := generateClusterRBACRoleBindingNamedResource(
			objectKind,
			object.GetName(),
			generateActualGroupNameFor(projectName, groupPrefix),
			metav1.OwnerReference{
				APIVersion: kubermaticv1.SchemeGroupVersion.String(),
				Kind:       objectKind,
				UID:        object.GetUID(),
				Name:       object.GetName(),
			},
		)
		sharedExistingRoleBinding, err := rbacClusterRoleBindingLister.Get(generatedRoleBinding.Name)
		if err != nil {
			if !kerrors.IsNotFound(err) {
				return err
			}
		}
		if sharedExistingRoleBinding != nil {
			if equality.Semantic.DeepEqual(sharedExistingRoleBinding.Subjects, generatedRoleBinding.Subjects) {
				continue
			}
			existingRoleBinding := sharedExistingRoleBinding.DeepCopy()
			existingRoleBinding.Subjects = generatedRoleBinding.Subjects
			if _, err = kubeClient.RbacV1().ClusterRoleBindings().Update(existingRoleBinding); err != nil {
				return err
			}
			continue
		}
		if _, err = kubeClient.RbacV1().ClusterRoleBindings().Create(generatedRoleBinding); err != nil {
			return err
		}
	}
	return nil
}
