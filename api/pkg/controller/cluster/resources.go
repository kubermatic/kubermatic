package cluster

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/controllermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/dns"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/ipamcontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/scheduler"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nodeDeletionFinalizer = "kubermatic.io/delete-nodes"
)

func (cc *Controller) ensureResourcesAreDeployed(cluster *kubermaticv1.Cluster) error {
	data, err := cc.getClusterTemplateData(cluster)
	if err != nil {
		return err
	}

	// check that all services are available
	if err := cc.ensureServices(cluster, data); err != nil {
		return err
	}

	// check that all secrets are available // New way of handling secrets
	if err := cc.ensureSecrets(cluster, data); err != nil {
		return err
	}

	// check that all ConfigMaps are available
	if err := cc.ensureConfigMaps(cluster, data); err != nil {
		return err
	}

	// check that all Deployments are available
	if err := cc.ensureDeployments(cluster, data); err != nil {
		return err
	}

	// check that all StatefulSets are created
	if err := cc.ensureStatefulSets(cluster, data); err != nil {
		return err
	}

	// check that all CronJobs are created
	if err := cc.ensureCronJobs(cluster, data); err != nil {
		return err
	}

	// check that all PodDisruptionBudgets are created
	if err := cc.ensurePodDisruptionBudgets(cluster, data); err != nil {
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
		cc.monitoringScrapeAnnotationPrefix,
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
		openvpn.Service,
		etcd.Service,
		dns.Service,
		machinecontroller.Service,
		metricsserver.Service,
	}
}

func (cc *Controller) ensureServices(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators()

	for _, create := range creators {
		if err := resources.EnsureService(data, create, cc.serviceLister.Services(c.Status.NamespaceName), cc.kubeClient.CoreV1().Services(c.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the service exists: %v", err)
		}
	}

	return nil
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(c *kubermaticv1.Cluster) []resources.DeploymentCreator {
	creators := []resources.DeploymentCreator{
		machinecontroller.Deployment,
		machinecontroller.WebhookDeployment,
		openvpn.Deployment,
		apiserver.Deployment,
		scheduler.Deployment,
		controllermanager.Deployment,
		dns.Deployment,
		metricsserver.Deployment,
	}

	if c != nil && len(c.Spec.MachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.Deployment)
	}

	return creators
}

func (cc *Controller) ensureDeployments(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(c)

	for _, create := range creators {
		if err := resources.EnsureDeployment(data, create, cc.deploymentLister.Deployments(c.Status.NamespaceName), cc.kubeClient.AppsV1().Deployments(c.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the Deployment exists: %v", err)
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
		{resources.OpenVPNCASecretName, openvpn.CertificateAuthority},
		{resources.OpenVPNServerCertificatesSecretName, openvpn.TLSServingCertificate},
		{resources.OpenVPNClientCertificatesSecretName, openvpn.InternalClientCertificate},
		{resources.TokensSecretName, apiserver.TokenUsers},
		{resources.AdminKubeconfigSecretName, resources.AdminKubeconfig},
		{resources.SchedulerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil)},
		{resources.KubeletDnatControllerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil)},
		{resources.MachineControllerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil)},
		{resources.ControllerManagerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil)},
		{resources.KubeStateMetricsKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil)},
		{resources.MachineControllerWebhookServingCertSecretName, machinecontroller.TLSServingCertificate},
		{resources.MetricsServerKubeconfigSecretName, resources.GetInternalKubeconfigCreator(resources.MetricsServerKubeconfigSecretName, resources.MetricsServerCertUsername, nil)},
	}
}

func (cc *Controller) ensureSecrets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	operations := GetSecretCreatorOperations(cc.dockerPullConfigJSON)

	for _, op := range operations {
		if err := resources.EnsureSecret(op.name, data, op.create, cc.secretLister.Secrets(c.Status.NamespaceName), cc.kubeClient.CoreV1().Secrets(c.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the Secret exists: %v", err)
		}
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use
func GetConfigMapCreators(data *resources.TemplateData) []resources.ConfigMapCreator {
	return []resources.ConfigMapCreator{
		cloudconfig.ConfigMapCreator(data),
		openvpn.ServerClientConfigsConfigMapCreator(data),
		dns.ConfigMapCreator(data),
	}
}

func (cc *Controller) ensureConfigMaps(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	for _, create := range creators {
		if err := resources.EnsureConfigMap(create, cc.configMapLister.ConfigMaps(c.Status.NamespaceName), cc.kubeClient.CoreV1().ConfigMaps(c.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the ConfigMap exists: %v", err)
		}
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use
func GetStatefulSetCreators() []resources.StatefulSetCreator {
	return []resources.StatefulSetCreator{
		etcd.StatefulSet,
	}
}

func (cc *Controller) ensureStatefulSets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetCreators()

	for _, create := range creators {
		if err := resources.EnsureStatefulSet(data, create, cc.statefulSetLister.StatefulSets(c.Status.NamespaceName), cc.kubeClient.AppsV1().StatefulSets(c.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the StatefulSet exists: %v", err)
		}
	}

	return nil
}

// GetPodDisruptionBudgetCreators returns all PodDisruptionBudgetCreators that are currently in use
func GetPodDisruptionBudgetCreators() []resources.PodDisruptionBudgetCreator {
	return []resources.PodDisruptionBudgetCreator{
		etcd.PodDisruptionBudget,
		apiserver.PodDisruptionBudget,
	}
}

func (cc *Controller) ensurePodDisruptionBudgets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetPodDisruptionBudgetCreators()

	for _, create := range creators {
		if err := resources.EnsurePodDisruptionBudget(data, create, cc.podDisruptionBudgetLister.PodDisruptionBudgets(c.Status.NamespaceName), cc.kubeClient.PolicyV1beta1().PodDisruptionBudgets(c.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the PodDisruptionBudget exists: %v", err)
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

func (cc *Controller) ensureCronJobs(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetCronJobCreators()

	for _, create := range creators {
		if err := resources.EnsureCronJob(data, create, cc.cronJobLister.CronJobs(c.Status.NamespaceName), cc.kubeClient.BatchV1beta1().CronJobs(c.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the CronJob exists: %v", err)
		}
	}

	return nil
}
