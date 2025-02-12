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

package kyvernopolicycontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kyverno-policy-controller"

	// Finalizer for cleaning up policies in user clusters
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-kyverno-policy"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	log            *zap.SugaredLogger
	workerName     string
	seedClient     ctrlruntimeclient.Client
	clientProvider UserClusterClientProvider
	versions       kubermatic.Versions
	recorder       record.EventRecorder
}

// TODO: Replaced after kyverno-policy-syncer controller has been added to user cluster controller manager
func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	clientProvider UserClusterClientProvider,
	versions kubermatic.Versions,
) error {
	reconciler := &Reconciler{
		log:            log.Named(ControllerName),
		workerName:     workerName,
		seedClient:     mgr.GetClient(),
		clientProvider: clientProvider,
		versions:       versions,
		recorder:       mgr.GetEventRecorderFor(ControllerName),
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.PolicyBinding{}).
		WithEventFilter(predicate.ByName(workerName)).
		Watches(
			&kubermaticv1.Cluster{},
			handler.EnqueueRequestsFromMapFunc(reconciler.enqueueAllPolicyBindings),
			builder.WithPredicates(predicate.ByName(workerName)),
		).
		Build(reconciler)

	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	return nil
}

// enqueueAllPolicyBindings returns a function that enqueues all PolicyBindings that might target the cluster.
func (r *Reconciler) enqueueAllPolicyBindings(ctx context.Context, obj ctrlruntimeclient.Object) []reconcile.Request {
	var requests []reconcile.Request

	cluster, ok := obj.(*kubermaticv1.Cluster)
	if !ok {
		err := fmt.Errorf("object was not a cluster but a %T", obj)
		r.log.Error(err)
		utilruntime.HandleError(err)
		return nil
	}

	// List all PolicyBindings
	policyBindingList := &kubermaticv1.PolicyBindingList{}
	if err := r.seedClient.List(ctx, policyBindingList); err != nil {
		r.log.Error(err)
		utilruntime.HandleError(fmt.Errorf("failed to list policy bindings: %w", err))
		return nil
	}

	// Add requests for all PolicyBindings that might target this cluster
	for _, pb := range policyBindingList.Items {
		// For global scope, check project selectors
		if pb.Spec.Scope == "global" {
			if shouldProcessGlobalBinding(&pb, cluster) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: pb.Name,
					},
				})
			}
			continue
		}

		// For project scope, check cluster selectors
		if pb.Spec.Scope == "project" {
			if shouldProcessProjectBinding(&pb, cluster) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: pb.Name,
					},
				})
			}
		}
	}

	return requests
}

// shouldProcessGlobalBinding checks if a global binding should be processed for a cluster
func shouldProcessGlobalBinding(binding *kubermaticv1.PolicyBinding, cluster *kubermaticv1.Cluster) bool {
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

// shouldProcessProjectBinding checks if a project binding should be processed for a cluster
func shouldProcessProjectBinding(binding *kubermaticv1.PolicyBinding, cluster *kubermaticv1.Cluster) bool {
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

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("policyBinding", request.Name)
	log.Debug("Reconciling PolicyBinding")

	// Get PolicyBinding
	policyBinding := &kubermaticv1.PolicyBinding{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, policyBinding); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get policy binding: %w", err)
	}

	// Handle deletion
	if !policyBinding.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, policyBinding)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(policyBinding, cleanupFinalizer) {
		controllerutil.AddFinalizer(policyBinding, cleanupFinalizer)
		if err := r.seedClient.Update(ctx, policyBinding); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
		return reconcile.Result{}, nil
	}

	// Get PolicyTemplate
	policyTemplate := &kubermaticv1.PolicyTemplate{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{
		Name: policyBinding.Spec.PolicyTemplateRef.Name,
	}, policyTemplate); err != nil {
		if apierrors.IsNotFound(err) {
			r.recorder.Event(policyBinding, corev1.EventTypeWarning, kubermaticv1.PolicyTemplateNotFound, "PolicyTemplate not found")
			return r.updatePolicyBindingCondition(ctx, policyBinding, kubermaticv1.PolicyTemplateNotFound, "PolicyTemplate not found", false)
		}
		return reconcile.Result{}, fmt.Errorf("failed to get policy template: %w", err)
	}

	// Get list of target clusters based on selector
	targetClusters, err := r.getTargetClusters(ctx, policyBinding)
	if err != nil {
		r.recorder.Event(policyBinding, corev1.EventTypeWarning, kubermaticv1.PolicyApplicationFailed, fmt.Sprintf("Failed to get target clusters: %v", err))
		return r.updatePolicyBindingCondition(ctx, policyBinding, kubermaticv1.PolicyApplicationFailed, fmt.Sprintf("Failed to get target clusters: %v", err), false)
	}

	// Process each target cluster
	var processingErrors []string
	for _, cluster := range targetClusters {
		if err := r.reconcileClusterPolicy(ctx, cluster, policyTemplate, policyBinding); err != nil {
			log.Errorw("Failed to reconcile cluster policy", "cluster", cluster.Name, "error", err)
			processingErrors = append(processingErrors, fmt.Sprintf("cluster %s: %v", cluster.Name, err))
		}
	}

	if len(processingErrors) > 0 {
		errorMsg := fmt.Sprintf("Failed to apply policy to some clusters: %v", processingErrors)
		r.recorder.Event(policyBinding, corev1.EventTypeWarning, kubermaticv1.PolicyApplicationFailed, errorMsg)
		return r.updatePolicyBindingCondition(ctx, policyBinding, kubermaticv1.PolicyApplicationFailed, errorMsg, false)
	}

	r.recorder.Event(policyBinding, corev1.EventTypeNormal, kubermaticv1.PolicyAppliedSuccessfully, "Successfully applied policy to all target clusters")
	return r.updatePolicyBindingCondition(ctx, policyBinding, kubermaticv1.PolicyAppliedSuccessfully, "Successfully applied policy to all target clusters", true)
}

func (r *Reconciler) handleDeletion(ctx context.Context, policyBinding *kubermaticv1.PolicyBinding) (reconcile.Result, error) {
	if !controllerutil.ContainsFinalizer(policyBinding, cleanupFinalizer) {
		return reconcile.Result{}, nil
	}

	// Get list of target clusters
	targetClusters, err := r.getTargetClusters(ctx, policyBinding)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get target clusters for cleanup: %w", err)
	}

	// Remove policies from all target clusters
	var cleanupErrors []string
	for _, cluster := range targetClusters {
		if err := r.cleanupClusterPolicy(ctx, cluster, policyBinding); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("cluster %s: %v", cluster.Name, err))
		}
	}

	if len(cleanupErrors) > 0 {
		return reconcile.Result{}, fmt.Errorf("failed to cleanup policies: %v", cleanupErrors)
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(policyBinding, cleanupFinalizer)
	if err := r.seedClient.Update(ctx, policyBinding); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) cleanupClusterPolicy(ctx context.Context, cluster *kubermaticv1.Cluster, binding *kubermaticv1.PolicyBinding) error {
	userClusterClient, err := r.clientProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	policy := &kyvernov1.ClusterPolicy{}
	err = userClusterClient.Get(ctx, types.NamespacedName{
		Name: fmt.Sprintf("%s-%s", binding.Spec.PolicyTemplateRef.Name, binding.Name),
	}, policy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get cluster policy: %w", err)
	}

	if err := userClusterClient.Delete(ctx, policy); err != nil {
		return fmt.Errorf("failed to delete cluster policy: %w", err)
	}

	return nil
}

func (r *Reconciler) updatePolicyBindingCondition(ctx context.Context, policyBinding *kubermaticv1.PolicyBinding, reason, message string, success bool) (reconcile.Result, error) {
	// Update Ready condition
	readyCondition := metav1.Condition{
		Type:               kubermaticv1.PolicyReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: policyBinding.Generation,
		LastTransitionTime: metav1.Now(),
	}

	if !success {
		readyCondition.Status = metav1.ConditionFalse
	}

	// Update Enforced condition
	enforcedCondition := metav1.Condition{
		Type:               kubermaticv1.PolicyEnforcedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: policyBinding.Generation,
		LastTransitionTime: metav1.Now(),
	}

	if !success {
		enforcedCondition.Status = metav1.ConditionFalse
	}

	// Update conditions
	meta.SetStatusCondition(&policyBinding.Status.Conditions, readyCondition)
	meta.SetStatusCondition(&policyBinding.Status.Conditions, enforcedCondition)

	// Update observed generation
	policyBinding.Status.ObservedGeneration = policyBinding.Generation

	if err := r.seedClient.Status().Update(ctx, policyBinding); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to update policy binding status: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) getTargetClusters(ctx context.Context, policyBinding *kubermaticv1.PolicyBinding) ([]*kubermaticv1.Cluster, error) {
	var clusters []*kubermaticv1.Cluster
	clusterList := &kubermaticv1.ClusterList{}

	if err := r.seedClient.List(ctx, clusterList); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	switch policyBinding.Spec.Scope {
	case kubermaticv1.PolicyBindingScopeGlobal:
		// For global scope, we first need to get the target projects
		targetProjectIDs, err := r.getTargetProjectIDs(ctx, policyBinding.Spec.Target.Projects)
		if err != nil {
			return nil, fmt.Errorf("failed to get target project IDs: %w", err)
		}

		// Then filter clusters that belong to these projects
		for i := range clusterList.Items {
			cluster := &clusterList.Items[i]
			projectID := cluster.Labels["project-id"]
			if projectID == "" {
				continue
			}

			for _, targetProjectID := range targetProjectIDs {
				if projectID == targetProjectID {
					clusters = append(clusters, cluster)
					break
				}
			}
		}

	case kubermaticv1.PolicyBindingScopeProject:
		// Get the PolicyTemplate to get the project ID
		policyTemplate := &kubermaticv1.PolicyTemplate{}
		if err := r.seedClient.Get(ctx, types.NamespacedName{
			Name: policyBinding.Spec.PolicyTemplateRef.Name,
		}, policyTemplate); err != nil {
			return nil, fmt.Errorf("failed to get policy template: %w", err)
		}

		if policyTemplate.Spec.ProjectID == "" {
			return nil, fmt.Errorf("project-scoped policy template must have a ProjectID")
		}

		// First, get all clusters from the specified project
		var projectClusters []*kubermaticv1.Cluster
		for i := range clusterList.Items {
			cluster := &clusterList.Items[i]
			if cluster.Labels["project-id"] == policyTemplate.Spec.ProjectID {
				projectClusters = append(projectClusters, cluster)
			}
		}

		// Then apply the cluster selector
		clusters = r.filterClustersBySelector(projectClusters, policyBinding.Spec.Target.Clusters)
	default:
		return nil, fmt.Errorf("invalid scope: %s", policyBinding.Spec.Scope)
	}

	return clusters, nil
}

// getTargetProjectIDs returns a list of project IDs based on the project selector
func (r *Reconciler) getTargetProjectIDs(ctx context.Context, projectSelector kubermaticv1.ResourceSelector) ([]string, error) {
	var projectList *kubermaticv1.ProjectList
	if projectSelector.SelectAll || projectSelector.LabelSelector != nil {
		projectList = &kubermaticv1.ProjectList{}
		if err := r.seedClient.List(ctx, projectList); err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}
	}

	// Handle SelectAll
	if projectSelector.SelectAll {
		projectIDs := make([]string, len(projectList.Items))
		for i, project := range projectList.Items {
			projectIDs[i] = project.Name
		}
		return projectIDs, nil
	}

	// Handle name-based selection
	if len(projectSelector.Name) > 0 {
		return projectSelector.Name, nil
	}

	// Handle label selector
	if projectSelector.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(projectSelector.LabelSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to parse label selector: %w", err)
		}

		var projectIDs []string
		for _, project := range projectList.Items {
			if selector.Matches(labels.Set(project.Labels)) {
				projectIDs = append(projectIDs, project.Name)
			}
		}
		return projectIDs, nil
	}

	return nil, fmt.Errorf("invalid project selector: must specify either selectAll, names, or labelSelector")
}

// filterClustersBySelector filters clusters based on the cluster selector
func (r *Reconciler) filterClustersBySelector(clusters []*kubermaticv1.Cluster, clusterSelector kubermaticv1.ResourceSelector) []*kubermaticv1.Cluster {
	if clusterSelector.SelectAll {
		return clusters
	}

	var filteredClusters []*kubermaticv1.Cluster

	// Filter by names (project IDs) if specified
	if len(clusterSelector.Name) > 0 {
		nameSet := make(map[string]struct{}, len(clusterSelector.Name))
		for _, name := range clusterSelector.Name {
			nameSet[name] = struct{}{}
		}

		for _, cluster := range clusters {
			if _, exists := nameSet[cluster.Name]; exists {
				filteredClusters = append(filteredClusters, cluster)
			}
		}
		return filteredClusters
	}

	// Filter by label selector if specified
	if clusterSelector.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(clusterSelector.LabelSelector)
		if err != nil {
			r.log.Error("failed to parse label selector", "error", err)
			return nil
		}

		for _, cluster := range clusters {
			if selector.Matches(labels.Set(cluster.Labels)) {
				filteredClusters = append(filteredClusters, cluster)
			}
		}
		return filteredClusters
	}

	return nil
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

func (r *Reconciler) reconcileClusterPolicy(ctx context.Context, cluster *kubermaticv1.Cluster, policyTemplate *kubermaticv1.PolicyTemplate, policyBinding *kubermaticv1.PolicyBinding) error {
	// Get client for user cluster
	userClusterClient, err := r.clientProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	// Create reconciler factory
	factories := []reconciling.NamedKyvernoClusterPolicyReconcilerFactory{
		kyvernoPolicyReconcilerFactory(policyTemplate, policyBinding),
	}

	// Reconcile the Kyverno ClusterPolicy
	if err := reconciling.ReconcileKyvernoClusterPolicys(ctx, factories, "", userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile cluster policy: %w", err)
	}

	return nil
}
