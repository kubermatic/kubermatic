//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

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

package gatewayapiprotectioncontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	gatewayapiresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/gateway-api-protection-controller/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kkp-kubelb-gateway-api-protection-controller"
)

type reconciler struct {
	seedClient ctrlruntimeclient.Client
	userClient ctrlruntimeclient.Client
	recorder   events.EventRecorder
	log        *zap.SugaredLogger
	versions   kubermatic.Versions

	// adminDisabled is set when an admin disabled the protection in the KubermaticConfiguration or the
	// datacenter; it arrives as the -kubelb-disable-gateway-api-protection flag.
	adminDisabled bool
}

func Add(
	seedMgr, userMgr manager.Manager,
	log *zap.SugaredLogger,
	clusterName string,
	versions kubermatic.Versions,
	adminDisabled bool,
) error {
	r := &reconciler{
		seedClient:    seedMgr.GetClient(),
		userClient:    userMgr.GetClient(),
		recorder:      seedMgr.GetEventRecorder(ControllerName),
		log:           log,
		versions:      versions,
		adminDisabled: adminDisabled,
	}

	enqueueCluster := func(context.Context, ctrlruntimeclient.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: clusterName}}}
	}

	// Only react to KKP's own policy and the upstream safe-upgrades policy it removes. Matched by name,
	// not label, so removing the label cannot stop us from restoring our policy. Each policy and its
	// binding share a name, so one predicate covers both.
	policyNames := predicate.ByName(gatewayapiresources.GatewayAPIAdmissionPolicyName, gatewayapiresources.UpstreamSafeUpgradesPolicyName)

	_, err := builder.ControllerManagedBy(userMgr).
		Named(ControllerName).
		WatchesRawSource(source.Kind(
			seedMgr.GetCache(),
			&kubermaticv1.Cluster{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, c *kubermaticv1.Cluster) []reconcile.Request {
				return enqueueCluster(ctx, c)
			}),
			predicate.TypedByName[*kubermaticv1.Cluster](clusterName),
			predicate.TypedFactory(func(cluster *kubermaticv1.Cluster) bool {
				// Only watch clusters that are in a state where they can be reconciled.
				// Pause-flag is checked by the ReconcileWrapper.
				return cluster.DeletionTimestamp == nil && cluster.Status.ExtendedHealth.ControlPlaneHealthy()
			}),
		)).
		Watches(&admissionregistrationv1.ValidatingAdmissionPolicy{}, handler.EnqueueRequestsFromMapFunc(enqueueCluster), builder.WithPredicates(policyNames)).
		Watches(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{}, handler.EnqueueRequestsFromMapFunc(enqueueCluster), builder.WithPredicates(policyNames)).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Resource is marked for deletion.
	if cluster.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r.seedClient,
		"",
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionKubeLBGatewayAPIProtectionReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return nil, r.reconcile(ctx, cluster)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
		return reconcile.Result{}, err
	}

	return *result, err
}

// reconcile manages the policy that reserves the Gateway API CRDs for the kubeLB CCM while kubeLB and its
// Gateway API support are enabled, and removes the upstream safe-upgrades policy while that protection
// is in place.
func (r *reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Remove the policy when it is not needed, or Gateway API CRD writes stay blocked.
	if !kubeLBGatewayAPIEnabled(cluster) {
		if err := r.ensureObjectsAreRemoved(ctx, gatewayapiresources.GatewayAPIAdmissionPolicyResourcesForDeletion()); err != nil {
			return err
		}

		// Nothing to protect, so report nothing rather than "unprotected".
		return r.clearGatewayAPIProtectedStatus(ctx, cluster)
	}

	if r.gatewayAPIProtectionDisabled(cluster) {
		if err := r.ensureObjectsAreRemoved(ctx, gatewayapiresources.GatewayAPIAdmissionPolicyResourcesForDeletion()); err != nil {
			return err
		}

		return r.setGatewayAPIProtectedStatus(ctx, cluster, false)
	}

	policyReconcilers := []kkpreconciling.NamedValidatingAdmissionPolicyReconcilerFactory{
		gatewayapiresources.GatewayAPIValidatingAdmissionPolicyReconciler(),
	}
	if err := kkpreconciling.ReconcileValidatingAdmissionPolicies(ctx, policyReconcilers, "", r.userClient); err != nil {
		return fmt.Errorf("failed to reconcile ValidatingAdmissionPolicies: %w", err)
	}

	// Reconcile the binding after the policy so it never references a missing policy.
	bindingReconcilers := []kkpreconciling.NamedValidatingAdmissionPolicyBindingReconcilerFactory{
		gatewayapiresources.GatewayAPIValidatingAdmissionPolicyBindingReconciler(),
	}
	if err := kkpreconciling.ReconcileValidatingAdmissionPolicyBindings(ctx, bindingReconcilers, "", r.userClient); err != nil {
		return fmt.Errorf("failed to reconcile ValidatingAdmissionPolicyBindings: %w", err)
	}

	// The upstream safe-upgrades policy comes with any Gateway API bundle and version-locks the CCM's
	// CRDs; a policy cannot block another policy, so remove it while KubeLB owns the CRDs.
	if err := r.ensureObjectsAreRemoved(ctx, gatewayapiresources.UpstreamSafeUpgradesResourcesForDeletion()); err != nil {
		return err
	}

	return r.setGatewayAPIProtectedStatus(ctx, cluster, true)
}

// setGatewayAPIProtectedStatus records whether the protection is in place. It is written after the
// policy was reconciled, so it reflects the effective result of all admin and cluster settings.
func (r *reconciler) setGatewayAPIProtectedStatus(ctx context.Context, cluster *kubermaticv1.Cluster, protected bool) error {
	current := cluster.Status.KubeLB
	if current != nil && current.GatewayAPIProtected == protected {
		return nil
	}

	return util.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		if c.Status.KubeLB == nil {
			c.Status.KubeLB = &kubermaticv1.KubeLBStatus{}
		}
		c.Status.KubeLB.GatewayAPIProtected = protected
	})
}

// clearGatewayAPIProtectedStatus removes the kubeLB status block, including a stale one left behind
// after Gateway API support was turned off.
func (r *reconciler) clearGatewayAPIProtectedStatus(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if cluster.Status.KubeLB == nil {
		return nil
	}

	// A nil pointer becomes null in the merge patch, which removes the key.
	return util.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.KubeLB = nil
	})
}

// gatewayAPIProtectionDisabled reports whether an admin (via the startup flag) or the cluster itself
// disabled the protection. Either one disabling it wins.
func (r *reconciler) gatewayAPIProtectionDisabled(cluster *kubermaticv1.Cluster) bool {
	if r.adminDisabled {
		return true
	}

	return cluster != nil && cluster.Spec.KubeLB != nil && cluster.Spec.KubeLB.DisableGatewayAPIProtection
}

// kubeLBGatewayAPIEnabled reports whether the kubeLB CCM runs with Gateway API support. Both switches
// are required: disabling kubeLB keeps enableGatewayAPI set but removes the CCM.
func kubeLBGatewayAPIEnabled(cluster *kubermaticv1.Cluster) bool {
	return cluster != nil && cluster.Spec.IsKubeLBEnabled() && cluster.Spec.KubeLB.IsGatewayAPIEnabled()
}

// ensureObjectsAreRemoved deletes the given objects in order, skipping those that do not exist.
func (r *reconciler) ensureObjectsAreRemoved(ctx context.Context, objects []ctrlruntimeclient.Object) error {
	for _, object := range objects {
		// Check the cache first to avoid a DELETE on every reconcile.
		if err := r.userClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(object), object); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("failed to get %T %q: %w", object, object.GetName(), err)
		}

		if err := r.userClient.Delete(ctx, object); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to ensure %T %q is removed/not present: %w", object, object.GetName(), err)
		}
	}

	return nil
}
