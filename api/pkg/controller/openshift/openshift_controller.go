package openshift

import (
	"context"
	"fmt"
	"time"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/clusterdeletion"
	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/openshift/resources"
	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/address"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/cloudconfig"
	"github.com/kubermatic/kubermatic/api/pkg/resources/dns"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/openvpn"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/kubermatic/kubermatic/api/pkg/resources/usercluster"

	"github.com/golang/glog"

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
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
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
	scheme               *runtime.Scheme
	recorder             record.EventRecorder
	dcs                  map[string]provider.DatacenterMeta
	dc                   string
	overwriteRegistry    string
	nodeAccessNetwork    string
	etcdDiskSize         resource.Quantity
	dockerPullConfigJSON []byte
	workerName           string
	externalURL          string
	oidc                 OIDCConfig
	features             Features
}

func Add(
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	dc string,
	dcs map[string]provider.DatacenterMeta,
	overwriteRegistry,
	nodeAccessNetwork string,
	etcdDiskSize resource.Quantity,
	dockerPullConfigJSON []byte,
	externalURL string,
	oidcConfig OIDCConfig,
	features Features,
) error {
	reconciler := &Reconciler{
		Client:               mgr.GetClient(),
		scheme:               mgr.GetScheme(),
		recorder:             mgr.GetRecorder(ControllerName),
		dc:                   dc,
		dcs:                  dcs,
		overwriteRegistry:    overwriteRegistry,
		nodeAccessNetwork:    nodeAccessNetwork,
		etcdDiskSize:         etcdDiskSize,
		dockerPullConfigJSON: dockerPullConfigJSON,
		workerName:           workerName,
		externalURL:          externalURL,
		oidc:                 oidcConfig,
		features:             features,
	}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	typesToWatch := []runtime.Object{
		&corev1.Service{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Namespace{},
		&appsv1.StatefulSet{},
		&appsv1.Deployment{},
		&batchv1beta1.CronJob{},
		&policyv1beta1.PodDisruptionBudget{},
		&autoscalingv1beta2.VerticalPodAutoscaler{},
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

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.Spec.Pause {
		glog.V(4).Infof("skipping paused cluster %s", cluster.Name)
		return reconcile.Result{}, nil
	}

	if cluster.Annotations["kubermatic.io/openshift"] == "" {
		return reconcile.Result{}, nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := r.reconcile(ctx, cluster)
	if err != nil {
		glog.Errorf("failed reconciling cluster %s: %v", cluster.Name, err)
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	glog.V(4).Infof("Reconciling cluster %s", cluster.Name)

	if cluster.DeletionTimestamp != nil {
		userClusterClient, err := r.getUserClusterClient(ctx, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to get user cluster client: %v", err)
		}
		return clusterdeletion.New(r.Client, userClusterClient).CleanupCluster(ctx, cluster)
	}

	// Ensure Namespace
	if err := r.ensureNamespace(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to ensure Namespace: %v", err)
	}

	dc, found := r.dcs[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find dc %s", cluster.Spec.Cloud.DatacenterName)
	}
	osData := &openshiftData{
		cluster:           cluster,
		client:            r.Client,
		dc:                &dc,
		overwriteRegistry: r.overwriteRegistry,
		nodeAccessNetwork: r.nodeAccessNetwork,
		oidc:              r.oidc,
		etcdDiskSize:      r.etcdDiskSize,
	}

	if err := r.address(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to reconcile the cluster address: %v", err)
	}

	if err := r.networkDefaults(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to setup cluster networking defaults: %v", err)
	}

	if err := r.services(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Services: %v", err)
	}

	if err := r.secrets(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets: %v", err)
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
	if !cluster.Status.Health.CloudProviderInfrastructure {
		return &reconcile.Result{RequeueAfter: 1 * time.Second}, nil
	}
	if err := r.configMaps(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}

	if err := r.deployments(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Deployments: %v", err)
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

	if !osData.Cluster().Status.Health.Apiserver {
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	reconcileRequest, err := r.createClusterAccessToken(ctx, osData)
	if reconcileRequest != nil || err != nil {
		return reconcileRequest, err
	}

	// Only add the node deletion finalizer when the cluster is actually running
	// Otherwise we fail to delete the nodes and are stuck in a loop
	if !kuberneteshelper.HasFinalizer(cluster, kubermaticapiv1.NodeDeletionFinalizer) {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Finalizers = append(c.Finalizers, kubermaticapiv1.NodeDeletionFinalizer)
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (r *Reconciler) syncHeath(ctx context.Context, osData *openshiftData) error {
	currentHealth := osData.Cluster().Status.Health.DeepCopy()
	type depInfo struct {
		healthy  *bool
		minReady int32
	}

	healthMapping := map[string]*depInfo{
		openshiftresources.ApiserverDeploymentName:         {healthy: &currentHealth.Apiserver, minReady: 1},
		openshiftresources.ControllerManagerDeploymentName: {healthy: &currentHealth.Controller, minReady: 1},
		resources.MachineControllerDeploymentName:          {healthy: &currentHealth.MachineController, minReady: 1},
		resources.OpenVPNServerDeploymentName:              {healthy: &currentHealth.OpenVPN, minReady: 1},
	}

	var err error
	for name := range healthMapping {
		*healthMapping[name].healthy, err = resources.HealthyDeployment(ctx, r.Client, nn(osData.Cluster().Status.NamespaceName, name), healthMapping[name].minReady)
		if err != nil {
			return fmt.Errorf("failed to get dep health %q: %v", name, err)
		}
	}

	currentHealth.Etcd, err = resources.HealthyStatefulSet(ctx, r.Client, nn(osData.Cluster().Status.NamespaceName, resources.EtcdStatefulSetName), 2)
	if err != nil {
		return fmt.Errorf("failed to get etcd health: %v", err)
	}

	//TODO: Revisit this. This is a tiny bit ugly, but Openshift doesn't have a distinct scheduler
	// and introducing a distinct health struct for Openshift means we have to change the API as well
	currentHealth.Scheduler = currentHealth.Controller

	if osData.Cluster().Status.Health != *currentHealth {
		return r.updateCluster(ctx, osData.Cluster(), func(c *kubermaticv1.Cluster) {
			c.Status.Health = *currentHealth
		})
	}

	return nil
}

func (r *Reconciler) updateCluster(ctx context.Context, c *kubermaticv1.Cluster, modify func(*kubermaticv1.Cluster)) error {
	// Store it here because it may be unset later on if an update request failed
	name := c.Name
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		//Get latest version
		if err := r.Get(ctx, nn("", name), c); err != nil {
			return err
		}
		// Apply modifications
		modify(c)
		// Update the cluster
		return r.Update(ctx, c)
	})
}

func (r *Reconciler) getUserClusterClient(ctx context.Context, cluster *kubermaticv1.Cluster) (client.Client, error) {
	kubeConfigSecret := &corev1.Secret{}
	if err := r.Get(ctx, nn(cluster.Status.NamespaceName, openshiftresources.ExternalX509KubeconfigName), kubeConfigSecret); err != nil {
		return nil, fmt.Errorf("failed to get userCluster kubeconfig secret: %v", err)
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigSecret.Data[resources.KubeconfigSecretKey])
	if err != nil {
		return nil, fmt.Errorf("failed to get config from secret: %v", err)
	}
	return client.New(cfg, client.Options{})
}

// Openshift doesn't seem to support a token-file-based authentication at all
// It can be passed down onto the kube-apiserver but does still not work, presumably because OS puts another authentication
// layer on top
// The workaround here is to create a serviceaccount and clusterrolebinding in the user cluster, then copy the token secret
// of that Serviceaccount into the admin kubeconfig.
// In its current form this is not a long-term solution as we wont notice if someone deletes the token Secret inside the user
// cluster, rendering our admin-kubeconfig invalid
// TODO: Find an alternate approach or move this to a controller that has informers in both the user cluster and the seed
func (r *Reconciler) createClusterAccessToken(ctx context.Context, osData *openshiftData) (*reconcile.Result, error) {
	userClusterClient, err := r.getUserClusterClient(ctx, osData.Cluster())
	if err != nil {
		return nil, fmt.Errorf("failed to get userClusterClient: %v", err)
	}

	// Ensure ServiceAccount in user cluster
	tokenOwnerServiceAccountName, tokenOwnerServiceAccountCreator := openshiftresources.TokenOwnerServiceAccount(ctx)
	err = reconciling.EnsureNamedObject(ctx,
		nn(metav1.NamespaceSystem, tokenOwnerServiceAccountName),
		reconciling.ServiceAccountObjectWrapper(tokenOwnerServiceAccountCreator),
		userClusterClient,
		&corev1.ServiceAccount{},
		false)
	if err != nil {
		return nil, fmt.Errorf("failed to create TokenOwnerServiceAccount in user cluster: %v", err)
	}

	// Ensure ClusterRoleBinding in user cluster
	tokenOwnerServiceAccountClusterRoleBindingName, tokenOwnerServiceAccountClusterRoleBindingCreator := openshiftresources.TokenOwnerServiceAccountClusterRoleBinding(ctx)
	err = reconciling.EnsureNamedObject(ctx,
		nn("", tokenOwnerServiceAccountClusterRoleBindingName),
		reconciling.ClusterRoleBindingObjectWrapper(tokenOwnerServiceAccountClusterRoleBindingCreator),
		userClusterClient, &rbacv1.ClusterRoleBinding{},
		false)
	if err != nil {
		return nil, fmt.Errorf("failed to create TokenOwnerServiceAccountClusterRoleBinding in user cluster: %v", err)
	}

	// Get the ServiceAccount to find out the name of its secret
	tokenOwnerServiceAccount := &corev1.ServiceAccount{}
	if err := userClusterClient.Get(ctx, nn(metav1.NamespaceSystem, tokenOwnerServiceAccountName), tokenOwnerServiceAccount); err != nil {
		return nil, fmt.Errorf("failed to get TokenOwnerServiceAccount after creating it: %v", err)
	}

	// Check if the secret already exists, if not try again later
	if len(tokenOwnerServiceAccount.Secrets) < 1 {
		return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Get the secret
	tokenSecret := &corev1.Secret{}
	if err := userClusterClient.Get(ctx, nn(metav1.NamespaceSystem, tokenOwnerServiceAccount.Secrets[0].Name), tokenSecret); err != nil {
		return nil, fmt.Errorf("failed to get token secret from user cluster: %v", err)
	}

	// Create the admin-kubeconfig in the seed cluster
	adminKubeconfigSecretName, adminKubeconfigCreator := resources.AdminKubeconfigCreator(osData, func(c *clientcmdapi.Config) {
		c.AuthInfos[resources.KubeconfigDefaultContextKey].Token = string(tokenSecret.Data["token"])
	})()
	err = reconciling.EnsureNamedObject(ctx,
		nn(osData.Cluster().Status.NamespaceName, adminKubeconfigSecretName),
		reconciling.SecretObjectWrapper(adminKubeconfigCreator),
		r.Client,
		&corev1.Secret{},
		false)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token secret: %v", err)
	}
	return nil, nil
}

func (r *Reconciler) getAllSecretCreators(ctx context.Context, osData *openshiftData) []reconciling.NamedSecretCreatorGetter {
	creators := []reconciling.NamedSecretCreatorGetter{
		certificates.RootCACreator(osData),
		openvpn.CACreator(),
		apiserver.DexCACertificateCreator(osData.GetDexCA),
		certificates.FrontProxyCACreator(),
		openshiftresources.ServiceSignerCA(),
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

		// Kubeconfigs
		resources.GetInternalKubeconfigCreator(resources.SchedulerKubeconfigSecretName, resources.SchedulerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.KubeletDnatControllerKubeconfigSecretName, resources.KubeletDnatControllerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.MachineControllerKubeconfigSecretName, resources.MachineControllerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.ControllerManagerKubeconfigSecretName, resources.ControllerManagerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.KubeStateMetricsKubeconfigSecretName, resources.KubeStateMetricsCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.MetricsServerKubeconfigSecretName, resources.MetricsServerCertUsername, nil, osData),
		resources.GetInternalKubeconfigCreator(resources.InternalUserClusterAdminKubeconfigSecretName, resources.InternalUserClusterAdminKubeconfigCertUsername, []string{"system:masters"}, osData),

		//TODO: This is only needed because of the ServiceAccount Token needed for Openshift
		//TODO: Streamline this by using it everywhere and use the clientprovider here or remove
		openshiftresources.ExternalX509KubeconfigCreator(osData),
		openshiftresources.GetLoopbackKubeconfigCreator(ctx, osData)}

	return creators
}

func (r *Reconciler) secrets(ctx context.Context, osData *openshiftData) error {
	for _, namedSecretCreator := range r.getAllSecretCreators(ctx, osData) {
		secretName, secretCreator := namedSecretCreator()
		if err := reconciling.EnsureNamedObject(ctx,
			nn(osData.Cluster().Status.NamespaceName, secretName), reconciling.SecretObjectWrapper(secretCreator), r.Client, &corev1.Secret{}, false); err != nil {
			return fmt.Errorf("failed to ensure Secret %s: %v", secretName, err)
		}
	}

	return nil
}

func GetStatefulSetCreators(osData *openshiftData, enableDataCorruptionChecks bool) []reconciling.NamedStatefulSetCreatorGetter {
	return []reconciling.NamedStatefulSetCreatorGetter{
		etcd.StatefulSetCreator(osData, enableDataCorruptionChecks),
	}
}

func (r *Reconciler) statefulSets(ctx context.Context, osData *openshiftData) error {
	for _, namedStatefulSetCreator := range GetStatefulSetCreators(osData, r.features.EtcdDataCorruptionChecks) {
		statefulSetName, statefulSetCreator := namedStatefulSetCreator()
		if err := reconciling.EnsureNamedObject(ctx,
			nn(osData.Cluster().Status.NamespaceName, statefulSetName),
			reconciling.StatefulSetObjectWrapper(statefulSetCreator),
			r.Client,
			&appsv1.StatefulSet{},
			false); err != nil {
			return fmt.Errorf("failed to ensure StatefulSet %q: %v", statefulSetName, err)
		}
	}

	return nil
}

func (r *Reconciler) getAllConfigmapCreators(ctx context.Context, osData *openshiftData) []reconciling.NamedConfigMapCreatorGetter {
	return []reconciling.NamedConfigMapCreatorGetter{
		cloudconfig.ConfigMapCreator(osData),
		openshiftresources.OpenshiftAPIServerConfigMapCreator(ctx, osData),
		openshiftresources.OpenshiftControllerMangerConfigMapCreator(ctx, osData),
		openvpn.ServerClientConfigsConfigMapCreator(osData),
		dns.ConfigMapCreator(osData),
	}
}

func (r *Reconciler) configMaps(ctx context.Context, osData *openshiftData) error {
	for _, namedConfigmapCreator := range r.getAllConfigmapCreators(ctx, osData) {
		configMapName, configMapCreator := namedConfigmapCreator()
		if err := reconciling.EnsureNamedObject(ctx,
			nn(osData.Cluster().Status.NamespaceName, configMapName), reconciling.ConfigMapObjectWrapper(configMapCreator), r.Client, &corev1.ConfigMap{}, false); err != nil {
			return fmt.Errorf("failed to ensure ConfigMap %s: %v", configMapName, err)
		}
	}
	return nil
}

func (r *Reconciler) getAllDeploymentCreators(ctx context.Context, osData *openshiftData) []reconciling.NamedDeploymentCreatorGetter {
	return []reconciling.NamedDeploymentCreatorGetter{openshiftresources.APIDeploymentCreator(ctx, osData),
		openshiftresources.ControllerManagerDeploymentCreator(ctx, osData),
		openshiftresources.MachineController(osData),
		openvpn.DeploymentCreator(osData),
		dns.DeploymentCreator(osData),
		machinecontroller.WebhookDeploymentCreator(osData),
		usercluster.DeploymentCreator(osData, true)}
}

func (r *Reconciler) deployments(ctx context.Context, osData *openshiftData) error {
	for _, namedDeploymentCreator := range r.getAllDeploymentCreators(ctx, osData) {
		deploymentName, deploymentCreator := namedDeploymentCreator()
		if err := reconciling.EnsureNamedObject(ctx,
			nn(osData.Cluster().Status.NamespaceName, deploymentName), reconciling.DeploymentObjectWrapper(deploymentCreator), r.Client, &appsv1.Deployment{}, false); err != nil {
			return fmt.Errorf("failed to ensure Deployment %s: %v", deploymentName, err)
		}
	}
	return nil
}

func GetCronJobCreators(osData *openshiftData) []reconciling.NamedCronJobCreatorGetter {
	return []reconciling.NamedCronJobCreatorGetter{
		etcd.CronJobCreator(osData),
	}
}

func (r *Reconciler) cronJobs(ctx context.Context, osData *openshiftData) error {
	for _, cronJobCreator := range GetCronJobCreators(osData) {
		cronJobName, cronJobCreator := cronJobCreator()
		if err := reconciling.EnsureNamedObject(ctx,
			nn(osData.Cluster().Status.NamespaceName, cronJobName), reconciling.CronJobObjectWrapper(cronJobCreator), r.Client, &batchv1beta1.CronJob{}, false); err != nil {
			return fmt.Errorf("failed to ensure CronJob %q: %v", cronJobName, err)
		}
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
		openshiftresources.ApiserverDeploymentName,
		openshiftresources.ControllerManagerDeploymentName}

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

func (r *Reconciler) address(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	modifiers, err := address.SyncClusterAddress(ctx, cluster, r.Client, r.externalURL, r.dc, r.dcs)
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
			c.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
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
	return []reconciling.NamedServiceCreatorGetter{
		apiserver.InternalServiceCreator(),
		apiserver.ExternalServiceCreator(),
		openvpn.ServiceCreator(),
		etcd.ServiceCreator(osData),
		dns.ServiceCreator(),
		machinecontroller.ServiceCreator(),
	}
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

// A cheap helper because I am too lazy to type this everytime
func nn(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
