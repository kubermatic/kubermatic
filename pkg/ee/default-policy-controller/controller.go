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

package defaultpolicycontroller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-default-policy-controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName   string
	recorder     record.EventRecorder
	seedGetter   provider.SeedGetter
	configGetter provider.KubermaticConfigurationGetter
	log          *zap.SugaredLogger
	versions     kubermatic.Versions
}

func Add(ctx context.Context, mgr manager.Manager, numWorkers int, workerName string, seedGetter provider.SeedGetter, kubermaticConfigurationGetter provider.KubermaticConfigurationGetter, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName:   workerName,
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		seedGetter:   seedGetter,
		configGetter: kubermaticConfigurationGetter,
		log:          log,
		versions:     versions,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		// Watch for clusters
		For(&kubermaticv1.Cluster{}).
		// Watch changes for PolicyTemplates that have been enforced.
		Watches(&kubermaticv1.PolicyTemplate{}, enqueueClusters(reconciler, log), builder.WithPredicates(withEventFilter())).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log := r.log.With("cluster", cluster.Name)

	if cluster.DeletionTimestamp != nil {
		// Cluster deletion in progress, skipping reconciliation
		log.Debug("Cluster deletion in progress, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionDefaultPolicyControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster, log)
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

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, log *zap.SugaredLogger) (*reconcile.Result, error) {
	// Ensure that the cluster is healthy first, this is important for the default policy controller, and for all policy controllers.
	if !cluster.Status.ExtendedHealth.AllHealthy() {
		log.Debug("Cluster is not healthy, skipping reconciliation")
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Determine whether to ignore default policies
	createdCondition := kubermaticv1.ClusterConditionDefaultPolicyBindingsControllerCreatedSuccessfully
	ignoreDefaultPolicies := cluster.Status.HasConditionValue(createdCondition, corev1.ConditionTrue)

	policyTemplates := &kubermaticv1.PolicyTemplateList{}
	if err := r.List(ctx, policyTemplates); err != nil {
		return nil, fmt.Errorf("failed to list PolicyTemplates: %w", err)
	}

	var reconcilers []reconciling.NamedPolicyBindingReconcilerFactory
	for _, policyTemplate := range policyTemplates.Items {
		if policyTemplate.DeletionTimestamp != nil {
			continue
		}

		// Check if the PolicyTemplate targets this cluster
		if !isClusterTargeted(cluster, &policyTemplate) {
			continue
		}

		if policyTemplate.Spec.Enforced || (policyTemplate.Spec.Default && !ignoreDefaultPolicies) {
			reconcilers = append(reconcilers, policyBindingReconcilerFactory(policyTemplate))
		}
	}

	if err := reconciling.ReconcilePolicyBindings(ctx, reconcilers, cluster.Status.NamespaceName, r.Client); err != nil {
		return nil, err
	}

	if !cluster.Status.HasConditionValue(createdCondition, corev1.ConditionTrue) {
		if err := util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
			util.SetClusterCondition(
				cluster,
				r.versions,
				createdCondition,
				corev1.ConditionTrue,
				"",
				"",
			)
		}); err != nil {
			return &reconcile.Result{}, err
		}
	}

	return nil, nil
}

// policyBindingReconcilerFactory creates a named factory for reconciling policy bindings.
func policyBindingReconcilerFactory(template kubermaticv1.PolicyTemplate) reconciling.NamedPolicyBindingReconcilerFactory {
	return func() (string, reconciling.PolicyBindingReconciler) {
		return template.Name, func(binding *kubermaticv1.PolicyBinding) (*kubermaticv1.PolicyBinding, error) {
			annotations := make(map[string]string)

			if template.Spec.Enforced {
				annotations[kubermaticv1.AnnotationPolicyEnforced] = strconv.FormatBool(true)
			}
			if template.Spec.Default {
				annotations[kubermaticv1.AnnotationPolicyDefault] = strconv.FormatBool(true)
			}

			kubernetes.EnsureAnnotations(binding, annotations)

			kubernetes.EnsureLabels(binding, template.Labels)

			binding.Spec = kubermaticv1.PolicyBindingSpec{
				PolicyTemplateRef: corev1.ObjectReference{
					Name: template.Name,
				},
			}

			return binding, nil
		}
	}
}

// isClusterTargeted checks if the PolicyTemplate targets the given cluster.
func isClusterTargeted(cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate) bool {
	projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	// If no target is specified, we check the visibility
	if template.Spec.Target == nil {
		// Global policies apply to all clusters
		if template.Spec.Visibility == kubermaticv1.PolicyTemplateVisibilityGlobal {
			return true
		}

		// Project policies apply to clusters in the same project
		if template.Spec.Visibility == kubermaticv1.PolicyTemplateVisibilityProject &&
			template.Spec.ProjectID != "" &&
			projectID == template.Spec.ProjectID {
			return true
		}

		return false
	}

	// Check project selector if specified
	if template.Spec.Target.ProjectSelector != nil && projectID != "" {
		selector, err := metav1.LabelSelectorAsSelector(template.Spec.Target.ProjectSelector)
		if err != nil {
			return false
		}

		projectLabels := map[string]string{kubermaticv1.ProjectIDLabelKey: projectID}
		if !selector.Matches(labels.Set(projectLabels)) {
			return false
		}
	}

	// Check cluster selector if specified
	if template.Spec.Target.ClusterSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(template.Spec.Target.ClusterSelector)
		if err != nil {
			return false
		}

		if !selector.Matches(labels.Set(cluster.Labels)) {
			return false
		}
	}

	return true
}

func enqueueClusters(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request
		policyTemplate := obj.(*kubermaticv1.PolicyTemplate)

		// Check if the policy template is enforced
		if !policyTemplate.Spec.Enforced {
			return requests
		}

		// List all clusters
		clusters := &kubermaticv1.ClusterList{}
		if err := client.List(ctx, clusters); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list clusters: %w", err))
			return requests
		}

		for _, cluster := range clusters.Items {
			if isClusterTargeted(&cluster, policyTemplate) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: cluster.Name,
					},
				})
			}
		}
		return requests
	})
}

func withEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			obj := e.Object.(*kubermaticv1.PolicyTemplate)

			return obj.Spec.Enforced
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*kubermaticv1.PolicyTemplate)
			newObj := e.ObjectNew.(*kubermaticv1.PolicyTemplate)

			if newObj.GetDeletionTimestamp() != nil {
				return false
			}

			// If the template became enforced
			if !oldObj.Spec.Enforced && newObj.Spec.Enforced {
				return true
			}

			// If the template is enforced and changed
			if newObj.Spec.Enforced && newObj.GetGeneration() != oldObj.GetGeneration() {
				return true
			}

			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			obj := e.Object.(*kubermaticv1.PolicyTemplate)
			if obj.GetDeletionTimestamp() != nil {
				return false
			}

			return obj.Spec.Enforced
		},
	}
}
