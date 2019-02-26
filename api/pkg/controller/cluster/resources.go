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
	metricsserver "github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/scheduler"
	"github.com/kubermatic/kubermatic/api/pkg/resources/usercluster"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1beta "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta1"
)

const (
	NodeDeletionFinalizer = "kubermatic.io/delete-nodes"
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

	// check that all StatefulSets are created
	if err := cc.ensureVerticalPodAutoscalers(cluster, data); err != nil {
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
		cc.oidcCAFile,
		cc.oidcIssuerURL,
		cc.oidcIssuerClientID,
		cc.enableEtcdDataCorruptionChecks,
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
func GetServiceCreators(data *resources.TemplateData) []resources.ServiceCreator {
	return []resources.ServiceCreator{
		apiserver.InternalServiceCreator(),
		apiserver.ExternalServiceCreator(),
		openvpn.ServiceCreator(),
		etcd.ServiceCreator(data),
		dns.ServiceCreator(),
		machinecontroller.ServiceCreator(),
		metricsserver.ServiceCreator(),
	}
}

func (cc *Controller) ensureServices(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators(data)

	return resources.ReconcileServices(creators, c.Status.NamespaceName, cc.dynamicClient, cc.dynamicCache, resources.ClusterRefWrapper(c))
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(data resources.DeploymentDataProvider) []resources.DeploymentCreator {
	creators := []resources.DeploymentCreator{
		machinecontroller.DeploymentCreator(data),
		machinecontroller.WebhookDeploymentCreator(data),
		openvpn.DeploymentCreator(data),
		dns.DeploymentCreator(data),
		metricsserver.DeploymentCreator(data),
		usercluster.DeploymentCreator(data),
	}

	if cluster := data.Cluster(); cluster != nil && cluster.Annotations["kubermatic.io/openshift"] == "" {
		creators = append(creators, apiserver.DeploymentCreator(data))
		creators = append(creators, scheduler.DeploymentCreator(data))
		creators = append(creators, controllermanager.DeploymentCreator(data))
	}

	if data.Cluster() != nil && len(data.Cluster().Spec.MachineNetworks) > 0 {
		creators = append(creators, ipamcontroller.DeploymentCreator(data))
	}

	return creators
}

func (cc *Controller) ensureDeployments(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data)

	return resources.ReconcileDeployments(creators, cluster.Status.NamespaceName, cc.dynamicClient, cc.dynamicCache, resources.ClusterRefWrapper(cluster))
}

// GetSecretCreators returns all SecretCreators that are currently in use
func GetSecretCreators(data *resources.TemplateData) []resources.NamedSecretCreatorGetter {
	creators := []resources.NamedSecretCreatorGetter{
		certificates.RootCACreator(data),
		openvpn.CACreator(),
		certificates.FrontProxyCACreator(data),
		resources.ImagePullSecretCreator(data.DockerPullConfigJSON),
		apiserver.FrontProxyClientCertificateCreator(data),
		etcd.TLSCertificateCreator(data),
		apiserver.EtcdClientCertificateCreator(data),
		apiserver.TLSServingCertificateCreator(data),
		apiserver.KubeletClientCertificateCreator(data),
		apiserver.ServiceAccountKeyCreator(),
		openvpn.TLSServingCertificateCreator(data),
		openvpn.InternalClientCertificateCreator(data),
		apiserver.TokenUsersCreator(data),
		resources.AdminKubeconfigCreator(data),
		machinecontroller.TLSServingCertificateCreator(data),

		// Kubeconfigs
		resources.GetInternalKubeconfigCreator(resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.MetricsServerKubeconfigSecretName, resources.MetricsServerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.UserClusterControllerKubeconfigSecretName, resources.UserClusterControllerCertUsername, []string{"system:masters"}, data),
	}

	if len(data.OIDCCAFile()) > 0 {
		creators = append(creators, apiserver.DexCACertificateCreator(data))
	}

	if len(data.Cluster().Spec.MachineNetworks) > 0 {
		creators = append(creators, resources.GetInternalKubeconfigCreator(resources.IPAMControllerKubeconfigSecretName, resources.IPAMControllerCertUsername, nil, data))
	}
	return creators
}

func (cc *Controller) ensureSecrets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretCreatorGetters := GetSecretCreators(data)

	if err := resources.ReconcileSecrets(namedSecretCreatorGetters, c.Status.NamespaceName, cc.dynamicClient, cc.dynamicCache); err != nil {
		return fmt.Errorf("failed to ensure that the Secret exists: %v", err)
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use
func GetConfigMapCreators(data *resources.TemplateData) []resources.NamedConfigMapCreatorGetter {
	return []resources.NamedConfigMapCreatorGetter{
		cloudconfig.ConfigMapCreator(data),
		openvpn.ServerClientConfigsConfigMapCreator(data),
		dns.ConfigMapCreator(data),
	}
}

func (cc *Controller) ensureConfigMaps(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	if err := resources.ReconcileConfigMaps(creators, c.Status.NamespaceName, cc.dynamicClient, cc.dynamicCache); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %v", err)
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use
func GetStatefulSetCreators(data *resources.TemplateData) []resources.StatefulSetCreator {
	return []resources.StatefulSetCreator{
		etcd.StatefulSetCreator(data),
	}
}

// GetPodDisruptionBudgetCreators returns all PodDisruptionBudgetCreators that are currently in use
func GetPodDisruptionBudgetCreators(data *resources.TemplateData) []resources.PodDisruptionBudgetCreator {
	return []resources.PodDisruptionBudgetCreator{
		etcd.PodDisruptionBudgetCreator(data),
		apiserver.PodDisruptionBudgetCreator(data),
		metricsserver.PodDisruptionBudgetCreator(data),
	}
}

func (cc *Controller) ensurePodDisruptionBudgets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetPodDisruptionBudgetCreators(data)

	if err := resources.ReconcilePodDisruptionBudgets(creators, c.Status.NamespaceName, cc.dynamicClient, cc.dynamicCache); err != nil {
		return fmt.Errorf("failed to ensure that the PodDisruptionBudget exists: %v", err)
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

func (cc *Controller) ensureVerticalPodAutoscalers(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	controlPlaneDeploymentNames := []string{
		"dns-resolver",
		"machine-controller",
		"machine-controller-webhook",
		"metrics-server",
		"openvpn-server",
	}
	if c.Annotations["kubermatic.io/openshift"] == "" {
		controlPlaneDeploymentNames = append(controlPlaneDeploymentNames, "apiserver", "controller-manager", "scheduler")
	}
	creators, err := resources.GetVerticalPodAutoscalersForAll(controlPlaneDeploymentNames, []string{"etcd"}, c.Status.NamespaceName, cc.dynamicCache)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %v", err)
	}

	if !cc.enableVPA {
		// If the feature is disabled, we just wrap the create function to disable the VPA.
		// This is easier than passing a bool to all required functions.
		for i, create := range creators {
			creators[i] = func(existing *autoscalingv1beta.VerticalPodAutoscaler) (*autoscalingv1beta.VerticalPodAutoscaler, error) {
				vpa, err := create(existing)
				if err != nil {
					return nil, err
				}
				if vpa.Spec.UpdatePolicy == nil {
					vpa.Spec.UpdatePolicy = &autoscalingv1beta.PodUpdatePolicy{}
				}
				mode := autoscalingv1beta.UpdateModeOff
				vpa.Spec.UpdatePolicy.UpdateMode = &mode

				return vpa, nil
			}
		}
	}

	return resources.ReconcileVerticalPodAutoscalers(creators, c.Status.NamespaceName, cc.dynamicClient, cc.dynamicCache)
}

func (cc *Controller) ensureStatefulSets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetCreators(data)

	return resources.ReconcileStatefulSets(creators, c.Status.NamespaceName, cc.dynamicClient, cc.dynamicCache, resources.ClusterRefWrapper(c))
}
