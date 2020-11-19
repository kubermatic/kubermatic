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

package openshift

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/clusterdeletion"
	openshiftresources "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/openshift/resources"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/address"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig"
	"k8c.io/kubermatic/v2/pkg/resources/clusterautoscaler"
	"k8c.io/kubermatic/v2/pkg/resources/dns"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"
	"k8c.io/kubermatic/v2/pkg/resources/machinecontroller"
	metricsserver "k8c.io/kubermatic/v2/pkg/resources/metrics-server"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/openvpn"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/usercluster"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "openshift_controller"

// Check if the Reconciler fulfills the interface
// at compile time
var _ reconcile.Reconciler = &Reconciler{}

type OIDCConfig struct {
	IssuerURL    string
	CAFile       string
	ClientID     string
	ClientSecret string
}

type Features struct {
	EtcdDataCorruptionChecks bool
	VPA                      bool
}

type Reconciler struct {
	ctrlruntimeclient.Client

	log                         *zap.SugaredLogger
	scheme                      *runtime.Scheme
	recorder                    record.EventRecorder
	seedGetter                  provider.SeedGetter
	userClusterConnProvider     *clusterclient.Provider
	overwriteRegistry           string
	nodeAccessNetwork           string
	etcdDiskSize                resource.Quantity
	dockerPullConfigJSON        []byte
	workerName                  string
	externalURL                 string
	oidc                        OIDCConfig
	kubermaticImage             string
	etcdLauncherImage           string
	dnatControllerImage         string
	features                    Features
	concurrentClusterUpdates    int
	etcdBackupRestoreController bool
	backupSchedule              time.Duration
	versions                    kubermatic.Versions
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	seedGetter provider.SeedGetter,
	userClusterConnProvider *clusterclient.Provider,
	overwriteRegistry,
	nodeAccessNetwork string,
	etcdDiskSize resource.Quantity,
	dockerPullConfigJSON []byte,
	externalURL string,
	kubermaticImage string,
	etcdLauncherImage string,
	dnatControllerImage string,
	etcdBackupRestoreController bool,
	backupSchedule time.Duration,
	features Features,
	concurrentClusterUpdates int,
	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		Client:                      mgr.GetClient(),
		log:                         log.Named(ControllerName),
		scheme:                      mgr.GetScheme(),
		recorder:                    mgr.GetEventRecorderFor(ControllerName),
		seedGetter:                  seedGetter,
		userClusterConnProvider:     userClusterConnProvider,
		overwriteRegistry:           overwriteRegistry,
		nodeAccessNetwork:           nodeAccessNetwork,
		etcdDiskSize:                etcdDiskSize,
		dockerPullConfigJSON:        dockerPullConfigJSON,
		workerName:                  workerName,
		externalURL:                 externalURL,
		kubermaticImage:             kubermaticImage,
		etcdLauncherImage:           etcdLauncherImage,
		dnatControllerImage:         dnatControllerImage,
		features:                    features,
		concurrentClusterUpdates:    concurrentClusterUpdates,
		etcdBackupRestoreController: etcdBackupRestoreController,
		backupSchedule:              backupSchedule,
		versions:                    versions,
	}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	typesToWatch := []ctrlruntimeclient.Object{
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Namespace{},
		&appsv1.StatefulSet{},
		&appsv1.Deployment{},
		&batchv1beta1.CronJob{},
		&policyv1beta1.PodDisruptionBudget{},
		&autoscalingv1beta2.VerticalPodAutoscaler{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	log = log.With("cluster", cluster.Name)

	if !cluster.IsOpenshift() {
		log.Debug("Skipping because the cluster is an Kubernetes cluster")
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionOpenshiftControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			// only reconcile this cluster if there are not yet too many updates running
			if available, err := controllerutil.ClusterAvailableForReconciling(ctx, r, cluster, r.concurrentClusterUpdates); !available || err != nil {
				log.Infow("Concurrency limit reached, checking again in 10 seconds", "concurrency-limit", r.concurrentClusterUpdates)
				return &reconcile.Result{RequeueAfter: 10 * time.Second}, err
			}

			return r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	if result == nil {
		result = &reconcile.Result{}
	}

	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.DeletionTimestamp != nil {
		log.Debug("Cleaning up cluster")

		// Defer getting the client to make sure we only request it if we actually need it
		userClusterClientGetter := func() (ctrlruntimeclient.Client, error) {
			userClusterClient, err := r.userClusterConnProvider.GetClient(ctx, cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to get user cluster client: %v", err)
			}

			log.Debugw("Getting client for cluster", "cluster", cluster.Name)
			return userClusterClient, nil
		}

		// Always requeue a cluster after we executed the cleanup.
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, clusterdeletion.New(r.Client, userClusterClientGetter, r.etcdBackupRestoreController).CleanupCluster(ctx, log, cluster)
	}

	if cluster.Spec.Openshift == nil {
		return nil, errors.New("openshift cluster but .Spec.Openshift is unset")
	}

	// Ensure Namespace
	if err := r.ensureNamespace(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to ensure Namespace: %v", err)
	}

	if err := r.networkDefaults(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to setup cluster networking defaults: %v", err)
	}

	if err := r.reconcileResources(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to reconcile resources: %v", err)
	}

	if err := r.syncHeath(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to sync health: %v", err)
	}

	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		// We have to wait for the APIServer to not be completely down. Changes for it
		// will trigger us, so we can just return here.
		return nil, nil
	}

	reachable, err := r.clusterIsReachable(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to check if cluster is reachable: %v", err)
	}
	if !reachable {
		log.Debug("Cluster is not reachable yet, retrying later")
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Only add the node deletion finalizer when the cluster is actually running
	// Otherwise we fail to delete the nodes and are stuck in a loop
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer,
		kubermaticapiv1.InClusterImageRegistryConfigCleanupFinalizer,
		kubermaticapiv1.InClusterCredentialsRequestsCleanupFinalizer) {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer,
				kubermaticapiv1.InClusterImageRegistryConfigCleanupFinalizer,
				kubermaticapiv1.InClusterCredentialsRequestsCleanupFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// clusterIsReachable checks if the cluster is reachable via its external name
func (r *Reconciler) clusterIsReachable(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	client, err := r.userClusterConnProvider.GetClient(ctx, c)
	if err != nil {
		return false, err
	}

	if err := client.List(ctx, &corev1.NamespaceList{}); err != nil {
		return false, nil
	}

	return true, nil
}

func (r *Reconciler) syncHeath(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	currentHealth := cluster.Status.ExtendedHealth.DeepCopy()
	type depInfo struct {
		healthStatus *kubermaticv1.HealthStatus
		minReady     int32
	}

	healthMapping := map[string]*depInfo{
		resources.ApiserverDeploymentName:             {healthStatus: &currentHealth.Apiserver, minReady: 1},
		resources.ControllerManagerDeploymentName:     {healthStatus: &currentHealth.Controller, minReady: 1},
		resources.MachineControllerDeploymentName:     {healthStatus: &currentHealth.MachineController, minReady: 1},
		resources.OpenVPNServerDeploymentName:         {healthStatus: &currentHealth.OpenVPN, minReady: 1},
		resources.UserClusterControllerDeploymentName: {healthStatus: &currentHealth.UserClusterControllerManager, minReady: 1},
	}

	for name := range healthMapping {
		status, err := resources.HealthyDeployment(ctx, r.Client, nn(cluster.Status.NamespaceName, name), healthMapping[name].minReady)
		if err != nil {
			return fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthStatus = kubermaticv1helper.GetHealthStatus(status, cluster, r.versions)
	}

	status, err := resources.HealthyStatefulSet(ctx, r.Client, nn(cluster.Status.NamespaceName, resources.EtcdStatefulSetName), 2)
	if err != nil {
		return fmt.Errorf("failed to get etcd health: %v", err)
	}
	currentHealth.Etcd = kubermaticv1helper.GetHealthStatus(status, cluster, r.versions)

	//TODO: Revisit this. This is a tiny bit ugly, but Openshift doesn't have a distinct scheduler
	// and introducing a distinct health struct for Openshift means we have to change the API as well
	currentHealth.Scheduler = currentHealth.Controller

	if cluster.Status.ExtendedHealth != *currentHealth {
		return r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Status.ExtendedHealth = *currentHealth
		})
	}

	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionClusterInitialized, corev1.ConditionTrue) && kubermaticv1helper.IsClusterInitialized(cluster, r.versions) {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			kubermaticv1helper.SetClusterCondition(
				c,
				r.versions,
				kubermaticv1.ClusterConditionClusterInitialized,
				corev1.ConditionTrue,
				"",
				"Cluster has been initialized successfully",
			)
		})
	}

	return err
}

func (r *Reconciler) updateCluster(ctx context.Context, c *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster)) error {
	oldCluster := c.DeepCopy()
	modify(c)
	return r.Patch(ctx, c, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (r *Reconciler) getAllSecretCreators(ctx context.Context, osData *openshiftData) []reconciling.NamedSecretCreatorGetter {
	creators := []reconciling.NamedSecretCreatorGetter{
		certificates.RootCACreator(osData),
		openvpn.CACreator(),
		certificates.FrontProxyCACreator(),
		openshiftresources.OpenShiftTLSServingCertificateCreator(osData),
		openshiftresources.ServiceSignerCA(),
		openshiftresources.OpenshiftControllerManagerServingCertSecretCreator(osData.GetRootCA),
		resources.ImagePullSecretCreator(r.dockerPullConfigJSON),
		apiserver.FrontProxyClientCertificateCreator(osData),
		etcd.TLSCertificateCreator(osData),
		apiserver.EtcdClientCertificateCreator(osData),
		apiserver.TLSServingCertificateCreator(osData),
		apiserver.KubeletClientCertificateCreator(osData),
		apiserver.ServiceAccountKeyCreator(),
		openvpn.TLSServingCertificateCreator(osData),
		openvpn.InternalClientCertificateCreator(osData),
		machinecontroller.TLSServingCertificateCreator(osData),
		openshiftresources.KubeSchedulerServingCertCreator(osData.GetRootCA),
		metricsserver.TLSServingCertSecretCreator(osData.GetRootCA),

		// Kubeconfigs
		resources.GetInternalKubeconfigCreator(resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.MetricsServerKubeconfigSecretName, resources.MetricsServerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.InternalUserClusterAdminKubeconfigSecretName, resources.InternalUserClusterAdminKubeconfigCertUsername, []string{"system:masters"}, osData),
		resources.GetInternalKubeconfigCreator(resources.ClusterAutoscalerKubeconfigSecretName, resources.ClusterAutoscalerCertUsername, nil, osData),
		openshiftresources.ImagePullSecretCreator(osData.Cluster()),
		openshiftresources.OauthSessionSecretCreator,
		openshiftresources.OauthOCPBrandingSecretCreator,
		openshiftresources.OauthTLSServingCertCreator(osData),
		openshiftresources.ConsoleServingCertCreator(osData.GetRootCA),

		//TODO: This is only needed because of the ServiceAccount Token needed for Openshift
		//TODO: Streamline this by using it everywhere and use the clientprovider here or remove
		openshiftresources.ExternalX509KubeconfigCreator(osData),
		openshiftresources.GetLoopbackKubeconfigCreator(ctx, osData, r.log)}

	if osData.cluster.Spec.Cloud.GCP != nil {
		creators = append(creators, resources.ServiceAccountSecretCreator(osData))
	}

	return creators
}

func (r *Reconciler) secrets(ctx context.Context, osData *openshiftData) error {
	ns := osData.Cluster().Status.NamespaceName
	if err := reconciling.ReconcileSecrets(ctx, r.getAllSecretCreators(ctx, osData), ns, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	return nil
}

func GetStatefulSetCreators(osData *openshiftData, enableDataCorruptionChecks bool) []reconciling.NamedStatefulSetCreatorGetter {
	return []reconciling.NamedStatefulSetCreatorGetter{
		etcd.StatefulSetCreator(osData, enableDataCorruptionChecks),
	}
}

func (r *Reconciler) statefulSets(ctx context.Context, osData *openshiftData) error {
	creators := GetStatefulSetCreators(osData, r.features.EtcdDataCorruptionChecks)
	return reconciling.ReconcileStatefulSets(ctx, creators, osData.Cluster().Status.NamespaceName, r.Client)
}

func GetEtcdBackupConfigCreators(osData *openshiftData) []reconciling.NamedEtcdBackupConfigCreatorGetter {
	creators := []reconciling.NamedEtcdBackupConfigCreatorGetter{
		etcd.BackupConfigCreator(osData),
	}
	return creators
}

func (r *Reconciler) etcdBackupConfigs(ctx context.Context, c *kubermaticv1.Cluster, osData *openshiftData) error {
	creators := GetEtcdBackupConfigCreators(osData)

	return reconciling.ReconcileEtcdBackupConfigs(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c)))
}

func (r *Reconciler) getAllConfigmapCreators(ctx context.Context, osData *openshiftData) []reconciling.NamedConfigMapCreatorGetter {
	return []reconciling.NamedConfigMapCreatorGetter{
		openshiftresources.APIServerOauthMetadataConfigMapCreator(osData),
		openshiftresources.OpenshiftAPIServerConfigMapCreator(osData),
		openshiftresources.OpenshiftKubeAPIServerConfigMapCreator(osData),
		openshiftresources.KubeControllerManagerConfigMapCreatorFactory(osData),
		openshiftresources.OpenshiftControllerManagerConfigMapCreator(osData),
		openvpn.ServerClientConfigsConfigMapCreator(osData),
		openshiftresources.KubeSchedulerConfigMapCreator,
		dns.ConfigMapCreator(osData),
		openshiftresources.OauthConfigMapCreator(osData),
		openshiftresources.ConsoleConfigCreator(osData),
		// Put the cloudconfig at the end, it may need data from the cloud controller, this reduces the likelihood
		// that we instantly rotate the apiserver due to cloudconfig changes
		cloudconfig.ConfigMapCreator(osData),
	}
}

func (r *Reconciler) configMaps(ctx context.Context, osData *openshiftData) error {
	creators := r.getAllConfigmapCreators(ctx, osData)
	return reconciling.ReconcileConfigMaps(ctx, creators, osData.Cluster().Status.NamespaceName, r.Client)
}

func (r *Reconciler) getAllDeploymentCreators(ctx context.Context, osData *openshiftData) []reconciling.NamedDeploymentCreatorGetter {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		openshiftresources.OpenshiftAPIServerDeploymentCreator(ctx, osData),
		openshiftresources.APIDeploymentCreator(ctx, osData),
		openshiftresources.KubeControllerManagerDeploymentCreatorFactory(osData),
		openshiftresources.OpenshiftControllerManagerDeploymentCreator(ctx, osData),
		openshiftresources.MachineController(osData),
		openshiftresources.KubeSchedulerDeploymentCreator(osData),
		openvpn.DeploymentCreator(osData),
		dns.DeploymentCreator(osData),
		machinecontroller.WebhookDeploymentCreator(osData),
		usercluster.DeploymentCreator(osData, true),
		openshiftresources.OpenshiftNetworkOperatorCreatorFactory(osData),
		openshiftresources.OpenshiftDNSOperatorFactory(osData),
		openshiftresources.OauthDeploymentCreator(osData),
		openshiftresources.ConsoleDeployment(osData),
		openshiftresources.CloudCredentialOperator(osData),
		openshiftresources.RegistryOperatorFactory(osData),
		metricsserver.DeploymentCreator(osData),
	}

	if osData.Cluster().Annotations[kubermaticv1.AnnotationNameClusterAutoscalerEnabled] != "" {
		creators = append(creators, clusterautoscaler.DeploymentCreator(osData))
	}

	return creators
}

func (r *Reconciler) deployments(ctx context.Context, osData *openshiftData) error {
	return reconciling.ReconcileDeployments(ctx, r.getAllDeploymentCreators(ctx, osData), osData.Cluster().Status.NamespaceName, r.Client)
}

func GetCronJobCreators(osData *openshiftData) []reconciling.NamedCronJobCreatorGetter {
	return []reconciling.NamedCronJobCreatorGetter{
		etcd.CronJobCreator(osData),
	}
}

func (r *Reconciler) cronJobs(ctx context.Context, osData *openshiftData) error {
	creators := GetCronJobCreators(osData)
	if err := reconciling.ReconcileCronJobs(ctx, creators, osData.Cluster().Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure that the CronJobs exists: %v", err)
	}
	return nil
}

func GetPodDisruptionBudgetCreators(osData *openshiftData) []reconciling.NamedPodDisruptionBudgetCreatorGetter {
	return []reconciling.NamedPodDisruptionBudgetCreatorGetter{
		etcd.PodDisruptionBudgetCreator(osData),
		apiserver.PodDisruptionBudgetCreator(),
		dns.PodDisruptionBudgetCreator(),
		metricsserver.PodDisruptionBudgetCreator(),
	}
}

func (r *Reconciler) podDisruptionBudgets(ctx context.Context, osData *openshiftData) error {
	for _, podDisruptionBudgetCreator := range GetPodDisruptionBudgetCreators(osData) {
		pdbName, pdbCreator := podDisruptionBudgetCreator()
		if err := reconciling.EnsureNamedObject(ctx,
			nn(osData.Cluster().Status.NamespaceName, pdbName),
			reconciling.PodDisruptionBudgetObjectWrapper(pdbCreator),
			r.Client,
			&policyv1beta1.PodDisruptionBudget{},
			true); err != nil {
			return fmt.Errorf("failed to ensure PodDisruptionBudget %q: %v", pdbName, err)
		}
	}
	return nil
}

func (r *Reconciler) verticalPodAutoscalers(ctx context.Context, osData *openshiftData) error {
	controlPlaneDeploymentNames := []string{
		resources.DNSResolverDeploymentName,
		resources.MachineControllerDeploymentName,
		resources.MachineControllerWebhookDeploymentName,
		resources.OpenVPNServerDeploymentName,
		resources.ApiserverDeploymentName,
		resources.ControllerManagerDeploymentName,
		openshiftresources.OpenshiftAPIServerDeploymentName,
		openshiftresources.OpenshiftControllerManagerDeploymentName}

	creatorGetters, err := resources.GetVerticalPodAutoscalersForAll(ctx, r.Client, controlPlaneDeploymentNames, []string{resources.EtcdStatefulSetName}, osData.Cluster().Status.NamespaceName, r.features.VPA)
	if err != nil {
		return fmt.Errorf("failed to create the functions to handle VPA resources: %v", err)
	}

	for _, vpaCreatorGetter := range creatorGetters {
		name, creator := vpaCreatorGetter()
		err := reconciling.EnsureNamedObject(ctx,
			nn(osData.Cluster().Status.NamespaceName, name),
			reconciling.VerticalPodAutoscalerObjectWrapper(creator),
			r.Client,
			&autoscalingv1beta2.VerticalPodAutoscaler{},
			false)
		if err != nil {
			return fmt.Errorf("failed to create VerticalPodAutoscaler %q: %v", name, err)
		}
	}

	return nil
}

func (r *Reconciler) ensureNamespace(ctx context.Context, c *kubermaticv1.Cluster) error {
	if c.Status.NamespaceName == "" {
		if err := r.updateCluster(ctx, c, func(c *kubermaticv1.Cluster) {
			c.Status.NamespaceName = fmt.Sprintf("cluster-%s", c.Name)
		}); err != nil {
			return fmt.Errorf("failed to set .Status.NamespaceName: %v", err)
		}
	}

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, nn("", c.Status.NamespaceName), ns); err != nil {
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to get Namespace %q: %v", c.Status.NamespaceName, err)
		}
		ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:            c.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(c, kubermaticv1.SchemeGroupVersion.WithKind("Cluster"))}}}
		if err := r.Create(ctx, ns); err != nil {
			return fmt.Errorf("failed to create Namespace %q: %v", ns.Name, err)
		}
	}

	return nil
}

func (r *Reconciler) address(ctx context.Context, cluster *kubermaticv1.Cluster, seed *kubermaticv1.Seed) error {
	modifiers, err := address.NewModifiersBuilder(r.log.With("cluster", cluster.Name)).
		Cluster(cluster).
		Client(r.Client).
		ExternalURL(r.externalURL).
		Seed(seed).
		Build(ctx)
	if err != nil {
		return err
	}
	if len(modifiers) > 0 {
		if err := r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			for _, modifier := range modifiers {
				modifier(c)
			}
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) networkDefaults(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	var modifiers []func(*kubermaticv1.Cluster)

	if len(cluster.Spec.ClusterNetwork.Services.CIDRBlocks) == 0 {
		setServiceNetwork := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.240.16.0/20"}
		}
		modifiers = append(modifiers, setServiceNetwork)
	}

	if len(cluster.Spec.ClusterNetwork.Pods.CIDRBlocks) == 0 {
		setPodNetwork := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
		}
		modifiers = append(modifiers, setPodNetwork)
	}

	if cluster.Spec.ClusterNetwork.DNSDomain == "" {
		setDNSDomain := func(c *kubermaticv1.Cluster) {
			c.Spec.ClusterNetwork.DNSDomain = "cluster.local"
		}
		modifiers = append(modifiers, setDNSDomain)
	}

	return r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
		for _, modify := range modifiers {
			modify(c)
		}
	})
}

// GetServiceCreators returns all service creators that are currently in use
func getAllServiceCreators(osData *openshiftData) []reconciling.NamedServiceCreatorGetter {
	creators := []reconciling.NamedServiceCreatorGetter{
		apiserver.ServiceCreator(osData.Cluster().Spec.ExposeStrategy, osData.Cluster().Address.InternalName),
		openshiftresources.OpenshiftAPIServiceCreator,
		openvpn.ServiceCreator(osData.Cluster().Spec.ExposeStrategy),
		etcd.ServiceCreator(osData),
		dns.ServiceCreator(),
		machinecontroller.ServiceCreator(),
		openshiftresources.OauthServiceCreator(osData.Cluster().Spec.ExposeStrategy),
		metricsserver.ServiceCreator(),
	}

	if osData.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		creators = append(creators, nodeportproxy.FrontLoadBalancerServiceCreator())
	}

	return creators
}

func (r *Reconciler) services(ctx context.Context, osData *openshiftData) error {
	for _, namedServiceCreator := range getAllServiceCreators(osData) {
		serviceName, serviceCreator := namedServiceCreator()
		if err := reconciling.EnsureNamedObject(ctx,
			nn(osData.Cluster().Status.NamespaceName, serviceName), reconciling.ServiceObjectWrapper(serviceCreator), r.Client, &corev1.Service{}, false); err != nil {
			return fmt.Errorf("failed to ensure Service %s: %v", serviceName, err)
		}
	}
	return nil
}

func (r *Reconciler) ensureServiceAccounts(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedServiceAccountCreatorGetters := []reconciling.NamedServiceAccountCreatorGetter{
		usercluster.ServiceAccountCreator,
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, namedServiceAccountCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure ServiceAccounts: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureRoles(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedRoleCreatorGetters := []reconciling.NamedRoleCreatorGetter{
		usercluster.RoleCreator,
	}
	if err := reconciling.ReconcileRoles(ctx, namedRoleCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure Roles: %v", err)
	}

	return nil
}

func (r *Reconciler) ensureRoleBindings(ctx context.Context, c *kubermaticv1.Cluster) error {
	namedRoleBindingCreatorGetters := []reconciling.NamedRoleBindingCreatorGetter{
		usercluster.RoleBindingCreator,
	}
	if err := reconciling.ReconcileRoleBindings(ctx, namedRoleBindingCreatorGetters, c.Status.NamespaceName, r.Client); err != nil {
		return fmt.Errorf("failed to ensure RoleBindings: %v", err)
	}
	return nil
}

// A cheap helper because I am too lazy to type this every time
func nn(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
