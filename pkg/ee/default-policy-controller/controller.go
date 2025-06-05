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
	"strings"
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
		Watches(&kubermaticv1.PolicyTemplate{}, reconciler.enqueueClusters(), builder.WithPredicates(withEventFilter())).
		// Watch changes for PolicyBinding resources.
		Watches(&kubermaticv1.PolicyBinding{}, reconciler.enqueueClustersOnPolicyBindingDeletion(), builder.WithPredicates(withPolicyBindingEventFilter(reconciler))).
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
		if !r.isClusterTargeted(ctx, cluster, &policyTemplate) {
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
func (r *Reconciler) isClusterTargeted(ctx context.Context, cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate) bool {
	clusterProjectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]

	// If no target is specified, we check the visibility
	if template.Spec.Target == nil {
		return handleVisibilityOnly(cluster, template, clusterProjectID)
	}

	// Handle based on visibility and target combinations
	switch template.Spec.Visibility {
	case kubermaticv1.PolicyTemplateVisibilityGlobal:
		return r.handleGlobalWithTarget(ctx, cluster, template, clusterProjectID)

	case kubermaticv1.PolicyTemplateVisibilityProject:
		return handleProjectWithTarget(cluster, template, clusterProjectID)

	default:
		return false
	}
}

// handleVisibilityOnly handles when no target is specified - uses visibility rules only.
func handleVisibilityOnly(cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate, clusterProjectID string) bool {
	switch template.Spec.Visibility {
	case kubermaticv1.PolicyTemplateVisibilityGlobal:
		return true
	case kubermaticv1.PolicyTemplateVisibilityProject:
		return template.Spec.ProjectID != "" && clusterProjectID == template.Spec.ProjectID
	default:
		return false
	}
}

// handleGlobalWithTarget handles Global visibility with Target specified, using the provided client.
func (r *Reconciler) handleGlobalWithTarget(ctx context.Context, cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate, clusterProjectID string) bool {
	target := template.Spec.Target
	hasProjectSelector := target.ProjectSelector != nil
	hasClusterSelector := target.ClusterSelector != nil

	switch {
	case hasProjectSelector && hasClusterSelector:
		// Global + Target [ProjectSelector, ClusterSelector]
		return r.handleGlobalProjectAndClusterSelectors(ctx, cluster, template, clusterProjectID)
	case hasProjectSelector && !hasClusterSelector:
		// Global + Target [ProjectSelector]
		return r.handleGlobalProjectSelectorOnly(ctx, cluster, template, clusterProjectID)
	case !hasProjectSelector && hasClusterSelector:
		// Global + Target [ClusterSelector]
		return handleGlobalClusterSelectorOnly(cluster, template)
	default:
		return false
	}
}

// handleProjectWithTarget handles Project visibility with Target specified.
func handleProjectWithTarget(cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate, clusterProjectID string) bool {
	if template.Spec.ProjectID != "" && clusterProjectID != template.Spec.ProjectID {
		return false
	}

	if template.Spec.Target.ClusterSelector != nil {
		return matchesClusterSelector(cluster, template.Spec.Target.ClusterSelector)
	}

	return false
}

// handleGlobalProjectAndClusterSelectors handles Global + Project + Cluster selectors (AND filtering).
func (r *Reconciler) handleGlobalProjectAndClusterSelectors(ctx context.Context, cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate, clusterProjectID string) bool {
	if !matchesClusterSelector(cluster, template.Spec.Target.ClusterSelector) {
		return false
	}

	if !r.matchesProjectSelector(ctx, clusterProjectID, template.Spec.Target.ProjectSelector) {
		return false
	}

	return true
}

// handleGlobalProjectSelectorOnly handles Global + Project selector only.
func (r *Reconciler) handleGlobalProjectSelectorOnly(ctx context.Context, cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate, clusterProjectID string) bool {
	return r.matchesProjectSelector(ctx, clusterProjectID, template.Spec.Target.ProjectSelector)
}

// handleGlobalClusterSelectorOnly handles Global + Cluster selector only.
func handleGlobalClusterSelectorOnly(cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate) bool {
	return matchesClusterSelector(cluster, template.Spec.Target.ClusterSelector)
}

// matchesClusterSelector checks if a cluster matches the given cluster selector.
func matchesClusterSelector(cluster *kubermaticv1.Cluster, clusterSelector *metav1.LabelSelector) bool {
	if isLabelSelectorEmpty(clusterSelector) {
		return true
	}

	selector, err := metav1.LabelSelectorAsSelector(clusterSelector)
	if err != nil {
		return false
	}

	return selector.Matches(labels.Set(cluster.Labels))
}

// matchesProjectSelector checks if a project (by ID) matches the given project selector.
func (r *Reconciler) matchesProjectSelector(ctx context.Context, projectID string, projectSelector *metav1.LabelSelector) bool {
	if isLabelSelectorEmpty(projectSelector) {
		return true
	}

	projects := &kubermaticv1.ProjectList{}
	if err := r.List(ctx, projects); err != nil {
		return false
	}

	selector, err := metav1.LabelSelectorAsSelector(projectSelector)
	if err != nil {
		return false
	}

	for _, project := range projects.Items {
		if project.Name == projectID && selector.Matches(labels.Set(project.Labels)) {
			return true
		}
	}

	return false
}

func (r *Reconciler) enqueueClusters() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request
		policyTemplate := obj.(*kubermaticv1.PolicyTemplate)

		if policyTemplate.DeletionTimestamp != nil {
			return r.handlePolicyTemplateDeletion(ctx, policyTemplate)
		}

		// Check if the policy template is enforced
		if !policyTemplate.Spec.Enforced {
			return requests
		}

		// List all clusters
		clusters := &kubermaticv1.ClusterList{}
		if err := r.List(ctx, clusters); err != nil {
			r.log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list clusters: %w", err))
			return requests
		}

		for _, cluster := range clusters.Items {
			if r.isClusterTargeted(ctx, &cluster, policyTemplate) {
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

// handlePolicyTemplateDeletion handles cleanup when a PolicyTemplate is deleted
func (r *Reconciler) handlePolicyTemplateDeletion(ctx context.Context, policyTemplate *kubermaticv1.PolicyTemplate) []reconcile.Request {
	var requests []reconcile.Request

	bindings := &kubermaticv1.PolicyBindingList{}
	if err := r.List(ctx, bindings); err != nil {
		return requests
	}

	for _, binding := range bindings.Items {
		if binding.Spec.PolicyTemplateRef.Name == policyTemplate.Name {
			if err := r.Delete(ctx, &binding); err != nil {
				r.log.Error("Failed to delete PolicyBinding for deleted template", "error", err)
			}
		}
	}

	return requests
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
			return true
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

func (r *Reconciler) enqueueClustersOnPolicyBindingDeletion() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request
		policyBinding := obj.(*kubermaticv1.PolicyBinding)

		clusterNamespace := policyBinding.Namespace
		if !strings.HasPrefix(clusterNamespace, "cluster-") {
			return requests
		}
		clusterName := strings.TrimPrefix(clusterNamespace, "cluster-")

		cluster := &kubermaticv1.Cluster{}
		if err := r.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
			if !apierrors.IsNotFound(err) {
				r.log.Error("Failed to get cluster for PolicyBinding deletion", "cluster", clusterName, "error", err)
			}
			return requests
		}

		if policyBinding.Spec.PolicyTemplateRef.Name != "" {
			policyTemplate := &kubermaticv1.PolicyTemplate{}
			if err := r.Get(ctx, types.NamespacedName{Name: policyBinding.Spec.PolicyTemplateRef.Name}, policyTemplate); err != nil {
				if !apierrors.IsNotFound(err) {
					r.log.Error("Failed to get PolicyTemplate for PolicyBinding deletion", "template", policyBinding.Spec.PolicyTemplateRef.Name, "error", err)
				}
				return requests
			}

			if policyTemplate.Spec.Enforced && r.isClusterTargeted(ctx, cluster, policyTemplate) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: clusterName,
					},
				})
			}
		}

		return requests
	})
}

func withPolicyBindingEventFilter(reconciler *Reconciler) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			obj := e.Object.(*kubermaticv1.PolicyBinding)

			return obj.Spec.PolicyTemplateRef.Name != ""
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// isLabelSelectorEmpty checks if a LabelSelector is semantically empty.
func isLabelSelectorEmpty(selector *metav1.LabelSelector) bool {
	if selector == nil {
		return true
	}
	return len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0
}
