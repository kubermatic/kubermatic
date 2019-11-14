package openshift

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	clusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/clusterdeletion"
	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/openshift/resources"
	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1helper "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/address"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/clusterautoscaler"
	"github.com/kubermatic/kubermatic/api/pkg/resources/dns"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/nodeportproxy"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/usercluster"

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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kubermatic_openshift_controller"

// Check if the Reconciler fullfills the interface
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
	client.Client
	log                      *zap.SugaredLogger
	scheme                   *runtime.Scheme
	recorder                 record.EventRecorder
	seedGetter               provider.SeedGetter
	userClusterConnProvider  *clusterclient.Provider
	overwriteRegistry        string
	nodeAccessNetwork        string
	etcdDiskSize             resource.Quantity
	dockerPullConfigJSON     []byte
	workerName               string
	externalURL              string
	oidc                     OIDCConfig
	kubermaticImage          string
	dnatControllerImage      string
	features                 Features
	concurrentClusterUpdates int
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
	oidcConfig OIDCConfig,
	kubermaticImage string,
	dnatControllerImage string,
	features Features,
	concurrentClusterUpdates int,
) error {
	reconciler := &Reconciler{
		Client:                   mgr.GetClient(),
		log:                      log.Named(ControllerName),
		scheme:                   mgr.GetScheme(),
		recorder:                 mgr.GetEventRecorderFor(ControllerName),
		seedGetter:               seedGetter,
		userClusterConnProvider:  userClusterConnProvider,
		overwriteRegistry:        overwriteRegistry,
		nodeAccessNetwork:        nodeAccessNetwork,
		etcdDiskSize:             etcdDiskSize,
		dockerPullConfigJSON:     dockerPullConfigJSON,
		workerName:               workerName,
		externalURL:              externalURL,
		oidc:                     oidcConfig,
		kubermaticImage:          kubermaticImage,
		dnatControllerImage:      dnatControllerImage,
		features:                 features,
		concurrentClusterUpdates: concurrentClusterUpdates,
	}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	typesToWatch := []runtime.Object{
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

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	if cluster.Annotations["kubermatic.io/openshift"] == "" {
		log.Debug("Skipping because the cluster is an Kubernetes cluster")
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
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
		userClusterClientGetter := func() (client.Client, error) {
			userClusterClient, err := r.userClusterConnProvider.GetClient(cluster)
			if err != nil {
				return nil, fmt.Errorf("failed to get user cluster client: %v", err)
			}
			log.Debugw("Getting client for cluster", "cluster", cluster.Name)
			return userClusterClient, nil
		}

		// Always requeue a cluster after we executed the cleanup.
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, clusterdeletion.New(r.Client, userClusterClientGetter).CleanupCluster(ctx, log, cluster)
	}

	if cluster.Spec.Openshift == nil {
		return nil, errors.New("openshift cluster but .Spec.Openshift is unset")
	}

	// Ensure Namespace
	if err := r.ensureNamespace(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to ensure Namespace: %v", err)
	}

	seed, err := r.seedGetter()
	if err != nil {
		return nil, err
	}
	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find dc %s", cluster.Spec.Cloud.DatacenterName)
	}
	supportsFailureDomainZoneAntiAffinity, err := controllerutil.SupportsFailureDomainZoneAntiAffinity(ctx, r.Client)
	if err != nil {
		return nil, err
	}

	osData := &openshiftData{
		cluster:                               cluster,
		client:                                r.Client,
		dc:                                    &datacenter,
		overwriteRegistry:                     r.overwriteRegistry,
		nodeAccessNetwork:                     r.nodeAccessNetwork,
		oidc:                                  r.oidc,
		etcdDiskSize:                          r.etcdDiskSize,
		kubermaticImage:                       r.kubermaticImage,
		dnatControllerImage:                   r.dnatControllerImage,
		supportsFailureDomainZoneAntiAffinity: supportsFailureDomainZoneAntiAffinity,
		userClusterClient: func() (client.Client, error) {
			return r.userClusterConnProvider.GetClient(cluster)
		},
		externalURL: r.externalURL,
		seed:        seed.DeepCopy(),
	}

	if err := r.networkDefaults(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to setup cluster networking defaults: %v", err)
	}

	if err := r.services(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Services: %v", err)
	}

	if err := r.address(ctx, cluster, seed); err != nil {
		return nil, fmt.Errorf("failed to reconcile the cluster address: %v", err)
	}

	if err := r.secrets(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	if err := r.ensureServiceAccounts(ctx, cluster); err != nil {
		return nil, err
	}

	if err := r.ensureRoles(ctx, cluster); err != nil {
		return nil, err
	}

	if err := r.ensureRoleBindings(ctx, cluster); err != nil {
		return nil, err
	}

	if err := r.statefulSets(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile StatefulSets: %v", err)
	}

	// Wait until the cloud provider infra is ready before attempting
	// to render the cloud-config
	// TODO: Model resource deployment as a DAG so we don't need hacks
	// like this combined with tribal knowledge and "someone is noticing this
	// isn't working correctly"
	// https://github.com/kubermatic/kubermatic/issues/2948
	if kubermaticv1.HealthStatusUp != cluster.Status.ExtendedHealth.CloudProviderInfrastructure {
		return &reconcile.Result{RequeueAfter: 1 * time.Second}, nil
	}
	if err := r.configMaps(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}

	if err := r.deployments(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	if osData.Cluster().Spec.ExposeStrategy == corev1.ServiceTypeLoadBalancer {
		if err := nodeportproxy.EnsureResources(ctx, r.Client, osData); err != nil {
			return nil, fmt.Errorf("failed to ensure NodePortProxy resources: %v", err)
		}
	}

	if err := r.cronJobs(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile CronJobs: %v", err)
	}

	if err := r.podDisruptionBudgets(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile PodDisruptionBudgets: %v", err)
	}

	if err := r.verticalPodAutoscalers(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile VerticalPodAutoscalers: %v", err)
	}

	if err := r.syncHeath(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to sync health: %v", err)
	}

	if kubermaticv1.HealthStatusDown == osData.Cluster().Status.ExtendedHealth.Apiserver {
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	reachable, err := r.clusterIsReachable(ctx, osData.Cluster())
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

	// We must put the hash for this into the usercluster and the raw value to expose in the
	// seed. The usercluster secret must be created max 1h after creation timestamp of the
	// kube-system ns: https://github.com/openshift/origin/blob/e774f85c15aef11d76db1ffc458484867e503293/pkg/oauthserver/authenticator/password/bootstrap/bootstrap.go#L131
	// TODO: Move this into the usercluster controller
	if err := r.ensureConsoleBootstrapPassword(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to create bootstrap password for openshift console: %v", err)
	}

	// This requires both the cluster to be up and a CRD we deploy via the AddonController
	// to exist, so do this at the very end
	// TODO: Move this into the usercluster controller
	if err := r.ensureConsoleOAuthSecret(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to create oauth secret for Openshift console: %v", err)
	}

	return nil, nil
}

func (r *Reconciler) ensureConsoleBootstrapPassword(ctx context.Context, osData *openshiftData) error {
	getter := []reconciling.NamedSecretCreatorGetter{
		openshiftresources.BootStrapPasswordSecretGenerator(osData)}
	ns := osData.Cluster().Status.NamespaceName
	return reconciling.ReconcileSecrets(ctx, getter, ns, r.Client)
}

func (r *Reconciler) ensureConsoleOAuthSecret(ctx context.Context, osData *openshiftData) error {
	getter := []reconciling.NamedSecretCreatorGetter{
		openshiftresources.ConsoleOAuthClientSecretCreator(osData)}
	ns := osData.Cluster().Status.NamespaceName
	return reconciling.ReconcileSecrets(ctx, getter, ns, r.Client)
}

// clusterIsReachable checks if the cluster is reachable via its external name
func (r *Reconciler) clusterIsReachable(ctx context.Context, c *kubermaticv1.Cluster) (bool, error) {
	client, err := r.userClusterConnProvider.GetClient(c)
	if err != nil {
		return false, err
	}

	if err := client.List(ctx, &corev1.NamespaceList{}); err != nil {
		return false, nil
	}

	return true, nil
}

func (r *Reconciler) syncHeath(ctx context.Context, osData *openshiftData) error {
	currentHealth := osData.Cluster().Status.ExtendedHealth.DeepCopy()
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
		status, err := resources.HealthyDeployment(ctx, r.Client, nn(osData.Cluster().Status.NamespaceName, name), healthMapping[name].minReady)
		if err != nil {
			return fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
		*healthMapping[name].healthStatus = status
	}

	status, err := resources.HealthyStatefulSet(ctx, r.Client, nn(osData.Cluster().Status.NamespaceName, resources.EtcdStatefulSetName), 2)
	if err != nil {
		return fmt.Errorf("failed to get etcd health: %v", err)
	}
	currentHealth.Etcd = status

	//TODO: Revisit this. This is a tiny bit ugly, but Openshift doesn't have a distinct scheduler
	// and introducing a distinct health struct for Openshift means we have to change the API as well
	currentHealth.Scheduler = currentHealth.Controller

	if osData.Cluster().Status.ExtendedHealth != *currentHealth {
		return r.updateCluster(ctx, osData.Cluster(), func(c *kubermaticv1.Cluster) {
			c.Status.ExtendedHealth = *currentHealth
		})
	}

	return nil
}

func (r *Reconciler) updateCluster(ctx context.Context, c *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster)) error {
	oldCluster := c.DeepCopy()
	modify(c)
	return r.Patch(ctx, c, client.MergeFrom(oldCluster))
}

func (r *Reconciler) getAllSecretCreators(ctx context.Context, osData *openshiftData) []reconciling.NamedSecretCreatorGetter {
	creators := []reconciling.NamedSecretCreatorGetter{
		certificates.RootCACreator(osData),
		openvpn.CACreator(),
		apiserver.DexCACertificateCreator(osData.GetDexCA),
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

func (r *Reconciler) getAllConfigmapCreators(ctx context.Context, osData *openshiftData) []reconciling.NamedConfigMapCreatorGetter {
	return []reconciling.NamedConfigMapCreatorGetter{
		openshiftresources.APIServerOauthMetadataConfigMapCreator(osData),
		openshiftresources.OpenshiftAPIServerConfigMapCreator(osData),
		openshiftresources.OpenshiftKubeAPIServerConfigMapCreator(osData),
		openshiftresources.KubeControllerManagerConfigMapCreatorFactory(osData),
		openshiftresources.OpenshiftControllerManagerConfigMapCreator(osData.Cluster().Spec.Version.String()),
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
		openshiftresources.RegistryOperatorFactory(osData)}

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
	modifiers, err := address.SyncClusterAddress(ctx, cluster, r.Client, r.externalURL, seed)
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
		apiserver.InternalServiceCreator(),
		apiserver.ExternalServiceCreator(osData.Cluster().Spec.ExposeStrategy),
		openshiftresources.OpenshiftAPIServiceCreator,
		openvpn.ServiceCreator(osData.Cluster().Spec.ExposeStrategy),
		etcd.ServiceCreator(osData),
		dns.ServiceCreator(),
		machinecontroller.ServiceCreator(),
		openshiftresources.OauthServiceCreator(osData.Cluster().Spec.ExposeStrategy),
	}

	if osData.Cluster().Spec.ExposeStrategy == corev1.ServiceTypeLoadBalancer {
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

// A cheap helper because I am too lazy to type this everytime
func nn(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
