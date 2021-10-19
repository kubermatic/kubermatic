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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
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
	GrafanaOrgAnnotationKey = "mla.k8c.io/organization"
)

// orgGrafanaReconciler stores necessary components that are required to manage MLA(Monitoring, Logging, and Alerting) setup.
type orgGrafanaReconciler struct {
	ctrlruntimeclient.Client

	log                  *zap.SugaredLogger
	workerName           string
	recorder             record.EventRecorder
	versions             kubermatic.Versions
	orgGrafanaController *orgGrafanaController
}

// Add creates a new MLA controller that is responsible for
// managing Monitoring, Logging and Alerting for user clusters.
func newOrgGrafanaReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	orgGrafanaController *orgGrafanaController,
) error {
	client := mgr.GetClient()

	reconciler := &orgGrafanaReconciler{
		Client: client,

		log:                  log,
		workerName:           workerName,
		recorder:             mgr.GetEventRecorderFor(ControllerName),
		versions:             versions,
		orgGrafanaController: orgGrafanaController,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch Projects: %w", err)
	}
	return err
}

func (r *orgGrafanaReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	project := &kubermaticv1.Project{}
	if err := r.Get(ctx, request.NamespacedName, project); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !project.DeletionTimestamp.IsZero() {
		if err := r.orgGrafanaController.handleDeletion(ctx, project); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if !kubernetes.HasFinalizer(project, mlaFinalizer) {
		kubernetes.AddFinalizer(project, mlaFinalizer)
		if err := r.Update(ctx, project); err != nil {
			return reconcile.Result{}, fmt.Errorf("updating finalizers: %w", err)
		}
	}

	org := grafanasdk.Org{
		Name: getOrgNameForProject(project),
	}
	orgID, err := r.orgGrafanaController.ensureOrganization(ctx, log, project, org, GrafanaOrgAnnotationKey)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana Organization: %w", err)
	}

	if err := r.orgGrafanaController.ensureDashboards(ctx, log, orgID); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana Dashboards: %w", err)
	}

	return reconcile.Result{}, nil
}

type orgGrafanaController struct {
	ctrlruntimeclient.Client
	grafanaClient *grafanasdk.Client
	mlaNamespace  string

	log *zap.SugaredLogger
}

func newOrgGrafanaController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	mlaNamespace string,
	grafanaClient *grafanasdk.Client,
) *orgGrafanaController {
	return &orgGrafanaController{
		Client:        client,
		grafanaClient: grafanaClient,
		mlaNamespace:  mlaNamespace,

		log: log,
	}
}

func (r *orgGrafanaController) cleanUp(ctx context.Context) error {
	projectList := &kubermaticv1.ProjectList{}
	if err := r.List(ctx, projectList); err != nil {
		return err
	}
	for _, project := range projectList.Items {
		if err := r.handleDeletion(ctx, &project); err != nil {
			return err
		}
	}
	return nil
}

func (r *orgGrafanaController) handleDeletion(ctx context.Context, project *kubermaticv1.Project) error {
	update := false
	orgID, ok := project.GetAnnotations()[GrafanaOrgAnnotationKey]
	if ok {
		update = true
		delete(project.Annotations, GrafanaOrgAnnotationKey)
		id, err := strconv.ParseUint(orgID, 10, 32)
		if err != nil {
			return err
		}
		_, err = r.grafanaClient.DeleteOrg(ctx, uint(id))
		if err != nil {
			return err
		}
	}
	if kubernetes.HasFinalizer(project, mlaFinalizer) {
		update = true
		kubernetes.RemoveFinalizer(project, mlaFinalizer)
	}
	if update {
		if err := r.Update(ctx, project); err != nil {
			return fmt.Errorf("updating Project: %w", err)
		}
	}
	return nil
}

func (r *orgGrafanaController) createGrafanaOrg(ctx context.Context, expected grafanasdk.Org) (grafanasdk.Org, error) {
	status, err := r.grafanaClient.CreateOrg(ctx, expected)
	if err != nil {
		return expected, fmt.Errorf("unable to add organization: %w (status: %s, message: %s)",
			err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
	}
	if status.OrgID == nil {
		// possibly organization already exists
		org, err := r.grafanaClient.GetOrgByOrgName(ctx, expected.Name)
		if err != nil {
			return org, fmt.Errorf("unable to get organization by name %+v %w", expected, err)
		}
		return org, nil
	}
	expected.ID = *status.OrgID

	userList := &kubermaticv1.UserList{}
	if err := r.List(ctx, userList); err != nil {
		return expected, err
	}
	for _, user := range userList.Items {
		if !user.Spec.IsAdmin {
			continue
		}
		grafanaUser, err := r.grafanaClient.LookupUser(ctx, user.Spec.Email)
		if err != nil {
			return expected, err
		}
		if err := addUserToOrg(ctx, r.grafanaClient, expected, &grafanaUser, grafanasdk.ROLE_EDITOR); err != nil {
			return expected, err
		}

	}

	return expected, nil
}

func (r *orgGrafanaController) ensureDashboards(ctx context.Context, log *zap.SugaredLogger, orgID uint) error {
	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, ctrlruntimeclient.InNamespace(r.mlaNamespace)); err != nil {
		return fmt.Errorf("Failed to list configmaps: %w", err)
	}
	for _, configMap := range configMapList.Items {
		if !strings.HasPrefix(configMap.GetName(), grafanaDashboardsConfigmapNamePrefix) {
			continue
		}
		if err := addDashboards(ctx, log, r.grafanaClient.WithOrgIDHeader(orgID), &configMap); err != nil {
			return err
		}
	}
	return nil
}

func (r *orgGrafanaController) ensureOrganization(ctx context.Context, log *zap.SugaredLogger, project *kubermaticv1.Project, expected grafanasdk.Org, annotationKey string) (uint, error) {
	orgID, ok := project.GetAnnotations()[annotationKey]
	if !ok {
		org, err := r.createGrafanaOrg(ctx, expected)
		if err != nil {
			return 0, fmt.Errorf("unable to create grafana org: %w", err)
		}
		if err := r.setAnnotation(ctx, project, annotationKey, strconv.FormatUint(uint64(org.ID), 10)); err != nil {
			// revert org creation, if deletion failed, we can't do much about it
			// if we failed at this moment and the project would be renamed quickly, that organization will be orphaned and we will never remove it.
			if status, err := r.grafanaClient.DeleteOrg(ctx, org.ID); err != nil {
				log.Debugf("unable to delete organization: %w (status: %s, message: %s)",
					err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
			}
			return 0, err
		}
		return org.ID, nil
	}
	id, err := strconv.ParseUint(orgID, 10, 32)
	if err != nil {
		return 0, err
	}

	org, err := r.grafanaClient.GetOrgById(ctx, uint(id))
	if err != nil {
		// possibly not found
		org, err := r.createGrafanaOrg(ctx, expected)
		if err != nil {
			return 0, fmt.Errorf("unable to create grafana org: %w", err)
		}
		if err := r.setAnnotation(ctx, project, annotationKey, strconv.FormatUint(uint64(org.ID), 10)); err != nil {
			// revert org creation, if deletion failed, we can't do much about it
			if status, err := r.grafanaClient.DeleteOrg(ctx, org.ID); err != nil {
				log.Debugf("unable to delete organization: %w (status: %s, message: %s)",
					err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
			}
			return 0, err
		}
		return org.ID, nil
	}
	expected.ID = uint(id)
	if !reflect.DeepEqual(org, expected) {
		if status, err := r.grafanaClient.UpdateOrg(ctx, expected, uint(id)); err != nil {
			return 0, fmt.Errorf("unable to update organization: %w (status: %s, message: %s)",
				err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
	}
	return org.ID, nil

}

func (r *orgGrafanaController) setAnnotation(ctx context.Context, project *kubermaticv1.Project, key, value string) error {
	annotations := project.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[key] = value
	project.SetAnnotations(annotations)
	if err := r.Update(ctx, project); err != nil {
		return fmt.Errorf("updating Project: %w", err)
	}
	return nil
}

func getOrgNameForProject(project *kubermaticv1.Project) string {
	return fmt.Sprintf("%s-%s", project.Spec.Name, project.Name)
}
