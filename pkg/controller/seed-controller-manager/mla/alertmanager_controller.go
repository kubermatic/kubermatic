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
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	alertmanagerFinalizer      = "kubermatic.io/alertmanager"
	alertmanagerConfigEndpoint = "/api/v1/alerts"
	defaultConfig              = `
alertmanager_config: |
  route:
    receiver: 'null'
  receivers:
    - name: 'null'
`
)

type mlaGatewayURLGetter interface {
	mlaGatewayURL(cluster *kubermaticv1.Cluster) string
}

type defaultMLAGatewayURLGetter struct {
}

func newDefaultMLAGatewayURLGetter() *defaultMLAGatewayURLGetter {
	return &defaultMLAGatewayURLGetter{}
}

func (d *defaultMLAGatewayURLGetter) mlaGatewayURL(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("http://mla-gateway-alert.%s.svc.cluster.local", cluster.Status.NamespaceName)
}

type alertmanagerReconciler struct {
	ctrlruntimeclient.Client
	httpClient *http.Client

	log        *zap.SugaredLogger
	workerName string
	recorder   record.EventRecorder
	versions   kubermatic.Versions

	mlaGatewayURLGetter mlaGatewayURLGetter
}

func newAlertmanagerReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	httpClient *http.Client,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &alertmanagerReconciler{
		Client:     client,
		httpClient: httpClient,

		log:                 log,
		workerName:          workerName,
		recorder:            mgr.GetEventRecorderFor(ControllerName),
		versions:            versions,
		mlaGatewayURLGetter: newDefaultMLAGatewayURLGetter(),
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch Cluster: %w", err)
	}

	enqueueClusterForAlertmanager := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		if err := client.List(context.Background(), clusterList); err != nil {
			log.Errorw("Failed to list clusters", zap.Error(err))
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %w", err))
		}
		for _, cluster := range clusterList.Items {
			if cluster.Status.NamespaceName == a.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}
		return []reconcile.Request{}
	})
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Alertmanager{}}, enqueueClusterForAlertmanager); err != nil {
		return fmt.Errorf("failed to watch Alertmanager: %w", err)
	}
	enqueueClusterForSecret := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		ctx := context.Background()
		alertmangerList := &kubermaticv1.AlertmanagerList{}
		if err := client.List(ctx, alertmangerList, ctrlruntimeclient.InNamespace(a.GetNamespace())); err != nil {
			log.Errorw("Failed to list alertmanager object", zap.Error(err))
			utilruntime.HandleError(fmt.Errorf("failed to list alertmanager objects: %w", err))
		}
		for _, alertmanager := range alertmangerList.Items {
			if a.GetName() == alertmanager.Spec.ConfigSecret.Name {
				clusterList := &kubermaticv1.ClusterList{}
				if err := client.List(context.Background(), clusterList); err != nil {
					log.Errorw("Failed to list clusters", zap.Error(err))
					utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %w", err))
				}
				for _, cluster := range clusterList.Items {
					if cluster.Status.NamespaceName == a.GetNamespace() {
						return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
					}
				}

			}
		}
		return []reconcile.Request{}
	})
	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, enqueueClusterForSecret); err != nil {
		return fmt.Errorf("failed to watch Secret: %w", err)
	}
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
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
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

	if !cluster.DeletionTimestamp.IsZero() {
		// If this cluster is being deleted, we only clean up the Alertmanager configuration, we don't need to clean up
		// `Alertmanager` and `Secret` objects because they are in cluster namespace, and they will be garbage-collected
		// by Kubernetes itself.
		if err := r.cleanUpAlertmanagerConfiguration(cluster); err != nil {
			return nil, fmt.Errorf("failed to delete alertmanager conifugration: %w", err)
		}
		if kubernetes.HasFinalizer(cluster, alertmanagerFinalizer) {
			kubernetes.RemoveFinalizer(cluster, alertmanagerFinalizer)
			if err := r.Update(ctx, cluster); err != nil {
				return nil, fmt.Errorf("updating Cluster: %w", err)
			}
		}
		return nil, nil
	}

	monitoringEnabled := cluster.Spec.MLA != nil && cluster.Spec.MLA.MonitoringEnabled
	// Currently, we don't have a dedicated flag for enabling/disabling Alertmanager, and Alertmanager will be enabled
	// or disabled based on the monitoring flag.
	if !monitoringEnabled {
		// If monitoring is disabled, we clean up `Alertmanager` and `Secret` objects, and also Alertmanager configuration.
		if err := r.cleanUpAlertmanagerConfiguration(cluster); err != nil {
			return nil, fmt.Errorf("failed to delete alertmanager conifugration: %w", err)
		}
		if err := r.cleanUpAlertmanagerObjects(ctx, cluster); err != nil {
			return nil, fmt.Errorf("failed to remove alertmanager objects: %w", err)
		}
		if kubernetes.HasFinalizer(cluster, alertmanagerFinalizer) {
			kubernetes.RemoveFinalizer(cluster, alertmanagerFinalizer)
			if err := r.Update(ctx, cluster); err != nil {
				return nil, fmt.Errorf("updating Cluster: %w", err)
			}
		}
		return nil, nil
	}

	if !kubernetes.HasFinalizer(cluster, alertmanagerFinalizer) {
		kubernetes.AddFinalizer(cluster, alertmanagerFinalizer)
		if err := r.Update(ctx, cluster); err != nil {
			return nil, fmt.Errorf("updating finalizers: %w", err)
		}
	}

	if err := r.ensureAlertmanagerConfiguration(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create alertmanager configuration: %w", err)
	}
	return nil, nil
}

func (r *alertmanagerReconciler) cleanUpAlertmanagerConfiguration(cluster *kubermaticv1.Cluster) error {
	req, err := http.NewRequest(http.MethodDelete,
		r.mlaGatewayURLGetter.mlaGatewayURL(cluster)+alertmanagerConfigEndpoint, nil)
	if err != nil {
		return err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	// https://cortexmetrics.io/docs/api/#delete-alertmanager-configuration
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (r *alertmanagerReconciler) cleanUpAlertmanagerObjects(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Currently, we need to clean up the `Secret` of Alertmanager configuration before we clean up `Alertmanager` object.
	// In the future, when we implement the Alertmanager api endpoint, we can set owner reference to the secret when it
	// is created, so `Secret` will be GCed when `Alertmanager` object is deleted.
	alertmanager := &kubermaticv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.DefaultAlertmanagerName,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      resources.DefaultAlertmanagerName,
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
		if err := r.Delete(ctx, secret); err != nil && !errors.IsNotFound(err) {
			return err
		}
	}
	if err := r.Delete(ctx, alertmanager); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}
	return nil
}

func (r *alertmanagerReconciler) ensureAlertmanagerConfiguration(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	config, err := r.getAlertmanagerConfigForCluster(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get alertmanager config: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost,
		r.mlaGatewayURLGetter.mlaGatewayURL(cluster)+alertmanagerConfigEndpoint,
		bytes.NewBuffer(config))
	if err != nil {
		return err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	// https://cortexmetrics.io/docs/api/#set-alertmanager-configuration
	if resp.StatusCode != http.StatusCreated {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d,error: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (r *alertmanagerReconciler) getAlertmanagerConfigForCluster(ctx context.Context, cluster *kubermaticv1.Cluster) ([]byte, error) {
	configuration := []byte(defaultConfig)
	alertmanager := &kubermaticv1.Alertmanager{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      resources.DefaultAlertmanagerName,
		Namespace: cluster.Status.NamespaceName,
	}, alertmanager); err != nil {
		if errors.IsNotFound(err) {
			return configuration, nil
		}
		return nil, fmt.Errorf("failed to get alertmanager: %w", err)
	}
	// If Alertmanager object is found, we try to get configuration from config secret.
	secret := &corev1.Secret{}
	if alertmanager.Spec.ConfigSecret.Name == "" {
		alertmanager.Spec.ConfigSecret.Name = "alertmanager"
		if err := r.Update(ctx, alertmanager); err != nil {
			return nil, fmt.Errorf("failed to update alertmanager object: %w", err)
		}
	}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      alertmanager.Spec.ConfigSecret.Name,
		Namespace: cluster.Status.NamespaceName,
	}, secret); err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get alertmanger config secret: %w", err)
	}

	if secret.Data == nil || len(secret.Data[resources.AlertmanagerConfigSecretKey]) == 0 {
		if _, err := controllerruntime.CreateOrUpdate(ctx, r.Client, secret, func() error {
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
