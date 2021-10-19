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
	"net/http"
	"reflect"
	"time"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	PrometheusType = "prometheus"
	lokiType       = "loki"
)

// datasourceGrafanaReconciler stores necessary components that are required to manage MLA(Monitoring, Logging, and Alerting) setup.
type datasourceGrafanaReconciler struct {
	ctrlruntimeclient.Client

	log                         *zap.SugaredLogger
	workerName                  string
	recorder                    record.EventRecorder
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

	reconciler := &datasourceGrafanaReconciler{
		Client: client,

		log:                         log,
		workerName:                  workerName,
		recorder:                    mgr.GetEventRecorderFor(ControllerName),
		versions:                    versions,
		datasourceGrafanaController: datasourceGrafanaController,
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
		return fmt.Errorf("failed to watch Clusters: %w", err)
	}
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.MLAAdminSetting{}},
		controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient()), predicateutil.ByName(resources.MLAAdminSettingsName)); err != nil {
		return fmt.Errorf("failed to watch MLAAdminSetting: %w", err)
	}
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

	// Add a wrapping here so we can emit an event on error
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionMLAControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.datasourceGrafanaController.reconcile(ctx, cluster)
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

type datasourceGrafanaController struct {
	ctrlruntimeclient.Client
	httpClient   *http.Client
	grafanaURL   string
	grafanaAuth  string
	mlaNamespace string

	log               *zap.SugaredLogger
	overwriteRegistry string
}

func newDatasourceGrafanaController(
	client ctrlruntimeclient.Client,
	httpClient *http.Client,
	grafanaURL string,
	grafanaAuth string,
	mlaNamespace string,

	log *zap.SugaredLogger,
	overwriteRegistry string,
) *datasourceGrafanaController {
	return &datasourceGrafanaController{
		Client:       client,
		mlaNamespace: mlaNamespace,
		httpClient:   httpClient,
		grafanaURL:   grafanaURL,
		grafanaAuth:  grafanaAuth,

		log:               log,
		overwriteRegistry: overwriteRegistry,
	}
}

func (r *datasourceGrafanaController) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// disabled by default
	if cluster.Spec.MLA == nil {
		cluster.Spec.MLA = &kubermaticv1.MLASettings{}
	}
	grafanaClient, err := grafanasdk.NewClient(r.grafanaURL, r.grafanaAuth, r.httpClient)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize grafana client")
	}
	projectID, ok := cluster.GetLabels()[kubermaticv1.ProjectIDLabelKey]
	if !ok {
		return nil, fmt.Errorf("unable to get project name from label")
	}

	project := &kubermaticv1.Project{}
	err = r.Get(ctx, types.NamespacedName{Name: projectID}, project)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			// if project removed before cluster we need only to remove resources and finalizer
			if err := r.handleDeletion(ctx, nil, cluster); err != nil {
				return nil, fmt.Errorf("handling deletion: %w", err)
			}
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	org, err := getOrgByProject(ctx, grafanaClient, project)
	if err != nil {
		return nil, err
	}
	// set header from the very beginning so all other calls will be within this organization
	grafanaClient.SetOrgIDHeader(org.ID)

	mlaDisabled := !cluster.Spec.MLA.LoggingEnabled && !cluster.Spec.MLA.MonitoringEnabled
	if !cluster.DeletionTimestamp.IsZero() || mlaDisabled {
		if err := r.handleDeletion(ctx, grafanaClient, cluster); err != nil {
			return nil, fmt.Errorf("handling deletion: %w", err)
		}
		return nil, nil
	}

	if !kubernetes.HasFinalizer(cluster, mlaFinalizer) {
		kubernetes.AddFinalizer(cluster, mlaFinalizer)
		if err := r.Update(ctx, cluster); err != nil {
			return nil, fmt.Errorf("updating finalizers: %w", err)
		}
	}

	data := resources.NewTemplateDataBuilder().
		WithContext(ctx).
		WithClient(r.Client).
		WithCluster(cluster).
		WithOverwriteRegistry(r.overwriteRegistry).
		Build()

	settings := &kubermaticv1.MLAAdminSetting{}
	if err := r.Get(ctx, types.NamespacedName{Name: resources.MLAAdminSettingsName, Namespace: cluster.Status.NamespaceName}, settings); err != nil && !apiErrors.IsNotFound(err) {
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
	if err := r.ensureServices(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to reconcile Services in namespace %s: %w", "mla", err)
	}

	lokiDS := grafanasdk.Datasource{
		OrgID:  org.ID,
		UID:    getDatasourceUIDForCluster(lokiType, cluster),
		Name:   getLokiDatasourceNameForCluster(cluster),
		Type:   lokiType,
		Access: "proxy",
		URL:    fmt.Sprintf("http://mla-gateway.%s.svc.cluster.local", cluster.Status.NamespaceName),
	}
	if err := r.reconcileDatasource(ctx, grafanaClient, cluster.Spec.MLA.LoggingEnabled, lokiDS); err != nil {
		return nil, fmt.Errorf("failed to ensure Grafana Loki Datasources: %w", err)
	}

	prometheusDS := grafanasdk.Datasource{
		OrgID:  org.ID,
		UID:    getDatasourceUIDForCluster(PrometheusType, cluster),
		Name:   getPrometheusDatasourceNameForCluster(cluster),
		Type:   PrometheusType,
		Access: "proxy",
		URL:    fmt.Sprintf("http://mla-gateway.%s.svc.cluster.local/api/prom", cluster.Status.NamespaceName),
	}
	if err := r.reconcileDatasource(ctx, grafanaClient, cluster.Spec.MLA.MonitoringEnabled, prometheusDS); err != nil {
		return nil, fmt.Errorf("failed to ensure Grafana Prometheus Datasources: %w", err)
	}

	return nil, nil
}

func (r *datasourceGrafanaController) reconcileDatasource(ctx context.Context, grafanaClient *grafanasdk.Client, enabled bool, expected grafanasdk.Datasource) error {
	if enabled {
		ds, err := grafanaClient.GetDatasourceByUID(ctx, expected.UID)
		if err != nil {
			if errors.As(err, &grafanasdk.ErrNotFound{}) {
				status, err := grafanaClient.CreateDatasource(ctx, expected)
				if err != nil {
					return fmt.Errorf("unable to add datasource: %w (status: %s, message: %s)",
						err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
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
		if !reflect.DeepEqual(ds, expected) {
			if status, err := grafanaClient.UpdateDatasource(ctx, expected); err != nil {
				return fmt.Errorf("unable to update datasource: %w (status: %s, message: %s)",
					err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
			}
		}
	} else if status, err := grafanaClient.DeleteDatasourceByUID(ctx, expected.UID); err != nil {
		return fmt.Errorf("unable to delete datasource: %w (status: %s, message: %s)",
			err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
	}
	return nil

}

func (r *datasourceGrafanaController) ensureDeployments(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData, settings *kubermaticv1.MLAAdminSetting) error {
	creators := []reconciling.NamedDeploymentCreatorGetter{
		GatewayDeploymentCreator(data, settings),
	}
	if err := reconciling.ReconcileDeployments(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return err
	}
	return nil
}

func (r *datasourceGrafanaController) ensureConfigMaps(ctx context.Context, c *kubermaticv1.Cluster, settings *kubermaticv1.MLAAdminSetting) error {
	creators := []reconciling.NamedConfigMapCreatorGetter{
		GatewayConfigMapCreator(c, r.mlaNamespace, settings),
	}
	if err := reconciling.ReconcileConfigMaps(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the ConfigMap exists: %v", err)
	}
	return nil
}

func (r *datasourceGrafanaController) ensureSecrets(ctx context.Context, c *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := []reconciling.NamedSecretCreatorGetter{
		GatewayCACreator(),
		GatewayCertificateCreator(c, data.GetMLAGatewayCA),
	}
	if err := reconciling.ReconcileSecrets(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c))); err != nil {
		return fmt.Errorf("failed to ensure that the Secrets exist: %v", err)
	}
	return nil
}

func (r *datasourceGrafanaController) ensureServices(ctx context.Context, c *kubermaticv1.Cluster) error {
	creators := []reconciling.NamedServiceCreatorGetter{
		GatewayInternalServiceCreator(),
		GatewayExternalServiceCreator(c),
	}
	return reconciling.ReconcileServices(ctx, creators, c.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(c)))
}

func (r *datasourceGrafanaController) cleanUp(ctx context.Context) error {
	clusterList := &kubermaticv1.ClusterList{}
	if err := r.List(ctx, clusterList); err != nil {
		return err
	}
	for _, cluster := range clusterList.Items {
		if err := r.handleDeletion(ctx, nil, &cluster); err != nil {
			return err
		}
	}
	return nil
}

func (r *datasourceGrafanaController) handleDeletion(ctx context.Context, grafanaClient *grafanasdk.Client, cluster *kubermaticv1.Cluster) error {
	if grafanaClient != nil {
		// that's mostly means that Grafana organization doesn't exists anymore
		if status, err := grafanaClient.DeleteDatasourceByUID(ctx, getDatasourceUIDForCluster(lokiType, cluster)); err != nil {
			return fmt.Errorf("unable to delete datasource: %w (status: %s, message: %s)",
				err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
		if status, err := grafanaClient.DeleteDatasourceByUID(ctx, getDatasourceUIDForCluster(PrometheusType, cluster)); err != nil {
			return fmt.Errorf("unable to delete datasource: %w (status: %s, message: %s)",
				err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
	}

	if cluster.DeletionTimestamp.IsZero() {
		for _, resource := range ResourcesOnDeletion(cluster.Status.NamespaceName) {
			if err := r.Client.Delete(ctx, resource); err != nil && !apiErrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete %s: %v", resource.GetName(), err)
			}
		}
	}

	if kubernetes.HasFinalizer(cluster, mlaFinalizer) {
		kubernetes.RemoveFinalizer(cluster, mlaFinalizer)
		if err := r.Update(ctx, cluster); err != nil {
			return fmt.Errorf("updating Cluster: %w", err)
		}
	}

	return nil
}
