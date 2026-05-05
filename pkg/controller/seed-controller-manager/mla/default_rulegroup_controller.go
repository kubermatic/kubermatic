/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package mla

// defaultRuleGroupController creates a predefined RuleGroup the first time
// user-cluster monitoring is enabled on a Cluster.
//
// Design intent: this controller is intentionally NOT a continuous reconciler.
// It creates the RuleGroup once and then leaves it completely alone.
// End-users are free to edit, extend, or delete the object — the controller
// will never overwrite their changes or restore a deleted object.
//
// Contrast with rulegroup_sync_controller, which continuously syncs template
// RuleGroups from the MLA namespace into every cluster namespace.
//
// To add or change the default alert rules, edit default_rulegroup_rules.yaml
// in this package — no Go code changes required.

import (
	"context"
	_ "embed"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// defaultRuleGroupName is the name of the RuleGroup seeded by this controller.
const defaultRuleGroupName = "kkp-default-monitoring-rules"

//go:embed default_rulegroup_rules.yaml
var defaultRuleGroupRulesYAML []byte

// ruleGroupData mirrors the flat single-group format used in default_rulegroup_rules.yaml:
//
//	name: my-group
//	rules:
//	  - alert: ...
//
// Only the fields we need to manipulate are declared; everything else
// round-trips through yaml.v3 as-is.
type ruleGroupData struct {
	Name     string      `yaml:"name"`
	Interval string      `yaml:"interval,omitempty"`
	Rules    []alertRule `yaml:"rules"`
}

type alertRule struct {
	Alert       string            `yaml:"alert,omitempty"`
	Record      string            `yaml:"record,omitempty"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// buildRuleData parses the embedded YAML and injects seed_cluster: <clusterID>
// into the labels of every rule, then re-serialises to bytes.
func buildRuleData(clusterID string) ([]byte, error) {
	var rg ruleGroupData
	if err := yaml.Unmarshal(defaultRuleGroupRulesYAML, &rg); err != nil {
		return nil, fmt.Errorf("failed to parse default rule group YAML: %w", err)
	}

	for i := range rg.Rules {
		rule := &rg.Rules[i]
		if rule.Labels == nil {
			rule.Labels = make(map[string]string)
		}
		rule.Labels["seed_cluster"] = clusterID
	}

	data, err := yaml.Marshal(rg)
	if err != nil {
		return nil, fmt.Errorf("failed to serialise rule group YAML: %w", err)
	}
	return data, nil
}

// -----------------------------------------------------------------------
// Reconciler
// -----------------------------------------------------------------------

type defaultRuleGroupReconciler struct {
	ctrlruntimeclient.Client
	log        *zap.SugaredLogger
	workerName string
	recorder   events.EventRecorder
	versions   kubermatic.Versions
}

func newDefaultRuleGroupReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	_ *defaultRuleGroupController, // reserved for future use; kept for consistency with other sub-controllers
) error {
	log = log.Named(ControllerName)
	subname := "default-rulegroup"

	reconciler := &defaultRuleGroupReconciler{
		Client:     mgr.GetClient(),
		log:        log.Named(subname),
		workerName: workerName,
		recorder:   mgr.GetEventRecorder(controllerName(subname)),
		versions:   versions,
	}

	// Only trigger on transitions that matter for first-time RuleGroup creation.
	clusterPredicate := predicate.Funcs{
		// Create: fire for newly created clusters that already have monitoring on.
		// NOTE: Status.NamespaceName is typically empty at this point; the reconciler
		// will return early, but the namespace-ready update below will re-trigger it.
		CreateFunc: func(e event.CreateEvent) bool {
			c := e.Object.(*kubermaticv1.Cluster)
			return c.Spec.MLA != nil && c.Spec.MLA.MonitoringEnabled
		},
		// Update: fire on two distinct transitions:
		//   1. MonitoringEnabled false → true  (user enables monitoring on existing cluster)
		//   2. NamespaceName "" → non-empty while monitoring is already on
		//      (covers clusters created with monitoring pre-enabled: the CREATE event fires
		//       but the namespace is not ready yet, so we must re-trigger once it appears)
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldCluster := e.ObjectOld.(*kubermaticv1.Cluster)
			newCluster := e.ObjectNew.(*kubermaticv1.Cluster)
			wasEnabled := oldCluster.Spec.MLA != nil && oldCluster.Spec.MLA.MonitoringEnabled
			nowEnabled := newCluster.Spec.MLA != nil && newCluster.Spec.MLA.MonitoringEnabled
			namespaceJustReady := oldCluster.Status.NamespaceName == "" && newCluster.Status.NamespaceName != ""
			return (!wasEnabled && nowEnabled) || (nowEnabled && namespaceJustReady)
		},
		// Delete / Generic: nothing to do.
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName(subname)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}, builder.WithPredicates(clusterPredicate)).
		Build(reconciler)

	return err
}

func (r *defaultRuleGroupReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	// Guard: skip clusters that are being deleted.
	if !cluster.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}
	// Guard: monitoring must be enabled and the cluster namespace must exist.
	if cluster.Spec.MLA == nil || !cluster.Spec.MLA.MonitoringEnabled {
		return reconcile.Result{}, nil
	}
	if cluster.Status.NamespaceName == "" {
		log.Debug("Cluster namespace not yet available, skipping")
		return reconcile.Result{}, nil
	}

	if err := r.createIfAbsent(ctx, log, cluster); err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "DefaultRuleGroupError", "Seeding", err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// createIfAbsent creates the default RuleGroup only when it does not yet exist.
// If the object is already present (whether the original or a user-modified copy),
// this function does nothing — preserving any customizations.
func (r *defaultRuleGroupReconciler) createIfAbsent(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	key := types.NamespacedName{
		Name:      defaultRuleGroupName,
		Namespace: cluster.Status.NamespaceName,
	}

	existing := &kubermaticv1.RuleGroup{}
	err := r.Get(ctx, key, existing)
	if err == nil {
		// Already exists — leave it alone.
		log.Debugw("Default RuleGroup already present, skipping", "rulegroup", key)
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing default RuleGroup: %w", err)
	}

	// Not found — build the rule data with the cluster ID injected, then create.
	ruleData, err := buildRuleData(cluster.Name)
	if err != nil {
		return fmt.Errorf("failed to build rule data for cluster %s: %w", cluster.Name, err)
	}

	rg := &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultRuleGroupName,
			Namespace: cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: kubermaticv1.RuleGroupTypeMetrics,
			Cluster: corev1.ObjectReference{
				Name: cluster.Name,
			},
			Data: ruleData,
		},
	}

	if err := r.Create(ctx, rg); err != nil {
		return fmt.Errorf("failed to create default RuleGroup: %w", err)
	}

	log.Infow("Created a starter Alerting RuleGroup", "cluster", cluster.Name, "rulegroup", key)
	r.recorder.Eventf(cluster, nil, corev1.EventTypeNormal, "DefaultRuleGroupCreated", "Seeding", "Created a starter Alerting RuleGroup %s", defaultRuleGroupName)
	return nil
}

// defaultRuleGroupController is a placeholder kept so the Add() call signature
// matches the other sub-controllers. It also implements the cleaner interface
// so that the cleanup controller can remove any seeded RuleGroups when MLA is
// globally disabled.
type defaultRuleGroupController struct {
	ctrlruntimeclient.Client
}

func newDefaultRuleGroupController(client ctrlruntimeclient.Client, _ *zap.SugaredLogger) *defaultRuleGroupController {
	return &defaultRuleGroupController{Client: client}
}

// CleanUp deletes all RuleGroups named defaultRuleGroupName across all namespaces.
// It is called by the cleanup controller when MLA is disabled at the operator level.
func (c *defaultRuleGroupController) CleanUp(ctx context.Context) error {
	ruleGroupList := &kubermaticv1.RuleGroupList{}
	if err := c.List(ctx, ruleGroupList); err != nil {
		return fmt.Errorf("failed to list RuleGroups during cleanup: %w", err)
	}
	for _, rg := range ruleGroupList.Items {
		if rg.Name != defaultRuleGroupName {
			continue
		}
		if err := c.Delete(ctx, &rg); ctrlruntimeclient.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to delete default RuleGroup %s/%s: %w", rg.Namespace, rg.Name, err)
		}
	}
	return nil
}
