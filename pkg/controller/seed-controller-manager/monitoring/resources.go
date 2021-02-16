/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package monitoring

import (
	"context"
	"fmt"
	"time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/kubestatemetrics"
	"k8c.io/kubermatic/v2/pkg/resources/prometheus"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8s.io/apimachinery/pkg/api/resource"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) getClusterTemplateData(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (*resources.TemplateData, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return nil, err
	}

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	return resources.NewTemplateData(
		ctx,
		client,
		cluster,
		&datacenter,
		seed.DeepCopy(),
		r.overwriteRegistry,
		r.nodePortRange,
		r.nodeAccessNetwork,
		resource.Quantity{},
		r.monitoringScrapeAnnotationPrefix,
		r.inClusterPrometheusRulesFile,
		r.inClusterPrometheusDisableDefaultRules,
		r.inClusterPrometheusDisableDefaultScrapingConfigs,
		r.inClusterPrometheusScrapingConfigsFile,
		"",
		"",
		"",
		false,
		"",
		"",
		"",
		20*time.Minute,
		false,
		r.versions,
	), nil
}

func (r *Reconciler) ensureRoles(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	getters := []reconciling.NamedRoleCreatorGetter{
		prometheus.RoleCreator(),
	}

	return reconciling.ReconcileRoles(ctx, getters, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

func (r *Reconciler) ensureRoleBindings(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	getters := []reconciling.NamedRoleBindingCreatorGetter{
		prometheus.RoleBindingCreator(cluster.Status.NamespaceName),
	}

	return reconciling.ReconcileRoleBindings(ctx, getters, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(data *resources.TemplateData) []reconciling.NamedDeploymentCreatorGetter {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		kubestatemetrics.DeploymentCreator(data),
	}

	return creators
}

func (r *Reconciler) ensureDeployments(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data)

	return reconciling.ReconcileDeployments(ctx, creators, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
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

func (r *Reconciler) ensureSecrets(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretCreatorGetters := GetSecretCreatorOperations(data)

	if err := reconciling.ReconcileSecrets(ctx, namedSecretCreatorGetters, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster))); err != nil {
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

func (r *Reconciler) ensureConfigMaps(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	if err := reconciling.ReconcileConfigMaps(ctx, creators, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster))); err != nil {
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

func (r *Reconciler) ensureStatefulSets(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetCreators(data)

	return reconciling.ReconcileStatefulSets(ctx, creators, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

func (r *Reconciler) ensureVerticalPodAutoscalers(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	deploymentNames := []string{
		resources.KubeStateMetricsDeploymentName,
	}
	statefulSetNames := []string{
		resources.PrometheusStatefulSetName,
	}

	creators, err := resources.GetVerticalPodAutoscalersForAll(
		ctx,
		r.Client,
		deploymentNames,
		statefulSetNames,
		cluster.Status.NamespaceName,
		r.features.VPA)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %v", err)
	}
	return reconciling.ReconcileVerticalPodAutoscalers(ctx, creators, cluster.Status.NamespaceName, r.Client)
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceCreators(data *resources.TemplateData) []reconciling.NamedServiceCreatorGetter {
	return []reconciling.NamedServiceCreatorGetter{
		prometheus.ServiceCreator(data),
	}
}

func (r *Reconciler) ensureServices(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators(data)

	return reconciling.ReconcileServices(ctx, creators, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceAccountCreators() []reconciling.NamedServiceAccountCreatorGetter {
	return []reconciling.NamedServiceAccountCreatorGetter{
		prometheus.ServiceAccountCreator(),
	}
}

func (r *Reconciler) ensureServiceAccounts(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceAccountCreators()

	return reconciling.ReconcileServiceAccounts(ctx, creators, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}
