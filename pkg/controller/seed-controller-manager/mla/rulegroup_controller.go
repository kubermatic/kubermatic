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
	"io/ioutil"
	"net/http"
	"reflect"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ruleGroupFinalizer             = "kubermatic.io/rule-group"
	metricsRuleGroupConfigEndpoint = "/api/v1/rules"
	ruleGroupTenantHeaderName      = "X-Scope-OrgID"
	defaultNamespace               = "/default"
)

type ruleGroupReconciler struct {
	ctrlruntimeclient.Client
	log                 *zap.SugaredLogger
	workerName          string
	recorder            record.EventRecorder
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

	reconciler := &ruleGroupReconciler{
		Client:              client,
		log:                 log,
		workerName:          workerName,
		recorder:            mgr.GetEventRecorderFor(ControllerName),
		versions:            versions,
		ruleGroupController: ruleGroupController,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.RuleGroup{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch RuleGroup: %w", err)
	}

	enqueueRuleGroupsForCluster := handler.EnqueueRequestsFromMapFunc(func(object ctrlruntimeclient.Object) []reconcile.Request {
		cluster := object.(*kubermaticv1.Cluster)
		if cluster.Status.NamespaceName == "" {
			return nil
		}
		ruleGroupList := &kubermaticv1.RuleGroupList{}
		if err := client.List(context.Background(), ruleGroupList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
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
		client.List(context.Background(), ruleGroupList, ctrlruntimeclient.MatchingFields{})
		return requests
	})

	clusterPredicate := predicate.Funcs{
		// For Update event, only trigger reconciliation when MonitoringEnabled changes.
		UpdateFunc: func(event event.UpdateEvent) bool {
			oldCluster := event.ObjectOld.(*kubermaticv1.Cluster)
			newCluster := event.ObjectNew.(*kubermaticv1.Cluster)
			oldMonitoringEnabled := oldCluster.Spec.MLA != nil && oldCluster.Spec.MLA.MonitoringEnabled
			newMonitoringEnabled := newCluster.Spec.MLA != nil && newCluster.Spec.MLA.MonitoringEnabled
			return oldMonitoringEnabled != newMonitoringEnabled
		},
	}
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, enqueueRuleGroupsForCluster, clusterPredicate); err != nil {
		return fmt.Errorf("failed to watch Cluster: %w", err)
	}

	return nil
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
		return reconcile.Result{}, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMLAControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.ruleGroupController.reconcile(ctx, cluster, ruleGroup)
		},
	)
	if err != nil {
		r.log.Errorw("Failed to reconcile cluster", "cluster", cluster.Name, zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err

}

type ruleGroupController struct {
	ctrlruntimeclient.Client
	httpClient *http.Client

	log            *zap.SugaredLogger
	cortexRulerURL string
}

func newRuleGroupController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	httpClient *http.Client,
	cortexRulerURL string,
) *ruleGroupController {
	return &ruleGroupController{
		Client:         client,
		httpClient:     httpClient,
		log:            log,
		cortexRulerURL: cortexRulerURL,
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

	monitoringEnabled := cluster.Spec.MLA != nil && cluster.Spec.MLA.MonitoringEnabled
	if !cluster.DeletionTimestamp.IsZero() || !monitoringEnabled {
		// If this cluster is being deleted, or monitoring is disabled for this cluster, we just delete this `RuleGroup`,
		// and the clean up of `RuleGroup` will be triggered in the next reconciliation loop.
		if err := r.Delete(ctx, ruleGroup); err != nil {
			return nil, ctrlruntimeclient.IgnoreNotFound(err)
		}
		return nil, nil
	}

	if !kubernetes.HasFinalizer(ruleGroup, ruleGroupFinalizer) {
		kubernetes.AddFinalizer(ruleGroup, ruleGroupFinalizer)
		if err := r.Update(ctx, ruleGroup); err != nil {
			return nil, fmt.Errorf("updating finalizers for ruleGroup object: %w", err)
		}
	}

	if err := r.ensureRuleGroup(ruleGroup, requestURL); err != nil {
		return nil, fmt.Errorf("failed to create rule group: %w", err)
	}
	return nil, nil
}

func (r *ruleGroupController) cleanUp(ctx context.Context) error {
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
			return fmt.Errorf("failed to handle deletion: %w", err)
		}
		if err := r.Delete(ctx, &ruleGroup); err != nil {
			return err
		}
	}
	return nil
}

func (r *ruleGroupController) getRequestURL(ruleGroup *kubermaticv1.RuleGroup) (string, error) {
	if ruleGroup.Spec.RuleGroupType == kubermaticv1.RuleGroupTypeMetrics {
		return fmt.Sprintf("%s%s%s", r.cortexRulerURL, metricsRuleGroupConfigEndpoint, defaultNamespace), nil
	}
	return "", fmt.Errorf("unknown rule group type: %s", ruleGroup.Spec.RuleGroupType)
}

func (r *ruleGroupController) handleDeletion(ctx context.Context, ruleGroup *kubermaticv1.RuleGroup, requestURL string) error {
	req, err := http.NewRequest(http.MethodDelete,
		fmt.Sprintf("%s/%s", requestURL, ruleGroup.Name), nil)
	if err != nil {
		return err
	}
	req.Header.Add(ruleGroupTenantHeaderName, ruleGroup.Spec.Cluster.Name)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	// https://cortexmetrics.io/docs/api/#delete-rule-group
	if resp.StatusCode != http.StatusAccepted {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}
	if kubernetes.HasFinalizer(ruleGroup, ruleGroupFinalizer) {
		kubernetes.RemoveFinalizer(ruleGroup, ruleGroupFinalizer)
		if err := r.Update(ctx, ruleGroup); err != nil {
			return fmt.Errorf("updating ruleGroup finalizer: %w", err)
		}
	}
	return nil
}

func (r *ruleGroupController) ensureRuleGroup(ruleGroup *kubermaticv1.RuleGroup, requestURL string) error {
	currentRuleGroup, err := r.getCurrentRuleGroup(ruleGroup, requestURL)
	if err != nil {
		return err
	}
	expectedRuleGroup := map[string]interface{}{}
	if err := yaml.Unmarshal(ruleGroup.Spec.Data, &expectedRuleGroup); err != nil {
		return fmt.Errorf("unable to unmarshal expected rule group: %w", err)
	}
	if reflect.DeepEqual(currentRuleGroup, expectedRuleGroup) {
		return nil
	}

	req, err := http.NewRequest(http.MethodPost,
		requestURL,
		bytes.NewBuffer(ruleGroup.Spec.Data))
	if err != nil {
		return err
	}
	req.Header.Add(ruleGroupTenantHeaderName, ruleGroup.Spec.Cluster.Name)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	// https://cortexmetrics.io/docs/api/#set-rule-group
	if resp.StatusCode != http.StatusAccepted {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (r *ruleGroupController) getCurrentRuleGroup(ruleGroup *kubermaticv1.RuleGroup, requestURL string) (map[string]interface{}, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", requestURL, ruleGroup.Name), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(ruleGroupTenantHeaderName, ruleGroup.Spec.Cluster.Name)
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
		body, err := ioutil.ReadAll(resp.Body)
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
