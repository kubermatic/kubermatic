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

package defaultpolicycontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	PolicyTemplateCleanupControllerName = "kkp-default-policy-template-cleanup-controller"
)

type policyTemplateCleanupReconciler struct {
	ctrlruntimeclient.Client

	log *zap.SugaredLogger
}

func addPolicyTemplateCleanupController(mgr manager.Manager, numWorkers int, log *zap.SugaredLogger) error {
	reconciler := &policyTemplateCleanupReconciler{
		Client: mgr.GetClient(),
		log:    log.Named(PolicyTemplateCleanupControllerName),
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(PolicyTemplateCleanupControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.PolicyTemplate{}, builder.WithPredicates(withPolicyTemplateCleanupEventFilter())).
		Build(reconciler)

	return err
}

func (r *policyTemplateCleanupReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("policyTemplate", request.Name)
	log.Debug("Processing")

	policyTemplate := &kubermaticv1.PolicyTemplate{}
	if err := r.Get(ctx, request.NamespacedName, policyTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, r.cleanupPolicyBindingsForDeletedTemplate(ctx, request.Name)
		}
		return reconcile.Result{}, fmt.Errorf("failed to get PolicyTemplate: %w", err)
	}

	if policyTemplate.GetDeletionTimestamp() != nil {
		if err := r.cleanupPolicyBindingsForDeletedTemplate(ctx, policyTemplate.Name); err != nil {
			return reconcile.Result{}, err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.Client, policyTemplate, kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to remove PolicyTemplate cleanup finalizer: %w", err)
		}

		return reconcile.Result{}, nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.Client, policyTemplate, kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add PolicyTemplate cleanup finalizer: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *policyTemplateCleanupReconciler) cleanupPolicyBindingsForDeletedTemplate(ctx context.Context, policyTemplateName string) error {
	bindings := &kubermaticv1.PolicyBindingList{}
	if err := r.List(ctx, bindings); err != nil {
		return fmt.Errorf("failed to list PolicyBindings during template deletion: %w", err)
	}

	for _, binding := range bindings.Items {
		if binding.Spec.PolicyTemplateRef.Name == policyTemplateName {
			if err := r.Delete(ctx, &binding); ctrlruntimeclient.IgnoreNotFound(err) != nil {
				return fmt.Errorf("failed to delete PolicyBinding %s/%s for deleted template: %w", binding.Namespace, binding.Name, err)
			}
		}
	}

	return nil
}

func withPolicyTemplateCleanupEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			newObj := e.ObjectNew.(*kubermaticv1.PolicyTemplate)

			if newObj.GetDeletionTimestamp() != nil {
				return true
			}

			return !kuberneteshelper.HasFinalizer(newObj, kubermaticv1.PolicyTemplatePolicyBindingCleanupFinalizer)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
}
