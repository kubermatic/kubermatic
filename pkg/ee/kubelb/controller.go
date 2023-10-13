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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"text/template"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubelbmanagementresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/kubelb-cluster"
	kubelbseedresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/seed-cluster"
	kubelbuserclusterresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/user-cluster"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName   = "kkp-kubelb-controller"
	CleanupFinalizer = "kubermatic.k8c.io/cleanup-kubelb-ccm"
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

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &kubermaticv1.Cluster{}), &handler.EnqueueRequestForObject{}, workerlabel.Predicates(workerName), predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster := o.(*kubermaticv1.Cluster)
		// Only watch clusters that are in a state where they can be reconciled.
		return !cluster.Spec.Pause && cluster.DeletionTimestamp == nil && cluster.Status.NamespaceName != ""
	})); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}
	return nil
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

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionApplicationInstallationControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)
	if err != nil {
		r.log.Errorw("Failed to reconcile cluster", "cluster", cluster.Name, zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
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
	kubeLBManagementClient, cfg, err := r.getKubeLBManagementClusterClient(ctx, seed, datacenter)
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
	if err := r.createOrUpdateKubeLBUserClusterResources(ctx, cluster); err != nil {
		return nil, err
	}

	// Create/update required resources in user cluster namespace in seed.
	if err := r.createOrUpdateKubeLBSeedClusterResources(ctx, cluster, kubeLBManagementClient, cfg); err != nil {
		return nil, err
	}

	return nil, nil
}

func (r *reconciler) createOrUpdateKubeLBManagementClusterResources(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	namespace := cluster.Status.NamespaceName

	// Create namespace; which is equivalent to registering the tenant.
	nsReconcilers := []reconciling.NamedNamespaceReconcilerFactory{
		kubelbmanagementresources.NamespaceReconciler(namespace),
	}

	if err := reconciling.ReconcileNamespaces(ctx, nsReconcilers, "", client); err != nil {
		return fmt.Errorf("failed to reconcile namespace: %w", err)
	}

	// Create RBAC for the tenant.
	saReconcilers := []reconciling.NamedServiceAccountReconcilerFactory{
		kubelbmanagementresources.ServiceAccountReconciler(),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, saReconcilers, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile service account: %w", err)
	}

	roleReconcilers := []reconciling.NamedRoleReconcilerFactory{
		kubelbmanagementresources.RoleReconciler(),
	}

	if err := reconciling.ReconcileRoles(ctx, roleReconcilers, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile role: %w", err)
	}

	roleBindingReconcilers := []reconciling.NamedRoleBindingReconcilerFactory{
		kubelbmanagementresources.RoleBindingReconciler(namespace),
	}

	if err := reconciling.ReconcileRoleBindings(ctx, roleBindingReconcilers, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile role binding: %w", err)
	}

	// Create service account token secret.
	secretReconcilers := []reconciling.NamedSecretReconcilerFactory{
		kubelbmanagementresources.SecretReconciler(),
	}

	if err := reconciling.ReconcileSecrets(ctx, secretReconcilers, namespace, client); err != nil {
		return fmt.Errorf("failed to reconcile secret: %w", err)
	}
	return nil
}

func (r *reconciler) createOrUpdateKubeLBUserClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
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
		kubelbuserclusterresources.ClusterRoleReconciler(),
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

	return nil
}

func (r *reconciler) createOrUpdateKubeLBSeedClusterResources(ctx context.Context, cluster *kubermaticv1.Cluster, kubeLBManagementClient ctrlruntimeclient.Client, kubeconfig []byte) error {
	namespace := cluster.Status.NamespaceName

	// Generate kubeconfig secret.
	tenantKubeconfig, err := r.generateKubeconfig(ctx, kubeLBManagementClient, namespace, string(kubeconfig))
	if err != nil {
		return fmt.Errorf("failed to generate kubeconfig: %w", err)
	}

	secretReconcilers := []reconciling.NamedSecretReconcilerFactory{
		kubelbmanagementresources.TenantKubeconfigSecretReconciler(tenantKubeconfig),
	}

	// Create kubeconfig secret in the user cluster namespace.
	if err := reconciling.ReconcileSecrets(ctx, secretReconcilers, namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile kubeLB tenant kubeconfig secret: %w", err)
	}

	// Create RBAC for the user cluster.
	saReconcilers := []reconciling.NamedServiceAccountReconcilerFactory{
		kubelbseedresources.ServiceAccountReconciler(),
	}

	if err := reconciling.ReconcileServiceAccounts(ctx, saReconcilers, namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile service account: %w", err)
	}

	// Create/update kubeLB deployment.
	deploymentReconcilers := []reconciling.NamedDeploymentReconcilerFactory{
		kubelbseedresources.DeploymentReconciler(kubelbseedresources.NewKubeLBData(ctx, cluster, r, r.overwriteRegistry)),
	}
	if err := reconciling.ReconcileDeployments(ctx, deploymentReconcilers, namespace, r.Client); err != nil {
		return fmt.Errorf("failed to reconcile the Deployments: %w", err)
	}

	return nil
}

func (r *reconciler) generateKubeconfig(ctx context.Context, client ctrlruntimeclient.Client, namespace string, kubeconfig string) (string, error) {
	managementKubeconfig, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	secret := corev1.Secret{}
	err = client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: kubelbmanagementresources.ServiceAccountTokenSecretName}, &secret)
	if err != nil {
		return "", fmt.Errorf("failed to get ServiceAccount token Secret: %w", err)
	}

	var serverUrl string
	// Get the first cluster from the kubeconfig.
	for _, cluster := range managementKubeconfig.Clusters {
		serverUrl = cluster.Server
		break
	}

	ca := secret.Data[corev1.ServiceAccountRootCAKey]
	token := secret.Data[corev1.ServiceAccountTokenKey]

	// Generate kubeconfig.
	data := struct {
		CA_Certificate string
		Token          string
		Namespace      string
		ServerURL      string
	}{
		CA_Certificate: base64.StdEncoding.EncodeToString(ca),
		Token:          string(token),
		Namespace:      namespace,
		ServerURL:      serverUrl,
	}

	tmpl, err := template.New("kubeconfig").Parse(kubeconfigTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse kubeconfig template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, data); err != nil {
		return "", fmt.Errorf("failed to execute kubeconfig template: %w", err)
	}
	return buf.String(), nil
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
	if seed.Spec.KubeLB == nil || seed.Spec.KubeLB.Kubeconfig.Name == "" {
		if dc.Spec.KubeLB == nil || dc.Spec.KubeLB.Kubeconfig.Name == "" {
			return nil, fmt.Errorf("kubeLB management kubeconfig not found")
		}
		name = dc.Spec.KubeLB.Kubeconfig.Name
		namespace = dc.Spec.KubeLB.Kubeconfig.Namespace
	} else {
		name = seed.Spec.KubeLB.Kubeconfig.Name
		namespace = seed.Spec.KubeLB.Kubeconfig.Namespace
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

const kubeconfigTemplate = `apiVersion: v1
kind: Config
clusters:
- name: kubelb-cluster
  cluster:
    certificate-authority-data: {{ .CA_Certificate }}
    server: {{ .ServerURL }}
contexts:
- name: default-context
  context:
    cluster: kubelb-cluster
    namespace: {{ .Namespace }}
    user: default-user
current-context: default-context
users:
- name: default-user
  user:
    token: {{ .Token }}
`
