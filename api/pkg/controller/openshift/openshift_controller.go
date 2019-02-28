package openshift

import (
	"context"
	"fmt"
	"time"

	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/openshift/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	"github.com/golang/glog"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

type Reconciler struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func Add(mgr manager.Manager, numWorkers int, workerName string) error {
	clusterPredicates := workerlabel.Predicates(workerName)

	dynamicClient := mgr.GetClient()
	reconciler := &Reconciler{Client: dynamicClient,
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder(ControllerName)}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	enqueueClusterForNamespacedObject := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		if err := dynamicClient.List(context.Background(), &client.ListOptions{}, clusterList); err != nil {
			// TODO: Is there a better way to handle errors that occur here?
			glog.Errorf("failed to list Clusters: %v", err)
		}
		for _, cluster := range clusterList.Items {
			// Predicates are used on the watched object itself, hence we can not
			// use them for anything other than the Cluster CR itself
			if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != workerName {
				continue
			}
			if cluster.Status.NamespaceName == a.Meta.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}
		return []reconcile.Request{}
	})}
	if err := c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, enqueueClusterForNamespacedObject); err != nil {
		return fmt.Errorf("failed to create watch for ConfigMaps: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, enqueueClusterForNamespacedObject); err != nil {
		return fmt.Errorf("failed to create watch for Secrets: %v", err)
	}
	if err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, enqueueClusterForNamespacedObject); err != nil {
		return fmt.Errorf("failed to create watch for Deployments: %v", err)
	}

	//TODO: Ensure only openshift clusters are handled via a predicate
	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, clusterPredicates)
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

	// Add a wrapping here so we can emit an event on error
	result, err := r.reconcile(ctx, cluster)
	if err != nil {
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.Spec.Pause {
		glog.V(6).Infof("skipping paused cluster %s", cluster.Name)
		return nil, nil
	}

	if cluster.Annotations["kubermatic.io/openshift"] == "" {
		return nil, nil
	}

	glog.V(4).Infof("Reconciling cluster %s", cluster.Name)

	// Wait for namespace
	if cluster.Status.NamespaceName == "" {
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.Status.NamespaceName}, ns); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, err
		}
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	osData := &openshiftData{cluster: cluster, client: r.Client}

	if err := r.secrets(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	if err := r.configMaps(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}
	if err := r.deployments(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	reconcileRequest, err := r.createClusterAccessToken(ctx, osData)
	if reconcileRequest != nil || err != nil {
		return reconcileRequest, err
	}

	if err := r.syncHeath(ctx, osData); err != nil {
		return nil, fmt.Errorf("failed to sync health: %v", err)
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
		return r.updateCluster(ctx, osData.Cluster().Name, func(c *kubermaticv1.Cluster) {
			c.Status.Health = *currentHealth
		})
	}

	return nil
}

func (r *Reconciler) updateCluster(ctx context.Context, name string, modify func(*kubermaticv1.Cluster)) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() (err error) {
		//Get latest version
		cluster := &kubermaticv1.Cluster{}
		if err := r.Get(ctx, nn("", name), cluster); err != nil {
			return err
		}
		// Apply modifications
		modify(cluster)
		// Update the cluster
		return r.Update(ctx, cluster)
	})
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
	kubeConfigSecret := &corev1.Secret{}
	if err := r.Get(ctx, nn(osData.Cluster().Status.NamespaceName, openshiftresources.ExternalX509KubeconfigName), kubeConfigSecret); err != nil {
		return nil, fmt.Errorf("failed to get userCluster kubeconfig secret: %v", err)
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigSecret.Data[resources.KubeconfigSecretKey])
	if err != nil {
		return nil, fmt.Errorf("failed to get config from secret: %v", err)
	}
	userClusterClient, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to get userClusterClient: %v", err)
	}

	// Ensure ServiceAccount in user cluster
	tokenOwnerServiceAccountName, tokenOwnerServiceAccountCreator := openshiftresources.TokenOwnerServiceAccount(ctx)
	err = resources.EnsureNamedObjectV2(ctx,
		nn(metav1.NamespaceSystem, tokenOwnerServiceAccountName),
		resources.ServiceAccountObjectWrapper(tokenOwnerServiceAccountCreator),
		userClusterClient,
		&corev1.ServiceAccount{})
	if err != nil {
		return nil, fmt.Errorf("failed to create TokenOwnerServiceAccount in user cluster: %v", err)
	}

	// Ensure ClusterRoleBinding in user cluster
	tokenOwnerServiceAccountClusterRoleBindingName, tokenOwnerServiceAccountClusterRoleBindingCreator := openshiftresources.TokenOwnerServiceAccountClusterRoleBinding(ctx)
	err = resources.EnsureNamedObjectV2(ctx,
		nn("", tokenOwnerServiceAccountClusterRoleBindingName),
		resources.ClusterRoleBindingObjectWrapper(tokenOwnerServiceAccountClusterRoleBindingCreator),
		userClusterClient, &rbacv1.ClusterRoleBinding{})
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
	err = resources.EnsureNamedObjectV2(ctx,
		nn(osData.Cluster().Status.NamespaceName, adminKubeconfigSecretName),
		resources.SecretObjectWrapper(adminKubeconfigCreator),
		r.Client,
		&corev1.Secret{})
	if err != nil {
		return nil, fmt.Errorf("failed to ensure token secret: %v", err)
	}
	return nil, nil
}

func (r *Reconciler) getAllSecretCreators(ctx context.Context, osData *openshiftData) []resources.NamedSecretCreatorGetter {
	return []resources.NamedSecretCreatorGetter{openshiftresources.ServiceSignerCA(),
		//TODO: This is only needed because of the ServiceAccount Token needed for Openshift
		//TODO: Streamline this by using it everywhere and use the clientprovider here or remove
		openshiftresources.ExternalX509KubeconfigCreator(osData),
		openshiftresources.GetLoopbackKubeconfigCreator(ctx, osData)}
}

func (r *Reconciler) secrets(ctx context.Context, osData *openshiftData) error {
	for _, namedSecretCreator := range r.getAllSecretCreators(ctx, osData) {
		secretName, secretCreator := namedSecretCreator()
		if err := resources.EnsureNamedObjectV2(ctx,
			nn(osData.Cluster().Status.NamespaceName, secretName), resources.SecretObjectWrapper(secretCreator), r.Client, &corev1.Secret{}); err != nil {
			return fmt.Errorf("failed to ensure Secret %s: %v", secretName, err)
		}
	}

	return nil
}

func (r *Reconciler) getAllConfigmapCreators() []openshiftresources.NamedConfigMapCreator {
	return []openshiftresources.NamedConfigMapCreator{openshiftresources.OpenshiftAPIServerConfigMapCreator,
		openshiftresources.OpenshiftControllerMangerConfigMapCreator}
}

func (r *Reconciler) configMaps(ctx context.Context, osData *openshiftData) error {
	for _, namedConfigmapCreator := range r.getAllConfigmapCreators() {
		configMapName, configMapCreator := namedConfigmapCreator(ctx, osData)
		if err := resources.EnsureNamedObjectV2(ctx,
			nn(osData.Cluster().Status.NamespaceName, configMapName), resources.ConfigMapObjectWrapper(configMapCreator), r.Client, &corev1.ConfigMap{}); err != nil {
			return fmt.Errorf("failed to ensure ConfigMap %s: %v", configMapName, err)
		}
	}
	return nil
}

func (r *Reconciler) getAllDeploymentCreators() []openshiftresources.NamedDeploymentCreator {
	return []openshiftresources.NamedDeploymentCreator{openshiftresources.APIDeploymentCreator, openshiftresources.DeploymentCreator}
}

func (r *Reconciler) deployments(ctx context.Context, osData *openshiftData) error {
	for _, namedDeploymentCreator := range r.getAllDeploymentCreators() {
		deploymentName, deploymentCreator := namedDeploymentCreator(ctx, osData)
		if err := resources.EnsureNamedObjectV2(ctx,
			nn(osData.Cluster().Status.NamespaceName, deploymentName), resources.DeploymentObjectWrapper(deploymentCreator), r.Client, &appsv1.Deployment{}); err != nil {
			return fmt.Errorf("failed to ensure Deployment %s: %v", deploymentName, err)
		}
	}
	return nil
}

// A cheap helper because I am too lazy to type this everytime
func nn(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
