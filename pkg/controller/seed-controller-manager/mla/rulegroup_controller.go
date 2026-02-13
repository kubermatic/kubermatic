/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
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
	ruleGroupFinalizer             = "kubermatic.k8c.io/rule-group"
	MetricsRuleGroupConfigEndpoint = "/api/v1/rules"
	LogRuleGroupConfigEndpoint     = "/loki/api/v1/rules"
	RuleGroupTenantHeaderName      = "X-Scope-OrgID"
	defaultNamespace               = "/default"
)

type ruleGroupReconciler struct {
	ctrlruntimeclient.Client
	log                 *zap.SugaredLogger
	workerName          string
	recorder            events.EventRecorder
	versions            kubermatic.Versions
	ruleGroupController *ruleGroupController
}

func newRuleGroupReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	ruleGroupController *ruleGroupController,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()
	subname := "rulegroup"

	reconciler := &ruleGroupReconciler{
		Client:              client,
		log:                 log.Named(subname),
		workerName:          workerName,
		recorder:            mgr.GetEventRecorder(controllerName(subname)),
		versions:            versions,
		ruleGroupController: ruleGroupController,
	}

	ruleGroupPredicate := predicateutil.Factory(func(o ctrlruntimeclient.Object) bool {
		// We don't want to enqueue RuleGroup objects in mla namespace since those are regarded as rulegroup template,
		// and will be rollout to cluster namespaces in rulegroup_sync_controller.
		if o.GetNamespace() == ruleGroupController.mlaNamespace {
			return false
		}
		// If the cluster name in empty, we just ignore the ruleGroup.
		ruleGroup := o.(*kubermaticv1.RuleGroup)
		return ruleGroup.Spec.Cluster.Name != ""
	})

	enqueueRuleGroupsForCluster := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object ctrlruntimeclient.Object) []reconcile.Request {
		cluster := object.(*kubermaticv1.Cluster)
		if cluster.Status.NamespaceName == "" {
			return nil
		}
		ruleGroupList := &kubermaticv1.RuleGroupList{}
		if err := client.List(ctx, ruleGroupList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
			log.Errorw("failed to list ruleGroups for cluster", zap.Error(err), "cluster", cluster.Name)
			utilruntime.HandleError(fmt.Errorf("failed to list ruleGroups: %w", err))
		}
		var requests []reconcile.Request
		for _, ruleGroup := range ruleGroupList.Items {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      ruleGroup.Name,
					Namespace: ruleGroup.Namespace,
				},
			})
		}
		return requests
	})

	clusterPredicate := predicate.Funcs{
		// For Update event, only trigger reconciliation when MonitoringEnabled or LoggingEnabled changes.
		UpdateFunc: func(event event.UpdateEvent) bool {
			oldCluster := event.ObjectOld.(*kubermaticv1.Cluster)
			newCluster := event.ObjectNew.(*kubermaticv1.Cluster)
			oldMonitoringEnabled := oldCluster.Spec.MLA != nil && oldCluster.Spec.MLA.MonitoringEnabled
			newMonitoringEnabled := newCluster.Spec.MLA != nil && newCluster.Spec.MLA.MonitoringEnabled
			oldLoggingEnabled := oldCluster.Spec.MLA != nil && oldCluster.Spec.MLA.LoggingEnabled
			newLoggingEnabled := newCluster.Spec.MLA != nil && newCluster.Spec.MLA.LoggingEnabled
			return (oldMonitoringEnabled != newMonitoringEnabled) || (oldLoggingEnabled != newLoggingEnabled)
		},
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName(subname)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.RuleGroup{}, builder.WithPredicates(ruleGroupPredicate)).
		Watches(&kubermaticv1.Cluster{}, enqueueRuleGroupsForCluster, builder.WithPredicates(clusterPredicate)).
		Build(reconciler)

	return err
}

func (r *ruleGroupReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	ruleGroup := &kubermaticv1.RuleGroup{}
	if err := r.Get(ctx, request.NamespacedName, ruleGroup); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, types.NamespacedName{Name: ruleGroup.Spec.Cluster.Name}, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			// If cluster object is already gone, but rule group is still found in the namespace, we need to handle deletion
			// and remove finalizer for it so that it will not block the deletion of cluster namespace.
			requestURL, err := r.ruleGroupController.getRequestURL(ruleGroup)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to get request URL: %w", err)
			}
			if err := r.ruleGroupController.handleDeletion(ctx, ruleGroup, requestURL); err != nil {
				return reconcile.Result{}, fmt.Errorf("failed to delete ruleGroup: %w", err)
			}
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMLAControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.ruleGroupController.reconcile(ctx, cluster, ruleGroup)
		},
	)

	if result == nil || err != nil {
		result = &reconcile.Result{}
	}

	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return *result, err
}

type ruleGroupController struct {
	ctrlruntimeclient.Client
	httpClient *http.Client

	log            *zap.SugaredLogger
	cortexRulerURL string
	lokiRulerURL   string
	mlaNamespace   string
}

func newRuleGroupController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	httpClient *http.Client,
	cortexRulerURL string,
	lokiRulerURL string,
	mlaNamespace string,
) *ruleGroupController {
	return &ruleGroupController{
		Client:         client,
		httpClient:     httpClient,
		log:            log,
		cortexRulerURL: cortexRulerURL,
		lokiRulerURL:   lokiRulerURL,
		mlaNamespace:   mlaNamespace,
	}
}

func (r *ruleGroupController) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, ruleGroup *kubermaticv1.RuleGroup) (*reconcile.Result, error) {
	requestURL, err := r.getRequestURL(ruleGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to get request URL: %w", err)
	}

	if !ruleGroup.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, ruleGroup, requestURL); err != nil {
			return nil, fmt.Errorf("failed to delete ruleGroup: %w", err)
		}
		return nil, nil
	}

	mlaEnabled := cluster.Spec.MLA != nil && (cluster.Spec.MLA.MonitoringEnabled || cluster.Spec.MLA.LoggingEnabled)
	if !cluster.DeletionTimestamp.IsZero() || !mlaEnabled {
		// If this cluster is being deleted, or MLA is disabled for this cluster, we just delete this `RuleGroup`,
		// and the clean up of `RuleGroup` will be triggered in the next reconciliation loop.
		if err := r.Delete(ctx, ruleGroup); err != nil {
			return nil, ctrlruntimeclient.IgnoreNotFound(err)
		}
		return nil, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r, ruleGroup, ruleGroupFinalizer); err != nil {
		return nil, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.ensureRuleGroup(ctx, ruleGroup, requestURL); err != nil {
		return nil, fmt.Errorf("failed to create rule group: %w", err)
	}
	return nil, nil
}

func (r *ruleGroupController) CleanUp(ctx context.Context) error {
	ruleGroupList := &kubermaticv1.RuleGroupList{}
	if err := r.List(ctx, ruleGroupList); err != nil {
		return err
	}
	for _, ruleGroup := range ruleGroupList.Items {
		requestURL, err := r.getRequestURL(&ruleGroup)
		if err != nil {
			return fmt.Errorf("failed to get request URL: %w", err)
		}
		if err := r.handleDeletion(ctx, &ruleGroup, requestURL); err != nil {
			return fmt.Errorf("failed to handle rule group cleanup for RuleGroup %s/%s: %w", ruleGroup.Namespace, ruleGroup.Name, err)
		}
		if err := r.Delete(ctx, &ruleGroup); err != nil {
			return err
		}
	}
	return nil
}

func (r *ruleGroupController) getRequestURL(ruleGroup *kubermaticv1.RuleGroup) (string, error) {
	if ruleGroup.Spec.RuleGroupType == kubermaticv1.RuleGroupTypeMetrics {
		return fmt.Sprintf("%s%s%s", r.cortexRulerURL, MetricsRuleGroupConfigEndpoint, defaultNamespace), nil
	}
	if ruleGroup.Spec.RuleGroupType == kubermaticv1.RuleGroupTypeLogs {
		return fmt.Sprintf("%s%s%s", r.lokiRulerURL, LogRuleGroupConfigEndpoint, defaultNamespace), nil
	}
	return "", fmt.Errorf("unknown rule group type: %s", ruleGroup.Spec.RuleGroupType)
}

func (r *ruleGroupController) handleDeletion(ctx context.Context, ruleGroup *kubermaticv1.RuleGroup, requestURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		fmt.Sprintf("%s/%s", requestURL, ruleGroup.Name), nil)
	if err != nil {
		return err
	}
	req.Header.Add(RuleGroupTenantHeaderName, ruleGroup.Spec.Cluster.Name)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	// https://cortexmetrics.io/docs/api/#delete-rule-group
	if resp.StatusCode != http.StatusAccepted {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	return kubernetes.TryRemoveFinalizer(ctx, r, ruleGroup, ruleGroupFinalizer)
}

func (r *ruleGroupController) ensureRuleGroup(ctx context.Context, ruleGroup *kubermaticv1.RuleGroup, requestURL string) error {
	currentRuleGroup, err := r.getCurrentRuleGroup(ctx, ruleGroup, requestURL)
	if err != nil {
		return err
	}
	expectedRuleGroup := map[string]interface{}{}
	decoder := yaml.NewDecoder(bytes.NewReader(ruleGroup.Spec.Data))
	if err := decoder.Decode(&expectedRuleGroup); err != nil {
		return fmt.Errorf("unable to unmarshal expected rule group: %w", err)
	}
	if reflect.DeepEqual(currentRuleGroup, expectedRuleGroup) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		requestURL,
		bytes.NewBuffer(ruleGroup.Spec.Data))
	if err != nil {
		return err
	}
	req.Header.Add(RuleGroupTenantHeaderName, ruleGroup.Spec.Cluster.Name)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	// https://cortexmetrics.io/docs/api/#set-rule-group
	if resp.StatusCode != http.StatusAccepted {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (r *ruleGroupController) getCurrentRuleGroup(ctx context.Context, ruleGroup *kubermaticv1.RuleGroup, requestURL string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s", requestURL, ruleGroup.Name), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(RuleGroupTenantHeaderName, ruleGroup.Spec.Cluster.Name)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// https://cortexmetrics.io/docs/api/#get-rule-group
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}
	config := map[string]interface{}{}
	decoder := yaml.NewDecoder(resp.Body)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("unable to decode response body: %w", err)
	}
	return config, nil
}
