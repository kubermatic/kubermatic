//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

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

package policybindingcontroller

import (
	"context"
	"encoding/json"
	"fmt"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	crpredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName   = "kkp-policy-binding-controller"
	CleanupFinalizer = "kubermatic.k8c.io/cleanup-policy-binding"
)

type reconciler struct {
	seedClient ctrlruntimeclient.Client
	userClient ctrlruntimeclient.Client

	log             *zap.SugaredLogger
	recorder        record.EventRecorder
	namespace       string
	clusterName     string
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

// Add creates the controller and registers watches.
func Add(seedMgr, userMgr manager.Manager, log *zap.SugaredLogger, namespace, clusterName string, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	r := &reconciler{
		seedClient:      seedMgr.GetClient(),
		userClient:      userMgr.GetClient(),
		log:             log.Named(ControllerName),
		recorder:        userMgr.GetEventRecorderFor(ControllerName),
		namespace:       namespace,
		clusterName:     clusterName,
		clusterIsPaused: clusterIsPaused,
	}

	// Predicate to limit PolicyBindings to our namespace.
	inNamespace := kubermaticpred.ByNamespace(namespace)

	// Watch PolicyBinding resources.
	builderCtrl := builder.ControllerManagedBy(userMgr).
		Named(ControllerName).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		For(&kubermaticv1.PolicyBinding{}, builder.WithPredicates(inNamespace))

	// Watch PolicyTemplate changes.
	templateHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request
		template := obj.(*kubermaticv1.PolicyTemplate)

		bindings := &kubermaticv1.PolicyBindingList{}
		if err := r.seedClient.List(ctx, bindings, ctrlruntimeclient.InNamespace(namespace)); err != nil {
			r.log.Errorw("failed to list PolicyBindings", "err", err)
			return requests
		}
		for _, b := range bindings.Items {
			if b.Spec.PolicyTemplateRef.Name == template.Name {
				requests = append(requests, reconcile.Request{NamespacedName: ctrlruntimeclient.ObjectKey{
					Namespace: namespace,
					Name:      b.Name,
				}})
			}
		}
		return requests
	})

	templatePred := crpredicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return true },
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		UpdateFunc:  func(e event.UpdateEvent) bool { return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}

	builderCtrl = builderCtrl.Watches(&kubermaticv1.PolicyTemplate{}, templateHandler, builder.WithPredicates(templatePred))

	// Watch ClusterPolicy resources in the user cluster for drift.
	cpHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		cp := obj.(*kyvernov1.ClusterPolicy)
		bindingName, ok := cp.Labels["kubermatic.k8c.io/policy-binding"]
		if !ok {
			return nil
		}
		return []reconcile.Request{{NamespacedName: ctrlruntimeclient.ObjectKey{Namespace: namespace, Name: bindingName}}}
	})

	builderCtrl = builderCtrl.Watches(&kyvernov1.ClusterPolicy{}, cpHandler)

	_, err := builderCtrl.Build(r)
	return err
}

// Reconcile reconciles a single PolicyBinding.
func (r *reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("binding", req.NamespacedName)
	log.Debug("Reconciling")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	binding := &kubermaticv1.PolicyBinding{}
	if err := r.seedClient.Get(ctx, req.NamespacedName, binding); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: r.clusterName}, cluster); err != nil {
		return reconcile.Result{}, err
	}

	// Handle Kyverno disabled
	if !cluster.Spec.IsKyvernoEnabled() {
		return reconcile.Result{}, r.handleKyvernoDisabled(ctx, binding)
	}

	// Normal reconciling
	if err := r.reconcile(ctx, binding); err != nil {
		r.recorder.Event(binding, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// handleKyvernoDisabled ensures cleanup when the feature is disabled.
func (r *reconciler) handleKyvernoDisabled(ctx context.Context, binding *kubermaticv1.PolicyBinding) error {
	// Always attempt to delete ClusterPolicy first.
	if err := r.deleteClusterPolicy(ctx, binding.Spec.PolicyTemplateRef.Name); err != nil {
		return err
	}

	// Remove finalizer to allow deletion.
	if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.seedClient, binding, CleanupFinalizer); err != nil {
		return fmt.Errorf("failed to remove finalizer: %w", err)
	}

	// Delete the binding object itself (ignore NotFound to stay idempotent)
	return ctrlruntimeclient.IgnoreNotFound(r.seedClient.Delete(ctx, binding))
}

func (r *reconciler) reconcile(ctx context.Context, binding *kubermaticv1.PolicyBinding) error {
	if !binding.DeletionTimestamp.IsZero() {
		if err := r.deleteClusterPolicy(ctx, binding.Spec.PolicyTemplateRef.Name); err != nil {
			return err
		}
		return kuberneteshelper.TryRemoveFinalizer(ctx, r.seedClient, binding, CleanupFinalizer)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.seedClient, binding, CleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	template := &kubermaticv1.PolicyTemplate{}
	if err := r.seedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: binding.Spec.PolicyTemplateRef.Name}, template); err != nil {
		if apierrors.IsNotFound(err) {
			if err := r.deleteClusterPolicy(ctx, binding.Spec.PolicyTemplateRef.Name); err != nil {
				return err
			}
			return r.seedClient.Delete(ctx, binding)
		}
		return err
	}

	if template.DeletionTimestamp != nil {
		if err := r.deleteClusterPolicy(ctx, template.Name); err != nil {
			return err
		}
		return r.seedClient.Delete(ctx, binding)
	}

	if template.Spec.NamespacedPolicy {
		return r.updateStatus(ctx, binding, template, false)
	}

	factories := []reconciling.NamedKyvernoClusterPolicyReconcilerFactory{
		r.clusterPolicyFactory(template, binding),
	}

	if err := reconciling.ReconcileKyvernoClusterPolicys(ctx, factories, "", r.userClient); err != nil {
		_ = r.updateStatus(ctx, binding, template, false)
		return fmt.Errorf("failed to reconcile ClusterPolicy: %w", err)
	}

	return r.updateStatus(ctx, binding, template, true)
}

// deleteClusterPolicy removes the ClusterPolicy with the given name, ignoring NotFound.
func (r *reconciler) deleteClusterPolicy(ctx context.Context, policyName string) error {
	cp := &kyvernov1.ClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policyName,
		},
	}
	return ctrlruntimeclient.IgnoreNotFound(r.userClient.Delete(ctx, cp))
}

func (r *reconciler) updateStatus(ctx context.Context, binding *kubermaticv1.PolicyBinding, template *kubermaticv1.PolicyTemplate, active bool) error {
	updated := binding.DeepCopy()
	updated.Status.ObservedGeneration = binding.Generation
	updated.Status.TemplateEnforced = &template.Spec.Enforced
	updated.Status.Active = &active

	return r.seedClient.Status().Patch(ctx, updated, ctrlruntimeclient.MergeFrom(binding))
}

func (r *reconciler) clusterPolicyFactory(template *kubermaticv1.PolicyTemplate, binding *kubermaticv1.PolicyBinding) reconciling.NamedKyvernoClusterPolicyReconcilerFactory {
	return func() (string, reconciling.KyvernoClusterPolicyReconciler) {
		return template.Name, func(existing *kyvernov1.ClusterPolicy) (*kyvernov1.ClusterPolicy, error) {
			// Labels
			kubernetes.EnsureLabels(existing, map[string]string{
				"kubermatic.k8c.io/policy-binding":  binding.Name,
				"kubermatic.k8c.io/policy-template": template.Name,
			})

			// Annotations
			ann := map[string]string{
				"policies.kyverno.io/title":       template.Spec.Title,
				"policies.kyverno.io/description": template.Spec.Description,
			}
			if template.Spec.Category != "" {
				ann["policies.kyverno.io/category"] = template.Spec.Category
			}
			if template.Spec.Severity != "" {
				ann["policies.kyverno.io/severity"] = template.Spec.Severity
			}
			for k, v := range template.Annotations {
				ann[k] = v
			}
			kubernetes.EnsureAnnotations(existing, ann)

			// Kyverno Spec
			var spec kyvernov1.Spec
			if err := json.Unmarshal(template.Spec.PolicySpec.Raw, &spec); err != nil {
				return nil, fmt.Errorf("failed to unmarshal policySpec: %w", err)
			}
			existing.Spec = spec
			return existing, nil
		}
	}
}
