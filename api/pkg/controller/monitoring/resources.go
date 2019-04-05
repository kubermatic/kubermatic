package monitoring

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/kubestatemetrics"
	"github.com/kubermatic/kubermatic/api/pkg/resources/prometheus"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *Controller) getClusterTemplateData(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (*resources.TemplateData, error) {
	dc, found := c.dcs[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	return resources.NewTemplateData(
		ctx,
		client,
		cluster,
		&dc,
		c.dc,
		c.overwriteRegistry,
		c.nodePortRange,
		c.nodeAccessNetwork,
		c.etcdDiskSize,
		c.monitoringScrapeAnnotationPrefix,
		c.inClusterPrometheusRulesFile,
		c.inClusterPrometheusDisableDefaultRules,
		c.inClusterPrometheusDisableDefaultScrapingConfigs,
		c.inClusterPrometheusScrapingConfigsFile,
		"",
		"",
		"",
	), nil
}

func (c *Controller) ensureRoles(cluster *kubermaticv1.Cluster) error {
	getters := []reconciling.NamedRoleCreatorGetter{
		prometheus.RoleCreator(),
	}

	return reconciling.ReconcileRoles(getters, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

func (c *Controller) ensureRoleBindings(cluster *kubermaticv1.Cluster) error {
	getters := []reconciling.NamedRoleBindingCreatorGetter{
		prometheus.RoleBindingCreator(cluster.Status.NamespaceName),
	}

	return reconciling.ReconcileRoleBindings(getters, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(data *resources.TemplateData) []reconciling.NamedDeploymentCreatorGetter {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubestatemetrics.DeploymentCreator(data),
	}

	return creators
}

func (c *Controller) ensureDeployments(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data)

	return reconciling.ReconcileDeployments(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetSecretCreatorOperations returns all SecretCreators that are currently in use
func GetSecretCreatorOperations(data *resources.TemplateData) []reconciling.NamedSecretCreatorGetter {
	return []reconciling.NamedSecretCreatorGetter{
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

	if err := reconciling.ReconcileSecrets(namedSecretCreatorGetters, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache); err != nil {
		return fmt.Errorf("failed to ensure that the Secret exists: %v", err)
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use
func GetConfigMapCreators(data *resources.TemplateData) []reconciling.NamedConfigMapCreatorGetter {
	return []reconciling.NamedConfigMapCreatorGetter{
		prometheus.ConfigMapCreator(data),
	}
}

func (c *Controller) ensureConfigMaps(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	if err := reconciling.ReconcileConfigMaps(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %v", err)
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use
func GetStatefulSetCreators(data *resources.TemplateData) []reconciling.NamedStatefulSetCreatorGetter {
	return []reconciling.NamedStatefulSetCreatorGetter{
		prometheus.StatefulSetCreator(data),
	}
}

func (c *Controller) ensureStatefulSets(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetCreators(data)

	return reconciling.ReconcileStatefulSets(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
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
	return reconciling.ReconcileVerticalPodAutoscalers(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache)
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceCreators(data *resources.TemplateData) []reconciling.NamedServiceCreatorGetter {
	return []reconciling.NamedServiceCreatorGetter{
		prometheus.ServiceCreator(data),
	}
}

func (c *Controller) ensureServices(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators(data)

	return reconciling.ReconcileServices(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceAccountCreators() []reconciling.NamedServiceAccountCreatorGetter {
	return []reconciling.NamedServiceAccountCreatorGetter{
		prometheus.ServiceAccountCreator(),
	}
}

func (c *Controller) ensureServiceAccounts(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceAccountCreators()

	return reconciling.ReconcileServiceAccounts(creators, cluster.Status.NamespaceName, c.dynamicClient, c.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}
