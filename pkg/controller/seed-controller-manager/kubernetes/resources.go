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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/resources/controllermanager"
	"k8c.io/kubermatic/v2/pkg/resources/csi"
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
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"
	"k8c.io/kubermatic/v2/pkg/resources/scheduler"
	"k8c.io/kubermatic/v2/pkg/resources/usercluster"
	userclusterwebhook "k8c.io/kubermatic/v2/pkg/resources/usercluster-webhook"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	certutil "k8s.io/client-go/util/cert"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	// get default seed values.
	seed, err = defaulting.DefaultSeed(seed, config, r.log)
	if err != nil {
		return nil, fmt.Errorf("failed to apply default values to Seed: %w", err)
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
	if cluster.Status.Address.IP == "" && cluster.Spec.ExposeStrategy != kubermaticv1.ExposeStrategyTunneling {
		// This can happen e.g. if a LB external IP address has not yet been allocated by a CCM.
		// Try to reconcile after some time and do not return an error.
		r.log.Debugf("Cluster IP address not known, retry after %.0f s", clusterIPUnknownRetryTimeout.Seconds())
		return &reconcile.Result{RequeueAfter: clusterIPUnknownRetryTimeout}, nil
	}

	// check that all secrets are available
	if err := r.ensureSecrets(ctx, cluster, data); err != nil {
		return nil, err
	}

	// Ensure audit webhook backend secret is created & referenced in cluster spec.
	if cluster.Spec.AuditLogging != nil && cluster.Spec.AuditLogging.WebhookBackend != nil {
		if err := r.ensureAuditWebhook(ctx, cluster, data); err != nil {
			return nil, err
		}
	}
	if err := r.ensureRBAC(ctx, cluster, namespace); err != nil {
		return nil, err
	}

	if err := r.ensureNetworkPolicies(ctx, cluster, data, config); err != nil {
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
	if cluster.Spec.ClusterNetwork.KonnectivityEnabled != nil && *cluster.Spec.ClusterNetwork.KonnectivityEnabled { //nolint:staticcheck
		if err := r.ensureOpenVPNSetupIsRemoved(ctx, data); err != nil {
			return nil, err
		}
	} else {
		if err := r.ensureKonnectivitySetupIsRemoved(ctx, data); err != nil {
			return nil, err
		}
	}

	// clean up NetworkPolicy created before konnectivity-server's kubeconfig was changed
	// to use the internal API server endpoint.
	if data.IsKonnectivityEnabled() {
		if err := r.ensureKonnectivityNetworkPolicyIsRemoved(ctx, data); err != nil {
			return nil, err
		}
	}

	// Ensure that kubernetes-dashboard is completely removed, when disabled
	if !cluster.Spec.IsKubernetesDashboardEnabled() {
		if err := r.ensureKubernetesDashboardResourcesAreRemoved(ctx, data); err != nil {
			return nil, err
		}
	}

	if cluster.Spec.DisableCSIDriver {
		if err := r.ensureCSIDriverResourcesAreRemoved(ctx, data); err != nil {
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

	supportsFailureDomainZoneAntiAffinity, err := resources.SupportsFailureDomainZoneAntiAffinity(ctx, r)
	if err != nil {
		return nil, err
	}

	cbsl := &kubermaticv1.ClusterBackupStorageLocation{}
	if cluster.Spec.IsClusterBackupEnabled() {
		key := types.NamespacedName{
			Namespace: resources.KubermaticNamespace,
			Name:      cluster.Spec.BackupConfig.BackupStorageLocation.Name,
		}

		if err := r.Get(ctx, key, cbsl); err != nil {
			// A defunct CBSL reference is not nice, but should not cancel the entire
			// seed-level reconciling for this cluster.
			cbsl = nil
		}
	} else {
		cbsl = nil
	}

	konnectivityEnabled := cluster.Spec.ClusterNetwork.KonnectivityEnabled != nil && *cluster.Spec.ClusterNetwork.KonnectivityEnabled //nolint:staticcheck

	apiserverAltNames, err := r.listAPIServerAlternateNames(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to determine additional API server altnames: %w", err)
	}

	return resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithClient(r).
		WithCluster(cluster).
		WithDatacenter(&datacenter).
		WithSeed(seed.DeepCopy()).
		WithKubermaticConfiguration(config.DeepCopy()).
		WithOverwriteRegistry(r.overwriteRegistry).
		WithAPIServerAlternateNames(apiserverAltNames).
		WithNodePortRange(config.Spec.UserCluster.NodePortRange).
		WithNodeAccessNetwork(r.nodeAccessNetwork).
		WithEtcdDiskSize(r.etcdDiskSize).
		WithUserClusterMLAEnabled(r.userClusterMLAEnabled).
		WithKonnectivityEnabled(konnectivityEnabled).
		WithTunnelingAgentIP(r.tunnelingAgentIP).
		WithCABundle(r.caBundle).
		WithOIDCIssuerURL(r.oidcIssuerURL).
		WithOIDCIssuerClientID(r.oidcIssuerClientID).
		WithKubermaticImage(r.kubermaticImage).
		WithEtcdLauncherImage(r.etcdLauncherImage).
		WithDnatControllerImage(r.dnatControllerImage).
		WithMachineControllerImageTag(r.machineControllerImageTag).
		WithMachineControllerImageRepository(r.machineControllerImageRepository).
		WithBackupPeriod(r.backupSchedule).
		WithBackupCount(r.backupCount).
		WithFailureDomainZoneAntiaffinity(supportsFailureDomainZoneAntiAffinity).
		WithClusterBackupStorageLocation(cbsl).
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

	err = util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
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

	log.Infow("Creating cluster namespace", "namespace", namespace)
	ns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:            namespace,
			OwnerReferences: []metav1.OwnerReference{r.getOwnerRefForCluster(cluster)},
		},
	}
	if err := r.Create(ctx, ns); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return nil, fmt.Errorf("failed to create Namespace %s: %w", namespace, err)
	}

	// before returning the namespace and putting its name into the cluster status,
	// ensure that the namespace is in our cache, or else other controllers that
	// want to reconcile might get confused
	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 30*time.Second, true, func(ctx context.Context) (bool, error) {
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

// GetServiceReconcilers returns all service creators that are currently in use.
func GetServiceReconcilers(data *resources.TemplateData) []reconciling.NamedServiceReconcilerFactory {
	extName := data.Cluster().Status.Address.ExternalName
	apiServerServiceType := data.DC().Spec.APIServerServiceType

	creators := []reconciling.NamedServiceReconcilerFactory{
		apiserver.ServiceReconciler(data.Cluster().Spec.ExposeStrategy, extName, apiServerServiceType),
		etcd.ServiceReconciler(data),
		userclusterwebhook.ServiceReconciler(),
		operatingsystemmanager.ServiceReconciler(),
	}

	if data.Cluster().Spec.Cloud.Edge == nil {
		creators = append(creators, machinecontroller.ServiceReconciler())
	}

	if data.IsKonnectivityEnabled() {
		creators = append(creators, konnectivity.ServiceReconciler(data.Cluster().Spec.ExposeStrategy, extName))
	} else {
		creators = append(creators,
			openvpn.ServiceReconciler(data.Cluster().Spec.ExposeStrategy),
			metricsserver.ServiceReconciler(),
			dns.ServiceReconciler(),
		)
	}

	if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		creators = append(creators, nodeportproxy.FrontLoadBalancerServiceReconciler(data))
	}

	return creators
}

func (r *Reconciler) ensureServices(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceReconcilers(data)
	return reconciling.ReconcileServices(ctx, creators, c.Status.NamespaceName, r)
}

// GetDeploymentReconcilers returns all DeploymentReconcilers that are currently in use.
func GetDeploymentReconcilers(data *resources.TemplateData, enableAPIserverOIDCAuthentication bool, versions kubermatic.Versions) []reconciling.NamedDeploymentReconcilerFactory {
	deployments := []reconciling.NamedDeploymentReconcilerFactory{
		apiserver.DeploymentReconciler(data, enableAPIserverOIDCAuthentication),
		scheduler.DeploymentReconciler(data),
		controllermanager.DeploymentReconciler(data),
		usercluster.DeploymentReconciler(data),
		userclusterwebhook.DeploymentReconciler(data),
		operatingsystemmanager.DeploymentReconciler(data),
		operatingsystemmanager.WebhookDeploymentReconciler(data),
	}

	// BYO and Edge provider doesn't need machine controller.
	if data.Cluster().Spec.Cloud.Edge == nil {
		deployments = append(deployments, machinecontroller.DeploymentReconciler(data))
		deployments = append(deployments, machinecontroller.WebhookDeploymentReconciler(data))
	}

	if !data.Cluster().Spec.DisableCSIDriver {
		deployments = append(deployments, csi.DeploymentsReconcilers(data)...)
	}

	if data.Cluster().Spec.IsKubernetesDashboardEnabled() {
		deployments = append(deployments, kubernetesdashboard.DeploymentReconciler(data))
	}

	if !data.IsKonnectivityEnabled() {
		deployments = append(deployments,
			openvpn.DeploymentReconciler(data),
			metricsserver.DeploymentReconciler(data),
			dns.DeploymentReconciler(data),
		)
	}

	// If CCM migration is ongoing defer the deployment of the CCM to the
	// moment in which cloud controllers or the full in-tree cloud provider
	// have been deactivated.
	if data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] &&
		(!metav1.HasAnnotation(data.Cluster().ObjectMeta, kubermaticv1.CCMMigrationNeededAnnotation) ||
			data.KCMCloudControllersDeactivated()) {
		deployments = append(deployments, cloudcontroller.DeploymentReconciler(data))
	}

	if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		deployments = append(deployments,
			nodeportproxy.DeploymentEnvoyReconciler(data, versions),
			nodeportproxy.DeploymentLBUpdaterReconciler(data),
		)
	}

	return deployments
}

func (r *Reconciler) ensureDeployments(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	if cluster.Spec.Cloud.ProviderName == string(kubermaticv1.AzureCloudProvider) {
		if err := r.migrateAzureCCM(ctx, cluster); err != nil {
			return fmt.Errorf("failed to migrate Azure CCM Deployment: %w", err)
		}
	}

	modifiers := []reconciling.ObjectModifier{
		modifier.RelatedRevisionsLabels(ctx, r),
		modifier.ControlplaneComponent(cluster),
	}

	factories := GetDeploymentReconcilers(data, r.features.KubernetesOIDCAuthentication, r.versions)
	return reconciling.ReconcileDeployments(ctx, factories, cluster.Status.NamespaceName, r, modifiers...)
}

// In #13180 and its backports the label selectors for the Azure CCM were fixed, but since they are
// immutable, the old CCM Deployment has to be deleted once.
func (r *Reconciler) migrateAzureCCM(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	key := types.NamespacedName{
		Name:      cloudcontroller.AzureCCMDeploymentName,
		Namespace: cluster.Status.NamespaceName,
	}

	dep := appsv1.Deployment{}
	if err := r.Get(ctx, key, &dep); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	// already migrated
	if dep.Spec.Selector.MatchLabels[resources.AppLabelKey] == cloudcontroller.AzureCCMDeploymentName {
		return nil
	}

	if err := r.Delete(ctx, &dep); err != nil {
		return err
	}

	return nil
}

// GetSecretReconcilers returns all SecretReconcilers that are currently in use.
func (r *Reconciler) GetSecretReconcilers(ctx context.Context, data *resources.TemplateData) []reconciling.NamedSecretReconcilerFactory {
	namespace := data.Cluster().Status.NamespaceName

	creators := []reconciling.NamedSecretReconcilerFactory{
		cloudconfig.SecretReconciler(data, resources.CloudConfigSeedSecretName),
		certificates.RootCAReconciler(data),
		certificates.FrontProxyCAReconciler(),
		resources.ImagePullSecretReconciler(r.dockerPullConfigJSON),
		apiserver.FrontProxyClientCertificateReconciler(data),
		etcd.TLSCertificateReconciler(data),
		apiserver.EtcdClientCertificateReconciler(data),
		apiserver.TLSServingCertificateReconciler(data),
		apiserver.KubeletClientCertificateReconciler(data),
		apiserver.ServiceAccountKeyReconciler(),
		userclusterwebhook.TLSServingCertificateReconciler(data),

		// Kubeconfigs
		resources.GetInternalKubeconfigReconciler(namespace, resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigReconciler(namespace, resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigReconciler(namespace, resources.OperatingSystemManagerKubeconfigSecretName, resources.OperatingSystemManagerCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigReconciler(namespace, resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigReconciler(namespace, resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil, data, r.log),
		resources.GetInternalKubeconfigReconciler(namespace, resources.InternalUserClusterAdminKubeconfigSecretName, resources.InternalUserClusterAdminKubeconfigCertUsername, []string{"system:masters"}, data, r.log),
		resources.GetInternalKubeconfigReconciler(namespace, resources.VMwareCloudDirectorCSIKubeconfigSecretName, resources.VMwareCloudDirectorCSICertUsername, nil, data, r.log),
		resources.AdminKubeconfigReconciler(data),
		apiserver.TokenViewerReconciler(),
		apiserver.TokenUsersReconciler(data),
		resources.ViewerKubeconfigReconciler(data),

		// OSM
		resources.GetInternalKubeconfigReconciler(namespace, resources.OperatingSystemManagerWebhookKubeconfigSecretName, resources.OperatingSystemManagerWebhookCertUsername, nil, data, r.log),
		operatingsystemmanager.TLSServingCertificateReconciler(data),
	}

	if data.Cluster().Spec.Cloud.Edge == nil {
		creators = append(creators, machinecontroller.TLSServingCertificateReconciler(data))
	}

	if !data.Cluster().Spec.DisableCSIDriver {
		creators = append(creators, csi.SecretsReconcilers(ctx, data)...)
	}

	if data.Cluster().Spec.IsKubernetesDashboardEnabled() {
		creators = append(creators,
			resources.GetInternalKubeconfigReconciler(namespace, resources.KubernetesDashboardKubeconfigSecretName, resources.KubernetesDashboardCertUsername, nil, data, r.log),
		)
	}

	if data.Cluster().Spec.IsKubeLBEnabled() {
		creators = append(creators,
			resources.GetInternalKubeconfigReconciler(namespace, resources.KubeLBCCMKubeconfigSecretName, resources.KubeLBCCMCertUsername, nil, data, r.log),
		)
	}

	if data.IsKonnectivityEnabled() {
		creators = append(creators,
			konnectivity.TLSServingCertificateReconciler(data),
			resources.GetInternalKubeconfigReconciler(namespace, resources.KonnectivityKubeconfigSecretName, resources.KonnectivityKubeconfigUsername, nil, data, r.log),
		)
	} else {
		creators = append(creators,
			openvpn.CAReconciler(),
			openvpn.TLSServingCertificateReconciler(data),
			openvpn.InternalClientCertificateReconciler(data),
			metricsserver.TLSServingCertSecretReconciler(data.GetRootCA),
			resources.GetInternalKubeconfigReconciler(namespace, resources.MetricsServerKubeconfigSecretName, resources.MetricsServerCertUsername, nil, data, r.log),
			resources.GetInternalKubeconfigReconciler(namespace, resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil, data, r.log),
		)
	}

	if data.Cluster().Spec.AuditLogging != nil && data.Cluster().Spec.AuditLogging.Enabled {
		creators = append(creators, apiserver.FluentBitSecretReconciler(data))
	}

	if data.Cluster().IsEncryptionEnabled() || data.Cluster().IsEncryptionActive() {
		creators = append(creators, apiserver.EncryptionConfigurationSecretReconciler(data))
	}

	if flag := data.Cluster().Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider]; flag {
		creators = append(creators, resources.GetInternalKubeconfigReconciler(
			namespace, resources.CloudControllerManagerKubeconfigSecretName, resources.CloudControllerManagerCertUsername, nil, data, r.log,
		))

		if data.Cluster().Spec.Cloud.Kubevirt != nil {
			creators = append(creators, cloudconfig.KubeVirtInfraSecretReconciler(data))
		}
	}

	if data.Cluster().Spec.Cloud.GCP != nil {
		creators = append(creators, resources.ServiceAccountSecretReconciler(data))
	}

	return creators
}

func (r *Reconciler) ensureSecrets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretReconcilerFactories := r.GetSecretReconcilers(ctx, data)

	if err := reconciling.ReconcileSecrets(ctx, namedSecretReconcilerFactories, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure that the Secret exists: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureServiceAccounts(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedServiceAccountReconcilerFactories := []reconciling.NamedServiceAccountReconcilerFactory{
		etcd.ServiceAccountReconciler,
		usercluster.ServiceAccountReconciler,
		userclusterwebhook.ServiceAccountReconciler,
		operatingsystemmanager.ServiceAccountReconciler,
	}

	if c.Spec.Cloud.Edge == nil {
		namedServiceAccountReconcilerFactories = append(namedServiceAccountReconcilerFactories, machinecontroller.ServiceAccountReconciler)
		namedServiceAccountReconcilerFactories = append(namedServiceAccountReconcilerFactories, machinecontroller.WebhookServiceAccountReconciler)
	}

	if !c.Spec.DisableCSIDriver {
		namedServiceAccountReconcilerFactories = append(namedServiceAccountReconcilerFactories, csi.ServiceAccountReconcilers(c)...)
	}

	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		namedServiceAccountReconcilerFactories = append(namedServiceAccountReconcilerFactories, nodeportproxy.ServiceAccountReconciler)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, namedServiceAccountReconcilerFactories, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure ServiceAccounts: %w", err)
	}

	namedKubeSystemServiceAccountReconcilerFactories := []reconciling.NamedServiceAccountReconcilerFactory{
		etcd.KubeSystemServiceAccountReconciler(c),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, namedKubeSystemServiceAccountReconcilerFactories, metav1.NamespaceSystem, r); err != nil {
		return fmt.Errorf("failed to ensure ServiceAccounts in %s namespace: %w", metav1.NamespaceSystem, err)
	}

	return nil
}

func (r *Reconciler) ensureRoles(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedRoleReconcilerFactories := []reconciling.NamedRoleReconcilerFactory{
		usercluster.RoleReconciler,
	}

	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		namedRoleReconcilerFactories = append(namedRoleReconcilerFactories, nodeportproxy.RoleReconciler)
	}

	if err := reconciling.ReconcileRoles(ctx, namedRoleReconcilerFactories, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure Roles: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureRoleBindings(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedRoleBindingReconcilerFactories := []reconciling.NamedRoleBindingReconcilerFactory{
		usercluster.RoleBindingReconciler,
	}
	if !c.Spec.DisableCSIDriver {
		namedRoleBindingReconcilerFactories = append(namedRoleBindingReconcilerFactories, csi.RoleBindingsReconcilers(c)...)
	}

	if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		namedRoleBindingReconcilerFactories = append(namedRoleBindingReconcilerFactories, nodeportproxy.RoleBindingReconciler)
	}

	if err := reconciling.ReconcileRoleBindings(ctx, namedRoleBindingReconcilerFactories, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure RoleBindings: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureClusterRoles(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedClusterRoleReconcilerFactories := []reconciling.NamedClusterRoleReconcilerFactory{
		usercluster.ClusterRole(),
		userclusterwebhook.ClusterRole(),
	}

	if !c.Spec.DisableCSIDriver {
		namedClusterRoleReconcilerFactories = append(namedClusterRoleReconcilerFactories, csi.ClusterRolesReconcilers(c)...)
	}

	if err := reconciling.ReconcileClusterRoles(ctx, namedClusterRoleReconcilerFactories, "", r); err != nil {
		return fmt.Errorf("failed to ensure Cluster Roles: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureClusterRoleBindings(ctx context.Context, namespace *corev1.Namespace) error {
	namedClusterRoleBindingsReconcilerFactories := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		usercluster.ClusterRoleBinding(namespace),
		userclusterwebhook.ClusterRoleBinding(namespace),
	}
	if err := reconciling.ReconcileClusterRoleBindings(ctx, namedClusterRoleBindingsReconcilerFactories, "", r); err != nil {
		return fmt.Errorf("failed to ensure Cluster Role Bindings: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureNetworkPolicies(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData, cfg *kubermaticv1.KubermaticConfiguration) error {
	if c.Spec.Features[kubermaticv1.ApiserverNetworkPolicy] {
		namedNetworkPolicyReconcilerFactories := []reconciling.NamedNetworkPolicyReconcilerFactory{
			apiserver.DenyAllPolicyReconciler(),
			apiserver.DNSAllowReconciler(c, data),
			apiserver.EctdAllowReconciler(c),
			apiserver.MachineControllerWebhookAllowReconciler(c),
			apiserver.UserClusterWebhookAllowReconciler(c),
			apiserver.OSMWebhookAllowReconciler(c),
		}

		// one shared limited context for all hostname resolutions
		resolverCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if data.IsKonnectivityEnabled() {
			namedNetworkPolicyReconcilerFactories = append(namedNetworkPolicyReconcilerFactories, apiserver.ApiserverInternalAllowReconciler())
		} else {
			namedNetworkPolicyReconcilerFactories = append(namedNetworkPolicyReconcilerFactories,
				apiserver.OpenVPNServerAllowReconciler(c),
				apiserver.MetricsServerAllowReconciler(c),
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
			namedNetworkPolicyReconcilerFactories = append(namedNetworkPolicyReconcilerFactories, apiserver.OIDCIssuerAllowReconciler(ipList, cfg.Spec.Ingress.NamespaceOverride))
		}

		apiIPs, err := r.fetchKubernetesServiceIPList(ctx, resolverCtx)
		if err != nil {
			return fmt.Errorf("failed to fetch Kubernetes API service IP list: %w", err)
		}

		namedNetworkPolicyReconcilerFactories = append(namedNetworkPolicyReconcilerFactories, apiserver.SeedApiserverAllowReconciler(apiIPs))

		if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyReconcilerFactories, c.Status.NamespaceName, r); err != nil {
			return fmt.Errorf("failed to ensure Network Policies: %w", err)
		}
	}

	return nil
}

// GetConfigMapReconcilers returns all ConfigMapReconcilers that are currently in use.
func GetConfigMapReconcilers(data *resources.TemplateData) []reconciling.NamedConfigMapReconcilerFactory {
	creators := []reconciling.NamedConfigMapReconcilerFactory{
		apiserver.AuditConfigMapReconciler(data),
		apiserver.AdmissionControlReconciler(data),
		apiserver.CABundleReconciler(data),
	}
	if !data.Cluster().Spec.DisableCSIDriver {
		creators = append(creators, csi.ConfigMapsReconcilers(data)...)
	}

	if data.IsKonnectivityEnabled() {
		creators = append(creators, apiserver.EgressSelectorConfigReconciler())
	} else {
		creators = append(creators,
			openvpn.ServerClientConfigsConfigMapReconciler(data),
			dns.ConfigMapReconciler(data),
		)
	}

	return creators
}

func (r *Reconciler) ensureConfigMaps(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapReconcilers(data)

	if err := reconciling.ReconcileConfigMaps(ctx, creators, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %w", err)
	}

	return nil
}

// GetStatefulSetReconcilers returns all StatefulSetReconcilers that are currently in use.
func GetStatefulSetReconcilers(data *resources.TemplateData, enableDataCorruptionChecks, enableTLSOnly bool, quotaBackendGB int64) []reconciling.NamedStatefulSetReconcilerFactory {
	return []reconciling.NamedStatefulSetReconcilerFactory{
		etcd.StatefulSetReconciler(data, enableDataCorruptionChecks, enableTLSOnly, quotaBackendGB),
	}
}

// GetEtcdBackupConfigReconcilers returns all EtcdBackupConfigReconcilers that are currently in use.
func GetEtcdBackupConfigReconcilers(data *resources.TemplateData, seed *kubermaticv1.Seed) []kkpreconciling.NamedEtcdBackupConfigReconcilerFactory {
	creators := []kkpreconciling.NamedEtcdBackupConfigReconcilerFactory{
		etcd.BackupConfigReconciler(data, seed),
	}
	return creators
}

// GetPodDisruptionBudgetReconcilers returns all PodDisruptionBudgetReconcilers that are currently in use.
func GetPodDisruptionBudgetReconcilers(data *resources.TemplateData) []reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	creators := []reconciling.NamedPodDisruptionBudgetReconcilerFactory{
		etcd.PodDisruptionBudgetReconciler(data),
		apiserver.PodDisruptionBudgetReconciler(),
	}
	if !data.IsKonnectivityEnabled() {
		creators = append(creators,
			metricsserver.PodDisruptionBudgetReconciler(),
			dns.PodDisruptionBudgetReconciler(),
		)
	}

	if data.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		creators = append(creators, nodeportproxy.PodDisruptionBudgetReconciler())
	}

	return creators
}

func (r *Reconciler) ensurePodDisruptionBudgets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetPodDisruptionBudgetReconcilers(data)

	if err := reconciling.ReconcilePodDisruptionBudgets(ctx, creators, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure that the PodDisruptionBudget exists: %w", err)
	}

	return nil
}

// GetCronJobReconcilers returns all CronJobReconcilers that are currently in use.
func GetCronJobReconcilers(data *resources.TemplateData) []reconciling.NamedCronJobReconcilerFactory {
	return []reconciling.NamedCronJobReconcilerFactory{
		etcd.CronJobReconciler(data),
	}
}

func (r *Reconciler) ensureCronJobs(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetCronJobReconcilers(data)

	if err := reconciling.ReconcileCronJobs(ctx, creators, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure that the CronJobs exists: %w", err)
	}

	return nil
}

func (r *Reconciler) ensureVerticalPodAutoscalers(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	controlPlaneDeploymentNames := []string{
		resources.ApiserverDeploymentName,
		resources.ControllerManagerDeploymentName,
		resources.SchedulerDeploymentName,
	}

	// machine controller is not deployed for the edge clusters
	if c.Spec.Cloud.Edge == nil {
		controlPlaneDeploymentNames = append(controlPlaneDeploymentNames,
			resources.MachineControllerDeploymentName,
			resources.MachineControllerWebhookDeploymentName,
		)
	}

	if !data.IsKonnectivityEnabled() {
		controlPlaneDeploymentNames = append(controlPlaneDeploymentNames,
			resources.OpenVPNServerDeploymentName,
			resources.MetricsServerDeploymentName,
			resources.DNSResolverDeploymentName,
		)
	}

	creators, err := resources.GetVerticalPodAutoscalersForAll(ctx, r, controlPlaneDeploymentNames, []string{resources.EtcdStatefulSetName}, c.Status.NamespaceName, r.features.VPA)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %w", err)
	}

	return kkpreconciling.ReconcileVerticalPodAutoscalers(ctx, creators, c.Status.NamespaceName, r)
}

func (r *Reconciler) ensureStatefulSets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	useTLSOnly, err := r.etcdUseStrictTLS(ctx, c)
	if err != nil {
		return err
	}

	quotaBackendGB := int64(0)
	if b := c.Spec.ComponentsOverride.Etcd.QuotaBackendGB; b != nil {
		quotaBackendGB = *b
	}

	creators := GetStatefulSetReconcilers(data, r.features.EtcdDataCorruptionChecks, useTLSOnly, quotaBackendGB)

	modifiers := []reconciling.ObjectModifier{
		modifier.RelatedRevisionsLabels(ctx, r),
		modifier.ControlplaneComponent(c),
	}

	return reconciling.ReconcileStatefulSets(ctx, creators, c.Status.NamespaceName, r, modifiers...)
}

func (r *Reconciler) ensureEtcdBackupConfigs(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData,
	seed *kubermaticv1.Seed) error {
	if seed.IsDefaultEtcdAutomaticBackupEnabled() {
		creators := GetEtcdBackupConfigReconcilers(data, seed)
		return kkpreconciling.ReconcileEtcdBackupConfigs(ctx, creators, c.Status.NamespaceName, r)
	}
	// If default etcd automatic backups are not enabled, remove them if any
	ebc := &kubermaticv1.EtcdBackupConfig{}
	err := r.Get(ctx, types.NamespacedName{Name: resources.EtcdDefaultBackupConfigName, Namespace: c.Status.NamespaceName}, ebc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.Delete(ctx, ebc)
}

func (r *Reconciler) ensureOldOPAIntegrationIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range gatekeeper.GetResourcesToRemoveOnDelete(data.Cluster().Status.NamespaceName) {
		if err := r.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure old OPA integration version resources are removed/not present: %w", err)
		}
	}

	return nil
}

func (r *Reconciler) ensureKubernetesDashboardResourcesAreRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range kubernetesdashboard.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure kubernetes-dashboard resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureCSIDriverResourcesAreRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range csi.ResourcesForDeletion(data.Cluster()) {
		err := r.Delete(ctx, resource)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure CSI driver resources are removed/not present: %w", err)
		}
	}
	return nil
}

func (r *Reconciler) ensureOpenVPNSetupIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range openvpn.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure OpenVPN resources are removed/not present: %w", err)
		}
	}
	for _, resource := range metricsserver.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure metrics-server resources are removed/not present: %w", err)
		}
	}
	for _, resource := range dns.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure dns-resolver resources are removed/not present: %w", err)
		}
	}
	dnatControllerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.KubeletDnatControllerKubeconfigSecretName,
			Namespace: data.Cluster().Status.NamespaceName,
		},
	}
	if err := r.Delete(ctx, dnatControllerSecret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to ensure DNAT controller resources are removed/not present: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureKonnectivitySetupIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range konnectivity.ResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure Konnectivity resources are removed/not present: %w", err)
		}
	}
	return nil
}

// ensureKonnectivityNetworkPolicyIsRemoved removes the NetworkPolicy put in place for
// konnectivity-server -> external API server endpoint communication.
func (r *Reconciler) ensureKonnectivityNetworkPolicyIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	if err := r.Delete(ctx, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.NetworkPolicyClusterExternalAddrAllow,
			Namespace: data.Cluster().Status.NamespaceName,
		},
	},
	); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to ensure external konnectivity-server NetworkPolicyy is removed/not present: %w", err)
	}
	return nil
}

func (r *Reconciler) ensureEncryptionConfigurationIsRemoved(ctx context.Context, data *resources.TemplateData) error {
	for _, resource := range apiserver.EncryptionResourcesForDeletion(data.Cluster().Status.NamespaceName) {
		if err := r.Delete(ctx, resource); err != nil && !apierrors.IsNotFound(err) {
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

	if err := r.ensureClusterRoles(ctx, cluster); err != nil {
		return err
	}

	if err := r.ensureClusterRoleBindings(ctx, namespace); err != nil {
		return err
	}

	return nil
}

// fetchKubernetesServiceIPList looks up all IPs for the 'kubernetes' Service in the 'default' namespace
// so that we get a complete list of IPs under which the seed cluster's Kubernetes API is available.
func (r *Reconciler) fetchKubernetesServiceIPList(ctx context.Context, resolverCtx context.Context) ([]net.IP, error) {
	ips := []net.IP{}

	// fetch endpoint slices for the "kubernetes" Service.
	endpointSlices := &discoveryv1.EndpointSliceList{}
	if err := r.List(ctx, endpointSlices,
		ctrlruntimeclient.InNamespace(corev1.NamespaceDefault),
		ctrlruntimeclient.MatchingLabels{"kubernetes.io/service-name": "kubernetes"},
	); err != nil {
		return nil, fmt.Errorf("failed to list EndpointSlices: %w", err)
	}

	// loop over all EndpointSlices and extract IPs depending on what kind
	// of endpoints are configured for the 'kubernetes' Service.
	for _, slice := range endpointSlices.Items {
		for _, endpoint := range slice.Endpoints {
			for _, address := range endpoint.Addresses {
				if slice.AddressType == discoveryv1.AddressTypeFQDN {
					ipList, err := hostnameToIPList(resolverCtx, address)
					if err != nil {
						return nil, fmt.Errorf("failed to resolve FQDM %q: %w", address, err)
					}
					ips = append(ips, ipList...)
				} else {
					ips = append(ips, net.ParseIP(address))
				}
			}
		}
	}

	return ips, nil
}

// listAPIServerAlternateNames returns the alternate names for the apiserver certificate from the
// corresponding services. This ensures that if multiple hostnames or IPs have been assigned to the
// API server service or front-loadbalancer service, then all of them are included in the certificate.
func (r *Reconciler) listAPIServerAlternateNames(ctx context.Context, cluster *kubermaticv1.Cluster) (*certutil.AltNames, error) {
	dnsNames := []string{}
	ips := []net.IP{}

	// Get all the loadbalancer Ingresses from the API server service.
	// These services are managed by this controller and might not exist on the first reconciliation.
	// That's okay because even if they did, the CCM might not have assigned a LoadBalancer yet.

	service := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.ApiserverServiceName}, service); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return nil, fmt.Errorf("failed to get API server service: %w", err)
	}

	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				ips = append(ips, net.ParseIP(ingress.IP))
			}
			if ingress.Hostname != "" {
				dnsNames = append(dnsNames, ingress.Hostname)
			}
		}
	}

	if cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		service := &corev1.Service{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.FrontLoadBalancerServiceName}, service); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return nil, fmt.Errorf("failed to get front-loadbalancer service: %w", err)
		}

		if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
			for _, ingress := range service.Status.LoadBalancer.Ingress {
				if ingress.IP != "" {
					ips = append(ips, net.ParseIP(ingress.IP))
				}
				if ingress.Hostname != "" {
					dnsNames = append(dnsNames, ingress.Hostname)
				}
			}
		}
	}

	return &certutil.AltNames{
		DNSNames: dnsNames,
		IPs:      ips,
	}, nil
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

func (r *Reconciler) ensureAuditWebhook(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	if data.DC().Spec.AuditLogging != nil && data.DC().Spec.AuditLogging.WebhookBackend != nil {
		// if webhook backend is enabled on the DC then create the auditwebhookconfig secret in the user cluster ns.
		creators := []reconciling.NamedSecretReconcilerFactory{r.auditWebhookSecretReconciler(ctx, data)}
		if err := reconciling.ReconcileSecrets(ctx, creators, c.Status.NamespaceName, r); err != nil {
			return err
		}
	}
	return nil
}

// auditWebhookSecretReconciler returns a reconciling.NamedSecretReconcilerFactory for a secret that contains
// audit webhook configuration for api server audit logs.
func (r *Reconciler) auditWebhookSecretReconciler(ctx context.Context, data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return data.DC().Spec.AuditLogging.WebhookBackend.AuditWebhookConfig.Name, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}
			if data.DC().Spec.AuditLogging.WebhookBackend != nil {
				webhookBackendSecret := &corev1.Secret{}
				err := r.Get(ctx, types.NamespacedName{Name: data.DC().Spec.AuditLogging.WebhookBackend.AuditWebhookConfig.Name, Namespace: data.DC().Spec.AuditLogging.WebhookBackend.AuditWebhookConfig.Namespace}, webhookBackendSecret)
				if err != nil {
					return secret, err
				} else {
					secret.Data = webhookBackendSecret.Data
				}
			}
			return secret, nil
		}
	}
}
