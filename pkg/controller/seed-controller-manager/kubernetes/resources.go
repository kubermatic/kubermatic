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
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"sort"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
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
	userclusterwebhook "k8c.io/kubermatic/v2/pkg/resources/usercluster-webhook"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	clusterIPUnknownRetryTimeout = 5 * time.Second
)

func (r *Reconciler) ensureResourcesAreDeployed(ctx context.Context, cluster *kubermaticv1.Cluster, namespace *corev1.Namespace) (*reconcile.Result, error) {
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
		return nil, fmt.Errorf("failed to sync address: %w", err)
	}

	// We should not proceed without having an IP address unless tunneling
	// strategy is used. Its required for all Kubeconfigs & triggers errors
	// otherwise.
	if cluster.GetAddress().IP == "" && cluster.Spec.ExposeStrategy != kubermaticv1.ExposeStrategyTunneling {
		// This can happen e.g. if a LB external IP address has not yet been allocated by a CCM.
		// Try to reconcile after some time and do not return an error.
		r.log.Debugf("Cluster IP address not known, retry after %.0f s", clusterIPUnknownRetryTimeout.Seconds())
		return &reconcile.Result{RequeueAfter: clusterIPUnknownRetryTimeout}, nil
	}

	// check that all secrets are available // New way of handling secrets
	if err := r.ensureSecrets(ctx, cluster, data); err != nil {
		return nil, err
	}

	if err := r.ensureRBAC(ctx, cluster, namespace); err != nil {
		return nil, err
	}

	if err := r.ensureNetworkPolicies(ctx, cluster, data); err != nil {
		return nil, err
	}

	// check that all StatefulSets are created
	if ok, err := r.statefulSetHealthCheck(ctx, cluster); !ok || err != nil {
		r.log.Debug("Skipping reconcile for StatefulSets, etcd is not healthy yet")
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

	// Ensure that kubernetes-dashboard is completely removed, when disabled
	if !cluster.Spec.KubernetesDashboard.IsEnabled() {
		if err := r.ensureKubernetesDashboardResourcesAreRemoved(ctx, data); err != nil {
			return nil, err
		}
	}

	// Ensure that encryption-at-rest is completely removed when no longer enabled or active
	if !cluster.IsEncryptionEnabled() && !cluster.IsEncryptionActive() {
		if err := r.ensureEncryptionConfigurationIsRemoved(ctx, data); err != nil {
			return nil, err
		}
	}

	return &reconcile.Result{}, nil
}

func (r *Reconciler) getClusterTemplateData(ctx context.Context, cluster *kubermaticv1.Cluster, seed *kubermaticv1.Seed, config *kubermaticv1.KubermaticConfiguration) (*resources.TemplateData, error) {
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

// reconcileClusterNamespace will ensure that the cluster namespace is
// correctly initialized and created.
func (r *Reconciler) reconcileClusterNamespace(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*corev1.Namespace, error) {
	if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, kubermaticv1.NamespaceCleanupFinalizer); err != nil {
		return nil, fmt.Errorf("failed to set %q finalizer: %w", kubermaticv1.NamespaceCleanupFinalizer, err)
	}

	namespace, err := r.ensureNamespaceExists(ctx, log, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure cluster namespace: %w", err)
	}

	err = kubermaticv1helper.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		if c.Status.NamespaceName != namespace.Name {
			c.Status.NamespaceName = namespace.Name
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cluster namespace status: %w", err)
	}

	return namespace, nil
}

// ensureNamespaceExists will create the cluster namespace.
func (r *Reconciler) ensureNamespaceExists(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*corev1.Namespace, error) {
	namespace := cluster.Status.NamespaceName
	if namespace == "" {
		namespace = kubernetesprovider.NamespaceName(cluster.Name)
	}

	ns := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: namespace}, ns)
	if err == nil {
		return ns, nil // found it
	}
	if !apierrors.IsNotFound(err) {
		return nil, err // something bad happened when trying to get the namespace
	}

	log.Info("Creating cluster namespace", "namespace", namespace)
	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            namespace,
			OwnerReferences: []metav1.OwnerReference{r.getOwnerRefForCluster(cluster)},
		},
	}
	if err := r.Create(ctx, ns); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create Namespace %s: %w", namespace, err)
	}

	// before returning the namespace and putting its name into the cluster status,
	// ensure that the namespace is in our cache, or else other controllers that
	// want to reconcile might get confused
	err = wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
		ns := &corev1.Namespace{}
		err := r.Get(ctx, types.NamespacedName{Name: namespace}, ns)
		if err == nil {
			return true, nil
		}
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		return false, err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to wait for cluster namespace to appear in cache: %w", err)
	}

	// Creating() an object does not set the type meta, in fact it _resets_ it.
	// Since the namespace is later used to build an owner reference, we must ensure
	// type meta is set properly.
	ns.TypeMeta = metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Namespace",
	}

	return ns, nil
}

// GetServiceCreators returns all service creators that are currently in use.
func GetServiceCreators(data *resources.TemplateData) []reconciling.NamedServiceCreatorGetter {
	extName := data.Cluster().GetAddress().ExternalName

	creators := []reconciling.NamedServiceCreatorGetter{
		apiserver.ServiceCreator(data.Cluster().Spec.ExposeStrategy, extName),
		etcd.ServiceCreator(data),
		machinecontroller.ServiceCreator(),
		userclusterwebhook.ServiceCreator(),
	}

	if data.IsKonnectivityEnabled() {
		creators = append(creators, konnectivity.ServiceCreator(data.Cluster().Spec.ExposeStrategy, extName))
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
	return reconciling.ReconcileServices(ctx, creators, c.Status.NamespaceName, r)
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use.
func GetDeploymentCreators(data *resources.TemplateData, enableAPIserverOIDCAuthentication bool) []reconciling.NamedDeploymentCreatorGetter {
	deployments := []reconciling.NamedDeploymentCreatorGetter{
		apiserver.DeploymentCreator(data, enableAPIserverOIDCAuthentication),
		scheduler.DeploymentCreator(data),
		controllermanager.DeploymentCreator(data),
		machinecontroller.DeploymentCreator(data),
		machinecontroller.WebhookDeploymentCreator(data),
		usercluster.DeploymentCreator(data),
		userclusterwebhook.DeploymentCreator(data),
	}

	if data.Cluster().Spec.KubernetesDashboard.IsEnabled() {
		deployments = append(deployments, kubernetesdashboard.DeploymentCreator(data))
	}

	if !data.IsKonnectivityEnabled() {
		deployments = append(deployments,
			openvpn.DeploymentCreator(data),
			metricsserver.DeploymentCreator(data),
			dns.DeploymentCreator(data),
		)
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

	if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		deployments = append(deployments,
			nodeportproxy.DeploymentEnvoyCreator(data),
			nodeportproxy.DeploymentLBUpdaterCreator(data),
		)
	}

	return deployments
}

func (r *Reconciler) ensureDeployments(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data, r.features.KubernetesOIDCAuthentication)
	return reconciling.ReconcileDeployments(ctx, creators, cluster.Status.NamespaceName, r)
}

// GetSecretCreators returns all SecretCreators that are currently in use.
func (r *Reconciler) GetSecretCreators(data *resources.TemplateData) []reconciling.NamedSecretCreatorGetter {
	namespace := data.Cluster().Status.NamespaceName

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
		userclusterwebhook.TLSServingCertificateCreator(data),

		// Kubeconfigs
		resources.GetInternalKubeconfigCreator(namespace, resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigCreator(namespace, resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigCreator(namespace, resources.OperatingSystemManagerKubeconfigSecretName, resources.OperatingSystemManagerCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigCreator(namespace, resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigCreator(namespace, resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigCreator(namespace, resources.InternalUserClusterAdminKubeconfigSecretName, resources.InternalUserClusterAdminKubeconfigCertUsername, []string{"system:masters"}, data, r.log),
		resources.GetInternalKubeconfigCreator(namespace, resources.ClusterAutoscalerKubeconfigSecretName, resources.ClusterAutoscalerCertUsername, nil, data, r.log),
		resources.AdminKubeconfigCreator(data),
		apiserver.TokenViewerCreator(),
		apiserver.TokenUsersCreator(data),
		resources.ViewerKubeconfigCreator(data),
	}

	if data.Cluster().Spec.KubernetesDashboard.IsEnabled() {
		creators = append(creators,
			resources.GetInternalKubeconfigCreator(namespace, resources.KubernetesDashboardKubeconfigSecretName, resources.KubernetesDashboardCertUsername, nil, data, r.log),
		)
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
			resources.GetInternalKubeconfigCreator(namespace, resources.MetricsServerKubeconfigSecretName, resources.MetricsServerCertUsername, nil, data, r.log),
			resources.GetInternalKubeconfigCreator(namespace, resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil, data, r.log),
		)
	}

	if data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.Enabled {
		creators = append(creators, apiserver.FluentBitSecretCreator(data))
	}

	if data.Cluster().IsEncryptionEnabled() || data.Cluster().IsEncryptionActive() {
		creators = append(creators, apiserver.EncryptionConfigurationSecretCreator(data))
	}

	if flag := data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; flag {
		creators = append(creators, resources.GetInternalKubeconfigCreator(
			namespace, resources.CloudControllerManagerKubeconfigSecretName, resources.CloudControllerManagerCertUsername, nil, data, r.log,
		))

		if data.Cluster().Spec.Cloud.Kubevirt != nil {
			creators = append(creators, cloudcontroller.GetKubeVirtInfraKubeConfigCreator(data))
		}
	}

	if data.Cluster().Spec.Cloud.GCP != nil {
		creators = append(creators, resources.ServiceAccountSecretCreator(data))
	}

	return creators
}

func (r *Reconciler) ensureSecrets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretCreatorGetters := r.GetSecretCreators(data)

	if err := reconciling.ReconcileSecrets(ctx, namedSecretCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure that the Secret exists: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureServiceAccounts(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedServiceAccountCreatorGetters := []reconciling.NamedServiceAccountCreatorGetter{
		etcd.ServiceAccountCreator,
		usercluster.ServiceAccountCreator,
		machinecontroller.ServiceAccountCreator,
		machinecontroller.WebhookServiceAccountCreator,
		userclusterwebhook.ServiceAccountCreator,
	}

	if c.Spec.EnableOperatingSystemManager {
		namedServiceAccountCreatorGetters = append(namedServiceAccountCreatorGetters, operatingsystemmanager.ServiceAccountCreator)
	}

	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		namedServiceAccountCreatorGetters = append(namedServiceAccountCreatorGetters, nodeportproxy.ServiceAccountCreator)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, namedServiceAccountCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure ServiceAccounts: %w", err)
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

	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		namedRoleCreatorGetters = append(namedRoleCreatorGetters, nodeportproxy.RoleCreator)
	}

	if err := reconciling.ReconcileRoles(ctx, namedRoleCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure Roles: %w", err)
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

	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		namedRoleBindingCreatorGetters = append(namedRoleBindingCreatorGetters, nodeportproxy.RoleBindingCreator)
	}

	if err := reconciling.ReconcileRoleBindings(ctx, namedRoleBindingCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure RoleBindings: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureClusterRoles(ctx context.Context) error {
	namedClusterRoleCreatorGetters := []reconciling.NamedClusterRoleCreatorGetter{
		usercluster.ClusterRole(),
		userclusterwebhook.ClusterRole(),
	}
	if err := reconciling.ReconcileClusterRoles(ctx, namedClusterRoleCreatorGetters, "", r.Client); err != nil {
		return fmt.Errorf("failed to ensure Cluster Roles: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureClusterRoleBindings(ctx context.Context, c *kubermaticv1.Cluster, namespace *corev1.Namespace) error {
	namedClusterRoleBindingsCreatorGetters := []reconciling.NamedClusterRoleBindingCreatorGetter{
		usercluster.ClusterRoleBinding(namespace),
		userclusterwebhook.ClusterRoleBinding(namespace),
	}
	if err := reconciling.ReconcileClusterRoleBindings(ctx, namedClusterRoleBindingsCreatorGetters, "", r.Client); err != nil {
		return fmt.Errorf("failed to ensure Cluster Role Bindings: %w", err)
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

		// one shared limited context for all hostname resolutions
		resolverCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if data.IsKonnectivityEnabled() {
			extName := data.Cluster().GetAddress().ExternalName

			// allow egress traffic to all resolved cluster external IPs
			ipList, err := hostnameToIPList(resolverCtx, extName)
			if err != nil {
				return fmt.Errorf("failed to resolve cluster external name %q: %w", extName, err)
			}

			namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters, apiserver.ClusterExternalAddrAllowCreator(ipList, c.Spec.ExposeStrategy))
		} else {
			namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters,
				apiserver.OpenVPNServerAllowCreator(c),
				apiserver.MetricsServerAllowCreator(c),
			)
		}

		issuerURL := c.Spec.OIDC.IssuerURL
		if issuerURL == "" && r.features.KubernetesOIDCAuthentication {
			issuerURL = data.OIDCIssuerURL()
		}

		if issuerURL != "" {
			u, err := url.Parse(issuerURL)
			if err != nil {
				return fmt.Errorf("failed to parse OIDC issuer URL %q: %w", issuerURL, err)
			}

			// allow egress traffic to OIDC issuer's external IPs
			ipList, err := hostnameToIPList(resolverCtx, u.Hostname())
			if err != nil {
				return fmt.Errorf("failed to resolve OIDC issuer URL %q: %w", issuerURL, err)
			}

			namedNetworkPolicyCreatorGetters = append(namedNetworkPolicyCreatorGetters, apiserver.OIDCIssuerAllowCreator(ipList))
		}

		if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
			return fmt.Errorf("failed to ensure Network Policies: %w", err)
		}
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use.
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
		creators = append(creators, cloudconfig.VsphereCSIConfigMapCreator(data))
	}

	if data.Cluster().Spec.Cloud.Nutanix != nil && data.Cluster().Spec.Cloud.Nutanix.CSI != nil {
		creators = append(creators, cloudconfig.NutanixCSIConfigMapCreator(data))
	}

	if data.Cluster().Spec.Cloud.VMwareCloudDirector != nil {
		creators = append(creators, cloudconfig.VMwareCloudDirectorCSIConfigMapCreator(data))
	}

	return creators
}

func (r *Reconciler) ensureConfigMaps(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	if err := reconciling.ReconcileConfigMaps(ctx, creators, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %w", err)
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use.
func GetStatefulSetCreators(data *resources.TemplateData, enableDataCorruptionChecks bool, enableTLSOnly bool) []reconciling.NamedStatefulSetCreatorGetter {
	return []reconciling.NamedStatefulSetCreatorGetter{
		etcd.StatefulSetCreator(data, enableDataCorruptionChecks, enableTLSOnly),
	}
}

// GetEtcdBackupConfigCreators returns all EtcdBackupConfigCreators that are currently in use.
func GetEtcdBackupConfigCreators(data *resources.TemplateData, seed *kubermaticv1.Seed) []reconciling.NamedEtcdBackupConfigCreatorGetter {
	creators := []reconciling.NamedEtcdBackupConfigCreatorGetter{
		etcd.BackupConfigCreator(data, seed),
	}
	return creators
}

// GetPodDisruptionBudgetCreators returns all PodDisruptionBudgetCreators that are currently in use.
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

	if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		creators = append(creators, nodeportproxy.PodDisruptionBudgetCreator())
	}

	return creators
}

func (r *Reconciler) ensurePodDisruptionBudgets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetPodDisruptionBudgetCreators(data)

	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure that the PodDisruptionBudget exists: %w", err)
	}

	return nil
}

// GetCronJobCreators returns all CronJobCreators that are currently in use.
func GetCronJobCreators(data *resources.TemplateData) []reconciling.NamedCronJobCreatorGetter {
	return []reconciling.NamedCronJobCreatorGetter{
		etcd.CronJobCreator(data),
	}
}

func (r *Reconciler) ensureCronJobs(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetCronJobCreators(data)

	if err := reconciling.ReconcileCronJobs(ctx, creators, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure that the CronJobs exists: %w", err)
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
		return fmt.Errorf("failed to create the functions to handle VPA resources: %w", err)
	}

	return reconciling.ReconcileVerticalPodAutoscalers(ctx, creators, c.Status.NamespaceName, r.Client)
}

func (r *Reconciler) ensureStatefulSets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	useTLSOnly, err := r.etcdUseStrictTLS(ctx, c)
	if err != nil {
		return err
	}

	creators := GetStatefulSetCreators(data, r.features.EtcdDataCorruptionChecks, useTLSOnly)

	return reconciling.ReconcileStatefulSets(ctx, creators, c.Status.NamespaceName, r.Client)
}

func (r *Reconciler) ensureEtcdBackupConfigs(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData,
	seed *kubermaticv1.Seed) error {
	if seed.IsDefaultEtcdAutomaticBackupEnabled() {
		creators := GetEtcdBackupConfigCreators(data, seed)
		return reconciling.ReconcileEtcdBackupConfigs(ctx, creators, c.Status.NamespaceName, r.Client)
	}
	// If default etcd automatic backups are not enabled, remove them if any
	ebc := &kubermaticv1.EtcdBackupConfig{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: resources.EtcdDefaultBackupConfigName, Namespace: c.Status.NamespaceName}, ebc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.Client.Delete(ctx, ebc)
}

func (r *Reconciler) ensureOldOPAIntegrationIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range gatekeeper.GetResourcesToRemoveOnDelete(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure old OPA integration version resources are removed/not present: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) ensureKubernetesDashboardResourcesAreRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range kubernetesdashboard.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		err := r.Client.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure kubernetes-dashboard resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureOSMResourcesAreRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range operatingsystemmanager.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		err := r.Client.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OSM resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureOpenVPNSetupIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range openvpn.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OpenVPN resources are removed/not present: %w", err)
		}
	}
	for _, resource := range metricsserver.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure metrics-server resources are removed/not present: %w", err)
		}
	}
	for _, resource := range dns.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure dns-resolver resources are removed/not present: %w", err)
		}
	}
	dnatControllerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.KubeletDnatControllerKubeconfigSecretName,
			Namespace: data.Cluster().Status.NamespaceName,
		},
	}
	if err := r.Client.Delete(ctx, dnatControllerSecret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to ensure DNAT controller resources are removed/not present: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureKonnectivitySetupIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range konnectivity.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure Konnectivity resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureEncryptionConfigurationIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range apiserver.EncryptionResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Client.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure encryption-at-rest resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureRBAC(ctx context.Context, cluster *kubermaticv1.Cluster, namespace *corev1.Namespace) error {
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

	if err := r.ensureClusterRoleBindings(ctx, cluster, namespace); err != nil {
		return err
	}

	return nil
}

// hostnameToIPList returns a list of IP addresses used to reach the provided hostname.
// If it is an IP address, returns it. If it is a domain name, resolves it.
// The returned list of IPs is always sorted to produce the same result on each resolution attempt.
func hostnameToIPList(ctx context.Context, hostname string) ([]net.IP, error) {
	if ip := net.ParseIP(hostname); ip != nil {
		// hostname is an IP address
		return []net.IP{ip}, nil
	}

	// hostname is a domain name - resolve it
	var r net.Resolver
	ipList, err := r.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return nil, err
	}
	if len(ipList) == 0 {
		return nil, fmt.Errorf("no resolved IP address for hostname %s", hostname)
	}

	sort.Slice(ipList, func(i, j int) bool {
		return bytes.Compare(ipList[i], ipList[j]) < 0
	})

	return ipList, nil
}
