/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package kyvernopolicysyncer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	controllerName = "kyverno-policy-syncer"
)

type reconciler struct {
	log             *zap.SugaredLogger
	seedClient      ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	recorder        record.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
	cluster         string
	namespace       string
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, clusterName string, namespace string, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		seedClient:      seedMgr.GetClient(),
		userClient:      userMgr.GetClient(),
		recorder:        userMgr.GetEventRecorderFor(controllerName),
		clusterIsPaused: clusterIsPaused,
		cluster:         clusterName,
		namespace:       namespace,
	}

	// Build controller with PolicyBinding as primary resource and watches for ClusterPolicy and PolicyTemplate
	err := builder.ControllerManagedBy(userMgr).
		Named(controllerName).
		For(&kubermaticv1.PolicyBinding{}).
		Watches(
			&kubermaticv1.PolicyTemplate{},
			handler.EnqueueRequestsFromMapFunc(r.policyTemplateToBinding),
		).
		Watches(
			&kyvernov1.ClusterPolicy{},
			handler.EnqueueRequestsFromMapFunc(r.clusterPolicyToBinding),
		).
		Complete(r)

	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("policybinding", request.Name)
	log.Debug("Reconciling PolicyBinding")

	// Check if cluster is paused
	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	// Get PolicyBinding from seed cluster
	binding := &kubermaticv1.PolicyBinding{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{
		Name:      request.Name,
		Namespace: r.namespace,
	}, binding); err != nil {
		if apierrors.IsNotFound(err) {
			// PolicyBinding was deleted, clean up any associated ClusterPolicies
			return r.cleanupClusterPolicy(ctx, request.Name, log)
		}
		return reconcile.Result{}, fmt.Errorf("failed to get PolicyBinding: %w", err)
	}

	// Get the corresponding template from the same namespace
	template := &kubermaticv1.PolicyTemplate{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{
		Name:      binding.Spec.PolicyTemplateRef.Name,
		Namespace: r.namespace,
	}, template); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("PolicyTemplate not found, skipping reconciliation")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get PolicyTemplate: %w", err)
	}

	// Check if this binding applies to our cluster
	if !r.shouldApplyToCluster(binding, template) {
		// Clean up any existing policies if they exist
		return r.cleanupClusterPolicy(ctx, binding.Name, log)
	}

	// Reconcile the ClusterPolicy
	if err := r.reconcileClusterPolicy(ctx, binding, template); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcile ClusterPolicy: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) reconcileClusterPolicy(ctx context.Context, binding *kubermaticv1.PolicyBinding, template *kubermaticv1.PolicyTemplate) error {
	// Create or update the ClusterPolicy using the reconciler factory
	factories := []reconciling.NamedKyvernoClusterPolicyReconcilerFactory{
		kyvernoPolicyReconcilerFactory(template, binding),
	}

	// Reconcile the Kyverno ClusterPolicy
	if err := reconciling.ReconcileKyvernoClusterPolicys(ctx, factories, "", r.userClient); err != nil {
		return fmt.Errorf("failed to reconcile cluster policy: %w", err)
	}

	return nil
}

func (r *reconciler) cleanupClusterPolicy(ctx context.Context, bindingName string, log *zap.SugaredLogger) (reconcile.Result, error) {
	policy := &kyvernov1.ClusterPolicy{}
	err := r.userClient.Get(ctx, types.NamespacedName{
		Name: fmt.Sprintf("%s-%s", bindingName, r.cluster),
	}, policy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get ClusterPolicy: %w", err)
	}

	if err := r.userClient.Delete(ctx, policy); err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, fmt.Errorf("failed to delete ClusterPolicy: %w", err)
	}

	log.Info("Successfully deleted ClusterPolicy")
	return reconcile.Result{}, nil
}

func (r *reconciler) shouldApplyToCluster(binding *kubermaticv1.PolicyBinding, template *kubermaticv1.PolicyTemplate) bool {
	// Fetch the cluster object
	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(context.Background(), types.NamespacedName{Name: r.cluster}, cluster); err != nil {
		r.log.Errorw("Failed to get cluster", "error", err)
		return false
	}

	// First check if the binding is valid for this cluster based on scope
	switch binding.Spec.Scope {
	case kubermaticv1.PolicyBindingScopeGlobal:
		if !r.shouldProcessGlobalBinding(binding, cluster) {
			return false
		}
	case kubermaticv1.PolicyBindingScopeProject:
		// For project scope, first verify the cluster belongs to the template's project
		if template.Spec.ProjectID != cluster.Labels["project-id"] {
			return false
		}
		// Then check the cluster selectors
		if !r.shouldProcessProjectBinding(binding, cluster) {
			return false
		}
	default:
		return false
	}

	return true
}

func (r *reconciler) shouldProcessGlobalBinding(binding *kubermaticv1.PolicyBinding, cluster *kubermaticv1.Cluster) bool {
	// For global bindings, we need to check project selectors
	if binding.Spec.Target.Projects.SelectAll {
		return true
	}

	// Check project name list
	for _, name := range binding.Spec.Target.Projects.Name {
		if name == cluster.Labels["project-id"] {
			return true
		}
	}

	// Check project label selector
	if binding.Spec.Target.Projects.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(binding.Spec.Target.Projects.LabelSelector)
		if err != nil {
			return false
		}
		return selector.Matches(labels.Set(cluster.Labels))
	}

	return false
}

func (r *reconciler) shouldProcessProjectBinding(binding *kubermaticv1.PolicyBinding, cluster *kubermaticv1.Cluster) bool {
	// For project bindings, we need to check cluster selectors
	if binding.Spec.Target.Clusters.SelectAll {
		return true
	}

	// Check cluster name list
	for _, name := range binding.Spec.Target.Clusters.Name {
		if name == cluster.Name {
			return true
		}
	}

	// Check cluster label selector
	if binding.Spec.Target.Clusters.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(binding.Spec.Target.Clusters.LabelSelector)
		if err != nil {
			return false
		}
		return selector.Matches(labels.Set(cluster.Labels))
	}

	return false
}

func kyvernoPolicyReconcilerFactory(policyTemplate *kubermaticv1.PolicyTemplate, policyBinding *kubermaticv1.PolicyBinding) reconciling.NamedKyvernoClusterPolicyReconcilerFactory {
	return func() (string, reconciling.KyvernoClusterPolicyReconciler) {
		name := fmt.Sprintf("%s-%s", policyTemplate.Name, policyBinding.Name)
		return name, func(policy *kyvernov1.ClusterPolicy) (*kyvernov1.ClusterPolicy, error) {
			// Set metadata
			policy.Name = name
			policy.Labels = map[string]string{
				"app.kubernetes.io/managed-by": "kubermatic",
				"policy-binding":               policyBinding.Name,
				"policy-template":              policyTemplate.Name,
			}

			// Set spec
			policy.Spec = policyTemplate.Spec.KyvernoPolicySpec
			return policy, nil
		}
	}
}

func (r *reconciler) clusterPolicyToBinding(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
	policy, ok := obj.(*kyvernov1.ClusterPolicy)
	if !ok {
		r.log.Error("Failed to convert object to ClusterPolicy")
		return nil
	}

	// Extract binding name from policy name
	// Policy name format: <template-name>-<binding-name>
	// TODO: change this to use the policy binding name directly
	bindingName := policy.Labels["policy-binding"]
	if bindingName == "" {
		return nil
	}

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name: bindingName,
			},
		},
	}
}

func (r *reconciler) policyTemplateToBinding(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
	template, ok := obj.(*kubermaticv1.PolicyTemplate)
	if !ok {
		r.log.Error("Failed to convert object to PolicyTemplate")
		return nil
	}

	// List all PolicyBindings that reference this template
	bindingList := &kubermaticv1.PolicyBindingList{}
	if err := r.seedClient.List(ctx, bindingList); err != nil {
		r.log.Error("Failed to list PolicyBindings", "error", err)
		return nil
	}

	var requests []reconcile.Request
	for _, binding := range bindingList.Items {
		if binding.Spec.PolicyTemplateRef.Name == template.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: binding.Name,
				},
			})
		}
	}

	return requests
}
