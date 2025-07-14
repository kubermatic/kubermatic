//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kubelbcontroller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubelbv1alpha1 "k8c.io/kubelb/api/kubelb.k8c.io/v1alpha1"
	"k8c.io/kubermatic/sdk/v2/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubelbresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources"
	kubelbseedresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/seed-cluster"
	kubelbuserclusterresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/user-cluster"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName                = "kkp-kubelb-controller"
	CleanupFinalizer              = "kubermatic.k8c.io/cleanup-kubelb-ccm"
	kubeLBCCMKubeconfigSecretName = "kubelb-ccm-kubeconfig"
	kubeconfigSecretKey           = "kubelb"
	healthCheckPeriod             = 5 * time.Second
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type reconciler struct {
	ctrlruntimeclient.Client

	workerName                    string
	recorder                      record.EventRecorder
	seedGetter                    provider.SeedGetter
	projectsGetter                provider.ProjectsGetter
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	overwriteRegistry             string
	versions                      kubermatic.Versions
}

func Add(mgr manager.Manager, numWorkers int, workerName string, overwriteRegistry string, seedGetter provider.SeedGetter, projectsGetter provider.ProjectsGetter, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &reconciler{
		Client:                        mgr.GetClient(),
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		seedGetter:                    seedGetter,
		projectsGetter:                projectsGetter,
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
		overwriteRegistry:             overwriteRegistry,
	}

	clusterIsAlive := predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		// Only watch clusters that are in a state where they can be reconciled.
		return !cluster.Spec.Pause && cluster.Status.NamespaceName != ""
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		Watches(&kubermaticv1.Project{}, enqueueClustersForProject(reconciler.Client, reconciler.log)).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(workerlabel.Predicate(workerName), clusterIsAlive)).
		Build(reconciler)

	return err
}

func enqueueClustersForProject(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		project := a.(*kubermaticv1.Project)
		labelReq, err := labels.NewRequirement(kubermaticv1.ProjectIDLabelKey, selection.Equals, []string{project.ObjectMeta.Name})
		if err != nil {

		}
		clusterList := &kubermaticv1.ClusterList{}
		if err := client.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{
			LabelSelector: labels.NewSelector().Add(*labelReq),
		}); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list clusters for project: %w", err))
		}

		requests := make([]reconcile.Request, len(clusterList.Items))

		for i, cluster := range clusterList.Items {
			requests[i] = reconcile.Request{NamespacedName: types.NamespacedName{Name: cluster.Name}}
		}

		return requests
	})
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Resource is marked for deletion.
	if cluster.DeletionTimestamp != nil {
		if kuberneteshelper.HasFinalizer(cluster, CleanupFinalizer) {
			log.Debug("Cleaning up kubeLB resources")
			return reconcile.Result{}, r.handleKubeLBCleanup(ctx, cluster)
		}
		// Finalizer doesn't exist so clean up is already done.
		return reconcile.Result{}, nil
	}

	// Kubelb was disabled after it was enabled. Clean up resources.
	if kuberneteshelper.HasFinalizer(cluster, CleanupFinalizer) && !cluster.Spec.IsKubeLBEnabled() {
		log.Debug("Cleaning up kubeLB resources")
		return reconcile.Result{}, r.handleKubeLBCleanup(ctx, cluster)
	}

	// Kubelb is disabled. Nothing to do.
	if !cluster.Spec.IsKubeLBEnabled() {
		return reconcile.Result{}, nil
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because it has no namespace yet")
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		log.Debugf("API server is not running, trying again in %v", healthCheckPeriod)
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionKubeLBControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return *result, err
}

func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Ensure that kubeLB management cluster kubeconfig exists. If it doesn't, we can't continue.
	seed, err := r.seedGetter()
	if err != nil {
		return nil, err
	}

	// Ensure that cluster related project can be fetched to use default tenant configuration if specified
	projects, err := r.projectsGetter()
	if err != nil {
		return nil, err
	}

	// We need to get the project name from cluster labels
	clusterLabels := cluster.GetLabels()
	if clusterLabels == nil {
		return nil, fmt.Errorf("couldn't find project due to empty labels for cluster %q", cluster.Name)
	}

	projectName := clusterLabels[kubermaticv1.ProjectIDLabelKey]
	project, projectFound := projects[projectName]
	if !projectFound {
		return nil, fmt.Errorf("couldn't find project %q for cluster %q", projectName, cluster.Name)
	}

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find datacenter %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}

	// Get kubeLB management cluster client.
	kubeLBManagementClient, kubeLBManagementKubeConfig, err := r.getKubeLBManagementClusterClient(ctx, seed, datacenter)
	if err != nil {
		return nil, err
	}

	// At this point we have a client for the kubelb management cluster. We can now process the cluster.
	// Add finalizer to the cluster. We only add finalizer to the cluster if/when the kubeconfig for the
	// kubelb management cluster is configured.
	if !kuberneteshelper.HasFinalizer(cluster, CleanupFinalizer) {
		if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, CleanupFinalizer); err != nil {
			return nil, fmt.Errorf("failed to set %q finalizer: %w", CleanupFinalizer, err)
		}
	}

	// Create/update required resources in kubeLB management cluster.
	if err := r.createOrUpdateKubeLBManagementClusterResources(ctx, kubeLBManagementClient, cluster, project.Spec.DefaultTenantSpec); err != nil {
		return nil, err
	}

	// Create/update required resources in user cluster.
	if err := r.createOrUpdateKubeLBUserClusterResources(ctx, cluster, datacenter); err != nil {
		return nil, err
	}

	// Create/update required resources in user cluster namespace in seed.
	return r.createOrUpdateKubeLBSeedClusterResources(ctx, cluster, kubeLBManagementClient, kubeLBManagementKubeConfig, datacenter)
}

func (r *reconciler) createOrUpdateKubeLBManagementClusterResources(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, defaultTenantSpec *kubelbv1alpha1.TenantSpec) error {
	// Check if tenant exists or not, if it doesn't we create it.
	tenant := &kubelbv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
			Labels: map[string]string{
				// These labels are used to identify the tenant in the kubelb management cluster.
				"kubermatic.k8c.io/cluster-name":          cluster.Name,
				"kubermatic.k8c.io/cluster-external-name": cluster.Status.Address.ExternalName,
				"kubermatic.k8c.io/cluster-project-id":    cluster.Labels[kubermaticv1.ProjectIDLabelKey],
			},
		},
	}
	// Default tenant configuration from the related project should be used if specified
	if defaultTenantSpec != nil {
		tenant.Spec = *defaultTenantSpec
	}
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(tenant), tenant); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get tenant: %w", err)
		}
		if err := client.Create(ctx, tenant); err != nil {
			return fmt.Errorf("failed to create tenant: %w", err)
		}
	}

	// When default tenant spec has changed, we should update it in the kubelb management cluster
	if !equality.Semantic.DeepEqualWithNilDifferentFromEmpty(defaultTenantSpec, tenant.Spec) {
		tenant.Spec = *defaultTenantSpec
		if err := client.Update(ctx, tenant); err != nil {
			return fmt.Errorf("failed to update tenant: %w", err)
		}
	}

	return nil
}

func (r *reconciler) createOrUpdateKubeLBUserClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster, dc kubermaticv1.Datacenter) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	// Create RBAC for the user cluster.
	roleReconciler := []reconciling.NamedRoleReconcilerFactory{
		kubelbuserclusterresources.KubeSystemRoleReconciler(),
	}

	if err := reconciling.ReconcileRoles(ctx, roleReconciler, metav1.NamespaceSystem, userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile role: %w", err)
	}

	clusterRoleReconciler := []reconciling.NamedClusterRoleReconcilerFactory{
		kubelbuserclusterresources.ClusterRoleReconciler(dc, cluster),
	}

	if err := reconciling.ReconcileClusterRoles(ctx, clusterRoleReconciler, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile cluster role: %w", err)
	}

	roleBindingReconciler := []reconciling.NamedRoleBindingReconcilerFactory{
		kubelbuserclusterresources.KubeSystemRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, roleBindingReconciler, metav1.NamespaceSystem, userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile role binding: %w", err)
	}

	clusterRoleBindingReconciler := []reconciling.NamedClusterRoleBindingReconcilerFactory{
		kubelbuserclusterresources.ClusterRoleBindingReconciler(),
	}

	if err := reconciling.ReconcileClusterRoleBindings(ctx, clusterRoleBindingReconciler, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile cluster role binding: %w", err)
	}

	crdReconciler := []kkpreconciling.NamedCustomResourceDefinitionReconcilerFactory{
		kubelbuserclusterresources.SyncSecretCRDReconciler(),
	}

	if err := kkpreconciling.ReconcileCustomResourceDefinitions(ctx, crdReconciler, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile CustomResourceDefinitions: %w", err)
	}

	return nil
}

func (r *reconciler) createOrUpdateKubeLBSeedClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster, kubeLBManagementClient ctrlruntimeclient.Client, kubeLBManagementKubeConfig []byte, dc kubermaticv1.Datacenter) (*reconcile.Result, error) {
	seedNamespace := cluster.Status.NamespaceName
	tenantNamespace := fmt.Sprintf(kubelbresources.TenantNamespacePattern, cluster.Name)

	// Generate kubeconfig secret.
	kubelbKubeconfigSecret := &corev1.Secret{}
	err := kubeLBManagementClient.Get(ctx, types.NamespacedName{Name: kubeLBCCMKubeconfigSecretName, Namespace: tenantNamespace}, kubelbKubeconfigSecret)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get kubeLB kubeconfig secret: %w", err)
		}
		// KubeLB manager will create the secret eventually, so we requeue after 15 seconds.
		return &reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	tenantKubeconfig, err := normalizeTenantKubeconfig(kubelbKubeconfigSecret.Data[kubeconfigSecretKey], kubeLBManagementKubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize tenant kubeconfig: %w", err)
	}

	secretReconcilers := []reconciling.NamedSecretReconcilerFactory{
		kubelbseedresources.TenantKubeconfigSecretReconciler(string(tenantKubeconfig)),
	}

	// Create kubeconfig secret in the user cluster namespace.
	if err := reconciling.ReconcileSecrets(ctx, secretReconcilers, seedNamespace, r.Client); err != nil {
		return nil, fmt.Errorf("failed to reconcile kubeLB tenant kubeconfig secret: %w", err)
	}

	// Create RBAC for the user cluster.
	saReconcilers := []reconciling.NamedServiceAccountReconcilerFactory{
		kubelbseedresources.ServiceAccountReconciler(),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, saReconcilers, seedNamespace, r.Client); err != nil {
		return nil, fmt.Errorf("failed to reconcile service account: %w", err)
	}

	// Create/update kubeLB deployment.
	modifiers := []reconciling.ObjectModifier{
		modifier.RelatedRevisionsLabels(ctx, r),
		modifier.ControlplaneComponent(cluster),
	}
	deploymentReconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		kubelbseedresources.DeploymentReconciler(kubelbseedresources.NewKubeLBData(ctx, cluster, r, r.overwriteRegistry, dc)),
	}
	if err := reconciling.ReconcileDeployments(ctx, deploymentReconcilers, seedNamespace, r.Client, modifiers...); err != nil {
		return nil, fmt.Errorf("failed to reconcile the Deployments: %w", err)
	}

	return nil, nil
}

func (r *reconciler) getKubeLBManagementClusterClient(ctx context.Context, seed *kubermaticv1.Seed, dc kubermaticv1.Datacenter) (ctrlruntimeclient.Client, []byte, error) {
	kubeLBManagerKubeconfig, err := getKubeLBKubeconfigSecret(ctx, r.Client, seed, dc)
	if err != nil {
		return nil, nil, err
	}

	kubeconfigValue := kubeLBManagerKubeconfig.Data[resources.KubeconfigSecretKey]
	if len(kubeconfigValue) == 0 {
		return nil, nil, fmt.Errorf("no kubeconfig found")
	}

	kubeconfig, err := clientcmd.Load(kubeconfigValue)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}
	cfg, err := clientcmd.NewInteractiveClientConfig(*kubeconfig, "", nil, nil, nil).ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
	return client, kubeconfigValue, err
}

func getKubeLBKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, seed *kubermaticv1.Seed, dc kubermaticv1.Datacenter) (*corev1.Secret, error) {
	var name, namespace string

	switch {
	case dc.Spec.KubeLB != nil && dc.Spec.KubeLB.Kubeconfig.Name != "":
		name = dc.Spec.KubeLB.Kubeconfig.Name
		namespace = dc.Spec.KubeLB.Kubeconfig.Namespace
	case seed.Spec.KubeLB != nil && seed.Spec.KubeLB.Kubeconfig.Name != "":
		name = seed.Spec.KubeLB.Kubeconfig.Name
		namespace = seed.Spec.KubeLB.Kubeconfig.Namespace
	default:
		return nil, fmt.Errorf("kubeLB management kubeconfig not found")
	}

	secret := &corev1.Secret{}
	resourceName := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if resourceName.Namespace == "" {
		resourceName.Namespace = seed.Namespace
	}
	if err := client.Get(ctx, resourceName, secret); err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig secret %q: %w", resourceName.String(), err)
	}

	return secret, nil
}

// normalizeTenantKubeconfig is responsible for updating the API server and CA certificate in tenant kubeconfig.
// The tenant kubeconfig is generated by the KubeLB manager running inside the management cluster(using in-cluster kubeconfig). It's quite common that outside of that cluster,
// either DNS or IP forwarding is being used to access the API server of the management cluster. In that case, we need to replace the server URL
// and CA data in the tenant kubeconfig with the one from the management kubeconfig that has been provided to KKP in the seed resource.
func normalizeTenantKubeconfig(tenantKubeconfig, managementKubeconfig []byte) ([]byte, error) {
	managementCfg, err := clientcmd.Load(managementKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load management cluster kubeconfig: %w", err)
	}

	tenantCfg, err := clientcmd.Load(tenantKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load tenant cluster kubeconfig: %w", err)
	}

	// Get the first cluster from the kubeconfig.
	serverURL := ""
	var caCertificate []byte
	for _, cluster := range managementCfg.Clusters {
		serverURL = cluster.Server
		caCertificate = cluster.CertificateAuthorityData
		break
	}

	// Update the clusters in the tenant kubeconfig with the server URL and CA data from the management kubeconfig.
	// NOTE: Tenant kubeconfig is generated by KubeLB manager and we will never have multiple clusters here. But just for brevity sake, we are updating all clusters.
	for i := range tenantCfg.Clusters {
		tenantCfg.Clusters[i].Server = serverURL
		tenantCfg.Clusters[i].CertificateAuthorityData = caCertificate
	}

	return clientcmd.Write(*tenantCfg)
}
