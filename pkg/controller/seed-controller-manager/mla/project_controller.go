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

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

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
	grafanaOrgAnnotationKey = "mla.k8c.io/organization"
)

// projectReconciler stores necessary components that are required to manage MLA(Monitoring, Logging, and Alerting) setup.
type projectReconciler struct {
	ctrlruntimeclient.Client
	grafanaClient *grafanasdk.Client

	log        *zap.SugaredLogger
	workerName string
	recorder   record.EventRecorder
	versions   kubermatic.Versions
}

// Add creates a new MLA controller that is responsible for
// managing Monitoring, Logging and Alerting for user clusters.
func newProjectReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	grafanaClient *grafanasdk.Client,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &projectReconciler{
		Client:        client,
		grafanaClient: grafanaClient,

		log:        log,
		workerName: workerName,
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		versions:   versions,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	debugPredicate := predicate.ByLabel(kubermaticv1.WorkerNameLabelKey, workerName)

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, &handler.EnqueueRequestForObject{}, debugPredicate); err != nil {
		return fmt.Errorf("failed to watch Projects: %v", err)
	}
	return err
}

func (r *projectReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	project := &kubermaticv1.Project{}
	if err := r.Get(ctx, request.NamespacedName, project); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !project.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, project); err != nil {
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
	if err := r.ensureOrganization(ctx, log, project, org, grafanaOrgAnnotationKey); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana Organization Datasources: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *projectReconciler) handleDeletion(ctx context.Context, project *kubermaticv1.Project) error {
	org, err := r.grafanaClient.GetOrgByOrgName(ctx, getOrgNameForProject(project))
	if err != nil {
		return err
	}
	_, err = r.grafanaClient.DeleteOrg(ctx, org.ID)
	if err != nil {
		return err
	}
	kubernetes.RemoveFinalizer(project, mlaFinalizer)
	if err := r.Update(ctx, project); err != nil {
		return fmt.Errorf("updating Project: %w", err)
	}
	return nil
}

func (r *projectReconciler) createGrafanaOrg(ctx context.Context, org grafanasdk.Org) (grafanasdk.Org, error) {
	status, err := r.grafanaClient.CreateOrg(ctx, org)
	if err != nil {
		return org, fmt.Errorf("unable to add organization: %w (status: %s, message: %s)",
			err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
	}
	if status.OrgID == nil {
		// possibly organization already exists
		org, err := r.grafanaClient.GetOrgByOrgName(ctx, org.Name)
		if err != nil {
			return org, fmt.Errorf("unable to get organization by name %s", org.Name)
		}
		return org, nil
	}
	org.ID = *status.OrgID
	return org, nil
}

func (r *projectReconciler) ensureOrganization(ctx context.Context, log *zap.SugaredLogger, project *kubermaticv1.Project, expected grafanasdk.Org, annotationKey string) error {
	orgID, ok := project.GetAnnotations()[annotationKey]
	if !ok {
		org, err := r.createGrafanaOrg(ctx, expected)
		if err != nil {
			return fmt.Errorf("unable to create grafana org: %w", err)
		}
		if err := r.setAnnotation(ctx, project, annotationKey, strconv.FormatUint(uint64(org.ID), 10)); err != nil {
			// revert org creation, if deletion failed, we can't do much about it
			if status, err := r.grafanaClient.DeleteOrg(ctx, org.ID); err != nil {
				log.Errorf("unable to delete organization: %w (status: %s, message: %s)",
					err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
			}
			return err
		}
		return nil
	}
	id, err := strconv.ParseUint(orgID, 10, 32)
	if err != nil {
		return err
	}

	org, err := r.grafanaClient.GetOrgById(ctx, uint(id))
	if err != nil {
		// possibly not found
		org, err := r.createGrafanaOrg(ctx, expected)
		if err != nil {
			return fmt.Errorf("unable to create grafana org: %w", err)
		}
		if err := r.setAnnotation(ctx, project, annotationKey, strconv.FormatUint(uint64(org.ID), 10)); err != nil {
			// revert org creation, if deletion failed, we can't do much about it
			if status, err := r.grafanaClient.DeleteOrg(ctx, org.ID); err != nil {
				log.Errorf("unable to delete organization: %w (status: %s, message: %s)",
					err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
			}
			return err
		}
		return nil
	}
	expected.ID = uint(id)
	if !reflect.DeepEqual(org, expected) {
		if status, err := r.grafanaClient.UpdateOrg(ctx, expected, uint(id)); err != nil {
			return fmt.Errorf("unable to update organization: %w (status: %s, message: %s)",
				err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
	}
	return nil

}

func (r *projectReconciler) setAnnotation(ctx context.Context, project *kubermaticv1.Project, key, value string) error {
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
