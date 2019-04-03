package cluster

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/controllermanager"
	"github.com/kubermatic/kubermatic/api/pkg/resources/dns"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/scheduler"
	"github.com/kubermatic/kubermatic/api/pkg/resources/usercluster"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
)

const (
	NodeDeletionFinalizer = "kubermatic.io/delete-nodes"
)

func (r *Reconciler) ensureResourcesAreDeployed(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	data, err := r.getClusterTemplateData(ctx, cluster)
	if err != nil {
		return err
	}

	// check that all services are available
	if cluster.Annotations["kubermatic.io/openshift"] == "" {
		if err := r.ensureServices(cluster, data); err != nil {
			return err
		}
	}

	// check that all secrets are available // New way of handling secrets
	if cluster.Annotations["kubermatic.io/openshift"] == "" {
		if err := r.ensureSecrets(cluster, data); err != nil {
			return err
		}
	}

	// check that all StatefulSets are created
	if err := r.ensureStatefulSets(cluster, data); err != nil {
		return err
	}

	// Wait until the cloud provider infra is ready before attempting
	// to render the cloud-config
	// TODO: Model resource deployment as a DAG so we don't need hacks
	// like this combined with tribal knowledge and "someone is noticing this
	// isn't working correctly"
	// https://github.com/kubermatic/kubermatic/issues/2948
	if !cluster.Status.Health.CloudProviderInfrastructure {
		return nil
	}

	if cluster.Annotations["kubermatic.io/openshift"] == "" {
		// check that all ConfigMaps are available
		if err := r.ensureConfigMaps(cluster, data); err != nil {
			return err
		}
	}

	// check that all Deployments are available
	if err := r.ensureDeployments(cluster, data); err != nil {
		return err
	}

	// check that all CronJobs are created
	if err := r.ensureCronJobs(cluster, data); err != nil {
		return err
	}

	// check that all PodDisruptionBudgets are created
	if err := r.ensurePodDisruptionBudgets(cluster, data); err != nil {
		return err
	}

	// check that all StatefulSets are created
	if err := r.ensureVerticalPodAutoscalers(cluster, data); err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) getClusterTemplateData(ctx context.Context, cluster *kubermaticv1.Cluster) (*resources.TemplateData, error) {
	dc, found := r.dcs[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	return resources.NewTemplateData(
		ctx,
		r,
		cluster,
		&dc,
		r.dc,
		r.overwriteRegistry,
		r.nodePortRange,
		r.nodeAccessNetwork,
		r.etcdDiskSize,
		r.monitoringScrapeAnnotationPrefix,
		r.inClusterPrometheusRulesFile,
		r.inClusterPrometheusDisableDefaultRules,
		r.inClusterPrometheusDisableDefaultScrapingConfigs,
		r.inClusterPrometheusScrapingConfigsFile,
		r.oidcCAFile,
		r.oidcIssuerURL,
		r.oidcIssuerClientID,
		r.enableEtcdDataCorruptionChecks,
	), nil
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
	return []reconciling.NamedServiceCreatorGetter{
		apiserver.InternalServiceCreator(),
		apiserver.ExternalServiceCreator(),
		openvpn.ServiceCreator(),
		etcd.ServiceCreator(data),
		dns.ServiceCreator(),
		machinecontroller.ServiceCreator(),
		metricsserver.ServiceCreator(),
	}
}

func (r *Reconciler) ensureServices(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetServiceCreators(data)
	return reconciling.ReconcileServices(creators, c.Status.NamespaceName, r, r.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(c)))
}

// GetDeploymentCreators returns all DeploymentCreators that are currently in use
func GetDeploymentCreators(data *resources.TemplateData) []reconciling.NamedDeploymentCreatorGetter {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		openvpn.DeploymentCreator(data),
		dns.DeploymentCreator(data),
	}

	if cluster := data.Cluster(); cluster != nil && cluster.Annotations["kubermatic.io/openshift"] == "" {
		creators = append(creators, apiserver.DeploymentCreator(data))
		creators = append(creators, scheduler.DeploymentCreator(data))
		creators = append(creators, controllermanager.DeploymentCreator(data))
		creators = append(creators, machinecontroller.DeploymentCreator(data))
		creators = append(creators, machinecontroller.WebhookDeploymentCreator(data))
		creators = append(creators, metricsserver.DeploymentCreator(data))
		creators = append(creators, usercluster.DeploymentCreator(data, false))
	}

	return creators
}

func (r *Reconciler) ensureDeployments(cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data)
	return reconciling.ReconcileDeployments(creators, cluster.Status.NamespaceName, r, r.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

// GetSecretCreators returns all SecretCreators that are currently in use
func (r *Reconciler) GetSecretCreators(data *resources.TemplateData) []reconciling.NamedSecretCreatorGetter {
	creators := []reconciling.NamedSecretCreatorGetter{
		certificates.RootCACreator(data),
		openvpn.CACreator(),
		certificates.FrontProxyCACreator(),
		resources.ImagePullSecretCreator(r.dockerPullConfigJSON),
		apiserver.FrontProxyClientCertificateCreator(data),
		etcd.TLSCertificateCreator(data),
		apiserver.EtcdClientCertificateCreator(data),
		apiserver.TLSServingCertificateCreator(data),
		apiserver.KubeletClientCertificateCreator(data),
		apiserver.ServiceAccountKeyCreator(),
		openvpn.TLSServingCertificateCreator(data),
		openvpn.InternalClientCertificateCreator(data),
		machinecontroller.TLSServingCertificateCreator(data),

		// Kubeconfigs
		resources.GetInternalKubeconfigCreator(resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.MetricsServerKubeconfigSecretName, resources.MetricsServerCertUsername, nil, data),
		resources.GetInternalKubeconfigCreator(resources.InternalUserClusterAdminKubeconfigSecretName, resources.InternalUserClusterAdminKubeconfigCertUsername, []string{"system:masters"}, data),
		resources.AdminKubeconfigCreator(data),
		apiserver.TokenUsersCreator(data),
	}

	if len(data.OIDCCAFile()) > 0 {
		creators = append(creators, apiserver.DexCACertificateCreator(data))
	}

	return creators
}

func (r *Reconciler) ensureSecrets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	namedSecretCreatorGetters := r.GetSecretCreators(data)

	if err := reconciling.ReconcileSecrets(namedSecretCreatorGetters, c.Status.NamespaceName, r, r.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the Secret exists: %v", err)
	}

	return nil
}

// GetConfigMapCreators returns all ConfigMapCreators that are currently in use
func GetConfigMapCreators(data *resources.TemplateData) []reconciling.NamedConfigMapCreatorGetter {
	return []reconciling.NamedConfigMapCreatorGetter{
		cloudconfig.ConfigMapCreator(data),
		openvpn.ServerClientConfigsConfigMapCreator(data),
		dns.ConfigMapCreator(data),
	}
}

func (r *Reconciler) ensureConfigMaps(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	if err := reconciling.ReconcileConfigMaps(creators, c.Status.NamespaceName, r, r.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %v", err)
	}

	return nil
}

// GetStatefulSetCreators returns all StatefulSetCreators that are currently in use
func GetStatefulSetCreators(data *resources.TemplateData) []reconciling.NamedStatefulSetCreatorGetter {
	return []reconciling.NamedStatefulSetCreatorGetter{
		etcd.StatefulSetCreator(data),
	}
}

// GetPodDisruptionBudgetCreators returns all PodDisruptionBudgetCreators that are currently in use
func GetPodDisruptionBudgetCreators(data *resources.TemplateData) []reconciling.NamedPodDisruptionBudgetCreatorGetter {
	return []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		etcd.PodDisruptionBudgetCreator(data),
		apiserver.PodDisruptionBudgetCreator(),
		metricsserver.PodDisruptionBudgetCreator(),
	}
}

func (r *Reconciler) ensurePodDisruptionBudgets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetPodDisruptionBudgetCreators(data)

	if err := reconciling.ReconcilePodDisruptionBudgets(creators, c.Status.NamespaceName, r, r.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
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

func (r *Reconciler) ensureCronJobs(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetCronJobCreators(data)

	if err := reconciling.ReconcileCronJobs(creators, c.Status.NamespaceName, r, r.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the CronJobs exists: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureVerticalPodAutoscalers(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	controlPlaneDeploymentNames := []string{
		"dns-resolver",
		"machine-controller",
		"machine-controller-webhook",
		"openvpn-server",
	}
	if c.Annotations["kubermatic.io/openshift"] == "" {
		controlPlaneDeploymentNames = append(controlPlaneDeploymentNames, "apiserver", "controller-manager", "scheduler", "metrics-server")
	}
	creators, err := resources.GetVerticalPodAutoscalersForAll(controlPlaneDeploymentNames, []string{"etcd"}, c.Status.NamespaceName, r.dynamicCache)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %v", err)
	}

	if !r.enableVPA {
		// If the feature is disabled, we just wrap the create function to disable the VPA.
		// This is easier than passing a bool to all required functions.
		for i, getNameAndCreator := range creators {
			creators[i] = func() (string, reconciling.VerticalPodAutoscalerCreator) {
				name, create := getNameAndCreator()
				return name, disableVPAWrapper(create)
			}
		}
	}

	return reconciling.ReconcileVerticalPodAutoscalers(creators, c.Status.NamespaceName, r, r.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(c)))
}

// disableVPAWrapper is a wrapper function which sets the UpdateMode on the VPA to UpdateModeOff.
// This essentially disables any processing from the VerticalPodAutoscaler
func disableVPAWrapper(create reconciling.VerticalPodAutoscalerCreator) reconciling.VerticalPodAutoscalerCreator {
	return func(vpa *autoscalingv1beta2.VerticalPodAutoscaler) (*autoscalingv1beta2.VerticalPodAutoscaler, error) {
		vpa, err := create(vpa)
		if err != nil {
			return nil, err
		}

		if vpa.Spec.UpdatePolicy == nil {
			vpa.Spec.UpdatePolicy = &autoscalingv1beta2.PodUpdatePolicy{}
		}
		mode := autoscalingv1beta2.UpdateModeOff
		vpa.Spec.UpdatePolicy.UpdateMode = &mode

		return vpa, nil
	}
}

func (r *Reconciler) ensureStatefulSets(c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetStatefulSetCreators(data)

	return reconciling.ReconcileStatefulSets(creators, c.Status.NamespaceName, r, r.dynamicCache, reconciling.OwnerRefWrapper(resources.GetClusterRef(c)))
}
