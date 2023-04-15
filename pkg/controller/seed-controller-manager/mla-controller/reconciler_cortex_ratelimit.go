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

package mlacontroller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"

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
	RuntimeConfigMap      = "cortex-runtime-config"
	RuntimeConfigFileName = "runtime-config.yaml"
)

type TenantOverride struct {
	IngestionRate      *int32 `yaml:"ingestion_rate,omitempty"`
	MaxSeriesPerMetric *int32 `yaml:"max_series_per_metric,omitempty"`
	MaxSeriesPerQuery  *int32 `yaml:"max_series_per_query,omitempty"`
	MaxSamplesPerQuery *int32 `yaml:"max_samples_per_query,omitempty"`
	IngestionBurstSize *int32 `yaml:"ingestion_burst_size,omitempty"`
	MaxSeriesTotal     *int32 `yaml:"max_series_per_user,omitempty"`
}

type Overrides struct {
	Overrides map[string]TenantOverride `yaml:"overrides"`
}

// Updates Cortex runtime configuration with rate limits based on MLAAdminSetting.
type cortexRatelimitReconciler struct {
	seedClient   ctrlruntimeclient.Client
	log          *zap.SugaredLogger
	workerName   string
	recorder     record.EventRecorder
	mlaNamespace string
}

var _ cleaner = &cortexRatelimitReconciler{}

func newCortexRatelimitReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	mlaNamespace string,
) *cortexRatelimitReconciler {
	return &cortexRatelimitReconciler{
		seedClient:   mgr.GetClient(),
		log:          log.Named("cortex-ratelimit"),
		workerName:   workerName,
		recorder:     mgr.GetEventRecorderFor(ControllerName),
		mlaNamespace: mlaNamespace,
	}
}

func (r *cortexRatelimitReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.MLAAdminSetting{}}, &handler.EnqueueRequestForObject{}, predicateutil.ByName(resources.MLAAdminSettingsName)); err != nil {
		return fmt.Errorf("failed to watch MLAAdminSettings: %w", err)
	}

	return nil
}

func (r *cortexRatelimitReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("settings", request.NamespacedName)
	log.Debug("Processing")

	mlaAdminSetting := &kubermaticv1.MLAAdminSetting{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, mlaAdminSetting); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !mlaAdminSetting.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, mlaAdminSetting); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r.seedClient, mlaAdminSetting, mlaFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.ensureLimits(ctx, mlaAdminSetting); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure limits: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *cortexRatelimitReconciler) ensureLimits(ctx context.Context, mlaAdminSetting *kubermaticv1.MLAAdminSetting) error {
	configMap := &corev1.ConfigMap{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Namespace: r.mlaNamespace, Name: RuntimeConfigMap}, configMap); err != nil {
		return fmt.Errorf("unable to get cortex runtime config map: %w", err)
	}
	config, ok := configMap.Data[RuntimeConfigFileName]
	if !ok {
		return errors.New("unable to find runtime config file in configmap")
	}
	or := &Overrides{}

	decoder := yaml.NewDecoder(strings.NewReader(config))
	decoder.KnownFields(true)

	if err := decoder.Decode(or); err != nil {
		return fmt.Errorf("unable to unmarshal runtime config[%s]: %w", config, err)
	}

	tenantOr := TenantOverride{}

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
	if or.Overrides == nil {
		or.Overrides = make(map[string]TenantOverride)
	}
	or.Overrides[mlaAdminSetting.Spec.ClusterName] = tenantOr
	data, err := yaml.Marshal(or)
	if err != nil {
		return fmt.Errorf("unable to marshal runtime config[%+v]: %w", or, err)
	}
	configMap.Data[RuntimeConfigFileName] = string(data)
	return r.seedClient.Update(ctx, configMap)
}

func (r *cortexRatelimitReconciler) Cleanup(ctx context.Context, log *zap.SugaredLogger) error {
	mlaAdminSettingList := &kubermaticv1.MLAAdminSettingList{}
	if err := r.seedClient.List(ctx, mlaAdminSettingList); err != nil {
		return fmt.Errorf("Failed to list mlaAdminSetting: %w", err)
	}
	for _, mlaAdminSetting := range mlaAdminSettingList.Items {
		if err := r.handleDeletion(ctx, r.log, &mlaAdminSetting); err != nil {
			return fmt.Errorf("handling deletion: %w", err)
		}
	}
	return nil
}

func (r *cortexRatelimitReconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, mlaAdminSetting *kubermaticv1.MLAAdminSetting) error {
	configMap := &corev1.ConfigMap{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Namespace: r.mlaNamespace, Name: RuntimeConfigMap}, configMap); err != nil {
		return fmt.Errorf("unable to get cortex runtime config map: %w", err)
	}
	config, ok := configMap.Data[RuntimeConfigFileName]
	if !ok {
		return errors.New("unable to find runtime config file in configmap")
	}
	or := &Overrides{}

	decoder := yaml.NewDecoder(strings.NewReader(config))
	decoder.KnownFields(true)

	if err := decoder.Decode(or); err != nil {
		return fmt.Errorf("unable to unmarshal runtime config[%s]: %w", config, err)
	}

	if _, ok := or.Overrides[mlaAdminSetting.Spec.ClusterName]; ok {
		delete(or.Overrides, mlaAdminSetting.Spec.ClusterName)
		data, err := yaml.Marshal(or)
		if err != nil {
			return fmt.Errorf("unable to marshal runtime config[%+v]: %w", or, err)
		}
		configMap.Data[RuntimeConfigFileName] = string(data)
		if err := r.seedClient.Update(ctx, configMap); err != nil {
			return fmt.Errorf("unable to update configmap: %w", err)
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, mlaAdminSetting, mlaFinalizer)
}
