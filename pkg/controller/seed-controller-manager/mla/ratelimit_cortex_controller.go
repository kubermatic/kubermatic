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
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	runtimeConfigMap      = "cortex-runtime-config"
	runtimeConfigFileName = "runtime-config.yaml"
)

type tenantOverride struct {
	IngestionRate      *int32 `yaml:"ingestion_rate,omitempty"`
	MaxSeriesPerMetric *int32 `yaml:"max_series_per_metric,omitempty"`
	MaxSeriesPerQuery  *int32 `yaml:"max_series_per_query,omitempty"`
	MaxSamplesPerQuery *int32 `yaml:"max_samples_per_query,omitempty"`
	IngestionBurstSize *int32 `yaml:"ingestion_burst_size,omitempty"`
	MaxSeriesTotal     *int32 `yaml:"max_series_per_user,omitempty"`
}

type overrides struct {
	Overrides map[string]tenantOverride `yaml:"overrides"`
}

// ratelimitCortexReconciler stores necessary components that are required to manage MLA(Monitoring, Logging, and Alerting) setup.
type ratelimitCortexReconciler struct {
	ctrlruntimeclient.Client

	log                       *zap.SugaredLogger
	workerName                string
	recorder                  record.EventRecorder
	versions                  kubermatic.Versions
	ratelimitCortexController *ratelimitCortexController
}

// Add creates a new MLA controller that is responsible for
// managing Monitoring, Logging and Alerting for user clusters.
func newRatelimitCortexReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	ratelimitCortexController *ratelimitCortexController,
) error {
	client := mgr.GetClient()

	reconciler := &ratelimitCortexReconciler{
		Client: client,

		log:                       log,
		workerName:                workerName,
		recorder:                  mgr.GetEventRecorderFor(ControllerName),
		versions:                  versions,
		ratelimitCortexController: ratelimitCortexController,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.MLAAdminSetting{}}, &handler.EnqueueRequestForObject{}, predicateutil.ByName(resources.MLAAdminSettingsName)); err != nil {
		return fmt.Errorf("failed to watch MLAAdminSetting: %w", err)
	}

	return err
}

func (r *ratelimitCortexReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	mlaAdminSetting := &kubermaticv1.MLAAdminSetting{}
	if err := r.Get(ctx, request.NamespacedName, mlaAdminSetting); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !mlaAdminSetting.DeletionTimestamp.IsZero() {
		if err := r.ratelimitCortexController.handleDeletion(ctx, log, mlaAdminSetting); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if !kubernetes.HasFinalizer(mlaAdminSetting, mlaFinalizer) {
		kubernetes.AddFinalizer(mlaAdminSetting, mlaFinalizer)
		if err := r.Update(ctx, mlaAdminSetting); err != nil {
			return reconcile.Result{}, fmt.Errorf("updating finalizers: %w", err)
		}
	}

	if err := r.ratelimitCortexController.ensureLimits(ctx, mlaAdminSetting); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure limits: %w", err)
	}

	return reconcile.Result{}, nil
}

type ratelimitCortexController struct {
	ctrlruntimeclient.Client
	mlaNamespace string

	log *zap.SugaredLogger
}

func newRatelimitCortexController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	mlaNamespace string,
) *ratelimitCortexController {
	return &ratelimitCortexController{
		Client:       client,
		mlaNamespace: mlaNamespace,

		log: log,
	}
}

func (r *ratelimitCortexController) ensureLimits(ctx context.Context, mlaAdminSetting *kubermaticv1.MLAAdminSetting) error {
	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: r.mlaNamespace, Name: runtimeConfigMap}, configMap); err != nil {
		return fmt.Errorf("unable to get cortex runtime config map: %w", err)
	}
	config, ok := configMap.Data[runtimeConfigFileName]
	if !ok {
		return errors.New("unable to find runtime config file in configmap")
	}
	or := &overrides{}
	if err := yaml.Unmarshal([]byte(config), or); err != nil {
		return fmt.Errorf("unable to unmarshal runtime config[%s]: %w", config, err)
	}

	tenantOr := tenantOverride{}

	if mlaAdminSetting.Spec.MonitoringRateLimits != nil {
		if mlaAdminSetting.Spec.MonitoringRateLimits.IngestionRate > 0 {
			tenantOr.IngestionRate = &mlaAdminSetting.Spec.MonitoringRateLimits.IngestionRate
		}
		if mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesPerMetric > 0 {
			tenantOr.MaxSeriesPerMetric = &mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesPerMetric
		}
		if mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesPerQuery > 0 {
			tenantOr.MaxSeriesPerQuery = &mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesPerQuery
		}
		if mlaAdminSetting.Spec.MonitoringRateLimits.MaxSamplesPerQuery > 0 {
			tenantOr.MaxSamplesPerQuery = &mlaAdminSetting.Spec.MonitoringRateLimits.MaxSamplesPerQuery
		}
		if mlaAdminSetting.Spec.MonitoringRateLimits.IngestionBurstSize > 0 {
			tenantOr.IngestionBurstSize = &mlaAdminSetting.Spec.MonitoringRateLimits.IngestionBurstSize
		}
		if mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesTotal > 0 {
			tenantOr.MaxSeriesTotal = &mlaAdminSetting.Spec.MonitoringRateLimits.MaxSeriesTotal
		}
	}
	or.Overrides[mlaAdminSetting.Spec.ClusterName] = tenantOr
	data, err := yaml.Marshal(or)
	if err != nil {
		return fmt.Errorf("unable to marshal runtime config[%+v]: %w", or, err)
	}
	configMap.Data[runtimeConfigFileName] = string(data)
	return r.Update(ctx, configMap)
}

func (r *ratelimitCortexController) cleanUp(ctx context.Context) error {
	mlaAdminSettingList := &kubermaticv1.MLAAdminSettingList{}
	if err := r.List(ctx, mlaAdminSettingList); err != nil {
		return fmt.Errorf("Failed to list mlaAdminSetting: %w", err)
	}
	for _, mlaAdminSetting := range mlaAdminSettingList.Items {
		if err := r.handleDeletion(ctx, r.log, &mlaAdminSetting); err != nil {
			return fmt.Errorf("handling deletion: %w", err)
		}
	}
	return nil
}

func (r *ratelimitCortexController) handleDeletion(ctx context.Context, log *zap.SugaredLogger, mlaAdminSetting *kubermaticv1.MLAAdminSetting) error {
	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: r.mlaNamespace, Name: runtimeConfigMap}, configMap); err != nil {
		return fmt.Errorf("unable to get cortex runtime config map: %w", err)
	}
	config, ok := configMap.Data[runtimeConfigFileName]
	if !ok {
		return errors.New("unable to find runtime config file in configmap")
	}
	or := &overrides{}
	if err := yaml.Unmarshal([]byte(config), or); err != nil {
		return fmt.Errorf("unable to unmarshal runtime config[%s]: %w", config, err)
	}
	if _, ok := or.Overrides[mlaAdminSetting.Spec.ClusterName]; ok {
		delete(or.Overrides, mlaAdminSetting.Spec.ClusterName)
		data, err := yaml.Marshal(or)
		if err != nil {
			return fmt.Errorf("unable to marshal runtime config[%+v]: %w", or, err)
		}
		configMap.Data[runtimeConfigFileName] = string(data)
		if err := r.Update(ctx, configMap); err != nil {
			return fmt.Errorf("unable to update configmap: %w", err)
		}
	}
	if kubernetes.HasFinalizer(mlaAdminSetting, mlaFinalizer) {
		kubernetes.RemoveFinalizer(mlaAdminSetting, mlaFinalizer)
		if err := r.Update(ctx, mlaAdminSetting); err != nil {
			return fmt.Errorf("updating mlaAdminSetting: %w", err)
		}
	}
	return nil
}
