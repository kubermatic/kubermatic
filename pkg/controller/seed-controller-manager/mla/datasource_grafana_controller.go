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
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	PrometheusType   = "prometheus"
	lokiType         = "loki"
	alertmanagerType = "alertmanager"
)

// datasourceGrafanaReconciler stores necessary components that are required to manage MLA(Monitoring, Logging, and Alerting) setup.
type datasourceGrafanaReconciler struct {
	ctrlruntimeclient.Client

	log                         *zap.SugaredLogger
	workerName                  string
	recorder                    events.EventRecorder
	versions                    kubermatic.Versions
	datasourceGrafanaController *datasourceGrafanaController
}

func newDatasourceGrafanaReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	datasourceGrafanaController *datasourceGrafanaController,
) error {
	client := mgr.GetClient()
	subname := "grafana-datasource"

	reconciler := &datasourceGrafanaReconciler{
		Client: client,

		log:                         log.Named(subname),
		workerName:                  workerName,
		recorder:                    mgr.GetEventRecorder(controllerName(subname)),
		versions:                    versions,
		datasourceGrafanaController: datasourceGrafanaController,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName(subname)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}).
		Watches(&kubermaticv1.MLAAdminSetting{}, controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient()), builder.WithPredicates(predicateutil.ByName(resources.MLAAdminSettingsName))).
		Build(reconciler)

	return err
}

func (r *datasourceGrafanaReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
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

	if cluster.Status.Address.ExternalName == "" {
		log.Debug("Skipping cluster reconciling because it has no external name yet")
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Add a wrapping here so we can emit an event on error
	result, err := controllerutil.ClusterReconcileWrapper(
		ctx,
		r,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMLAControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.datasourceGrafanaController.reconcile(ctx, cluster, log)
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

type datasourceGrafanaController struct {
	ctrlruntimeclient.Client
	clientProvider grafanaClientProvider
	mlaNamespace   string

	log               *zap.SugaredLogger
	overwriteRegistry string
}

func newDatasourceGrafanaController(
	client ctrlruntimeclient.Client,
	clientProvider grafanaClientProvider,
	mlaNamespace string,

	log *zap.SugaredLogger,
	overwriteRegistry string,
) *datasourceGrafanaController {
	return &datasourceGrafanaController{
		Client:         client,
		mlaNamespace:   mlaNamespace,
		clientProvider: clientProvider,

		log:               log,
		overwriteRegistry: overwriteRegistry,
	}
}

func (r *datasourceGrafanaController) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster, log *zap.SugaredLogger) (*reconcile.Result, error) {
	// disabled by default
	if cluster.Spec.MLA == nil {
		cluster.Spec.MLA = &kubermaticv1.MLASettings{}
	}
	projectID, ok := cluster.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	if !ok {
		return nil, fmt.Errorf("unable to get project name from label")
	}

	grafanaClient, err := r.clientProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	project := &kubermaticv1.Project{}
	err = r.Get(ctx, types.NamespacedName{Name: projectID}, project)
	if (err != nil && apierrors.IsNotFound(err)) || (err == nil && !project.DeletionTimestamp.IsZero()) {
		// if project removed before cluster we need only to remove resources and finalizer
		if err := r.handleDeletion(ctx, cluster, nil); err != nil {
			return nil, fmt.Errorf("handling deletion: %w", err)
		}
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if grafanaClient == nil {
		return nil, nil
	}

	org, err := getOrgByProject(ctx, grafanaClient, project)
	if err != nil {
		// This fails very often because of racing between the controllers - can't get a grafana organization from a project before it gets assigned
		// Once the organization controller adds the annotation to the project, we will reconcile again, so we skip reconciliation in this case
		// This works around potential resources abuse documented in https://github.com/kubermatic/kubermatic/issues/9970
		log.Warnf("failed to get grafana org from a project, waiting until the next reconciliation: %s", err)
		return nil, nil
	}
	// set header from the very beginning so all other calls will be within this organization
	grafanaClient.SetOrgIDHeader(org.ID)

	mlaDisabled := !cluster.Spec.MLA.LoggingEnabled && !cluster.Spec.MLA.MonitoringEnabled
	if !cluster.DeletionTimestamp.IsZero() || mlaDisabled {
		if err := r.handleDeletion(ctx, cluster, grafanaClient); err != nil {
			return nil, fmt.Errorf("handling deletion: %w", err)
		}
		return nil, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r, cluster, mlaFinalizer); err != nil {
		return nil, fmt.Errorf("failed to add finalizer: %w", err)
	}

	data := resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithClient(r).
		WithCluster(cluster).
		WithOverwriteRegistry(r.overwriteRegistry).
		Build()

	settings := &kubermaticv1.MLAAdminSetting{}
	if err := r.Get(ctx, types.NamespacedName{Name: resources.MLAAdminSettingsName, Namespace: cluster.Status.NamespaceName}, settings); err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get MLAAdminSetting: %w", err)
	}

	if err := r.ensureConfigMaps(ctx, cluster, settings); err != nil {
		return nil, fmt.Errorf("failed to reconcile ConfigMaps in namespace %s: %w", cluster.Status.NamespaceName, err)
	}
	if err := r.ensureSecrets(ctx, cluster, data); err != nil {
		return nil, fmt.Errorf("failed to reconcile Secrets in namespace %s: %w", cluster.Status.NamespaceName, err)
	}
	if err := r.ensureDeployments(ctx, cluster, data, settings); err != nil {
		return nil, fmt.Errorf("failed to reconcile Deployments in namespace %s: %w", cluster.Status.NamespaceName, err)
	}
	err = r.mlaGatewayHealth(ctx, cluster)
	if err != nil {
		return nil, err
	}
	if err := r.ensureServices(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to reconcile Services in namespace %s: %w", "mla", err)
	}

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
	if err := r.reconcileDatasource(ctx, cluster.Spec.MLA.MonitoringEnabled || cluster.Spec.MLA.LoggingEnabled, alertmanagerDS, grafanaClient); err != nil {
		return nil, fmt.Errorf("failed to ensure Grafana Alertmanager Datasources: %w", err)
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
	if err := r.reconcileDatasource(ctx, cluster.Spec.MLA.LoggingEnabled, lokiDS, grafanaClient); err != nil {
		return nil, fmt.Errorf("failed to ensure Grafana Loki Datasources: %w", err)
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
	if err := r.reconcileDatasource(ctx, cluster.Spec.MLA.MonitoringEnabled, prometheusDS, grafanaClient); err != nil {
		return nil, fmt.Errorf("failed to ensure Grafana Prometheus Datasources: %w", err)
	}

	return nil, nil
}

// datasourcesEqual returns true if both datasources are identical with
// regards to the fields set by this reconciler. Grafana defaults some
// fields server-side which this function is meant to ignore.
func datasourcesEqual(a, b grafanasdk.Datasource) bool {
	if a.OrgID != b.OrgID || a.UID != b.UID || a.Name != b.Name || a.Type != b.Type || a.Access != b.Access || a.URL != b.URL {
		return false
	}

	jsonA, err := json.Marshal(a.JSONData)
	if err != nil {
		return false
	}

	jsonB, err := json.Marshal(b.JSONData)
	if err != nil {
		return false
	}

	return bytes.Equal(jsonA, jsonB)
}

func (r *datasourceGrafanaController) reconcileDatasource(ctx context.Context, enabled bool, expected grafanasdk.Datasource, grafanaClient *grafanasdk.Client) error {
	if enabled {
		ds, err := grafanaClient.GetDatasourceByUID(ctx, expected.UID)
		if err != nil {
			if errors.As(err, &grafanasdk.ErrNotFound{}) {
				status, err := grafanaClient.CreateDatasource(ctx, expected)
				if err != nil {
					return fmt.Errorf("unable to add datasource: %w (status: %s, message: %s)",
						err, ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
				}
				if status.ID != nil {
					return nil
				}
				// possibly already exists with such name
				ds, err = grafanaClient.GetDatasourceByName(ctx, expected.Name)
				if err != nil {
					return fmt.Errorf("unable to get datasource by name %s", expected.Name)
				}
			}
		}
		expected.ID = ds.ID
		if !datasourcesEqual(ds, expected) {
			if status, err := grafanaClient.UpdateDatasource(ctx, expected); err != nil {
				return fmt.Errorf("unable to update datasource: %w (status: %s, message: %s)",
					err, ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
			}
		}
	} else if status, err := grafanaClient.DeleteDatasourceByUID(ctx, expected.UID); err != nil {
		return fmt.Errorf("unable to delete datasource: %w (status: %s, message: %s)",
			err, ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
	}
	return nil
}

func (r *datasourceGrafanaController) ensureDeployments(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData, settings *kubermaticv1.MLAAdminSetting) error {
	creators := []reconciling.NamedDeploymentReconcilerFactory{
		GatewayDeploymentReconciler(data, settings),
	}
	if err := reconciling.ReconcileDeployments(ctx, creators, c.Status.NamespaceName, r); err != nil {
		return err
	}
	return nil
}

func (r *datasourceGrafanaController) ensureConfigMaps(ctx context.Context, c *kubermaticv1.Cluster, settings *kubermaticv1.MLAAdminSetting) error {
	creators := []reconciling.NamedConfigMapReconcilerFactory{
		GatewayConfigMapReconciler(c, r.mlaNamespace, settings),
	}
	if err := reconciling.ReconcileConfigMaps(ctx, creators, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %w", err)
	}
	return nil
}

func (r *datasourceGrafanaController) ensureSecrets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := []reconciling.NamedSecretReconcilerFactory{
		GatewayCAReconciler(),
		GatewayCertificateReconciler(c, data.GetMLAGatewayCA),
	}
	if err := reconciling.ReconcileSecrets(ctx, creators, c.Status.NamespaceName, r); err != nil {
		return fmt.Errorf("failed to ensure that the Secrets exist: %w", err)
	}
	return nil
}

func (r *datasourceGrafanaController) ensureServices(ctx context.Context, c *kubermaticv1.Cluster) error {
	creators := []reconciling.NamedServiceReconcilerFactory{
		GatewayInternalServiceReconciler(),
		GatewayExternalServiceReconciler(c),
	}
	return reconciling.ReconcileServices(ctx, creators, c.Status.NamespaceName, r)
}

func (r *datasourceGrafanaController) CleanUp(ctx context.Context) error {
	clusterList := &kubermaticv1.ClusterList{}
	if err := r.List(ctx, clusterList); err != nil {
		return err
	}
	for _, cluster := range clusterList.Items {
		if err := r.handleDeletion(ctx, &cluster, nil); err != nil {
			return fmt.Errorf("failed to handle Grafana datasource cleanup for cluster %s: %w", cluster.Name, err)
		}
	}
	return nil
}

func (r *datasourceGrafanaController) cleanUpMLAGatewayHealthStatus(ctx context.Context, cluster *kubermaticv1.Cluster, resourceDeletionErr error) error {
	return controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		// Remove the health status in Cluster CR
		c.Status.ExtendedHealth.MLAGateway = nil
		if resourceDeletionErr != nil && !apierrors.IsNotFound(resourceDeletionErr) {
			down := kubermaticv1.HealthStatusDown
			c.Status.ExtendedHealth.MLAGateway = &down
		}
	})
}

func (r *datasourceGrafanaController) handleDeletion(ctx context.Context, cluster *kubermaticv1.Cluster, grafanaClient *grafanasdk.Client) error {
	if grafanaClient != nil {
		// that's mostly means that Grafana organization doesn't exists anymore
		if status, err := grafanaClient.DeleteDatasourceByUID(ctx, getDatasourceUIDForCluster(alertmanagerType, cluster)); err != nil {
			return fmt.Errorf("unable to delete datasource: %w (status: %s, message: %s)",
				err, ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
		}
		if status, err := grafanaClient.DeleteDatasourceByUID(ctx, getDatasourceUIDForCluster(lokiType, cluster)); err != nil {
			return fmt.Errorf("unable to delete datasource: %w (status: %s, message: %s)",
				err, ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
		}
		if status, err := grafanaClient.DeleteDatasourceByUID(ctx, getDatasourceUIDForCluster(PrometheusType, cluster)); err != nil {
			return fmt.Errorf("unable to delete datasource: %w (status: %s, message: %s)",
				err, ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
		}
	}
	if cluster.DeletionTimestamp.IsZero() && cluster.Status.NamespaceName != "" {
		for _, resource := range ResourcesOnDeletion(cluster.Status.NamespaceName) {
			err := r.Delete(ctx, resource)
			// Update Health status even in case of error
			// If any resources could not be deleted (configmap, secret,.. not only deployment)
			// The status will be kubermaticv1.HealthStatusDown until everything is cleaned up.
			// Then the status will be removed
			if errH := r.cleanUpMLAGatewayHealthStatus(ctx, cluster, err); errH != nil {
				return fmt.Errorf("failed to update mlaGateway status in cluster: %w", errH)
			}
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete %s: %w", resource.GetName(), err)
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r, cluster, mlaFinalizer)
}

func (r *datasourceGrafanaController) mlaGatewayHealth(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	mlaGatewayHealth, err := resources.HealthyDeployment(ctx, r, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: gatewayName}, 1)
	if err != nil {
		return fmt.Errorf("failed to get dep health %s: %w", resources.MLAMonitoringAgentDeploymentName, err)
	}

	err = controllerutil.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.ExtendedHealth.MLAGateway = &mlaGatewayHealth
	})
	if err != nil {
		return fmt.Errorf("error patching cluster health status: %w", err)
	}

	return nil
}
