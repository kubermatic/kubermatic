/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package webterminaladmissionpolicy

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	webterminalresources "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/webterminal-admission-policy/resources"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources"
	kkpreconciling "k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kkp-webterminal-admission-policy"
)

type reconciler struct {
	log             *zap.SugaredLogger
	client          ctrlruntimeclient.Client
	clusterIsPaused userclustercontrollermanager.IsPausedChecker

	// enabled is set when the web terminal runs as a ServiceAccount in this KKP installation
	// (feature gate WebTerminalServiceAccount); it arrives as the -web-terminal-serviceaccount flag.
	enabled bool
}

func Add(
	log *zap.SugaredLogger,
	seedMgr, userMgr manager.Manager,
	clusterName string,
	clusterIsPaused userclustercontrollermanager.IsPausedChecker,
	enabled bool,
) error {
	r := &reconciler{
		log:             log.Named(ControllerName),
		client:          userMgr.GetClient(),
		clusterIsPaused: clusterIsPaused,
		enabled:         enabled,
	}

	enqueue := func(context.Context, ctrlruntimeclient.Object) []reconcile.Request {
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: clusterName}}}
	}

	// Matched by name, not by label, so removing a label cannot stop us from putting a policy back or
	// make us forget a session. Each policy and its binding share a name, so one predicate covers both.
	policyNames := predicateutil.ByName(webterminalresources.PolicyNames()...)
	sessionServiceAccounts := predicateutil.Factory(isSessionServiceAccount)

	_, err := builder.ControllerManagedBy(userMgr).
		Named(ControllerName).
		// The Cluster triggers the first reconciliation, and one when the cluster is unpaused,
		// which changes none of the objects in the user cluster.
		WatchesRawSource(source.Kind(
			seedMgr.GetCache(),
			&kubermaticv1.Cluster{},
			handler.TypedEnqueueRequestsFromMapFunc(func(ctx context.Context, c *kubermaticv1.Cluster) []reconcile.Request {
				return enqueue(ctx, c)
			}),
			predicateutil.TypedByName[*kubermaticv1.Cluster](clusterName),
			predicate.TypedFuncs[*kubermaticv1.Cluster]{
				UpdateFunc: func(e event.TypedUpdateEvent[*kubermaticv1.Cluster]) bool {
					return e.ObjectOld.Spec.Pause != e.ObjectNew.Spec.Pause
				},
			},
		)).
		Watches(&admissionregistrationv1.ValidatingAdmissionPolicy{}, handler.EnqueueRequestsFromMapFunc(enqueue), builder.WithPredicates(policyNames)).
		Watches(&admissionregistrationv1.ValidatingAdmissionPolicyBinding{}, handler.EnqueueRequestsFromMapFunc(enqueue), builder.WithPredicates(policyNames)).
		Watches(&corev1.ServiceAccount{}, handler.EnqueueRequestsFromMapFunc(enqueue), builder.WithPredicates(sessionServiceAccounts)).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {
	r.log.Debug("Reconciling")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, r.reconcile(ctx)
}

// reconcile keeps the policies in place while the web terminal runs as a ServiceAccount, so that
// they are there before the first session. If it does not, a session that still exists keeps them:
// one that was started before the mode was switched off, or by a KKP API that has the feature gate
// earlier than this controller. Without the mode and without a session nothing needs the policies,
// and they are removed.
func (r *reconciler) reconcile(ctx context.Context) error {
	needed := r.enabled
	if !needed {
		sessionExists, err := r.sessionExists(ctx)
		if err != nil {
			return err
		}
		needed = sessionExists
	}

	if !needed {
		return r.ensureObjectsAreRemoved(ctx, webterminalresources.ResourcesForDeletion())
	}

	if err := kkpreconciling.ReconcileValidatingAdmissionPolicies(ctx, webterminalresources.ValidatingAdmissionPolicyReconcilers(), "", r.client); err != nil {
		return fmt.Errorf("failed to reconcile ValidatingAdmissionPolicies: %w", err)
	}

	// Reconcile the bindings after the policies so they never reference a missing policy.
	if err := kkpreconciling.ReconcileValidatingAdmissionPolicyBindings(ctx, webterminalresources.ValidatingAdmissionPolicyBindingReconcilers(), "", r.client); err != nil {
		return fmt.Errorf("failed to reconcile ValidatingAdmissionPolicyBindings: %w", err)
	}

	return nil
}

// isSessionServiceAccount returns true for the ServiceAccount of a web terminal session. The
// KKP API creates it for a session and the cleanup of the session deletes it.
func isSessionServiceAccount(o ctrlruntimeclient.Object) bool {
	return o.GetNamespace() == metav1.NamespaceSystem && strings.HasPrefix(o.GetName(), resources.WebTerminalSessionNamePrefix)
}

func (r *reconciler) sessionExists(ctx context.Context) (bool, error) {
	serviceAccounts := &corev1.ServiceAccountList{}
	if err := r.client.List(ctx, serviceAccounts, ctrlruntimeclient.InNamespace(metav1.NamespaceSystem)); err != nil {
		return false, fmt.Errorf("failed to list ServiceAccounts: %w", err)
	}

	for i := range serviceAccounts.Items {
		if isSessionServiceAccount(&serviceAccounts.Items[i]) {
			return true, nil
		}
	}

	return false, nil
}

// ensureObjectsAreRemoved deletes the given objects in order, skipping those that do not exist.
func (r *reconciler) ensureObjectsAreRemoved(ctx context.Context, objects []ctrlruntimeclient.Object) error {
	for _, object := range objects {
		// Check the cache first to avoid a DELETE on every reconcile.
		if err := r.client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(object), object); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("failed to get %T %q: %w", object, object.GetName(), err)
		}

		if err := r.client.Delete(ctx, object); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to remove %T %q: %w", object, object.GetName(), err)
		}

		r.log.Infow("Removed web terminal admission policy object", "kind", fmt.Sprintf("%T", object), "name", object.GetName())
	}

	return nil
}
