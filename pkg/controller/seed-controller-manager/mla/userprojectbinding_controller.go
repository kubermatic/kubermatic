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
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

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

type userProjectBindingReconciler struct {
	ctrlruntimeclient.Client
	grafanaClient *grafanasdk.Client
	httpClient    *http.Client

	log           *zap.SugaredLogger
	workerName    string
	recorder      record.EventRecorder
	versions      kubermatic.Versions
	grafanaURL    string
	grafanaHeader string
}

func newUserProjectBindingReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	grafanaClient *grafanasdk.Client,
	httpClient *http.Client,
	grafanaURL string,
	grafanaHeader string,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &userProjectBindingReconciler{
		Client:        client,
		grafanaClient: grafanaClient,
		httpClient:    httpClient,

		log:           log,
		workerName:    workerName,
		recorder:      mgr.GetEventRecorderFor(ControllerName),
		versions:      versions,
		grafanaURL:    grafanaURL,
		grafanaHeader: grafanaHeader,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.UserProjectBinding{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch UserProjectBindings: %v", err)
	}
	return err
}

func (r *userProjectBindingReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	userProjectBinding := &kubermaticv1.UserProjectBinding{}
	if err := r.Get(ctx, request.NamespacedName, userProjectBinding); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !userProjectBinding.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, userProjectBinding); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if !kubernetes.HasFinalizer(userProjectBinding, mlaFinalizer) {
		kubernetes.AddFinalizer(userProjectBinding, mlaFinalizer)
		if err := r.Update(ctx, userProjectBinding); err != nil {
			return reconcile.Result{}, fmt.Errorf("updating finalizers: %w", err)
		}
	}

	org, err := r.getOrgByProjectID(ctx, userProjectBinding.Spec.ProjectID)
	if err != nil {
		return reconcile.Result{}, err
	}

	group := rbac.ExtractGroupPrefix(userProjectBinding.Spec.Group)
	role := groupToRole[group]

	// checking if user already exists in the corresponding organization
	user, err := r.getGrafanaOrgUser(ctx, org.ID, userProjectBinding.Spec.UserEmail)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to get user : %w", err)
	}
	// if there is no such user in project organization, let's create one
	if user == nil {
		if _, err := r.addGrafanaOrgUser(ctx, org.ID, userProjectBinding.Spec.UserEmail, string(role)); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to add grafana user : %w", err)
		}
		return reconcile.Result{}, nil
	}

	if user.Role != string(role) {
		userRole := grafanasdk.UserRole{
			LoginOrEmail: userProjectBinding.Spec.UserEmail,
			Role:         string(role),
		}
		if status, err := r.grafanaClient.UpdateOrgUser(ctx, userRole, org.ID, user.ID); err != nil {
			return reconcile.Result{}, fmt.Errorf("unable to update grafana user role: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
	}

	return reconcile.Result{}, nil
}

func (r *userProjectBindingReconciler) getOrgByProjectID(ctx context.Context, projectID string) (grafanasdk.Org, error) {
	project := &kubermaticv1.Project{}
	if err := r.Get(ctx, types.NamespacedName{Name: projectID}, project); err != nil {
		return grafanasdk.Org{}, fmt.Errorf("failed to get project: %w", err)
	}

	orgID, ok := project.GetAnnotations()[grafanaOrgAnnotationKey]
	if !ok {
		return grafanasdk.Org{}, fmt.Errorf("project should have grafana org annotation set")
	}
	id, err := strconv.ParseUint(orgID, 10, 32)
	if err != nil {
		return grafanasdk.Org{}, err
	}
	return r.grafanaClient.GetOrgById(ctx, uint(id))
}

func (r *userProjectBindingReconciler) handleDeletion(ctx context.Context, userProjectBinding *kubermaticv1.UserProjectBinding) error {
	org, err := r.getOrgByProjectID(ctx, userProjectBinding.Spec.ProjectID)
	if err != nil {
		return err
	}
	user, err := r.getGrafanaOrgUser(ctx, org.ID, userProjectBinding.Spec.UserEmail)
	if err != nil {
		return fmt.Errorf("unable to get user : %w", err)
	}
	if user != nil {
		status, err := r.grafanaClient.DeleteOrgUser(ctx, user.OrgId, user.ID)
		if err != nil {
			return fmt.Errorf("failed to delete org user: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
	}

	kubernetes.RemoveFinalizer(userProjectBinding, mlaFinalizer)
	if err := r.Update(ctx, userProjectBinding); err != nil {
		return fmt.Errorf("updating UserProjectBinding: %w", err)
	}

	return nil
}

func (r *userProjectBindingReconciler) getGrafanaOrgUser(ctx context.Context, orgID uint, email string) (*grafanasdk.OrgUser, error) {
	users, err := r.grafanaClient.GetOrgUsers(ctx, orgID)
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.Email == email {
			return &user, nil
		}
	}
	return nil, nil
}

func (r *userProjectBindingReconciler) addGrafanaOrgUser(ctx context.Context, orgID uint, email, role string) (*grafanasdk.OrgUser, error) {
	req, err := http.NewRequest("GET", r.grafanaURL+"/api/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(r.grafanaHeader, email)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	type response struct {
		ID uint `json:"id"`
	}
	res := &response{}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(res); err != nil || res.ID == 0 {
		return nil, fmt.Errorf("unable to decode response : %w", err)
	}
	// delete user from default org
	if status, err := r.grafanaClient.DeleteOrgUser(ctx, defaultOrgID, res.ID); err != nil {
		return nil, fmt.Errorf("failed to delete grafana user from default org: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
	}

	userRole := grafanasdk.UserRole{
		LoginOrEmail: email,
		Role:         role,
	}
	if status, err := r.grafanaClient.AddOrgUser(ctx, userRole, orgID); err != nil {
		return nil, fmt.Errorf("failed to add grafana user to org: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
	}
	return &grafanasdk.OrgUser{
		ID:    res.ID,
		OrgId: orgID,
		Email: email,
		Login: email,
		Role:  role,
	}, nil
}
