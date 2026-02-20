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
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	alertmanagerFinalizer        = "kubermatic.k8c.io/alertmanager"
	AlertmanagerConfigEndpoint   = "/api/v1/alerts"
	AlertmanagerTenantHeaderName = "X-Scope-OrgID"
)

type alertmanagerReconciler struct {
	ctrlruntimeclient.Client

	log        *zap.SugaredLogger
	workerName string
	recorder   events.EventRecorder
	versions   kubermatic.Versions

	alertmanagerController *alertmanagerController
}

func newAlertmanagerReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	alertmanagerController *alertmanagerController,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()
	subname := "alertmanager"

	reconciler := &alertmanagerReconciler{
		Client: client,

		log:                    log.Named(subname),
		workerName:             workerName,
		recorder:               mgr.GetEventRecorder(controllerName(subname)),
		versions:               versions,
		alertmanagerController: alertmanagerController,
	}

	enqueueClusterForSecret := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		alertmanager := &kubermaticv1.Alertmanager{}
		if err := client.Get(ctx, types.NamespacedName{
			Name:      resources.AlertmanagerName,
			Namespace: a.GetNamespace(),
		}, alertmanager); err != nil {
			if apierrors.IsNotFound(err) {
				return []reconcile.Request{}
			}
			utilruntime.HandleError(fmt.Errorf("failed to get alertmanager object: %w", err))
		}

		if alertmanager.Spec.ConfigSecret.Name == a.GetName() {
			cluster, err := kubernetesprovider.ClusterFromNamespace(ctx, client, a.GetNamespace())
			if err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %w", err))
				return []reconcile.Request{}
			}
			if cluster != nil {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}

		return []reconcile.Request{}
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName(subname)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}).
		Watches(&kubermaticv1.Alertmanager{}, util.EnqueueClusterForNamespacedObject(client)).
		Watches(&corev1.Secret{}, enqueueClusterForSecret).
		Build(reconciler)

	return err
}

func (r *alertmanagerReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because it has no namespace yet")
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
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
			return r.alertmanagerController.reconcile(ctx, cluster)
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

type alertmanagerController struct {
	ctrlruntimeclient.Client
	httpClient *http.Client

	log                   *zap.SugaredLogger
	cortexAlertmanagerURL string
}

func newAlertmanagerController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	httpClient *http.Client,
	cortexAlertmanagerURL string,
) *alertmanagerController {
	return &alertmanagerController{
		Client:     client,
		httpClient: httpClient,

		log:                   log,
		cortexAlertmanagerURL: cortexAlertmanagerURL,
	}
}

func (r *alertmanagerController) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	mlaEnabled := cluster.Spec.MLA != nil && (cluster.Spec.MLA.MonitoringEnabled || cluster.Spec.MLA.LoggingEnabled)
	// Currently, we don't have a dedicated flag for enabling/disabling Alertmanager, and Alertmanager will be enabled
	// or disabled based on MLA flag.
	if !cluster.DeletionTimestamp.IsZero() || !mlaEnabled {
		return nil, r.handleDeletion(ctx, cluster)
	}

	if err := kubernetes.TryAddFinalizer(ctx, r, cluster, alertmanagerFinalizer); err != nil {
		return nil, fmt.Errorf("failed to add finalizer: %w", err)
	}

	err := r.ensureAlertmanagerConfiguration(ctx, cluster)
	// Do not return immmediatly if alertmanger configuration update failed. Update the configuration health status first.
	if errC := r.ensureAlertManagerConfigStatus(ctx, cluster, err); errC != nil {
		return nil, fmt.Errorf("failed to update alertmanager configuration status: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create alertmanager configuration: %w", err)
	}
	return nil, nil
}

func (r *alertmanagerController) CleanUp(ctx context.Context) error {
	clusterList := &kubermaticv1.ClusterList{}
	if err := r.List(ctx, clusterList); err != nil {
		return err
	}
	for _, cluster := range clusterList.Items {
		if err := r.handleDeletion(ctx, &cluster); err != nil {
			return fmt.Errorf("failed to handle alertmanager cleanup for cluster %s: %w", cluster.Name, err)
		}
	}
	return nil
}

func (r *alertmanagerController) handleDeletion(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// only perform cleanup steps if the Cluster object is valid,
	// but do remove the finalizer at the end even if the object was junk,
	// so that we do not block cluster deletion
	if cluster.Status.NamespaceName != "" {
		// If monitoring is disabled, we clean up `Alertmanager` and `Secret` objects, and also Alertmanager configuration.
		err := r.cleanUpAlertmanagerConfiguration(ctx, cluster)
		// Do not return immmediatly if alertmanger configuration update failed. Update the configuration health status first.
		if errC := r.ensureAlertManagerConfigStatus(ctx, cluster, err); errC != nil && !apierrors.IsNotFound(errC) {
			return fmt.Errorf("failed to update alertmanager configuration status in cluster: %w", err)
		}
		if err != nil {
			return fmt.Errorf("failed to delete alertmanager configuration: %w", err)
		}
		if cluster.DeletionTimestamp.IsZero() {
			// if cluster is still there we need to delete objects manually
			err := r.cleanUpAlertmanagerObjects(ctx, cluster)
			// Do not return immmediatly if alertmanger configuration update failed. Update the configuration health status first.
			if errC := r.cleanUpAlertmanagerConfigurationStatus(ctx, cluster, err); errC != nil {
				return fmt.Errorf("failed to update alertmanager configuration status in cluster: %w", errC)
			}
			if err != nil {
				return fmt.Errorf("failed to remove alertmanager objects: %w", err)
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r, cluster, alertmanagerFinalizer)
}

func (r *alertmanagerController) cleanUpAlertmanagerConfiguration(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete,
		r.cortexAlertmanagerURL+AlertmanagerConfigEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Add(AlertmanagerTenantHeaderName, cluster.Name)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	// https://cortexmetrics.io/docs/api/#delete-alertmanager-configuration
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (r *alertmanagerController) cleanUpAlertmanagerConfigurationStatus(ctx context.Context, cluster *kubermaticv1.Cluster, errC error) error {
	return util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		// Remove the alertmanager config status in Cluster CR
		c.Status.ExtendedHealth.AlertmanagerConfig = nil
		if errC != nil && !apierrors.IsNotFound(errC) {
			down := kubermaticv1.HealthStatusDown
			c.Status.ExtendedHealth.AlertmanagerConfig = &down
		}
	})
}

func (r *alertmanagerController) cleanUpAlertmanagerObjects(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	alertmanager := &kubermaticv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.AlertmanagerName,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	if err := r.Get(ctx, types.NamespacedName{
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
		if err := r.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := r.Delete(ctx, alertmanager); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}
	return nil
}

func (r *alertmanagerController) ensureAlertmanagerConfiguration(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	config, err := r.getAlertmanagerConfigForCluster(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config: %w", err)
	}

	alertmanagerURL := r.cortexAlertmanagerURL + AlertmanagerConfigEndpoint
	currentConfig, err := r.getCurrentAlertmanagerConfig(ctx, alertmanagerURL, cluster)
	if err != nil {
		return err
	}

	expectedConfig := map[string]interface{}{}
	decoder := yaml.NewDecoder(bytes.NewReader(config))
	if err := decoder.Decode(&expectedConfig); err != nil {
		return fmt.Errorf("unable to unmarshal expected config: %w", err)
	}

	if reflect.DeepEqual(currentConfig, expectedConfig) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		alertmanagerURL,
		bytes.NewBuffer(config))
	if err != nil {
		return err
	}
	req.Header.Add(AlertmanagerTenantHeaderName, cluster.Name)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	// https://cortexmetrics.io/docs/api/#set-alertmanager-configuration
	if resp.StatusCode != http.StatusCreated {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (r *alertmanagerController) getAlertmanagerForCluster(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.Alertmanager, error) {
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
	if err := r.Get(ctx, alertNamespacedName, alertmanager); err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get alertmanager: %w", err)
	}
	return alertmanager, nil
}

func (r *alertmanagerController) getAlertmanagerConfigForCluster(ctx context.Context, cluster *kubermaticv1.Cluster) ([]byte, error) {
	configuration := []byte(resources.DefaultAlertmanagerConfig)

	alertmanager, err := r.getAlertmanagerForCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}

	if alertmanager.Spec.ConfigSecret.Name == "" {
		if _, err := controllerruntime.CreateOrUpdate(ctx, r, alertmanager, func() error {
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
	if err := r.Get(ctx, secretNamespacedName, secret); err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get alertmanager config secret: %w", err)
	}

	if secret.Data == nil || len(secret.Data[resources.AlertmanagerConfigSecretKey]) == 0 {
		if _, err := controllerruntime.CreateOrUpdate(ctx, r, secret, func() error {
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

func (r *alertmanagerController) getCurrentAlertmanagerConfig(ctx context.Context, alertmanagerURL string, cluster *kubermaticv1.Cluster) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, alertmanagerURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(AlertmanagerTenantHeaderName, cluster.Name)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// https://cortexmetrics.io/docs/api/#get-alertmanager-configuration
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

func (r *alertmanagerController) ensureAlertManagerConfigStatus(ctx context.Context, cluster *kubermaticv1.Cluster, configErr error) error {
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
		if err := r.Status().Patch(ctx, alertmanager, ctrlruntimeclient.MergeFrom(oldAlertmanager)); err != nil {
			return fmt.Errorf("error patching alertmanager config status: %w", err)
		}
	}

	// Update alertmanager config status in Cluster CR
	err = util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.AlertmanagerConfig = clusterStatus
	})
	if err != nil {
		return fmt.Errorf("error patching cluster health status: %w", err)
	}

	return nil
}
