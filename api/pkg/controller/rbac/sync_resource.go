package rbac

import (
	"fmt"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
)

// syncProjectResource generates RBAC Role and Binding for a resource that belongs to a project.
// in order to support multiple cluster this code doesn't retrieve the project from the kube-api server
// instead it assumes that all required information is stored in OwnerReferences or in Labels (for cluster resources)
//
// note:
// the project resources live only on master cluster and cluster resources are on master and seed clusters
// we cannot use OwnerReferences for cluster resources because they are on clusters that don't have corresponding
// project resource and will be automatically gc'ed
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
		projectName = item.metaObject.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	}

	if len(projectName) == 0 {
		return fmt.Errorf("unable to find owing project for the object name = %s, gvr = %s", item.metaObject.GetName(), item.gvr.String())
	}

	if len(item.namespace) == 0 {
		if err := c.ensureClusterRBACRoleForNamedResource(projectName, item.gvr.Resource, item.kind, item.metaObject, item.clusterProvider.kubeClient, item.clusterProvider.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoles().Lister()); err != nil {
			err = fmt.Errorf("failed to sync RBAC ClusterRole for %s resource for %s cluster provider, due to = %v", item.gvr.String(), item.clusterProvider.providerName, err)
			return err
		}
		err := c.ensureClusterRBACRoleBindingForNamedResource(projectName, item.gvr.Resource, item.kind, item.metaObject, item.clusterProvider.kubeClient, item.clusterProvider.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoleBindings().Lister())
		if err != nil {
			err = fmt.Errorf("failed to sync RBAC ClusterRoleBinding for %s resource for %s cluster provider, due to = %v", item.gvr.String(), item.clusterProvider.providerName, err)
		}
		return err
	}

	err := c.ensureRBACRoleForNamedResource(projectName,
		item.gvr,
		item.kind,
		item.namespace,
		item.metaObject,
		item.clusterProvider.kubeClient,
		item.clusterProvider.kubeInformerProvider.KubeInformerFactoryFor(item.namespace).Rbac().V1().Roles().Lister().Roles(item.namespace))
	if err != nil {
		return fmt.Errorf("failed to sync RBAC Role for %s resource for %s cluster provider in namespace %s, due to = %v", item.gvr.String(), item.clusterProvider.providerName, item.namespace, err)
	}

	err = c.ensureRBACRoleBindingForNamedResource(projectName,
		item.gvr,
		item.kind,
		item.namespace,
		item.metaObject,
		item.clusterProvider.kubeClient,
		item.clusterProvider.kubeInformerProvider.KubeInformerFactoryFor(item.namespace).Rbac().V1().RoleBindings().Lister().RoleBindings(item.namespace))
	if err != nil {
		return fmt.Errorf("failed to sync RBAC RoleBinding for %s resource for %s cluster provider in namespace %s, due to = %v", item.gvr.String(), item.clusterProvider.providerName, item.namespace, err)
	}

	return nil
}

func (c *Controller) ensureClusterRBACRoleForNamedResource(projectName string, objectResource string, objectKind string, object metav1.Object, kubeClient kubernetes.Interface, rbacClusterRoleLister rbaclister.ClusterRoleLister) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipClusterRBACRoleBindingForNamedResource(projectName, objectResource, objectKind, groupPrefix, object)
		if err != nil {
			return err
		}
		if skip {
			glog.V(5).Infof("skipping ClusterRole generation for named resource for group \"%s\" and resource \"%s\"", groupPrefix, objectResource)
			continue
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

func (c *Controller) ensureClusterRBACRoleBindingForNamedResource(projectName string, objectResource string, objectKind string, object metav1.Object, kubeClient kubernetes.Interface, rbacClusterRoleBindingLister rbaclister.ClusterRoleBindingLister) error {
	for _, groupPrefix := range AllGroupsPrefixes {

		skip, _, err := shouldSkipClusterRBACRoleBindingForNamedResource(projectName, objectResource, objectKind, groupPrefix, object)
		if err != nil {
			return err
		}
		if skip {
			glog.V(5).Infof("skipping operation on ClusterRoleBinding because corresponding ClusterRole was not(will not be) created for group %q and %q resource for project %q", groupPrefix, objectResource, projectName)
			continue
		}

		generatedRoleBinding := generateClusterRBACRoleBindingNamedResource(
			objectKind,
			object.GetName(),
			GenerateActualGroupNameFor(projectName, groupPrefix),
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

// shouldSkipClusterRBACRoleBindingForNamedResource will tell you if you should skip the generation of ClusterResource or not,
// because for some kinds we actually don't create ClusterRole
//
// note that this method returns generated role if is not meant to be skipped
func shouldSkipClusterRBACRoleBindingForNamedResource(projectName string, objectResource string, objectKind string, groupPrefix string, object metav1.Object) (bool, *rbacv1.ClusterRole, error) {
	generatedRole, err := generateClusterRBACRoleNamedResource(
		objectKind,
		GenerateActualGroupNameFor(projectName, groupPrefix),
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
		return false, generatedRole, err
	}
	if generatedRole == nil {
		return true, nil, nil
	}
	return false, generatedRole, nil
}

func (c *Controller) ensureRBACRoleForNamedResource(projectName string, objectGVR schema.GroupVersionResource, objectKind string, namespace string, object metav1.Object, kubeClient kubernetes.Interface, rbacRoleLister rbaclister.RoleNamespaceLister) error {
	for _, groupPrefix := range AllGroupsPrefixes {
		skip, generatedRole, err := shouldSkipRBACRoleBindingForNamedResource(projectName, objectGVR, objectKind, groupPrefix, namespace, object)
		if err != nil {
			return err
		}
		if skip {
			glog.V(5).Infof("skipping Role generation for named resource for group %q and resource %q in namespace %q", groupPrefix, objectGVR.Resource, namespace)
			continue
		}
		sharedExistingRole, err := rbacRoleLister.Get(generatedRole.Name)
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
			if _, err = kubeClient.RbacV1().Roles(namespace).Update(existingRole); err != nil {
				return err
			}
			continue
		}

		if _, err = kubeClient.RbacV1().Roles(namespace).Create(generatedRole); err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) ensureRBACRoleBindingForNamedResource(projectName string, objectGVR schema.GroupVersionResource, objectKind string, namespace string, object metav1.Object, kubeClient kubernetes.Interface, rbacRoleBindingLister rbaclister.RoleBindingNamespaceLister) error {
	for _, groupPrefix := range AllGroupsPrefixes {

		skip, _, err := shouldSkipRBACRoleBindingForNamedResource(projectName, objectGVR, objectKind, groupPrefix, namespace, object)
		if err != nil {
			return err
		}
		if skip {
			glog.V(5).Infof("skipping operation on RoleBinding because corresponding Role was not(will not be) created for group %q and %q resource for project %q in namespace %q", groupPrefix, objectGVR.Resource, projectName, namespace)
			continue
		}

		generatedRoleBinding := generateRBACRoleBindingNamedResource(
			objectKind,
			object.GetName(),
			GenerateActualGroupNameFor(projectName, groupPrefix),
			namespace,
			metav1.OwnerReference{
				APIVersion: objectGVR.GroupVersion().String(),
				Kind:       objectKind,
				UID:        object.GetUID(),
				Name:       object.GetName(),
			},
		)
		sharedExistingRoleBinding, err := rbacRoleBindingLister.Get(generatedRoleBinding.Name)
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
			if _, err = kubeClient.RbacV1().RoleBindings(namespace).Update(existingRoleBinding); err != nil {
				return err
			}
			continue
		}
		if _, err = kubeClient.RbacV1().RoleBindings(namespace).Create(generatedRoleBinding); err != nil {
			return err
		}
	}
	return nil
}

// shouldSkipRBACRoleBindingForNamedResource will tell you if you should skip the generation of ClusterResource or not,
// because for some kinds we actually don't create Role
//
// note that this method returns generated role if is not meant to be skipped
func shouldSkipRBACRoleBindingForNamedResource(projectName string, objectGVR schema.GroupVersionResource, objectKind string, groupPrefix string, namespace string, object metav1.Object) (bool, *rbacv1.Role, error) {
	generatedRole, err := generateRBACRoleNamedResource(
		objectKind,
		GenerateActualGroupNameFor(projectName, groupPrefix),
		objectGVR.Resource,
		objectGVR.Group,
		object.GetName(),
		namespace,
		metav1.OwnerReference{
			APIVersion: objectGVR.GroupVersion().String(),
			Kind:       objectKind,
			UID:        object.GetUID(),
			Name:       object.GetName(),
		},
	)

	if err != nil {
		return false, generatedRole, err
	}
	if generatedRole == nil {
		return true, nil, nil
	}
	return false, generatedRole, nil
}
