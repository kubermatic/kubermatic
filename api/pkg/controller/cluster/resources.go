package cluster

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"sort"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/controllermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/dns"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/ipamcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/kubestatemetrics"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/prometheus"
	"github.com/kubermatic/kubermatic/api/pkg/resources/scheduler"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nodeDeletionFinalizer = "kubermatic.io/delete-nodes"

	annotationPrefix   = "kubermatic.io/"
	checksumAnnotation = annotationPrefix + "checksum"
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

	// check that all cluster role bindings are created
	if err := cc.ensureClusterRoleBindings(cluster); err != nil {
		return err
	}

	// check that all services are available
	if err := cc.ensureServices(cluster); err != nil {
		return err
	}

	// check that all secrets are available // New way of handling secrets
	if err := cc.ensureSecrets(cluster); err != nil {
		return err
	}

	// check that all ConfigMaps are available
	if err := cc.ensureConfigMaps(cluster); err != nil {
		return err
	}

	// check that all Deployments are available
	if err := cc.ensureDeployments(cluster); err != nil {
		return err
	}

	// check that all StatefulSets are created
	if err := cc.ensureStatefulSets(cluster); err != nil {
		return err
	}

	// check that all CronJobs are created
	if err := cc.ensureCronJobs(cluster); err != nil {
		return err
	}

	// check that all PodDisruptionBudgets are created
	if err := cc.ensurePodDisruptionBudgets(cluster); err != nil {
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
		cc.dc,
		cc.secretLister,
		cc.configMapLister,
		cc.serviceLister,
		cc.overwriteRegistry,
		cc.nodePortRange,
		cc.nodeAccessNetwork,
		cc.etcdDiskSize,
		cc.inClusterPrometheusRulesFile,
		cc.inClusterPrometheusDisableDefaultRules,
		cc.inClusterPrometheusDisableDefaultScrapingConfigs,
		cc.inClusterPrometheusScrapingConfigsFile,
		cc.dockerPullConfigJSON,
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

// GetServiceCreators returns all service creators that are currently in use
func GetServiceCreators() []resources.ServiceCreator {
	return []resources.ServiceCreator{
		apiserver.Service,
		apiserver.ExternalService,
		prometheus.Service,
		openvpn.Service,
		etcd.Service,
		dns.Service,
	}
}

func (cc *Controller) ensureServices(c *kubermaticv1.Cluster) error {
	creators := GetServiceCreators()

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

		if resources.DeepEqual(service, existing) {
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
		if resources.DeepEqual(sa, existing) {
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
		if err := resources.EnsureRole(data, create, cc.roleLister.Roles(c.Status.NamespaceName), cc.kubeClient.RbacV1().Roles(c.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the role exists: %v", err)
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

		if resources.DeepEqual(rb, existing) {
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

		if resources.DeepEqual(crb, existing) {
			continue
		}

		if _, err = cc.kubeClient.RbacV1().ClusterRoleBindings().Update(crb); err != nil {
			return fmt.Errorf("failed to update ClusterRoleBinding %s: %v", crb.Name, err)
		}
	}

	return nil
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(c *kubermaticv1.Cluster) []resources.DeploymentCreator {
	creators := []resources.DeploymentCreator{
		machinecontroller.Deployment,
		openvpn.Deployment,
		apiserver.Deployment,
		scheduler.Deployment,
		controllermanager.Deployment,
		dns.Deployment,
		kubestatemetrics.Deployment,
	}

	if c != nil && len(c.Spec.MachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.Deployment)
	}

	return creators
}

func (cc *Controller) ensureDeployments(c *kubermaticv1.Cluster) error {
	creators := GetDeploymentCreators(c)

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

		if resources.DeepEqual(dep, existing) {
			continue
		}

		// In case we update something immutable we need to delete&recreate. Creation happens on next sync
		if !equality.Semantic.DeepEqual(dep.Spec.Selector.MatchLabels, existing.Spec.Selector.MatchLabels) {
			propagation := metav1.DeletePropagationForeground
			return cc.kubeClient.AppsV1().Deployments(c.Status.NamespaceName).Delete(dep.Name, &metav1.DeleteOptions{PropagationPolicy: &propagation})
		}

		if _, err = cc.kubeClient.AppsV1().Deployments(c.Status.NamespaceName).Update(dep); err != nil {
			return fmt.Errorf("failed to update Deployment %s: %v", dep.Name, err)
		}
	}

	return nil
}

// SecretOperation returns a wrapper struct to utilize a sorted slice instead of an unsorted map
type SecretOperation struct {
	name   string
	create resources.SecretCreator
}

// GetSecretCreatorOperations returns all SecretCreators that are currently in use
func GetSecretCreatorOperations(dockerPullConfigJSON []byte) []SecretOperation {
	return []SecretOperation{
		{resources.CASecretName, certificates.RootCA},
		{resources.FrontProxyCASecretName, certificates.FrontProxyCA},
		{resources.ImagePullSecretName, resources.ImagePullSecretCreator(resources.ImagePullSecretName, dockerPullConfigJSON)},
		{resources.ApiserverFrontProxyClientCertificateSecretName, apiserver.FrontProxyClientCertificate},
		{resources.EtcdTLSCertificateSecretName, etcd.TLSCertificate},
		{resources.ApiserverEtcdClientCertificateSecretName, apiserver.EtcdClientCertificate},
		{resources.ApiserverTLSSecretName, apiserver.TLSServingCertificate},
		{resources.KubeletClientCertificatesSecretName, apiserver.KubeletClientCertificate},
		{resources.ServiceAccountKeySecretName, apiserver.ServiceAccountKey},
		{resources.OpenVPNServerCertificatesSecretName, openvpn.TLSServingCertificate},
		{resources.OpenVPNClientCertificatesSecretName, openvpn.InternalClientCertificate},
		{resources.TokensSecretName, apiserver.TokenUsers},
		{resources.AdminKubeconfigSecretName, resources.AdminKubeconfig},
		{resources.SchedulerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil)},
		{resources.KubeletDnatControllerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil)},
		{resources.MachineControllerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil)},
		{resources.ControllerManagerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil)},
		{resources.KubeStateMetricsKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil)},
	}
}

func (cc *Controller) ensureSecrets(c *kubermaticv1.Cluster) error {
	creators := GetSecretCreatorOperations(cc.dockerPullConfigJSON)

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, op := range creators {

		var existing *corev1.Secret
		if existing, err = cc.secretLister.Secrets(c.Status.NamespaceName).Get(op.name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			se, err := op.create(data, nil)
			if err != nil {
				return fmt.Errorf("failed to build Secret %s: %v", op.name, err)
			}
			if se.Annotations == nil {
				se.Annotations = map[string]string{}
			}
			se.Annotations[checksumAnnotation] = getChecksumForMapStringByteSlice(se.Data)

			if _, err = cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName).Create(se); err != nil {
				return fmt.Errorf("failed to create Secret %s: %v", se.Name, err)
			}
			continue
		}

		se, err := op.create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build Secret: %v", err)
		}
		if se.Annotations == nil {
			se.Annotations = map[string]string{}
		}
		se.Annotations[checksumAnnotation] = getChecksumForMapStringByteSlice(se.Data)

		annotationVal, annotationExists := existing.Annotations[checksumAnnotation]
		if annotationExists && annotationVal == se.Annotations[checksumAnnotation] {
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
		openvpn.ServerClientConfigsConfigMap,
		prometheus.ConfigMap,
		dns.ConfigMap,
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
		if cm.Annotations == nil {
			cm.Annotations = map[string]string{}
		}
		cm.Annotations[checksumAnnotation] = getChecksumForMapStringString(cm.Data)

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
		if cm.Annotations == nil {
			cm.Annotations = map[string]string{}
		}
		cm.Annotations[checksumAnnotation] = getChecksumForMapStringString(cm.Data)

		if existing.Annotations == nil {
			existing.Annotations = map[string]string{}
		}
		annotationVal, annotationExists := existing.Annotations[checksumAnnotation]
		if annotationExists && annotationVal == cm.Annotations[checksumAnnotation] {
			continue
		}

		if _, err = cc.kubeClient.CoreV1().ConfigMaps(c.Status.NamespaceName).Update(cm); err != nil {
			return fmt.Errorf("failed to update ConfigMap %s: %v", cm.Name, err)
		}
	}

	return nil
}

func getChecksumForMapStringByteSlice(data map[string][]byte) string {
	// Maps are unordered so we have to sort it first
	var keyVals []string
	for k := range data {
		keyVals = append(keyVals, fmt.Sprintf("%s:%s", k, string(data[k])))
	}
	return getChecksumForStringSlice(keyVals)
}

func getChecksumForMapStringString(data map[string]string) string {
	// Maps are unordered so we have to sort it first
	var keyVals []string
	for k := range data {
		keyVals = append(keyVals, fmt.Sprintf("%s:%s", k, data[k]))
	}
	return getChecksumForStringSlice(keyVals)
}

func getChecksumForStringSlice(stringSlice []string) string {
	sort.Strings(stringSlice)
	buffer := bytes.NewBuffer(nil)
	for _, item := range stringSlice {
		buffer.WriteString(item)
	}
	return fmt.Sprintf("%v", crc32.ChecksumIEEE(buffer.Bytes()))
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

		if resources.DeepEqual(set, existing) {
			continue
		}

		// In case we update something immutable we need to delete&recreate. Creation happens on next sync
		if !equality.Semantic.DeepEqual(set.Spec.Selector.MatchLabels, existing.Spec.Selector.MatchLabels) {
			propagation := metav1.DeletePropagationForeground
			return cc.kubeClient.AppsV1().StatefulSets(c.Status.NamespaceName).Delete(set.Name, &metav1.DeleteOptions{PropagationPolicy: &propagation})
		}

		if _, err = cc.kubeClient.AppsV1().StatefulSets(c.Status.NamespaceName).Update(set); err != nil {
			return fmt.Errorf("failed to update StatefulSet %s: %v", set.Name, err)
		}
	}

	return nil
}

// GetPodDisruptionBudgetCreators returns all PodDisruptionBudgetCreators that are currently in use
func GetPodDisruptionBudgetCreators() []resources.PodDisruptionBudgetCreator {
	return []resources.PodDisruptionBudgetCreator{
		etcd.PodDisruptionBudget,
	}
}

func (cc *Controller) ensurePodDisruptionBudgets(c *kubermaticv1.Cluster) error {
	creators := GetPodDisruptionBudgetCreators()

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *v1beta1.PodDisruptionBudget
		pdb, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build PodDisruptionBudget: %v", err)
		}

		if existing, err = cc.podDisruptionBudgetLister.PodDisruptionBudgets(c.Status.NamespaceName).Get(pdb.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.PolicyV1beta1().PodDisruptionBudgets(c.Status.NamespaceName).Create(pdb); err != nil {
				return fmt.Errorf("failed to create PodDisruptionBudget %s: %v", pdb.Name, err)
			}
			continue
		}

		pdb, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build PodDisruptionBudget: %v", err)
		}

		if resources.DeepEqual(pdb, existing) {
			continue
		}

		if _, err = cc.kubeClient.PolicyV1beta1().PodDisruptionBudgets(c.Status.NamespaceName).Update(pdb); err != nil {
			return fmt.Errorf("failed to update PodDisruptionBudget %s: %v", pdb.Name, err)
		}
	}

	return nil
}

// GetCronJobCreators returns all CronJobCreators that are currently in use
func GetCronJobCreators() []resources.CronJobCreator {
	return []resources.CronJobCreator{
		etcd.CronJob,
	}
}

func (cc *Controller) ensureCronJobs(c *kubermaticv1.Cluster) error {
	creators := GetCronJobCreators()

	data, err := cc.getClusterTemplateData(c)
	if err != nil {
		return err
	}

	for _, create := range creators {
		var existing *batchv1beta1.CronJob
		job, err := create(data, nil)
		if err != nil {
			return fmt.Errorf("failed to build CronJob: %v", err)
		}

		if existing, err = cc.cronJobLister.CronJobs(c.Status.NamespaceName).Get(job.Name); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}

			if _, err = cc.kubeClient.BatchV1beta1().CronJobs(c.Status.NamespaceName).Create(job); err != nil {
				return fmt.Errorf("failed to create CronJob %s: %v", job.Name, err)
			}
			continue
		}

		job, err = create(data, existing.DeepCopy())
		if err != nil {
			return fmt.Errorf("failed to build CronJob: %v", err)
		}

		if resources.DeepEqual(job, existing) {
			continue
		}

		if _, err = cc.kubeClient.BatchV1beta1().CronJobs(c.Status.NamespaceName).Update(job); err != nil {
			return fmt.Errorf("failed to update CronJob %s: %v", job.Name, err)
		}
	}

	return nil
}
