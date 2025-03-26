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
	"strings"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	subname := "grafana-org"

	reconciler := &orgGrafanaReconciler{
		Client: client,

		log:                  log.Named(subname),
		workerName:           workerName,
		recorder:             mgr.GetEventRecorderFor(controllerName(subname)),
		versions:             versions,
		orgGrafanaController: orgGrafanaController,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName(subname)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Project{}).
		Build(reconciler)

	return err
}

func (r *orgGrafanaReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	project := &kubermaticv1.Project{}
	if err := r.Get(ctx, request.NamespacedName, project); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	grafanaClient, err := r.orgGrafanaController.clientProvider(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	if !project.DeletionTimestamp.IsZero() {
		if err := r.orgGrafanaController.handleDeletion(ctx, project, grafanaClient); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if grafanaClient == nil {
		return reconcile.Result{}, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r, project, mlaFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	orgID, err := r.orgGrafanaController.ensureOrganization(ctx, log, grafanaClient, project)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana Organization: %w", err)
	}

	if err := r.orgGrafanaController.ensureDashboards(ctx, log, orgID, grafanaClient); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana Dashboards: %w", err)
	}

	return reconcile.Result{}, nil
}

type orgGrafanaController struct {
	ctrlruntimeclient.Client
	clientProvider grafanaClientProvider
	mlaNamespace   string

	log *zap.SugaredLogger
}

func newOrgGrafanaController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	mlaNamespace string,
	clientProvider grafanaClientProvider,
) *orgGrafanaController {
	return &orgGrafanaController{
		Client:         client,
		clientProvider: clientProvider,
		mlaNamespace:   mlaNamespace,

		log: log,
	}
}

func (r *orgGrafanaController) CleanUp(ctx context.Context) error {
	projectList := &kubermaticv1.ProjectList{}
	if err := r.List(ctx, projectList); err != nil {
		return err
	}
	grafanaClient, err := r.clientProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}
	for _, project := range projectList.Items {
		if err := r.handleDeletion(ctx, &project, grafanaClient); err != nil {
			return fmt.Errorf("failed to handle Grafana org cleanup for Project %s: %w", project.Name, err)
		}
	}
	return nil
}

func (r *orgGrafanaController) handleDeletion(ctx context.Context, project *kubermaticv1.Project, grafanaClient *grafanasdk.Client) error {
	oldProject := project.DeepCopy()
	update := false

	orgID, ok := getOrgIDForProject(project)
	if ok {
		update = true
		delete(project.Annotations, GrafanaOrgAnnotationKey)

		if grafanaClient != nil {
			if _, err := grafanaClient.DeleteOrg(ctx, orgID); err != nil {
				return err
			}
		}
	}

	if kubernetes.HasFinalizer(project, mlaFinalizer) {
		update = true
		kubernetes.RemoveFinalizer(project, mlaFinalizer)
	}

	if update {
		if err := r.Patch(ctx, project, ctrlruntimeclient.MergeFrom(oldProject)); err != nil {
			return fmt.Errorf("failed to update Project: %w", err)
		}
	}

	return nil
}

func (r *orgGrafanaController) ensureOrganization(ctx context.Context, log *zap.SugaredLogger, grafanaClient *grafanasdk.Client, project *kubermaticv1.Project) (uint, error) {
	desiredOrgName := getOrgNameForProject(project)

	adoptOrCreateOrg := func() (uint, error) {
		org, err := r.createGrafanaOrg(ctx, grafanaClient, desiredOrgName)
		if err != nil {
			return 0, fmt.Errorf("unable to create Grafana org: %w", err)
		}

		if err := r.setOrgAnnotation(ctx, project, org.ID); err != nil {
			// Leave the org in Grafana and during the next reconciliation createGrafanaOrg() will
			// adopt it and then we can try again to set the annotation.
			return 0, fmt.Errorf("failed to update Project's org annotation: %w", err)
		}

		return org.ID, nil
	}

	// If the project has no org attached yet, try to create a new one;
	// org names in Grafana are *unique* and createGrafanaOrg() will try
	// to adopt an existing org if possible.
	orgID, ok := getOrgIDForProject(project)
	if !ok {
		return adoptOrCreateOrg()
	}

	// We have an org ID, which might be stale. Fetch the associated org.
	org, err := grafanaClient.GetOrgById(ctx, orgID)
	if err != nil {
		return adoptOrCreateOrg()
	}

	// Now we know that we point to an org and that org exists; however
	// in case of catastrophic storage failure, it's possible that the Grafana
	// PVC was removed and all annotations suddenly point to non-existing orgs.
	// Once this controller begins to reconcile and fix the first (random)
	// project, it will most likely create a new org with an ID that *another*
	// project was already using previously. Now 2 projects would point to the
	// same org.
	// To prevent this, all we can do is rely on the org name, which will always
	// be suffixed with the unchanging (!) project name (not the human readable
	// name, which can easily change at any time).
	if !orgNameMatchesProject(project, org.Name) {
		return adoptOrCreateOrg()
	}

	// Lastly, check if the org name is still using the correct human readable
	// project name; if not, update the organization.
	if org.Name != desiredOrgName {
		org.Name = desiredOrgName

		if _, err := grafanaClient.UpdateOrg(ctx, org, orgID); err != nil {
			return 0, fmt.Errorf("unable to update organization: %w", err)
		}
	}

	// All good!
	return orgID, nil
}

func (r *orgGrafanaController) createGrafanaOrg(ctx context.Context, grafanaClient *grafanasdk.Client, orgName string) (*grafanasdk.Org, error) {
	org := grafanasdk.Org{
		Name: orgName,
	}

	// Note that Grafana org names are unique, but a DuplicateName error is *not* returned as an error here.
	status, err := grafanaClient.CreateOrg(ctx, org)
	if err != nil {
		return nil, fmt.Errorf("unable to add organization: %w", err)
	}

	// possibly organization already exists
	if status.OrgID == nil {
		org, err := grafanaClient.GetOrgByOrgName(ctx, orgName)
		if err != nil {
			return nil, fmt.Errorf("unable to get organization by name: %w", err)
		}

		return &org, nil
	}

	org.ID = *status.OrgID

	// initially assign users; updating these relations is done by the user-grafana-controller.
	userList := &kubermaticv1.UserList{}
	if err := r.List(ctx, userList); err != nil {
		return nil, fmt.Errorf("failed to list KKP users: %w", err)
	}

	for _, user := range userList.Items {
		if !user.Spec.IsAdmin {
			continue
		}

		grafanaUser, err := grafanaClient.LookupUser(ctx, user.Spec.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to lookup Grafana user %q: %w", user.Spec.Email, err)
		}

		if err := addUserToOrg(ctx, grafanaClient, org, &grafanaUser, grafanasdk.ROLE_EDITOR); err != nil {
			return nil, err
		}
	}

	return &org, nil
}

func (r *orgGrafanaController) ensureDashboards(ctx context.Context, log *zap.SugaredLogger, orgID uint, grafanaClient *grafanasdk.Client) error {
	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, ctrlruntimeclient.InNamespace(r.mlaNamespace)); err != nil {
		return fmt.Errorf("Failed to list configmaps: %w", err)
	}
	for _, configMap := range configMapList.Items {
		if !strings.HasPrefix(configMap.GetName(), grafanaDashboardsConfigmapNamePrefix) {
			continue
		}
		if err := addDashboards(ctx, log, grafanaClient.WithOrgIDHeader(orgID), &configMap); err != nil {
			return err
		}
	}
	return nil
}

func (r *orgGrafanaController) setOrgAnnotation(ctx context.Context, project *kubermaticv1.Project, orgID uint) error {
	annotations := project.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	annotations[GrafanaOrgAnnotationKey] = fmt.Sprintf("%d", orgID)
	project.SetAnnotations(annotations)

	return r.Update(ctx, project)
}

func getOrgNameForProject(project *kubermaticv1.Project) string {
	return fmt.Sprintf("%s-%s", project.Spec.Name, project.Name)
}

func orgNameMatchesProject(project *kubermaticv1.Project, orgName string) bool {
	return strings.HasSuffix(orgName, fmt.Sprintf("-%s", project.Name))
}
