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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubelbresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources"
	kubelbseedresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/seed-cluster"
	kubelbuserclusterresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/user-cluster"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
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
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	overwriteRegistry             string
	versions                      kubermatic.Versions
}

func Add(mgr manager.Manager, numWorkers int, workerName string, overwriteRegistry string, seedGetter provider.SeedGetter, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &reconciler{
		Client:                        mgr.GetClient(),
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		seedGetter:                    seedGetter,
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
		overwriteRegistry:             overwriteRegistry,
	}

	clusterIsAlive := predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		// Only watch clusters that are in a state where they can be reconciled.
		return !cluster.Spec.Pause && cluster.DeletionTimestamp == nil && cluster.Status.NamespaceName != ""
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(workerlabel.Predicate(workerName), clusterIsAlive)).
		Build(reconciler)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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
			r.log.Debugw("Cleaning up kubeLB resources", "cluster", cluster.Name)
			return reconcile.Result{}, r.handleKubeLBCleanup(ctx, cluster)
		}
		// Finalizer doesn't exist so clean up is already done.
		return reconcile.Result{}, nil
	}

	// Kubelb was disabled after it was enabled. Clean up resources.
	if kuberneteshelper.HasFinalizer(cluster, CleanupFinalizer) && !cluster.Spec.IsKubeLBEnabled() {
		r.log.Debugw("Cleaning up kubeLB resources", "cluster", cluster.Name)
		return reconcile.Result{}, r.handleKubeLBCleanup(ctx, cluster)
	}

	// Kubelb is disabled. Nothing to do.
	if !cluster.Spec.IsKubeLBEnabled() {
		return reconcile.Result{}, nil
	}

	if cluster.Status.NamespaceName == "" {
		r.log.Debug("Skipping cluster reconciling because it has no namespace yet")
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		r.log.Debugf("API server is not running, trying again in %v", healthCheckPeriod)
		return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
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

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find datacenter %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}

	// Get kubeLB management cluster client.
	kubeLBManagementClient, err := r.getKubeLBManagementClusterClient(ctx, seed, datacenter)
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
	if err := r.createOrUpdateKubeLBManagementClusterResources(ctx, kubeLBManagementClient, cluster); err != nil {
		return nil, err
	}

	// Create/update required resources in user cluster.
	if err := r.createOrUpdateKubeLBUserClusterResources(ctx, cluster, datacenter); err != nil {
		return nil, err
	}

	// Create/update required resources in user cluster namespace in seed.
	return r.createOrUpdateKubeLBSeedClusterResources(ctx, cluster, kubeLBManagementClient, datacenter)
}

func (r *reconciler) createOrUpdateKubeLBManagementClusterResources(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	// Check if tenant exists or not, if it doesn't we create it.
	tenant := &kubelbv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name: cluster.Name,
		},
	}
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(tenant), tenant); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get tenant: %w", err)
		}
		if err := client.Create(ctx, tenant); err != nil {
			return fmt.Errorf("failed to create tenant: %w", err)
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

func (r *reconciler) createOrUpdateKubeLBSeedClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster, kubeLBManagementClient ctrlruntimeclient.Client, dc kubermaticv1.Datacenter) (*reconcile.Result, error) {
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

	tenantKubeconfig := string(kubelbKubeconfigSecret.Data[kubeconfigSecretKey])

	secretReconcilers := []reconciling.NamedSecretReconcilerFactory{
		kubelbseedresources.TenantKubeconfigSecretReconciler(tenantKubeconfig),
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
	deploymentReconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		kubelbseedresources.DeploymentReconciler(kubelbseedresources.NewKubeLBData(ctx, cluster, r, r.overwriteRegistry, dc)),
	}
	if err := reconciling.ReconcileDeployments(ctx, deploymentReconcilers, seedNamespace, r.Client); err != nil {
		return nil, fmt.Errorf("failed to reconcile the Deployments: %w", err)
	}

	return nil, nil
}

func (r *reconciler) getKubeLBManagementClusterClient(ctx context.Context, seed *kubermaticv1.Seed, dc kubermaticv1.Datacenter) (ctrlruntimeclient.Client, error) {
	kubeLBManagerKubeconfig, err := getKubeLBKubeconfigSecret(ctx, r.Client, seed, dc)
	if err != nil {
		return nil, err
	}

	kubeconfigValue := kubeLBManagerKubeconfig.Data[resources.KubeconfigSecretKey]
	if len(kubeconfigValue) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}

	kubeconfig, err := clientcmd.Load(kubeconfigValue)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}
	cfg, err := clientcmd.NewInteractiveClientConfig(*kubeconfig, "", nil, nil, nil).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
	return client, err
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
