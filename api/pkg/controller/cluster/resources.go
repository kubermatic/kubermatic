package cluster

import (
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/go-test/deep"
	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/controllermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroler"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/prometheus"
	"github.com/kubermatic/kubermatic/api/pkg/resources/scheduler"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	nodeDeletionFinalizer = "kubermatic.io/delete-nodes"

	annotationPrefix            = "kubermatic.io/"
	lastAppliedConfigAnnotation = annotationPrefix + "last-applied-configuration"
)

func (cc *Controller) ensureResourcesAreDeployed(cluster *kubermaticv1.Cluster) error {
	// check that all service accounts are created
	if err := cc.ensureCheckServiceAccounts(cluster); err != nil {
		return err
	}

	// check that all roles are created
	if err := cc.ensureRoles(cluster); err != nil {
		return err
	}

	// check that all role bindings are created
	if err := cc.ensureRoleBindings(cluster); err != nil {
		return err
	}

	// check that all role bindings are created
	if err := cc.ensureClusterRoleBindings(cluster); err != nil {
		return err
	}

	// check that all services are available
	if err := cc.ensureServices(cluster); err != nil {
		return err
	}

	// check that all secrets are available
	if err := cc.ensureSecrets(cluster); err != nil {
		return err
	}

	// check that all secrets are available // New way of handling secrets
	if err := cc.ensureSecretsV2(cluster); err != nil {
		return err
	}

	// check that all ConfigMap's are available
	if err := cc.ensureConfigMaps(cluster); err != nil {
		return err
	}

	// check that all deployments are available
	if err := cc.ensureDeployments(cluster); err != nil {
		return err
	}

	// check that all StatefulSet's are created
	if err := cc.ensureStatefulSets(cluster); err != nil {
		return err
	}

	return nil
}

func (cc *Controller) getClusterTemplateData(c *kubermaticv1.Cluster) (*resources.TemplateData, error) {
	dc, found := cc.dcs[c.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", c.Spec.Cloud.DatacenterName)
	}

	return resources.NewTemplateData(
		c,
		&dc,
		cc.secretLister,
		cc.configMapLister,
		cc.serviceLister,
		cc.overwriteRegistry,
		cc.nodePortRange,
		cc.nodeAccessNetwork,
		cc.etcdDiskSize,
	), nil
}

// ensureNamespaceExists will create the cluster namespace
func (cc *Controller) ensureNamespaceExists(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var err error
	if c.Status.NamespaceName == "" {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Status.NamespaceName = fmt.Sprintf("cluster-%s", c.Name)
		})
		if err != nil {
			return nil, err
		}
	}

	if _, err := cc.namespaceLister.Get(c.Status.NamespaceName); !errors.IsNotFound(err) {
		return c, err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            c.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{cc.getOwnerRefForCluster(c)},
		},
	}
	if _, err := cc.kubeClient.CoreV1().Namespaces().Create(ns); err != nil {
		return nil, fmt.Errorf("failed to create namespace %s: %v", c.Status.NamespaceName, err)
	}

	return c, nil
}

func getPatch(currentObj, updateObj metav1.Object) ([]byte, error) {
	currentData, err := json.Marshal(currentObj)
	if err != nil {
		return nil, err
	}

	modifiedData, err := json.Marshal(updateObj)
	if err != nil {
		return nil, err
	}

	originalData, exists := currentObj.GetAnnotations()[lastAppliedConfigAnnotation]
	if !exists {
		glog.V(2).Infof("no last applied found in annotation %s for %s/%s", lastAppliedConfigAnnotation, currentObj.GetNamespace(), currentObj.GetName())
	}

	return jsonmergepatch.CreateThreeWayJSONMergePatch([]byte(originalData), modifiedData, currentData)
}

// Deprecated
func (cc *Controller) ensureSecrets(c *kubermaticv1.Cluster) error {
	//We need to follow a specific order here...
	//And maps in go are not sorted
	type secretOp struct {
		name string
		gen  func(*kubermaticv1.Cluster, *corev1.Secret) (*corev1.Secret, string, error)
	}
	ops := []secretOp{
		{resources.CAKeySecretName, cc.getRootCAKeySecret},
		{resources.CACertSecretName, cc.getRootCACertSecret},
		{resources.ApiserverTLSSecretName, cc.getApiserverServingCertificatesSecret},
		{resources.KubeletClientCertificatesSecretName, cc.getKubeletClientCertificatesSecret},
		{resources.ServiceAccountKeySecretName, cc.getServiceAccountKeySecret},
		{resources.AdminKubeconfigSecretName, cc.getAdminKubeconfigSecret},
		{resources.TokensSecretName, cc.getTokenUsersSecret},
		{resources.OpenVPNServerCertificatesSecretName, cc.getOpenVPNServerCertificates},
		{resources.OpenVPNClientCertificatesSecretName, cc.getOpenVPNInternalClientCertificates},
	}

	for _, op := range ops {
		exists := false
		existingSecret, err := cc.secretLister.Secrets(c.Status.NamespaceName).Get(op.name)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to get secret %s from lister: %v", op.name, err)
			}
		} else {
			exists = true
		}

		generatedSecret, currentJSON, err := op.gen(c, existingSecret)
		if err != nil {
			return fmt.Errorf("failed to generate Secret %s: %v", op.name, err)
		}
		generatedSecret.Annotations[lastAppliedConfigAnnotation] = currentJSON
		generatedSecret.Name = op.name

		if !exists {
			if _, err = cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Create(generatedSecret); err != nil {
				return fmt.Errorf("failed to create secret for %s: %v", op.name, err)
			}

			secretExistsInLister := func() (bool, error) {
				_, err = cc.secretLister.Secrets(c.Status.NamespaceName).Get(generatedSecret.Name)
				if err != nil {
					if os.IsNotExist(err) {
						return false, nil
					}
					runtime.HandleError(fmt.Errorf("failed to check if a created secret %s/%s got published to lister: %v", c.Status.NamespaceName, generatedSecret.Name, err))
					return false, nil
				}
				return true, nil
			}

			if err := wait.Poll(100*time.Millisecond, 30*time.Second, secretExistsInLister); err != nil {
				return fmt.Errorf("failed waiting for secret '%s' to exist in the lister: %v", generatedSecret.Name, err)
			}
			continue
		} else {
			if existingSecret.Annotations[lastAppliedConfigAnnotation] != currentJSON {
				patch, err := getPatch(existingSecret, generatedSecret)
				if err != nil {
					return err
				}
				if _, err := cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Patch(op.name, types.MergePatchType, patch); err != nil {
					return fmt.Errorf("failed to patch secret '%s': %v", op.name, err)
				}
			}
		}
	}

	return nil
}

func (cc *Controller) ensureServices(c *kubermaticv1.Cluster) error {
	creators := []resources.ServiceCreator{
		apiserver.Service,
		apiserver.ExternalService,
		prometheus.Service,
		openvpn.Service,
		etcd.DiscoveryService,
		etcd.ClientService,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *corev1.Service
		service, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build Service: %v", err)
		}

		if existing, err = cc.serviceLister.Services(c.Status.NamespaceName).Get(service.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.CoreV1().Services(c.Status.NamespaceName).Create(service); err != nil {
				return fmt.Errorf("failed to create Service %s: %v", service.Name, err)
			}
			continue
		}

		service, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Service: %v", err)
		}

		if diff := deep.Equal(service, existing); diff == nil {
			continue
		}

		if _, err = cc.kubeClient.CoreV1().Services(c.Status.NamespaceName).Update(service); err != nil {
			return fmt.Errorf("failed to patch Service %s: %v", service.Name, err)
		}
	}

	return nil
}

func (cc *Controller) ensureCheckServiceAccounts(c *kubermaticv1.Cluster) error {
	names := []string{
		resources.PrometheusServiceAccountName,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}
	ref := data.GetClusterRef()

	for _, name := range names {
		var existing *corev1.ServiceAccount
		sa := resources.ServiceAccount(name, &ref, nil)

		if existing, err = cc.serviceAccountLister.ServiceAccounts(c.Status.NamespaceName).Get(sa.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.CoreV1().ServiceAccounts(c.Status.NamespaceName).Create(sa); err != nil {
				return fmt.Errorf("failed to create ServiceAccount %s: %v", sa.Name, err)
			}
			continue
		}

		// We update the existing SA
		sa = resources.ServiceAccount(name, &ref, existing.DeepCopy())
		if diff := deep.Equal(sa, existing); diff == nil {
			continue
		}
		if _, err = cc.kubeClient.CoreV1().ServiceAccounts(c.Status.NamespaceName).Update(sa); err != nil {
			return fmt.Errorf("failed to patch ServiceAccount %s: %v", sa.Name, err)
		}
	}

	return nil
}

func (cc *Controller) ensureRoles(c *kubermaticv1.Cluster) error {
	creators := []resources.RoleCreator{
		prometheus.Role,
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

		if existing, err = cc.roleLister.Roles(c.Status.NamespaceName).Get(role.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.RbacV1().Roles(c.Status.NamespaceName).Create(role); err != nil {
				return fmt.Errorf("failed to create Role %s: %v", role.Name, err)
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

		if _, err = cc.kubeClient.RbacV1().Roles(c.Status.NamespaceName).Update(role); err != nil {
			return fmt.Errorf("failed to update Role %s: %v", role.Name, err)
		}
	}

	return nil
}

func (cc *Controller) ensureRoleBindings(c *kubermaticv1.Cluster) error {
	creators := []resources.RoleBindingCreator{
		prometheus.RoleBinding,
	}

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *rbacv1.RoleBinding
		rb, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding: %v", err)
		}

		if existing, err = cc.roleBindingLister.RoleBindings(c.Status.NamespaceName).Get(rb.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.RbacV1().RoleBindings(c.Status.NamespaceName).Create(rb); err != nil {
				return fmt.Errorf("failed to create RoleBinding %s: %v", rb.Name, err)
			}
			continue
		}

		rb, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build RoleBinding: %v", err)
		}

		if diff := deep.Equal(rb, existing); diff == nil {
			continue
		}

		if _, err = cc.kubeClient.RbacV1().RoleBindings(c.Status.NamespaceName).Update(rb); err != nil {
			return fmt.Errorf("failed to update RoleBinding %s: %v", rb.Name, err)
		}
	}

	return nil
}

func (cc *Controller) ensureClusterRoleBindings(c *kubermaticv1.Cluster) error {
	creators := []resources.ClusterRoleBindingCreator{}

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

		if existing, err = cc.clusterRoleBindingLister.Get(crb.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.RbacV1().ClusterRoleBindings().Create(crb); err != nil {
				return fmt.Errorf("failed to create ClusterRoleBinding %s: %v", crb.Name, err)
			}
			continue
		}

		crb, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ClusterRoleBinding: %v", err)
		}

		if diff := deep.Equal(crb, existing); diff == nil {
			continue
		}

		if _, err = cc.kubeClient.RbacV1().ClusterRoleBindings().Update(crb); err != nil {
			return fmt.Errorf("failed to update ClusterRoleBinding %s: %v", crb.Name, err)
		}
	}

	return nil
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators() []resources.DeploymentCreator {
	return []resources.DeploymentCreator{
		machinecontroller.Deployment,
		openvpn.Deployment,
		apiserver.Deployment,
		scheduler.Deployment,
		controllermanager.Deployment,
	}
}

func (cc *Controller) ensureDeployments(c *kubermaticv1.Cluster) error {
	creators := GetDeploymentCreators()

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *appsv1.Deployment
		dep, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build Deployment: %v", err)
		}

		if existing, err = cc.deploymentLister.Deployments(c.Status.NamespaceName).Get(dep.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.AppsV1().Deployments(c.Status.NamespaceName).Create(dep); err != nil {
				return fmt.Errorf("failed to create Deployment %s: %v", dep.Name, err)
			}
			continue
		}

		dep, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Deployment: %v", err)
		}

		if diff := deep.Equal(dep, existing); diff == nil {
			continue
		}

		// In case we update something immutable we need to delete&recreate. Creation happens on next sync
		if !reflect.DeepEqual(dep.Spec.Selector.MatchLabels, existing.Spec.Selector.MatchLabels) {
			propagation := metav1.DeletePropagationForeground
			return cc.kubeClient.AppsV1().Deployments(c.Status.NamespaceName).Delete(dep.Name, &metav1.DeleteOptions{PropagationPolicy: &propagation})
		}

		if _, err = cc.kubeClient.AppsV1().Deployments(c.Status.NamespaceName).Update(dep); err != nil {
			return fmt.Errorf("failed to update Deployment %s: %v", dep.Name, err)
		}
	}

	return nil
}

// GetSecretCreators returns all SecretCreators that are currently in use
func GetSecretCreators() []resources.SecretCreator {
	return []resources.SecretCreator{
		etcd.TLSCertificate,
	}
}

func (cc *Controller) ensureSecretsV2(c *kubermaticv1.Cluster) error {
	creators := GetSecretCreators()

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *corev1.Secret
		se, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build Secret: %v", err)
		}
		if existing, err = cc.secretLister.Secrets(c.Status.NamespaceName).Get(se.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Create(se); err != nil {
				return fmt.Errorf("failed to create Secret %s: %v", se.Name, err)
			}
			continue
		}

		se, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Secret: %v", err)
		}

		if diff := deep.Equal(se, existing); diff == nil {
			continue
		}

		if _, err = cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Update(se); err != nil {
			return fmt.Errorf("failed to update Secret %s: %v", se.Name, err)
		}
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use
func GetConfigMapCreators() []resources.ConfigMapCreator {
	return []resources.ConfigMapCreator{
		cloudconfig.ConfigMap,
		openvpn.ConfigMap,
		prometheus.ConfigMap,
	}
}

func (cc *Controller) ensureConfigMaps(c *kubermaticv1.Cluster) error {
	creators := GetConfigMapCreators()

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *corev1.ConfigMap
		cm, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		if existing, err = cc.configMapLister.ConfigMaps(c.Status.NamespaceName).Get(cm.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.CoreV1().ConfigMaps(c.Status.NamespaceName).Create(cm); err != nil {
				return fmt.Errorf("failed to create ConfigMap %s: %v", cm.Name, err)
			}
			continue
		}

		cm, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build ConfigMap: %v", err)
		}

		if diff := deep.Equal(cm, existing); diff == nil {
			continue
		}

		if _, err = cc.kubeClient.CoreV1().ConfigMaps(c.Status.NamespaceName).Update(cm); err != nil {
			return fmt.Errorf("failed to update ConfigMap %s: %v", cm.Name, err)
		}
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use
func GetStatefulSetCreators() []resources.StatefulSetCreator {
	return []resources.StatefulSetCreator{
		prometheus.StatefulSet,
		etcd.StatefulSet,
	}
}

func (cc *Controller) ensureStatefulSets(c *kubermaticv1.Cluster) error {
	creators := GetStatefulSetCreators()

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *appsv1.StatefulSet
		set, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build StatefulSet: %v", err)
		}

		if existing, err = cc.statefulSetLister.StatefulSets(c.Status.NamespaceName).Get(set.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.AppsV1().StatefulSets(c.Status.NamespaceName).Create(set); err != nil {
				return fmt.Errorf("failed to create StatefulSet %s: %v", set.Name, err)
			}
			continue
		}

		set, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build StatefulSet: %v", err)
		}

		if diff := deep.Equal(set, existing); diff == nil {
			continue
		}

		// In case we update something immutable we need to delete&recreate. Creation happens on next sync
		if !reflect.DeepEqual(set.Spec.Selector.MatchLabels, existing.Spec.Selector.MatchLabels) {
			propagation := metav1.DeletePropagationForeground
			return cc.kubeClient.AppsV1().StatefulSets(c.Status.NamespaceName).Delete(set.Name, &metav1.DeleteOptions{PropagationPolicy: &propagation})
		}

		if _, err = cc.kubeClient.AppsV1().StatefulSets(c.Status.NamespaceName).Update(set); err != nil {
			return fmt.Errorf("failed to update StatefulSet %s: %v", set.Name, err)
		}
	}

	return nil
}
