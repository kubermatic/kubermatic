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

package kubernetes

import (
	"context"
	"fmt"
	"time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/resources/clusterautoscaler"
	"k8c.io/kubermatic/v2/pkg/resources/controllermanager"
	"k8c.io/kubermatic/v2/pkg/resources/dns"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"
	"k8c.io/kubermatic/v2/pkg/resources/gatekeeper"
	"k8c.io/kubermatic/v2/pkg/resources/konnectivity"
	kubernetesdashboard "k8c.io/kubermatic/v2/pkg/resources/kubernetes-dashboard"
	"k8c.io/kubermatic/v2/pkg/resources/machinecontroller"
	metricsserver "k8c.io/kubermatic/v2/pkg/resources/metrics-server"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/openvpn"
	"k8c.io/kubermatic/v2/pkg/resources/operatingsystemmanager"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/scheduler"
	"k8c.io/kubermatic/v2/pkg/resources/usercluster"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	clusterIPUnknownRetryTimeout = 5 * time.Second
)

func (r *Reconciler) ensureResourcesAreDeployed(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return nil, err
	}
	config, err := r.configGetter(ctx)
	if err != nil {
		return nil, err
	}
	data, err := r.getClusterTemplateData(ctx, cluster, seed, config)
	if err != nil {
		return nil, err
	}

	// check that all services are available
	if err := r.ensureServices(ctx, cluster, data); err != nil {
		return nil, err
	}

	// Set the hostname & url
	if err := r.syncAddress(ctx, r.log.With("cluster", cluster.Name), cluster, seed); err != nil {
		return nil, fmt.Errorf("failed to sync address: %v", err)
	}

	// We should not proceed without having an IP address unless tunneling
	// strategy is used. Its required for all Kubeconfigs & triggers errors
	// otherwise.
	if cluster.Address.IP == "" && cluster.Spec.ExposeStrategy != kubermaticv1.ExposeStrategyTunneling {
		// This can happen e.g. if a LB external IP address has not yet been allocated by a CCM.
		// Try to reconcile after some time and do not return an error.
		r.log.Debugf("Cluster IP address not known, retry after %.0f s", clusterIPUnknownRetryTimeout.Seconds())
		return &reconcile.Result{RequeueAfter: clusterIPUnknownRetryTimeout}, nil
	}

	// check that all secrets are available // New way of handling secrets
	if err := r.ensureSecrets(ctx, cluster, data); err != nil {
		return nil, err
	}

	if err := r.ensureRBAC(ctx, cluster); err != nil {
		return nil, err
	}

	if err := r.ensureNetworkPolicies(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all StatefulSets are created
	if ok, err := r.statefulSetHealthCheck(ctx, cluster); !ok || err != nil {
		r.log.Info("Skipping reconcile for StatefulSets, not healthy yet")
	} else if err := r.ensureStatefulSets(ctx, cluster, data); err != nil {
		return nil, err
	}

	if err := r.ensureEtcdBackupConfigs(ctx, cluster, data, seed); err != nil {
		return nil, err
	}

	// Wait until the cloud provider infra is ready before attempting
	// to render the cloud-config
	// TODO: Model resource deployment as a DAG so we don't need hacks
	// like this combined with tribal knowledge and "someone is noticing this
	// isn't working correctly"
	// https://github.com/kubermatic/kubermatic/issues/2948
	if kubermaticv1.HealthStatusUp != cluster.Status.ExtendedHealth.CloudProviderInfrastructure {
		return nil, nil
	}

	// check that all ConfigMaps are available
	if err := r.ensureConfigMaps(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all Deployments are available
	if err := r.ensureDeployments(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all CronJobs are created
	if err := r.ensureCronJobs(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all PodDisruptionBudgets are created
	if err := r.ensurePodDisruptionBudgets(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all VerticalPodAutoscalers are created
	if err := r.ensureVerticalPodAutoscalers(ctx, cluster, data); err != nil {
		return nil, err
	}

	if cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		if err := nodeportproxy.EnsureResources(ctx, r.Client, data); err != nil {
			return nil, fmt.Errorf("failed to ensure NodePortProxy resources: %v", err)
		}
	}

	// Remove possible leftovers of older version of Gatekeeper, remove this in 1.19
	if err := r.ensureOldOPAIntegrationIsRemoved(ctx, data); err != nil {
		return nil, err
	}

	// This code supports switching between OpenVPN and Konnectivity setup (in both directions).
	// It can be removed one release after deprecating OpenVPN.
	if r.features.Konnectivity {
		if cluster.Spec.ClusterNetwork.KonnectivityEnabled != nil && *cluster.Spec.ClusterNetwork.KonnectivityEnabled {
			if err := r.ensureOpenVPNSetupIsRemoved(ctx, data); err != nil {
				return nil, err
			}
		} else {
			if err := r.ensureKonnectivitySetupIsRemoved(ctx, data); err != nil {
				return nil, err
			}
		}
	}

	// Ensure that OSM is completely removed, when disabled
	if !cluster.Spec.EnableOperatingSystemManager {
		if err := r.ensureOSMResourcesAreRemoved(ctx, data); err != nil {
			return nil, err
		}
	}

	return &reconcile.Result{}, nil
}

func (r *Reconciler) getClusterTemplateData(ctx context.Context, cluster *kubermaticv1.Cluster, seed *kubermaticv1.Seed, config *operatorv1alpha1.KubermaticConfiguration) (*resources.TemplateData, error) {
	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	supportsFailureDomainZoneAntiAffinity, err := resources.SupportsFailureDomainZoneAntiAffinity(ctx, r.Client)
	if err != nil {
		return nil, err
	}

	// Konnectivity is enabled if the feature gate is enabled and the cluster flag is enabled as well
	konnectivityEnabled := r.features.Konnectivity && cluster.Spec.ClusterNetwork.KonnectivityEnabled != nil && *cluster.Spec.ClusterNetwork.KonnectivityEnabled

	return resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithClient(r).
		WithCluster(cluster).
		WithDatacenter(&datacenter).
		WithSeed(seed.DeepCopy()).
		WithKubermaticConfiguration(config.DeepCopy()).
		WithOverwriteRegistry(r.overwriteRegistry).
		WithNodePortRange(config.Spec.UserCluster.NodePortRange).
		WithNodeAccessNetwork(r.nodeAccessNetwork).
		WithEtcdDiskSize(r.etcdDiskSize).
		WithUserClusterMLAEnabled(r.userClusterMLAEnabled).
		WithKonnectivityEnabled(konnectivityEnabled).
		WithCABundle(r.caBundle).
		WithOIDCIssuerURL(r.oidcIssuerURL).
		WithOIDCIssuerClientID(r.oidcIssuerClientID).
		WithKubermaticImage(r.kubermaticImage).
		WithEtcdLauncherImage(r.etcdLauncherImage).
		WithDnatControllerImage(r.dnatControllerImage).
		WithMachineControllerImageTag(r.machineControllerImageTag).
		WithMachineControllerImageRepository(r.machineControllerImageRepository).
		WithBackupPeriod(r.backupSchedule).
		WithFailureDomainZoneAntiaffinity(supportsFailureDomainZoneAntiAffinity).
		WithVersions(r.versions).
		Build(), nil
}

// ensureNamespaceExists will create the cluster namespace
func (r *Reconciler) ensureNamespaceExists(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if cluster.Status.NamespaceName == "" {
		err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.NamespaceName = fmt.Sprintf("cluster-%s", c.Name)
		})
		if err != nil {
			return err
		}
	}

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.Status.NamespaceName}, ns); !errors.IsNotFound(err) {
		return err
	}

	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{r.getOwnerRefForCluster(cluster)},
		},
	}
	if err := r.Client.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create Namespace %s: %v", cluster.Status.NamespaceName, err)
	}

	return nil
}

// GetServiceCreators returns all service creators that are currently in use
func GetServiceCreators(data *resources.TemplateData) []reconciling.NamedServiceCreatorGetter {
	creators := []reconciling.NamedServiceCreatorGetter{
		apiserver.ServiceCreator(data.Cluster().Spec.ExposeStrategy, data.Cluster().Address.ExternalName),
		etcd.ServiceCreator(data),
		machinecontroller.ServiceCreator(),
	}

	if data.IsKonnectivityEnabled() {
		creators = append(creators, konnectivity.ServiceCreator(data.Cluster().Spec.ExposeStrategy, data.Cluster().Address.ExternalName))
	} else {
		creators = append(creators,
			openvpn.ServiceCreator(data.Cluster().Spec.ExposeStrategy),
			metricsserver.ServiceCreator(),
			dns.ServiceCreator(),
		)
	}

	if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		creators = append(creators, nodeportproxy.FrontLoadBalancerServiceCreator(data))
	}

	return creators
}

func (r *Reconciler) ensureServices(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators(data)
	return reconciling.ReconcileServices(ctx, creators, c.Status.NamespaceName, r, reconciling.OwnerRefWrapper(resources.GetClusterRef(c)))
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(data *resources.TemplateData, enableAPIserverOIDCAuthentication bool) []reconciling.NamedDeploymentCreatorGetter {
	deployments := []reconciling.NamedDeploymentCreatorGetter{
		apiserver.DeploymentCreator(data, enableAPIserverOIDCAuthentication),
		scheduler.DeploymentCreator(data),
		controllermanager.DeploymentCreator(data),
		machinecontroller.DeploymentCreator(data),
		machinecontroller.WebhookDeploymentCreator(data),
		usercluster.DeploymentCreator(data),
		kubernetesdashboard.DeploymentCreator(data),
	}

	if !data.IsKonnectivityEnabled() {
		deployments = append(deployments,
			openvpn.DeploymentCreator(data),
			metricsserver.DeploymentCreator(data),
			dns.DeploymentCreator(data),
		)
	}

	if data.Cluster().Annotations[kubermaticv1.AnnotationNameClusterAutoscalerEnabled] != "" {
		deployments = append(deployments, clusterautoscaler.DeploymentCreator(data))
	}
	// If CCM migration is ongoing defer the deployment of the CCM to the
	// moment in which cloud controllers or the full in-tree cloud provider
	// have been deactivated.
	if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] &&
		(!metav1.HasAnnotation(data.Cluster().ObjectMeta, kubermaticv1.CCMMigrationNeededAnnotation) ||
			data.KCMCloudControllersDeactivated()) {
		deployments = append(deployments, cloudcontroller.DeploymentCreator(data))
	}

	if data.Cluster().Spec.EnableOperatingSystemManager {
		deployments = append(deployments, operatingsystemmanager.DeploymentCreator(data))
	}

	return deployments
}

func (r *Reconciler) ensureDeployments(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data, r.features.KubernetesOIDCAuthentication)
	return reconciling.ReconcileDeployments(ctx, creators, cluster.Status.NamespaceName, r, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetSecretCreators returns all SecretCreators that are currently in use
func (r *Reconciler) GetSecretCreators(data *resources.TemplateData) []reconciling.NamedSecretCreatorGetter {
	creators := []reconciling.NamedSecretCreatorGetter{
		certificates.RootCACreator(data),
		certificates.FrontProxyCACreator(),
		resources.ImagePullSecretCreator(r.dockerPullConfigJSON),
		apiserver.FrontProxyClientCertificateCreator(data),
		etcd.TLSCertificateCreator(data),
		apiserver.EtcdClientCertificateCreator(data),
		apiserver.TLSServingCertificateCreator(data),
		apiserver.KubeletClientCertificateCreator(data),
		apiserver.ServiceAccountKeyCreator(),
		machinecontroller.TLSServingCertificateCreator(data),

		// Kubeconfigs
		resources.GetInternalKubeconfigCreator(resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.OperatingSystemManagerKubeconfigSecretName, resources.OperatingSystemManagerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.InternalUserClusterAdminKubeconfigSecretName, resources.InternalUserClusterAdminKubeconfigCertUsername, []string{"system:masters"}, data),
		resources.GetInternalKubeconfigCreator(resources.KubernetesDashboardKubeconfigSecretName, resources.KubernetesDashboardCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.ClusterAutoscalerKubeconfigSecretName, resources.ClusterAutoscalerCertUsername, nil, data),
		resources.AdminKubeconfigCreator(data),
		apiserver.TokenViewerCreator(),
		apiserver.TokenUsersCreator(data),
		resources.ViewerKubeconfigCreator(data),
	}

	if data.IsKonnectivityEnabled() {
		creators = append(creators,
			konnectivity.TLSServingCertificateCreator(data),
			konnectivity.ProxyKubeconfig(data),
		)
	} else {
		creators = append(creators,
			openvpn.CACreator(),
			openvpn.TLSServingCertificateCreator(data),
			openvpn.InternalClientCertificateCreator(data),
			metricsserver.TLSServingCertSecretCreator(data.GetRootCA),
			resources.GetInternalKubeconfigCreator(resources.MetricsServerKubeconfigSecretName, resources.MetricsServerCertUsername, nil, data),
			resources.GetInternalKubeconfigCreator(resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil, data),
		)
	}

	if flag := data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; flag {
		creators = append(creators, resources.GetInternalKubeconfigCreator(
			resources.CloudControllerManagerKubeconfigSecretName, resources.CloudControllerManagerCertUsername, nil, data,
		))
	}

	if data.Cluster().Spec.Cloud.GCP != nil {
		creators = append(creators, resources.ServiceAccountSecretCreator(data))
	}

	return creators
}

func (r *Reconciler) ensureSecrets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretCreatorGetters := r.GetSecretCreators(data)

	if err := reconciling.ReconcileSecrets(ctx, namedSecretCreatorGetters, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the Secret exists: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureServiceAccounts(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedServiceAccountCreatorGetters := []reconciling.NamedServiceAccountCreatorGetter{
		etcd.ServiceAccountCreator,
		usercluster.ServiceAccountCreator,
		machinecontroller.ServiceAccountCreator,
		machinecontroller.WebhookServiceAccountCreator,
	}

	if c.Spec.EnableOperatingSystemManager {
		namedServiceAccountCreatorGetters = append(namedServiceAccountCreatorGetters, operatingsystemmanager.ServiceAccountCreator)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, namedServiceAccountCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure ServiceAccounts: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureRoles(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedRoleCreatorGetters := []reconciling.NamedRoleCreatorGetter{
		usercluster.RoleCreator,
		machinecontroller.WebhookRoleCreator,
	}

	if c.Spec.EnableOperatingSystemManager {
		namedRoleCreatorGetters = append(namedRoleCreatorGetters, operatingsystemmanager.RoleCreator)
	}

	if err := reconciling.ReconcileRoles(ctx, namedRoleCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure Roles: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureRoleBindings(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedRoleBindingCreatorGetters := []reconciling.NamedRoleBindingCreatorGetter{
		usercluster.RoleBindingCreator,
		machinecontroller.WebhookRoleBindingCreator,
	}

	if c.Spec.EnableOperatingSystemManager {
		namedRoleBindingCreatorGetters = append(namedRoleBindingCreatorGetters, operatingsystemmanager.RoleBindingCreator)
	}

	if err := reconciling.ReconcileRoleBindings(ctx, namedRoleBindingCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure RoleBindings: %v", err)
	}
	return nil
}

func (r *Reconciler) ensureClusterRoles(ctx context.Context) error {
	namedClusterRoleCreatorGetters := []reconciling.NamedClusterRoleCreatorGetter{
		usercluster.ClusterRoleCreator,
	}
	if err := reconciling.ReconcileClusterRoles(ctx, namedClusterRoleCreatorGetters, "", r.Client); err != nil {
		return fmt.Errorf("failed to ensure Cluster Roles: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureClusterRoleBindings(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedClusterRoleBindingsCreatorGetters := []reconciling.NamedClusterRoleBindingCreatorGetter{
		usercluster.ClusterRoleBinding(c.Status.NamespaceName),
	}
	if err := reconciling.ReconcileClusterRoleBindings(ctx, namedClusterRoleBindingsCreatorGetters, "", r.Client); err != nil {
		return fmt.Errorf("failed to ensure Cluster Role Bindings: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureNetworkPolicies(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	if c.Spec.Features[kubermaticv1.ApiserverNetworkPolicy] {
		namedNetworkPolicyCreatorGetters := []reconciling.NamedNetworkPolicyCreatorGetter{
			apiserver.DenyAllPolicyCreator(),
			apiserver.DNSAllowCreator(c, data),
			apiserver.EctdAllowCreator(c),
			apiserver.MachineControllerWebhookCreator(c),
		}
		if data.IsKonnectivityEnabled() {
			namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters, apiserver.ClusterExternalAddrAllowCreator(c))
		} else {
			namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters,
				apiserver.OpenVPNServerAllowCreator(c),
				apiserver.MetricsServerAllowCreator(c),
			)
		}
		if c.Spec.OIDC.IssuerURL != "" {
			namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters, apiserver.OIDCIssuerAllowCreator(c.Spec.OIDC.IssuerURL))
		} else if r.features.KubernetesOIDCAuthentication {
			namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters, apiserver.OIDCIssuerAllowCreator(data.OIDCIssuerURL()))
		}
		if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
			return fmt.Errorf("failed to ensure Network Policies: %v", err)
		}
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use
func GetConfigMapCreators(data *resources.TemplateData) []reconciling.NamedConfigMapCreatorGetter {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		cloudconfig.ConfigMapCreator(data),
		apiserver.AuditConfigMapCreator(data),
		apiserver.AdmissionControlCreator(data),
		apiserver.CABundleCreator(data),
	}

	if data.IsKonnectivityEnabled() {
		creators = append(creators, apiserver.EgressSelectorConfigCreator())
	} else {
		creators = append(creators,
			openvpn.ServerClientConfigsConfigMapCreator(data),
			dns.ConfigMapCreator(data),
		)
	}

	if data.Cluster().Spec.Cloud.VSphere != nil {
		creators = append(creators, cloudconfig.ConfigmapVsphereCSICreator(data))
	}

	return creators
}

func (r *Reconciler) ensureConfigMaps(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	if err := reconciling.ReconcileConfigMaps(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %v", err)
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use
func GetStatefulSetCreators(data *resources.TemplateData, enableDataCorruptionChecks bool, enableTLSOnly bool) []reconciling.NamedStatefulSetCreatorGetter {
	return []reconciling.NamedStatefulSetCreatorGetter{
		etcd.StatefulSetCreator(data, enableDataCorruptionChecks, enableTLSOnly),
	}
}

// GetEtcdBackupConfigCreators returns all EtcdBackupConfigCreators that are currently in use
func GetEtcdBackupConfigCreators(data *resources.TemplateData, seed *kubermaticv1.Seed) []reconciling.NamedEtcdBackupConfigCreatorGetter {
	creators := []reconciling.NamedEtcdBackupConfigCreatorGetter{
		etcd.BackupConfigCreator(data, seed),
	}
	return creators
}

// GetPodDisruptionBudgetCreators returns all PodDisruptionBudgetCreators that are currently in use
func GetPodDisruptionBudgetCreators(data *resources.TemplateData) []reconciling.NamedPodDisruptionBudgetCreatorGetter {
	creators := []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		etcd.PodDisruptionBudgetCreator(data),
		apiserver.PodDisruptionBudgetCreator(),
	}
	if !data.IsKonnectivityEnabled() {
		creators = append(creators,
			metricsserver.PodDisruptionBudgetCreator(),
			dns.PodDisruptionBudgetCreator(),
		)
	}
	return creators
}

func (r *Reconciler) ensurePodDisruptionBudgets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetPodDisruptionBudgetCreators(data)

	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the PodDisruptionBudget exists: %v", err)
	}

	return nil
}

// GetCronJobCreators returns all CronJobCreators that are currently in use
func GetCronJobCreators(data *resources.TemplateData) []reconciling.NamedCronJobCreatorGetter {
	return []reconciling.NamedCronJobCreatorGetter{
		etcd.CronJobCreator(data),
	}
}

func (r *Reconciler) ensureCronJobs(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetCronJobCreators(data)

	if err := reconciling.ReconcileCronJobs(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the CronJobs exists: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureVerticalPodAutoscalers(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	controlPlaneDeploymentNames := []string{
		resources.MachineControllerDeploymentName,
		resources.MachineControllerWebhookDeploymentName,
		resources.ApiserverDeploymentName,
		resources.ControllerManagerDeploymentName,
		resources.SchedulerDeploymentName,
	}
	if !data.IsKonnectivityEnabled() {
		controlPlaneDeploymentNames = append(controlPlaneDeploymentNames,
			resources.OpenVPNServerDeploymentName,
			resources.MetricsServerDeploymentName,
			resources.DNSResolverDeploymentName,
		)
	}

	creators, err := resources.GetVerticalPodAutoscalersForAll(ctx, r.Client, controlPlaneDeploymentNames, []string{resources.EtcdStatefulSetName}, c.Status.NamespaceName, r.features.VPA)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %v", err)
	}

	return reconciling.ReconcileVerticalPodAutoscalers(ctx, creators, c.Status.NamespaceName, r.Client)
}

func (r *Reconciler) ensureStatefulSets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	useTLSOnly, err := r.etcdUseStrictTLS(ctx, c)
	if err != nil {
		return err
	}

	creators := GetStatefulSetCreators(data, r.features.EtcdDataCorruptionChecks, useTLSOnly)

	return reconciling.ReconcileStatefulSets(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c)))
}

func (r *Reconciler) ensureEtcdBackupConfigs(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData,
	seed *kubermaticv1.Seed) error {
	if seed.IsDefaultEtcdAutomaticBackupEnabled() {
		creators := GetEtcdBackupConfigCreators(data, seed)
		return reconciling.ReconcileEtcdBackupConfigs(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c)))
	}
	// If default etcd automatic backups are not enabled, remove them if any
	ebc := &kubermaticv1.EtcdBackupConfig{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: resources.EtcdDefaultBackupConfigName, Namespace: c.Status.NamespaceName}, ebc)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.Client.Delete(ctx, ebc)
}

func (r *Reconciler) ensureOldOPAIntegrationIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range gatekeeper.GetResourcesToRemoveOnDelete(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure old OPA integration version resources are removed/not present: %v", err)
		}
	}

	return nil
}

func (r *Reconciler) ensureOSMResourcesAreRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range operatingsystemmanager.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		err := r.Client.Delete(ctx, resource)
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OSM resources are removed/not present: %v", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureOpenVPNSetupIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range openvpn.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OpenVPN resources are removed/not present: %v", err)
		}
	}
	for _, resource := range metricsserver.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure metrics-server resources are removed/not present: %v", err)
		}
	}
	for _, resource := range dns.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure dns-resolver resources are removed/not present: %v", err)
		}
	}
	dnatControllerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.KubeletDnatControllerKubeconfigSecretName,
			Namespace: data.Cluster().Status.NamespaceName,
		},
	}
	if err := r.Client.Delete(ctx, dnatControllerSecret); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to ensure DNAT controller resources are removed/not present: %v", err)
	}
	return nil
}

func (r *Reconciler) ensureKonnectivitySetupIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range konnectivity.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure Konnectivity resources are removed/not present: %v", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureRBAC(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if err := r.ensureServiceAccounts(ctx, cluster); err != nil {
		return err
	}

	if err := r.ensureRoles(ctx, cluster); err != nil {
		return err
	}

	if err := r.ensureRoleBindings(ctx, cluster); err != nil {
		return err
	}

	if err := r.ensureClusterRoles(ctx); err != nil {
		return err
	}

	if err := r.ensureClusterRoleBindings(ctx, cluster); err != nil {
		return err
	}

	return nil
}
