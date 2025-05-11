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
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
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

	if cluster.DeletionTimestamp != nil {
		// Cluster deletion in progress, skipping reconciliation
		r.log.Debugw("Cluster deletion in progress, skipping reconciliation", "cluster", cluster.Name)
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

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// Ensure that the cluster is healthy first, this is important for the default policy controller, and for all policy controllers.
	if !cluster.Status.ExtendedHealth.AllHealthy() {
		r.log.Debugw("Cluster is not healthy, skipping reconciliation", "cluster", cluster.Name)
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Determine whether to ignore default policies
	ignoreDefaultPolicies := false

	// Default policies are already created.
	if cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionDefaultPolicyBindingsControllerCreatedSuccessfully, corev1.ConditionTrue) {
		ignoreDefaultPolicies = true
	}

	// List all PolicyTemplates
	policyTemplates := &kubermaticv1.PolicyTemplateList{}
	if err := r.List(ctx, policyTemplates); err != nil {
		return nil, fmt.Errorf("failed to list PolicyTemplates: %w", err)
	}

	if cluster.Status.NamespaceName == "" {
		return nil, fmt.Errorf("cluster %s has no namespace name", cluster.Name)
	}

	// Collect all policy templates that need to be installed/updated.
	templates := []kubermaticv1.PolicyTemplate{}
	for _, policyTemplate := range policyTemplates.Items {
		if policyTemplate.DeletionTimestamp != nil {
			continue
		}

		// Check if the PolicyTemplate targets this cluster
		if !isClusterTargeted(cluster, &policyTemplate) {
			continue
		}

		if policyTemplate.Spec.Enforced || (policyTemplate.Spec.Default && !ignoreDefaultPolicies) {
			templates = append(templates, policyTemplate)
		}
	}

	// Get the namespace name for the cluster's control plane components
	namespace := cluster.Status.NamespaceName

	var errors []error
	for _, template := range templates {
		err := r.ensurePolicyBinding(ctx, namespace, template, cluster)
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) == 0 && !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionDefaultPolicyBindingsControllerCreatedSuccessfully, corev1.ConditionTrue) {
		if err := util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
			util.SetClusterCondition(
				cluster,
				r.versions,
				kubermaticv1.ClusterConditionDefaultPolicyBindingsControllerCreatedSuccessfully,
				corev1.ConditionTrue,
				"",
				"",
			)
		}); err != nil {
			return &reconcile.Result{}, err
		}
	}

	return nil, kerrors.NewAggregate(errors)
}

func (r *Reconciler) ensurePolicyBinding(ctx context.Context, namespace string, template kubermaticv1.PolicyTemplate, cluster *kubermaticv1.Cluster) error {
	// Check if the binding already exists
	existingPolicyBindingList := &kubermaticv1.PolicyBindingList{}
	if err := r.List(ctx, existingPolicyBindingList, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list policy bindings: %w", err)
	}

	// Check if we already have a binding for this template
	var currentPolicyBinding *kubermaticv1.PolicyBinding
	for _, binding := range existingPolicyBindingList.Items {
		if binding.Spec.PolicyTemplateRef.Name == template.Name {
			// Check if this is an enforced or default binding
			policyEnforced, _ := strconv.ParseBool(binding.Annotations[kubermaticv1.AnnotationPolicyEnforced])
			policyDefault, _ := strconv.ParseBool(binding.Annotations[kubermaticv1.AnnotationPolicyDefault])

			if policyEnforced || policyDefault {
				currentPolicyBinding = &binding
				break
			}
		}
	}

	reconcilers := []reconciling.NamedPolicyBindingReconcilerFactory{
		PolicyBindingReconciler(r.log, template, currentPolicyBinding, cluster),
	}
	return reconciling.ReconcilePolicyBindings(ctx, reconcilers, namespace, r.Client)
}

func PolicyBindingReconciler(logger *zap.SugaredLogger, template kubermaticv1.PolicyTemplate, existingBinding *kubermaticv1.PolicyBinding, cluster *kubermaticv1.Cluster) reconciling.NamedPolicyBindingReconcilerFactory {
	return func() (string, reconciling.PolicyBindingReconciler) {
		bindingName := template.Name
		return bindingName, func(binding *kubermaticv1.PolicyBinding) (*kubermaticv1.PolicyBinding, error) {
			delete(template.Annotations, corev1.LastAppliedConfigAnnotation)

			annotations := template.Annotations
			if annotations == nil {
				annotations = make(map[string]string)
			}

			if template.Spec.Enforced {
				annotations[kubermaticv1.AnnotationPolicyEnforced] = "true"
			}
			if template.Spec.Default {
				annotations[kubermaticv1.AnnotationPolicyDefault] = "true"
			}

			// Copy labels from the template if any
			if len(template.Labels) > 0 {
				if binding.Labels == nil {
					binding.Labels = make(map[string]string)
				}
				for k, v := range template.Labels {
					binding.Labels[k] = v
				}
			}

			binding.Annotations = annotations
			binding.Spec = kubermaticv1.PolicyBindingSpec{
				PolicyTemplateRef: corev1.ObjectReference{
					Name: template.Name,
				},
			}

			logger.Infof("Reconciling PolicyBinding %s/%s", binding.Namespace, binding.Name)
			return binding, nil
		}
	}
}

// isClusterTargeted checks if the PolicyTemplate targets the given cluster
func isClusterTargeted(cluster *kubermaticv1.Cluster, template *kubermaticv1.PolicyTemplate) bool {
	// If no target is specified, we check the visibility
	if template.Spec.Target == nil {
		// Global policies apply to all clusters
		if template.Spec.Visibility == kubermaticv1.PolicyTemplateVisibilityGlobal {
			return true
		}

		// Project policies apply to clusters in the same project
		if template.Spec.Visibility == kubermaticv1.PolicyTemplateVisibilityProject &&
			template.Spec.ProjectID != "" &&
			cluster.Labels[kubermaticv1.ProjectIDLabelKey] == template.Spec.ProjectID {
			return true
		}

		return false
	}

	// Check project selector if specified
	if template.Spec.Target.ProjectSelector != nil && cluster.Labels[kubermaticv1.ProjectIDLabelKey] != "" {
		selector, err := metav1.LabelSelectorAsSelector(template.Spec.Target.ProjectSelector)
		if err != nil {
			return false
		}

		projectLabels := map[string]string{kubermaticv1.ProjectIDLabelKey: cluster.Labels[kubermaticv1.ProjectIDLabelKey]}
		if !selector.Matches(labels(projectLabels)) {
			return false
		}
	}

	// Check cluster selector if specified
	if template.Spec.Target.ClusterSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(template.Spec.Target.ClusterSelector)
		if err != nil {
			return false
		}

		if !selector.Matches(labels(cluster.Labels)) {
			return false
		}
	}

	return true
}

// labels is a helper that converts a map[string]string to a labels.Labels type
type labels map[string]string

func (l labels) Get(key string) string {
	return l[key]
}

func (l labels) Has(key string) bool {
	_, exists := l[key]
	return exists
}

func enqueueClusters(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request
		policyTemplate := a.(*kubermaticv1.PolicyTemplate)

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
			if obj.GetDeletionTimestamp() != nil {
				return false
			}

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
