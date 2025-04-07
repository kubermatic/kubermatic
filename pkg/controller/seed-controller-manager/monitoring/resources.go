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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/kubestatemetrics"
	"k8c.io/kubermatic/v2/pkg/resources/prometheus"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"
	"k8c.io/reconciler/pkg/reconciling"

	"k8s.io/apimachinery/pkg/api/resource"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) getClusterTemplateData(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) (*resources.TemplateData, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return nil, err
	}
	config, err := r.configGetter(ctx)
	if err != nil {
		return nil, err
	}

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	konnectivityEnabled := cluster.Spec.ClusterNetwork.KonnectivityEnabled != nil && *cluster.Spec.ClusterNetwork.KonnectivityEnabled //nolint:staticcheck

	return resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithClient(client).
		WithCluster(cluster).
		WithDatacenter(&datacenter).
		WithSeed(seed.DeepCopy()).
		WithKubermaticConfiguration(config.DeepCopy()).
		WithOverwriteRegistry(r.overwriteRegistry).
		WithNodePortRange(config.Spec.UserCluster.NodePortRange).
		WithNodeAccessNetwork(r.nodeAccessNetwork).
		WithEtcdDiskSize(resource.Quantity{}).
		WithBackupPeriod(20 * time.Minute).
		WithVersions(r.versions).
		WithKonnectivityEnabled(konnectivityEnabled).
		Build(), nil
}

func (r *Reconciler) ensureRoles(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	getters := []reconciling.NamedRoleReconcilerFactory{
		prometheus.RoleReconciler(),
	}

	return reconciling.ReconcileRoles(ctx, getters, cluster.Status.NamespaceName, r)
}

func (r *Reconciler) ensureRoleBindings(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	getters := []reconciling.NamedRoleBindingReconcilerFactory{
		prometheus.RoleBindingReconciler(cluster.Status.NamespaceName),
	}

	return reconciling.ReconcileRoleBindings(ctx, getters, cluster.Status.NamespaceName, r)
}

// GetDeploymentReconcilers returns all DeploymentReconcilers that are currently in use.
func GetDeploymentReconcilers(data *resources.TemplateData) []reconciling.NamedDeploymentReconcilerFactory {
	creators := []reconciling.NamedDeploymentReconcilerFactory{
		kubestatemetrics.DeploymentReconciler(data),
	}

	return creators
}

func (r *Reconciler) ensureDeployments(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentReconcilers(data)

	modifiers := []reconciling.ObjectModifier{
		modifier.RelatedRevisionsLabels(ctx, r),
		modifier.ControlplaneComponent(cluster),
	}

	return reconciling.ReconcileDeployments(ctx, creators, cluster.Status.NamespaceName, r, modifiers...)
}

// GetSecretReconcilerOperations returns all SecretReconcilers that are currently in use.
func GetSecretReconcilerOperations(data *resources.TemplateData) []reconciling.NamedSecretReconcilerFactory {
	return []reconciling.NamedSecretReconcilerFactory{
		certificates.GetClientCertificateReconciler(
			resources.PrometheusApiserverClientCertificateSecretName,
			resources.PrometheusCertUsername, nil,
			resources.PrometheusClientCertificateCertSecretKey,
			resources.PrometheusClientCertificateKeySecretKey,
			data.GetRootCA,
		),
	}
}

func (r *Reconciler) ensureSecrets(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretReconcilerFactories := GetSecretReconcilerOperations(data)

	if err := reconciling.ReconcileSecrets(ctx, namedSecretReconcilerFactories, cluster.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure that the Secret exists: %w", err)
	}

	return nil
}

// GetConfigMapReconcilers returns all ConfigMapReconcilers that are currently in use.
func GetConfigMapReconcilers(data *resources.TemplateData) []reconciling.NamedConfigMapReconcilerFactory {
	return []reconciling.NamedConfigMapReconcilerFactory{
		prometheus.ConfigMapReconciler(data),
	}
}

func (r *Reconciler) ensureConfigMaps(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapReconcilers(data)

	if err := reconciling.ReconcileConfigMaps(ctx, creators, cluster.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %w", err)
	}

	return nil
}

// GetStatefulSetReconcilers returns all StatefulSetReconcilers that are currently in use.
func GetStatefulSetReconcilers(data *resources.TemplateData) []reconciling.NamedStatefulSetReconcilerFactory {
	return []reconciling.NamedStatefulSetReconcilerFactory{
		prometheus.StatefulSetReconciler(data),
	}
}

func (r *Reconciler) ensureStatefulSets(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetReconcilers(data)

	modifiers := []reconciling.ObjectModifier{
		modifier.RelatedRevisionsLabels(ctx, r),
		modifier.ControlplaneComponent(cluster),
	}

	return reconciling.ReconcileStatefulSets(ctx, creators, cluster.Status.NamespaceName, r, modifiers...)
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
		r,
		deploymentNames,
		statefulSetNames,
		cluster.Status.NamespaceName,
		r.features.VPA)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %w", err)
	}
	return kkpreconciling.ReconcileVerticalPodAutoscalers(ctx, creators, cluster.Status.NamespaceName, r)
}

// GetServiceReconcilers returns all service creators that are currently in use.
func GetServiceReconcilers(data *resources.TemplateData) []reconciling.NamedServiceReconcilerFactory {
	return []reconciling.NamedServiceReconcilerFactory{
		prometheus.ServiceReconciler(data),
	}
}

func (r *Reconciler) ensureServices(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceReconcilers(data)

	return reconciling.ReconcileServices(ctx, creators, cluster.Status.NamespaceName, r)
}

// GetServiceReconcilers returns all service creators that are currently in use.
func GetServiceAccountReconcilers() []reconciling.NamedServiceAccountReconcilerFactory {
	return []reconciling.NamedServiceAccountReconcilerFactory{
		prometheus.ServiceAccountReconciler(),
	}
}

func (r *Reconciler) ensureServiceAccounts(ctx context.Context, cluster *kubermaticv1.Cluster, _ *resources.TemplateData) error {
	creators := GetServiceAccountReconcilers()

	return reconciling.ReconcileServiceAccounts(ctx, creators, cluster.Status.NamespaceName, r)
}
