//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

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

package kubevirtnetworkcontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName                = "kubevirt-network-controller"
	WorkloadSubnetLabel           = "k8c.io/kubevirt-workload-subnet"
	NetworkPolicyPodSelectorLabel = "cluster.x-k8s.io/cluster-name"
	NetworkPolicyCleanupFinalizer = "kubermatic.k8c.io/cleanup-kubevirt-infra-network-policy"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	seedGetter  provider.SeedGetter
	infraGetter kubevirtInfranGetter

	log        *zap.SugaredLogger
	versions   kubermatic.Versions
	workerName string
	recorder   record.EventRecorder
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,

	numWorkers int,
	workerName string,

	seedGetter provider.SeedGetter,

	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		log:         log.Named(ControllerName),
		Client:      mgr.GetClient(),
		seedGetter:  seedGetter,
		infraGetter: SetupKubeVirtInfraClient,
		workerName:  workerName,
		recorder:    mgr.GetEventRecorderFor(ControllerName),
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
			cluster := o.(*kubermaticv1.Cluster)
			return cluster.Spec.Cloud.ProviderName == string(kubermaticv1.KubevirtCloudProvider)
		}))).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	seed, err := r.seedGetter()
	if err != nil {
		return reconcile.Result{}, err
	}

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return reconcile.Result{}, fmt.Errorf("couldn't find datacenter %q for cluster %q", cluster.Spec.Cloud.DatacenterName, cluster.Name)
	}

	// Check if the datacenter is kubevirt and if kubevirt is configured with NamespacedMode
	if datacenter.Spec.Kubevirt == nil || datacenter.Spec.Kubevirt.NamespacedMode == nil || !datacenter.Spec.Kubevirt.NamespacedMode.Enabled {
		log.Debug("Skipping reconciliation as the datacenter is not kubevirt or kubevirt is not configured with NamespacedMode")
		return reconcile.Result{}, nil
	}

	if datacenter.Spec.Kubevirt.ProviderNetwork == nil {
		log.Debug("Skipping reconciliation as the provider network is not configured")
		return reconcile.Result{}, nil
	}

	if !datacenter.Spec.Kubevirt.ProviderNetwork.NetworkPolicyEnabled {
		log.Debug("Skipping reconciliation as the network policy is not enabled")
		return reconcile.Result{}, nil
	}

	kubeVirtInfraClient, err := r.infraGetter(ctx, cluster, r.Client)
	if err != nil {
		return &reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionKubeVirtNetworkControllerSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, kubeVirtInfraClient, cluster, datacenter.Spec.Kubevirt)
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

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, kubeVirtInfraClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, dc *kubermaticv1.DatacenterSpecKubevirt) (*reconcile.Result, error) {
	// Cluster is marked for deletion.
	if cluster.DeletionTimestamp != nil {
		if kuberneteshelper.HasFinalizer(cluster, NetworkPolicyCleanupFinalizer) {
			log.Debug("Cleaning up cloud provider")
			if err := cleanUpKubevirtCloudProviderNetworkPolicy(ctx, kubeVirtInfraClient, cluster, dc); err != nil {
				return &reconcile.Result{}, fmt.Errorf("failed cloud provider cleanup: %w", err)
			}
			return &reconcile.Result{}, kuberneteshelper.TryRemoveFinalizer(ctx, r.Client, cluster, NetworkPolicyCleanupFinalizer)
		}
		// Finalizer doesn't exist so clean up is already done.
		return &reconcile.Result{}, nil
	}

	// add the cleanup finalizer first
	if !kuberneteshelper.HasFinalizer(cluster, NetworkPolicyCleanupFinalizer) {
		if err := kuberneteshelper.TryAddFinalizer(ctx, r, cluster, NetworkPolicyCleanupFinalizer); err != nil {
			return nil, fmt.Errorf("failed to add finalizer: %w", err)
		}
	}

	switch dc.ProviderNetwork.NetworkPolicy.Mode {
	case kubermaticv1.NetworkPolicyModeDeny:
		log.Debug("Setting up cluster-isolation NetworkPolicy for the tenant cluster in default deny mode")
		if err := reconcileNamespacedClusterIsolationNetworkPolicyDefaultDeny(ctx, kubeVirtInfraClient, cluster, dc.NamespacedMode.Namespace, dc.DNSConfig.Nameservers); err != nil {
			return &reconcile.Result{}, err
		}
	case kubermaticv1.NetworkPolicyModeAllow:
		gateways := make([]string, 0)
		cidrs := make([]string, 0)
		for _, vpc := range dc.ProviderNetwork.VPCs {
			for _, subnet := range vpc.Subnets {
				log.Debug("Fetching gateway and cidr for subnet: %s", subnet.Name)
				gateway, cidr, err := processSubnet(ctx, kubeVirtInfraClient, subnet.Name)
				if err != nil {
					return &reconcile.Result{}, err
				}
				gateways = append(gateways, gateway)
				cidrs = append(cidrs, cidr)
			}
		}

		if len(gateways) > 0 && len(cidrs) > 0 {
			log.Debug("Setting up cluster-isolation NetworkPolicy for the tenant cluster in default allow mode")
			if err := reconcileNamespacedClusterIsolationNetworkPolicyDefaultAllow(ctx, kubeVirtInfraClient, cluster, cidrs, gateways, dc.NamespacedMode.Namespace); err != nil {
				return &reconcile.Result{}, err
			}
		}
	default:
		return &reconcile.Result{}, fmt.Errorf("no valid mode %v", dc.ProviderNetwork.NetworkPolicy.Mode)
	}

	return nil, nil
}

func processSubnet(ctx context.Context, kvInfraClient ctrlruntimeclient.Client, subnetName string) (string, string, error) {
	subnetUS := &unstructured.Unstructured{}
	subnetUS.SetAPIVersion("kubeovn.io/v1")
	subnetUS.SetKind("Subnet")

	if err := kvInfraClient.Get(ctx, types.NamespacedName{Name: subnetName}, subnetUS); err != nil {
		return "", "", err
	}

	gateway, _, err := unstructured.NestedString(subnetUS.Object, "spec", "gateway")
	if err != nil {
		return "", "", fmt.Errorf("invalid kubeovn Subnet: .spec.gateway is not a string: %w", err)
	}

	cidrBlock, _, err := unstructured.NestedString(subnetUS.Object, "spec", "cidrBlock")
	if err != nil {
		return "", "", fmt.Errorf("invalid kubeovn Subnet: .spec.cidrBlock is not a string: %w", err)
	}

	return gateway, cidrBlock, nil
}

func cleanUpKubevirtCloudProviderNetworkPolicy(ctx context.Context, kvInfraClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, dc *kubermaticv1.DatacenterSpecKubevirt) error {
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("cluster-isolation-%s", cluster.Name),
			Namespace: dc.NamespacedMode.Namespace,
		},
	}
	return kvInfraClient.Delete(ctx, networkPolicy)
}
