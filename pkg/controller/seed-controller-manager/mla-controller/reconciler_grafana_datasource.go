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
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	"k8c.io/kubermatic/v3/pkg/controller/util"
	controllerutil "k8c.io/kubermatic/v3/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/util/workerlabel"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	PrometheusType             = "prometheus"
	lokiType                   = "loki"
	alertmanagerType           = "alertmanager"
	datasourceCleanupFinalizer = "kubermatic.k8c.io/mla-cleanup-datasources"
)

// Create/update/delete Grafana Datasources to organizations based on Kubermatic Clusters.
type grafanaDatasourceReconciler struct {
	seedClient        ctrlruntimeclient.Client
	log               *zap.SugaredLogger
	workerName        string
	recorder          record.EventRecorder
	clientProvider    grafana.ClientProvider
	versions          kubermatic.Versions
	overwriteRegistry string
	mlaNamespace      string
}

var _ cleaner = &grafanaDatasourceReconciler{}

func newGrafanaDatasourceReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	versions kubermatic.Versions,
	clientProvider grafana.ClientProvider,
	overwriteRegistry string,
	mlaNamespace string,
) *grafanaDatasourceReconciler {
	return &grafanaDatasourceReconciler{
		seedClient:        mgr.GetClient(),
		log:               log.Named("grafana-datasource"),
		workerName:        workerName,
		recorder:          mgr.GetEventRecorderFor(ControllerName),
		clientProvider:    clientProvider,
		versions:          versions,
		overwriteRegistry: overwriteRegistry,
		mlaNamespace:      mlaNamespace,
	}
}

func (r *grafanaDatasourceReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	clusterEnqueuer := controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, workerlabel.Predicates(r.workerName)); err != nil {
		return fmt.Errorf("failed to watch Clusters: %w", err)
	}

	// we keep the health status of the MLA gateway up-to-date
	if err := c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, clusterEnqueuer); err != nil {
		return fmt.Errorf("failed to watch Deployments: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.MLAAdminSetting{}}, clusterEnqueuer, predicateutil.ByName(resources.MLAAdminSettingsName)); err != nil {
		return fmt.Errorf("failed to watch MLAAdminSetting: %w", err)
	}

	return err
}

func (r *grafanaDatasourceReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.NamespacedName)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if cluster.Status.NamespaceName == "" {
		log.Debug("Skipping cluster reconciling because it has no namespace yet")
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if cluster.Status.Address.ExternalName == "" {
		log.Debug("Skipping cluster reconciling because it has no external name yet")
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	result, err := util.ClusterReconcileWrapper(
		ctx,
		r.seedClient,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMLAControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster, log)
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

func (r *grafanaDatasourceReconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, log *zap.SugaredLogger) (*reconcile.Result, error) {
	// disabled by default
	if cluster.Spec.MLA == nil {
		cluster.Spec.MLA = &kubermaticv1.MLASettings{}
	}

	gClient, err := r.clientProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	if gClient == nil {
		return nil, nil
	}

	org, err := gClient.GetOrgByOrgName(ctx, GrafanaOrganization)
	if err != nil {
		return nil, fmt.Errorf("failed to get Grafana organization %q: %w", GrafanaOrganization, err)
	}

	// set header from the very beginning so all other calls will be within this organization
	gClient.SetOrgIDHeader(org.ID)

	mlaDisabled := !cluster.Spec.MLA.LoggingEnabled && !cluster.Spec.MLA.MonitoringEnabled
	if !cluster.DeletionTimestamp.IsZero() || mlaDisabled {
		return nil, r.handleDeletion(ctx, cluster, gClient)
	}

	if err := kubernetes.TryAddFinalizer(ctx, r.seedClient, cluster, datasourceCleanupFinalizer); err != nil {
		return nil, fmt.Errorf("failed to add finalizer: %w", err)
	}

	// update health status early, so failing reconcilings will not lead to outdated info
	err = r.updateGatewayHealthStatus(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to update MLA gateway health: %w", err)
	}

	data := resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithClient(r.seedClient).
		WithCluster(cluster).
		WithOverwriteRegistry(r.overwriteRegistry).
		Build()

	settings := &kubermaticv1.MLAAdminSetting{}
	if err := r.seedClient.Get(ctx, types.NamespacedName{Name: resources.MLAAdminSettingsName, Namespace: cluster.Status.NamespaceName}, settings); err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get MLAAdminSetting: %w", err)
	}

	if err := r.ensureConfigMaps(ctx, cluster, settings); err != nil {
		return nil, fmt.Errorf("failed to reconcile ConfigMaps: %w", err)
	}
	if err := r.ensureSecrets(ctx, cluster, data); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets: %w", err)
	}
	if err := r.ensureDeployments(ctx, cluster, data, settings); err != nil {
		return nil, fmt.Errorf("failed to reconcile Deployments: %w", err)
	}
	if err := r.ensureServices(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to reconcile Services: %w", err)
	}
	if err := r.ensureDatasources(ctx, gClient, cluster, org); err != nil {
		return nil, fmt.Errorf("failed to reconcile Grafana datasources: %w", err)
	}

	return nil, nil
}

func (r *grafanaDatasourceReconciler) ensureDatasources(ctx context.Context, gClient grafana.Client, cluster *kubermaticv1.Cluster, org grafanasdk.Org) error {
	alertmanagerDS := grafanasdk.Datasource{
		OrgID:  org.ID,
		UID:    getDatasourceUIDForCluster(alertmanagerType, cluster),
		Name:   getAlertmanagerDatasourceNameForCluster(cluster),
		Type:   alertmanagerType,
		Access: "proxy",
		URL:    fmt.Sprintf("http://mla-gateway.%s.svc.cluster.local/api/prom", cluster.Status.NamespaceName),
		JSONData: map[string]interface{}{
			"handleGrafanaManagedAlerts": true,
			"implementation":             "cortex",
		},
	}
	if err := r.reconcileDatasource(ctx, cluster.Spec.MLA.MonitoringEnabled || cluster.Spec.MLA.LoggingEnabled, alertmanagerDS, gClient); err != nil {
		return fmt.Errorf("failed to ensure Grafana Alertmanager datasource: %w", err)
	}

	lokiDS := grafanasdk.Datasource{
		OrgID:  org.ID,
		UID:    getDatasourceUIDForCluster(lokiType, cluster),
		Name:   getLokiDatasourceNameForCluster(cluster),
		Type:   lokiType,
		Access: "proxy",
		URL:    fmt.Sprintf("http://mla-gateway.%s.svc.cluster.local", cluster.Status.NamespaceName),
		JSONData: map[string]interface{}{
			"alertmanagerUid": getDatasourceUIDForCluster(alertmanagerType, cluster),
		},
	}
	if err := r.reconcileDatasource(ctx, cluster.Spec.MLA.LoggingEnabled, lokiDS, gClient); err != nil {
		return fmt.Errorf("failed to ensure Grafana Loki datasource: %w", err)
	}

	prometheusDS := grafanasdk.Datasource{
		OrgID:  org.ID,
		UID:    getDatasourceUIDForCluster(PrometheusType, cluster),
		Name:   getPrometheusDatasourceNameForCluster(cluster),
		Type:   PrometheusType,
		Access: "proxy",
		URL:    fmt.Sprintf("http://mla-gateway.%s.svc.cluster.local/api/prom", cluster.Status.NamespaceName),
		JSONData: map[string]interface{}{
			"alertmanagerUid": getDatasourceUIDForCluster(alertmanagerType, cluster),
			"httpMethod":      "POST",
		},
	}
	if err := r.reconcileDatasource(ctx, cluster.Spec.MLA.MonitoringEnabled, prometheusDS, gClient); err != nil {
		return fmt.Errorf("failed to ensure Grafana Prometheus datasource: %w", err)
	}

	return nil
}

func (r *grafanaDatasourceReconciler) reconcileDatasource(ctx context.Context, enabled bool, expected grafanasdk.Datasource, gClient grafana.Client) error {
	if enabled {
		ds, err := gClient.GetDatasourceByUID(ctx, expected.UID)
		if err != nil {
			if grafana.IsNotFoundErr(err) {
				status, err := gClient.CreateDatasource(ctx, expected)
				if err != nil {
					return fmt.Errorf("unable to add datasource: %w", err)
				}
				if status.ID != nil {
					return nil
				}
				// possibly already exists with such name
				ds, err = gClient.GetDatasourceByName(ctx, expected.Name)
				if err != nil {
					return fmt.Errorf("unable to get datasource by name %s", expected.Name)
				}
			}
		}
		expected.ID = ds.ID
		if !reflect.DeepEqual(ds, expected) {
			if _, err := gClient.UpdateDatasource(ctx, expected); err != nil {
				return fmt.Errorf("unable to update datasource: %w", err)
			}
		}
	} else if _, err := gClient.DeleteDatasourceByUID(ctx, expected.UID); err != nil && !grafana.IsNotFoundErr(err) {
		return fmt.Errorf("unable to delete datasource: %w", err)
	}

	return nil
}

func (r *grafanaDatasourceReconciler) ensureDeployments(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData, settings *kubermaticv1.MLAAdminSetting) error {
	creators := []reconciling.NamedDeploymentReconcilerFactory{
		GatewayDeploymentReconciler(data, settings),
	}

	return reconciling.ReconcileDeployments(ctx, creators, c.Status.NamespaceName, r.seedClient)
}

func (r *grafanaDatasourceReconciler) ensureConfigMaps(ctx context.Context, c *kubermaticv1.Cluster, settings *kubermaticv1.MLAAdminSetting) error {
	creators := []reconciling.NamedConfigMapReconcilerFactory{
		GatewayConfigMapReconciler(c, r.mlaNamespace, settings),
	}

	return reconciling.ReconcileConfigMaps(ctx, creators, c.Status.NamespaceName, r.seedClient)
}

func (r *grafanaDatasourceReconciler) ensureSecrets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := []reconciling.NamedSecretReconcilerFactory{
		GatewayCAReconciler(),
		GatewayCertificateReconciler(c, data.GetMLAGatewayCA),
	}

	return reconciling.ReconcileSecrets(ctx, creators, c.Status.NamespaceName, r.seedClient)
}

func (r *grafanaDatasourceReconciler) ensureServices(ctx context.Context, c *kubermaticv1.Cluster) error {
	creators := []reconciling.NamedServiceReconcilerFactory{
		GatewayInternalServiceReconciler(),
		GatewayExternalServiceReconciler(c),
	}

	return reconciling.ReconcileServices(ctx, creators, c.Status.NamespaceName, r.seedClient)
}

func (r *grafanaDatasourceReconciler) Cleanup(ctx context.Context) error {
	clusterList := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusterList); err != nil {
		return err
	}
	for _, cluster := range clusterList.Items {
		if err := r.handleDeletion(ctx, &cluster, nil); err != nil {
			return err
		}
	}
	return nil
}

func (r *grafanaDatasourceReconciler) handleDeletion(ctx context.Context, cluster *kubermaticv1.Cluster, gClient grafana.Client) error {
	if gClient != nil {
		datasources := []string{
			getDatasourceUIDForCluster(alertmanagerType, cluster),
			getDatasourceUIDForCluster(lokiType, cluster),
			getDatasourceUIDForCluster(PrometheusType, cluster),
		}

		for _, ds := range datasources {
			if _, err := gClient.DeleteDatasourceByUID(ctx, ds); err != nil && !grafana.IsNotFoundErr(err) {
				return fmt.Errorf("unable to delete datasource %q: %w", ds, err)
			}
		}
	}

	if cluster.DeletionTimestamp.IsZero() && cluster.Status.NamespaceName != "" {
		for _, resource := range ResourcesOnDeletion(cluster.Status.NamespaceName) {
			err := r.seedClient.Delete(ctx, resource)
			// Update Health status even in case of error
			// If any resources could not be deleted (configmap, secret,.. not only deployment)
			// The status will be kubermaticv1.HealthStatusDown until everything is cleaned up.
			// Then the status will be removed
			if errH := r.cleanupGatewayHealthStatus(ctx, cluster, err); errH != nil {
				return fmt.Errorf("failed to update mlaGateway status in cluster: %w", errH)
			}
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete %s: %w", resource.GetName(), err)
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, cluster, datasourceCleanupFinalizer)
}

func (r *grafanaDatasourceReconciler) updateGatewayHealthStatus(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	mlaGatewayHealth, err := resources.HealthyDeployment(ctx, r.seedClient, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: gatewayName}, 1)
	if err != nil {
		return fmt.Errorf("failed to get health for Deployment %s: %w", resources.MLAMonitoringAgentDeploymentName, err)
	}

	return kubernetes.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.MLAGateway = &mlaGatewayHealth
	})
}

func (r *grafanaDatasourceReconciler) cleanupGatewayHealthStatus(ctx context.Context, cluster *kubermaticv1.Cluster, resourceDeletionErr error) error {
	return kubernetes.UpdateClusterStatus(ctx, r.seedClient, cluster, func(c *kubermaticv1.Cluster) {
		// Remove the health status in Cluster CR
		c.Status.ExtendedHealth.MLAGateway = nil
		if resourceDeletionErr != nil && !apierrors.IsNotFound(resourceDeletionErr) {
			down := kubermaticv1.HealthStatusDown
			c.Status.ExtendedHealth.MLAGateway = &down
		}
	})
}
