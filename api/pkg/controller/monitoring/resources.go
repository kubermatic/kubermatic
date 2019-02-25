package monitoring

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/kubestatemetrics"
	"github.com/kubermatic/kubermatic/api/pkg/resources/prometheus"
)

func (c *Controller) getClusterTemplateData(cluster *kubermaticv1.Cluster) (*resources.TemplateData, error) {
	dc, found := c.dcs[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	return resources.NewTemplateData(
		cluster,
		&dc,
		c.dc,
		c.secretLister,
		c.configMapLister,
		c.serviceLister,
		c.overwriteRegistry,
		c.nodePortRange,
		c.nodeAccessNetwork,
		c.etcdDiskSize,
		c.monitoringScrapeAnnotationPrefix,
		c.inClusterPrometheusRulesFile,
		c.inClusterPrometheusDisableDefaultRules,
		c.inClusterPrometheusDisableDefaultScrapingConfigs,
		c.inClusterPrometheusScrapingConfigsFile,
		c.dockerPullConfigJSON,
		"",
		"",
		"",
		false,
	), nil
}

func (c *Controller) ensureServiceAccounts(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := []resources.ServiceAccountCreator{
		prometheus.ServiceAccount,
	}

	for _, create := range creators {
		if err := resources.EnsureServiceAccount(data, create, c.serviceAccountLister.ServiceAccounts(cluster.Status.NamespaceName), c.kubeClient.CoreV1().ServiceAccounts(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the ServiceAccount exists: %v", err)
		}
	}

	return nil
}

func (c *Controller) ensureRoles(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := []resources.RoleCreator{
		prometheus.Role,
	}

	for _, create := range creators {
		if err := resources.EnsureRole(data, create, c.roleLister.Roles(cluster.Status.NamespaceName), c.kubeClient.RbacV1().Roles(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the Role exists: %v", err)
		}
	}

	return nil
}

func (c *Controller) ensureRoleBindings(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := []resources.RoleBindingCreator{
		prometheus.RoleBinding,
	}

	for _, create := range creators {
		if err := resources.EnsureRoleBinding(data, create, c.roleBindingLister.RoleBindings(cluster.Status.NamespaceName), c.kubeClient.RbacV1().RoleBindings(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the RoleBinding exists: %v", err)
		}
	}

	return nil
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(data resources.DeploymentDataProvider) []resources.DeploymentCreator {
	creators := []resources.DeploymentCreator{
		kubestatemetrics.DeploymentCreator(data),
	}

	return creators
}

func (c *Controller) ensureDeployments(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data)

	return resources.ReconcileDeployments(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, resources.ClusterRefWrapper(cluster))
}

// GetSecretCreatorOperations returns all SecretCreators that are currently in use
func GetSecretCreatorOperations(data *resources.TemplateData) []resources.NamedSecretCreatorGetter {
	return []resources.NamedSecretCreatorGetter{
		certificates.GetClientCertificateCreator(
			resources.PrometheusApiserverClientCertificateSecretName,
			resources.PrometheusCertUsername, nil,
			resources.PrometheusClientCertificateCertSecretKey,
			resources.PrometheusClientCertificateKeySecretKey,
			data.GetRootCA,
		),
	}
}

func (c *Controller) ensureSecrets(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretCreatorGetters := GetSecretCreatorOperations(data)

	if err := resources.ReconcileSecrets(namedSecretCreatorGetters, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache); err != nil {
		return fmt.Errorf("failed to ensure that the Secret exists: %v", err)
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use
func GetConfigMapCreators(data *resources.TemplateData) []resources.NamedConfigMapCreatorGetter {
	return []resources.NamedConfigMapCreatorGetter{
		prometheus.ConfigMapCreator(data),
	}
}

func (c *Controller) ensureConfigMaps(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	if err := resources.ReconcileConfigMaps(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %v", err)
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use
func GetStatefulSetCreators(data *resources.TemplateData) []resources.StatefulSetCreator {
	return []resources.StatefulSetCreator{
		prometheus.StatefulSetCreator(data),
	}
}

func (c *Controller) ensureStatefulSets(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetCreators(data)

	return resources.ReconcileStatefulSets(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, resources.ClusterRefWrapper(cluster))
}

func (c *Controller) ensureVerticalPodAutoscalers(cluster *kubermaticv1.Cluster) error {
	creators, err := resources.GetVerticalPodAutoscalersForAll([]string{
		"kube-state-metrics",
	},
		[]string{
			"prometheus",
		}, cluster.Status.NamespaceName,
		c.dynamicCache)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %v", err)
	}
	return resources.ReconcileVerticalPodAutoscalers(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache)
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceCreators(data *resources.TemplateData) []resources.ServiceCreator {
	return []resources.ServiceCreator{
		prometheus.ServiceCreator(data),
	}
}

func (c *Controller) ensureServices(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators(data)

	return resources.ReconcileServices(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, resources.ClusterRefWrapper(cluster))
}
