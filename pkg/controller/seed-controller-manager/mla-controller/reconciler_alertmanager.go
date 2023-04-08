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
	"bytes"
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/cortex"
	"k8c.io/kubermatic/v3/pkg/controller/util"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	kubernetesprovider "k8c.io/kubermatic/v3/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	controllerruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	alertmanagerFinalizer = "kubermatic.k8c.io/alertmanager"
)

// Manage alertmanager configuration based on Kubermatic Clusters.
type alertmanagerReconciler struct {
	seedClient           ctrlruntimeclient.Client
	log                  *zap.SugaredLogger
	workerName           string
	recorder             record.EventRecorder
	versions             kubermatic.Versions
	cortexClientProvider cortex.ClientProvider
}

var _ cleaner = &alertmanagerReconciler{}

func newAlertmanagerReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	versions kubermatic.Versions,
	cortexClientProvider cortex.ClientProvider,
) *alertmanagerReconciler {
	return &alertmanagerReconciler{
		seedClient:           mgr.GetClient(),
		log:                  log.Named("alertmanager"),
		workerName:           workerName,
		recorder:             mgr.GetEventRecorderFor(ControllerName),
		versions:             versions,
		cortexClientProvider: cortexClientProvider,
	}
}

func (r *alertmanagerReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch Clusters: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Alertmanager{}}, util.EnqueueClusterForNamespacedObject(r.seedClient)); err != nil {
		return fmt.Errorf("failed to watch Alertmanagers: %w", err)
	}

	enqueueClusterForSecret := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		alertmanager := &kubermaticv1.Alertmanager{}
		if err := r.seedClient.Get(ctx, types.NamespacedName{
			Name:      resources.AlertmanagerName,
			Namespace: a.GetNamespace(),
		}, alertmanager); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			utilruntime.HandleError(fmt.Errorf("failed to get alertmanager object: %w", err))
			return nil
		}

		if alertmanager.Spec.ConfigSecret.Name == a.GetName() {
			cluster, err := kubernetesprovider.ClusterFromNamespace(ctx, r.seedClient, a.GetNamespace())
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %w", err))
				return nil
			}
			if cluster != nil {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}

		return nil
	})

	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, enqueueClusterForSecret); err != nil {
		return fmt.Errorf("failed to watch Secrets: %w", err)
	}

	return err
}

func (r *alertmanagerReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.NamespacedName)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because it has no namespace yet")
		return reconcile.Result{}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := util.ClusterReconcileWrapper(
		ctx,
		r.seedClient,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMLAControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
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

func (r *alertmanagerReconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	mlaEnabled := cluster.Spec.MLA != nil && (cluster.Spec.MLA.MonitoringEnabled || cluster.Spec.MLA.LoggingEnabled)
	// Currently, we don't have a dedicated flag for enabling/disabling Alertmanager, and Alertmanager will be enabled
	// or disabled based on MLA flag.
	if !cluster.DeletionTimestamp.IsZero() || !mlaEnabled {
		return nil, r.handleDeletion(ctx, cluster)
	}

	if err := kubernetes.TryAddFinalizer(ctx, r.seedClient, cluster, alertmanagerFinalizer); err != nil {
		return nil, fmt.Errorf("failed to add finalizer: %w", err)
	}

	err := r.ensureAlertmanagerConfiguration(ctx, cluster)
	// Do not return immediately if alertmanger configuration update failed. Update the configuration health status first.
	if errC := r.ensureAlertManagerConfigStatus(ctx, cluster, err); errC != nil {
		return nil, fmt.Errorf("failed to update alertmanager configuration status: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create alertmanager configuration: %w", err)
	}
	return nil, nil
}

func (r *alertmanagerReconciler) Cleanup(ctx context.Context) error {
	clusterList := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusterList); err != nil {
		return err
	}
	for _, cluster := range clusterList.Items {
		if err := r.handleDeletion(ctx, &cluster); err != nil {
			return err
		}
	}
	return nil
}

func (r *alertmanagerReconciler) handleDeletion(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// only perform cleanup steps if the Cluster object is valid,
	// but do remove the finalizer at the end even if the object was junk,
	// so that we do not block cluster deletion
	if cluster.Status.NamespaceName != "" {
		// If monitoring is disabled, we clean up `Alertmanager` and `Secret` objects, and also Alertmanager configuration.
		err := r.cleanupAlertmanagerConfiguration(ctx, cluster)
		// Do not return immmediatly if alertmanger configuration update failed. Update the configuration health status first.
		if errC := r.ensureAlertManagerConfigStatus(ctx, cluster, err); errC != nil && !apierrors.IsNotFound(errC) {
			return fmt.Errorf("failed to update alertmanager configuration status in cluster: %w", err)
		}
		if err != nil {
			return fmt.Errorf("failed to delete alertmanager configuration: %w", err)
		}
		if cluster.DeletionTimestamp.IsZero() {
			// if cluster is still there we need to delete objects manually
			err := r.cleanupAlertmanagerObjects(ctx, cluster)
			// Do not return immmediatly if alertmanger configuration update failed. Update the configuration health status first.
			if errC := r.cleanupAlertmanagerConfigurationStatus(ctx, cluster, err); errC != nil {
				return fmt.Errorf("failed to update alertmanager configuration status in cluster: %w", errC)
			}
			if err != nil {
				return fmt.Errorf("failed to remove alertmanager objects: %w", err)
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, cluster, alertmanagerFinalizer)
}

func (r *alertmanagerReconciler) cleanupAlertmanagerConfiguration(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	return r.cortexClientProvider().DeleteAlertmanagerConfiguration(ctx, cluster.Name)
}

func (r *alertmanagerReconciler) cleanupAlertmanagerConfigurationStatus(ctx context.Context, cluster *kubermaticv1.Cluster, errC error) error {
	return kubernetes.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		// Remove the alertmanager config status in Cluster CR
		c.Status.ExtendedHealth.AlertmanagerConfig = nil
		if errC != nil && !apierrors.IsNotFound(errC) {
			down := kubermaticv1.HealthStatusDown
			c.Status.ExtendedHealth.AlertmanagerConfig = &down
		}
	})
}

func (r *alertmanagerReconciler) cleanupAlertmanagerObjects(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	alertmanager := &kubermaticv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.AlertmanagerName,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	if err := r.seedClient.Get(ctx, types.NamespacedName{
		Name:      resources.AlertmanagerName,
		Namespace: cluster.Status.NamespaceName,
	}, alertmanager); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}
	if alertmanager.Spec.ConfigSecret.Name != "" {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertmanager.Spec.ConfigSecret.Name,
				Namespace: cluster.Status.NamespaceName,
			},
		}
		if err := r.seedClient.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := r.seedClient.Delete(ctx, alertmanager); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}
	return nil
}

func (r *alertmanagerReconciler) ensureAlertmanagerConfiguration(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	config, err := r.getAlertmanagerConfigForCluster(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config: %w", err)
	}

	currentConfig, err := r.getCurrentAlertmanagerConfig(ctx, cluster)
	if err != nil {
		return err
	}

	expectedConfig := map[string]interface{}{}
	decoder := yaml.NewDecoder(bytes.NewReader(config))
	if err := decoder.Decode(&expectedConfig); err != nil {
		return fmt.Errorf("failed to unmarshal expected config: %w", err)
	}

	if reflect.DeepEqual(currentConfig, expectedConfig) {
		return nil
	}

	cClient := r.cortexClientProvider()

	if err := cClient.SetAlertmanagerConfiguration(ctx, cluster.Name, config); err != nil {
		return fmt.Errorf("failed to update Cortex Alertmanager configuration: %w", err)
	}

	return nil
}

func (r *alertmanagerReconciler) getAlertmanagerForCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.Alertmanager, error) {
	alertNamespacedName := types.NamespacedName{
		Name:      resources.AlertmanagerName,
		Namespace: cluster.Status.NamespaceName,
	}
	alertmanager := &kubermaticv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertNamespacedName.Name,
			Namespace: alertNamespacedName.Namespace,
		},
	}
	if err := r.seedClient.Get(ctx, alertNamespacedName, alertmanager); err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get alertmanager: %w", err)
	}
	return alertmanager, nil
}

func (r *alertmanagerReconciler) getAlertmanagerConfigForCluster(ctx context.Context, cluster *kubermaticv1.Cluster) ([]byte, error) {
	configuration := []byte(resources.DefaultAlertmanagerConfig)

	alertmanager, err := r.getAlertmanagerForCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}

	if alertmanager.Spec.ConfigSecret.Name == "" {
		if _, err := controllerruntime.CreateOrUpdate(ctx, r.seedClient, alertmanager, func() error {
			alertmanager.Spec.ConfigSecret.Name = resources.DefaultAlertmanagerConfigSecretName
			return nil
		}); err != nil {
			return nil, fmt.Errorf("failed to create or update alertmanager object: %w", err)
		}
	}

	secretNamespacedName := types.NamespacedName{
		Name:      alertmanager.Spec.ConfigSecret.Name,
		Namespace: cluster.Status.NamespaceName,
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNamespacedName.Name,
			Namespace: secretNamespacedName.Namespace,
		},
	}
	if err := r.seedClient.Get(ctx, secretNamespacedName, secret); err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get alertmanager config secret: %w", err)
	}

	if secret.Data == nil || len(secret.Data[resources.AlertmanagerConfigSecretKey]) == 0 {
		if _, err := controllerruntime.CreateOrUpdate(ctx, r.seedClient, secret, func() error {
			secret.Data = map[string][]byte{
				resources.AlertmanagerConfigSecretKey: configuration,
			}
			return nil
		}); err != nil {
			return nil, fmt.Errorf("failed to create or update alertmanager config secret: %w", err)
		}
	}
	return secret.Data[resources.AlertmanagerConfigSecretKey], nil
}

func (r *alertmanagerReconciler) getCurrentAlertmanagerConfig(ctx context.Context, cluster *kubermaticv1.Cluster) (map[string]interface{}, error) {
	cClient := r.cortexClientProvider()

	configEncoded, err := cClient.GetAlertmanagerConfiguration(ctx, cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get Cortex Alertmanager configuration: %w", err)
	}
	if configEncoded == nil {
		return nil, nil
	}

	config := map[string]interface{}{}
	decoder := yaml.NewDecoder(bytes.NewReader(configEncoded))
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return config, nil
}

func (r *alertmanagerReconciler) ensureAlertManagerConfigStatus(ctx context.Context, cluster *kubermaticv1.Cluster, configErr error) error {
	alertmanager, err := r.getAlertmanagerForCluster(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get alertmanager: %w", err)
	}
	oldAlertmanager := alertmanager.DeepCopy()

	up := kubermaticv1.HealthStatusUp
	down := kubermaticv1.HealthStatusDown

	alertmanager.Status.ConfigStatus.ErrorMessage = "" // reset error message
	alertmanager.Status.ConfigStatus.Status = corev1.ConditionTrue
	alertmanager.Status.ConfigStatus.LastUpdated = metav1.Now()
	clusterStatus := &up
	if configErr != nil {
		alertmanager.Status.ConfigStatus.ErrorMessage = configErr.Error()
		alertmanager.Status.ConfigStatus.Status = corev1.ConditionFalse
		alertmanager.Status.ConfigStatus.LastUpdated = oldAlertmanager.Status.ConfigStatus.LastUpdated
		clusterStatus = &down
	}

	// Update alertmanager config status in Alertmanager CR
	if oldAlertmanager.Status.ConfigStatus != alertmanager.Status.ConfigStatus {
		if err := r.seedClient.Status().Patch(ctx, alertmanager, ctrlruntimeclient.MergeFrom(oldAlertmanager)); err != nil {
			return fmt.Errorf("error patching alertmanager config status: %w", err)
		}
	}

	// Update alertmanager config status in Cluster CR
	err = kubernetes.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.AlertmanagerConfig = clusterStatus
	})
	if err != nil {
		return fmt.Errorf("error patching cluster health status: %w", err)
	}

	return nil
}
