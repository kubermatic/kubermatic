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
	"k8s.io/apimachinery/pkg/api/resource"

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
		resource.Quantity{},
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

func (c *Controller) ensureRoles(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	getters := []reconciling.NamedRoleCreatorGetter{
		prometheus.RoleCreator(),
	}

	return reconciling.ReconcileRoles(ctx, getters, cluster.Status.NamespaceName, c.dynamicClient, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

func (c *Controller) ensureRoleBindings(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	getters := []reconciling.NamedRoleBindingCreatorGetter{
		prometheus.RoleBindingCreator(cluster.Status.NamespaceName),
	}

	return reconciling.ReconcileRoleBindings(ctx, getters, cluster.Status.NamespaceName, c.dynamicClient, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(data *resources.TemplateData) []reconciling.NamedDeploymentCreatorGetter {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubestatemetrics.DeploymentCreator(data),
	}

	return creators
}

func (c *Controller) ensureDeployments(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data)

	return reconciling.ReconcileDeployments(ctx, creators, cluster.Status.NamespaceName, c.dynamicClient, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
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

func (c *Controller) ensureSecrets(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretCreatorGetters := GetSecretCreatorOperations(data)

	if err := reconciling.ReconcileSecrets(ctx, namedSecretCreatorGetters, cluster.Status.NamespaceName, c.dynamicClient); err != nil {
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

func (c *Controller) ensureConfigMaps(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	if err := reconciling.ReconcileConfigMaps(ctx, creators, cluster.Status.NamespaceName, c.dynamicClient); err != nil {
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

func (c *Controller) ensureStatefulSets(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetCreators(data)

	return reconciling.ReconcileStatefulSets(ctx, creators, cluster.Status.NamespaceName, c.dynamicClient, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

func (c *Controller) ensureVerticalPodAutoscalers(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	deploymentNames := []string{
		resources.KubeStateMetricsDeploymentName,
	}
	statefulSetNames := []string{
		resources.PrometheusStatefulSetName,
	}

	creators, err := resources.GetVerticalPodAutoscalersForAll(
		ctx,
		c.dynamicClient,
		deploymentNames,
		statefulSetNames,
		cluster.Status.NamespaceName,
	)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %v", err)
	}
	return reconciling.ReconcileVerticalPodAutoscalers(ctx, creators, cluster.Status.NamespaceName, c.dynamicClient)
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceCreators(data *resources.TemplateData) []reconciling.NamedServiceCreatorGetter {
	return []reconciling.NamedServiceCreatorGetter{
		prometheus.ServiceCreator(data),
	}
}

func (c *Controller) ensureServices(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators(data)

	return reconciling.ReconcileServices(ctx, creators, cluster.Status.NamespaceName, c.dynamicClient, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceAccountCreators() []reconciling.NamedServiceAccountCreatorGetter {
	return []reconciling.NamedServiceAccountCreatorGetter{
		prometheus.ServiceAccountCreator(),
	}
}

func (c *Controller) ensureServiceAccounts(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceAccountCreators()

	return reconciling.ReconcileServiceAccounts(ctx, creators, cluster.Status.NamespaceName, c.dynamicClient, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}
