package monitoring

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
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
		c.inClusterPrometheusRulesFile,
		c.inClusterPrometheusDisableDefaultRules,
		c.inClusterPrometheusDisableDefaultScrapingConfigs,
		c.inClusterPrometheusScrapingConfigsFile,
		c.dockerPullConfigJSON,
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
func GetDeploymentCreators(c *kubermaticv1.Cluster) []resources.DeploymentCreator {
	creators := []resources.DeploymentCreator{
		kubestatemetrics.Deployment,
	}

	return creators
}

func (c *Controller) ensureDeployments(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(cluster)

	for _, create := range creators {
		if err := resources.EnsureDeployment(data, create, c.deploymentLister.Deployments(cluster.Status.NamespaceName), c.kubeClient.AppsV1().Deployments(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the Deployment exists: %v", err)
		}
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use
func GetConfigMapCreators(data *resources.TemplateData) []resources.ConfigMapCreator {
	return []resources.ConfigMapCreator{
		prometheus.ConfigMapCreator(data),
	}
}

func (c *Controller) ensureConfigMaps(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	for _, create := range creators {
		if err := resources.EnsureConfigMap(create, c.configMapLister.ConfigMaps(cluster.Status.NamespaceName), c.kubeClient.CoreV1().ConfigMaps(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the ConfigMap exists: %v", err)
		}
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use
func GetStatefulSetCreators() []resources.StatefulSetCreator {
	return []resources.StatefulSetCreator{
		prometheus.StatefulSet,
	}
}

func (c *Controller) ensureStatefulSets(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetCreators()

	for _, create := range creators {
		if err := resources.EnsureStatefulSet(data, create, c.statefulSetLister.StatefulSets(cluster.Status.NamespaceName), c.kubeClient.AppsV1().StatefulSets(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the StatefulSet exists: %v", err)
		}
	}

	return nil
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceCreators() []resources.ServiceCreator {
	return []resources.ServiceCreator{
		prometheus.Service,
	}
}

func (c *Controller) ensureServices(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators()

	for _, create := range creators {
		if err := resources.EnsureService(data, create, c.serviceLister.Services(cluster.Status.NamespaceName), c.kubeClient.CoreV1().Services(cluster.Status.NamespaceName)); err != nil {
			return fmt.Errorf("failed to ensure that the service exists: %v", err)
		}
	}

	return nil
}
