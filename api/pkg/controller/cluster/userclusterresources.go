package cluster

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/go-test/deep"
	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/ipamcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	admissionv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (cc *Controller) reconcileUserClusterResources(cluster *kubermaticv1.Cluster, client kubernetes.Interface) (*kubermaticv1.Cluster, error) {
	var err error
	if err = cc.launchingCreateOpenVPNClientCertificates(cluster); err != nil {
		return nil, err
	}

	if len(cluster.Spec.MachineNetworks) > 0 {
		if err = cc.userClusterEnsureInitializerConfiguration(cluster, client); err != nil {
			return nil, err
		}
	}

	if err = cc.userClusterEnsureRoles(cluster, client); err != nil {
		return nil, err
	}

	if err = cc.userClusterEnsureConfigMaps(cluster, client); err != nil {
		return nil, err
	}

	if err = cc.userClusterEnsureRoleBindings(cluster, client); err != nil {
		return nil, err
	}

	if err = cc.userClusterEnsureClusterRoles(cluster, client); err != nil {
		return nil, err
	}

	if err = cc.userClusterEnsureClusterRoleBindings(cluster, client); err != nil {
		return nil, err
	}

	data, err := cc.getClusterTemplateData(cluster)
	if err != nil {
		return nil, err
	}

	if err = cc.userClusterEnsureMutatingWebhookConfigurations(cluster, data); err != nil {
		return nil, err
	}

	if err = cc.userClusterEnsureCustomResourceDefinitions(cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

func (cc *Controller) userClusterEnsureInitializerConfiguration(c *kubermaticv1.Cluster, client kubernetes.Interface) error {
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

func (cc *Controller) userClusterEnsureRoles(c *kubermaticv1.Cluster, client kubernetes.Interface) error {

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

		if resources.DeepEqual(role, existing) {
			continue
		}

		if _, err = client.RbacV1().Roles(role.Namespace).Update(role); err != nil {
			return fmt.Errorf("failed to update Role %s in namespace %s: %v", role.Name, role.Namespace, err)
		}
		glog.V(4).Infof("Updated Role %s inside user-cluster %s", role.Name, c.Name)
	}

	return nil
}

func (cc *Controller) userClusterEnsureRoleBindings(c *kubermaticv1.Cluster, client kubernetes.Interface) error {
	creators := []resources.RoleBindingCreator{
		machinecontroller.DefaultRoleBinding,
		machinecontroller.KubeSystemRoleBinding,
		machinecontroller.KubePublicRoleBinding,
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

// GetUserClusterRoleCreators returns a list of GetUserClusterRoleCreators
func GetUserClusterRoleCreators(c *kubermaticv1.Cluster) []resources.ClusterRoleCreator {
	creators := []resources.ClusterRoleCreator{
		machinecontroller.ClusterRole,
		vpnsidecar.DnatControllerClusterRole,
	}

	if len(c.Spec.MachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.ClusterRole)
	}

	return creators
}

func (cc *Controller) userClusterEnsureClusterRoles(c *kubermaticv1.Cluster, client kubernetes.Interface) error {
	creators := GetUserClusterRoleCreators(c)

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *rbacv1.ClusterRole
		cRole, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole: %v", err)
		}

		if existing, err = client.RbacV1().ClusterRoles().Get(cRole.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = client.RbacV1().ClusterRoles().Create(cRole); err != nil {
				return fmt.Errorf("failed to create ClusterRole %s: %v", cRole.Name, err)
			}
			glog.V(4).Infof("Created ClusterRole %s inside user-cluster %s", cRole.Name, c.Name)
			continue
		}

		cRole, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRole: %v", err)
		}

		if equality.Semantic.DeepEqual(cRole, existing) {
			continue
		}

		if _, err = client.RbacV1().ClusterRoles().Update(cRole); err != nil {
			return fmt.Errorf("failed to update ClusterRole %s: %v", cRole.Name, err)
		}
		glog.V(4).Infof("Updated ClusterRole %s inside user-cluster %s", cRole.Name, c.Name)
	}

	return nil
}

// GetUserClusterRoleBindingCreators returns a list of ClusterRoleBindingCreators which should be used to - guess what - create user cluster role bindings.
func GetUserClusterRoleBindingCreators(c *kubermaticv1.Cluster) []resources.ClusterRoleBindingCreator {
	creators := []resources.ClusterRoleBindingCreator{
		machinecontroller.ClusterRoleBinding,
		machinecontroller.NodeBootstrapperClusterRoleBinding,
		machinecontroller.NodeSignerClusterRoleBinding,
		vpnsidecar.DnatControllerClusterRoleBinding,
	}

	if len(c.Spec.MachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.ClusterRoleBinding)
	}

	return creators
}

func (cc *Controller) userClusterEnsureClusterRoleBindings(c *kubermaticv1.Cluster, client kubernetes.Interface) error {
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

func (cc *Controller) userClusterEnsureConfigMaps(c *kubermaticv1.Cluster, client kubernetes.Interface) error {

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	creators := []resources.ConfigMapCreator{
		openvpn.ClientConfigConfigMapCreator(data),
	}

	for _, create := range creators {
		var existing *corev1.ConfigMap
		cm, err := create(nil)
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		if existing, err = client.CoreV1().ConfigMaps(cm.Namespace).Get(cm.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = client.CoreV1().ConfigMaps(cm.Namespace).Create(cm); err != nil {
				return fmt.Errorf("failed to create ConfigMap %s: %v", cm.Name, err)
			}
			glog.V(4).Infof("Created ConfigMap %s inside user-cluster %s", cm.Name, c.Name)
			continue
		}

		cm, err = create(existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		if equality.Semantic.DeepEqual(cm, existing) {
			continue
		}

		if _, err = client.CoreV1().ConfigMaps(cm.Namespace).Update(cm); err != nil {
			return fmt.Errorf("failed to update ConfigMap %s: %v", cm.Name, err)
		}
		glog.V(4).Infof("Updated ConfigMap %s inside user-cluster %s", cm.Name, c.Name)
	}

	return nil
}

// GetCRDCreators reuturns a list of CRDCreateors
func GetCRDCreators() []resources.CRDCreateor {
	return []resources.CRDCreateor{
		machinecontroller.MachineCRD,
		machinecontroller.MachineSetCRD,
		machinecontroller.MachineDeploymentCRD,
		machinecontroller.ClusterCRD,
	}
}

func (cc *Controller) userClusterEnsureCustomResourceDefinitions(c *kubermaticv1.Cluster) error {
	client, err := cc.userClusterConnProvider.GetApiextensionsClient(c)
	if err != nil {
		return err
	}

	version, err := semver.NewVersion(c.Spec.Version)
	if err != nil {
		return fmt.Errorf("failed to extract version from cluster %s: %v", c.Name, err)
	}

	for _, create := range GetCRDCreators() {
		var existing *apiextensionsv1beta1.CustomResourceDefinition
		crd, err := create(*version, nil)
		if err != nil {
			return fmt.Errorf("failed to build CustomResourceDefinitions: %v", err)
		}
		if existing, err = client.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{}); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			if _, err = client.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd); err != nil {
				return fmt.Errorf("failed to create CustomResourceDefinition %s: %v", crd.Name, err)
			}
			glog.V(4).Infof("Created CustomResourceDefinition %s inside user cluster %s", crd.Name, c.Name)
			continue
		}

		crd, err = create(*version, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build CustomResourceDefinition: %v", err)
		}

		if equality.Semantic.DeepEqual(crd, existing) {
			continue
		}

		if _, err = client.ApiextensionsV1beta1().CustomResourceDefinitions().Update(crd); err != nil {
			return fmt.Errorf("failed to update CustomResourceDefinition %s: %v", crd.Name, err)
		}
		glog.V(4).Infof("Updated CustomResourceDefinition %s inside user cluster %s", crd.Name, c.Name)
	}

	return nil
}

// GetUserClusterMutatingWebhookConfigurationCreators returns all UserClusterMutatingWebhookConfigurationCreators
func GetUserClusterMutatingWebhookConfigurationCreators() []resources.MutatingWebhookConfigurationCreator {
	return []resources.MutatingWebhookConfigurationCreator{
		machinecontroller.MutatingwebhookConfiguration,
	}
}

func (cc *Controller) userClusterEnsureMutatingWebhookConfigurations(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	client, err := cc.userClusterConnProvider.GetAdmissionRegistrationClient(c)
	if err != nil {
		return err
	}

	for _, creator := range GetUserClusterMutatingWebhookConfigurationCreators() {
		mutatingWebhookConfiguration, err := creator(c, data, nil)
		if err != nil {
			return fmt.Errorf("failed to build MutatingwebhookConfiguration: %v", err)
		}
		existing, err := client.MutatingWebhookConfigurations().Get(mutatingWebhookConfiguration.Name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = client.MutatingWebhookConfigurations().Create(mutatingWebhookConfiguration); err != nil {
				return fmt.Errorf("failed to create MutatingWebhookConfiguration: %v", err)
			}
			glog.V(4).Infof("Created MutatingWebhookConfiguration %s inside user cluster %s", mutatingWebhookConfiguration.Name, c.Name)
		}

		mutatingWebhookConfiguration, err = creator(c, data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build MutatingWebhookConfiguration: %v", err)
		}

		if equality.Semantic.DeepEqual(mutatingWebhookConfiguration, existing) {
			return nil
		}

		if _, err = client.MutatingWebhookConfigurations().Update(mutatingWebhookConfiguration); err != nil {
			return fmt.Errorf("failed to update MutatingWebhookConfigurations %s: %v", mutatingWebhookConfiguration.Name, err)
		}
		glog.V(4).Infof("Updated MutatingWebhookConfigurations %s inside user cluster %s", mutatingWebhookConfiguration.Name, c.Name)
	}

	return nil
}
