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
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/grafana/grafana/pkg/models"
	grafanasdk "github.com/kubermatic/grafanasdk"
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

type userGrafanaReconciler struct {
	ctrlruntimeclient.Client

	log                   *zap.SugaredLogger
	workerName            string
	recorder              record.EventRecorder
	versions              kubermatic.Versions
	userGrafanaController *userGrafanaController
}

func newUserGrafanaReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	userGrafanaController *userGrafanaController,
) error {
	log = log.Named(ControllerName)
	client := mgr.GetClient()

	reconciler := &userGrafanaReconciler{
		Client: client,

		log:                   log,
		workerName:            workerName,
		recorder:              mgr.GetEventRecorderFor(ControllerName),
		versions:              versions,
		userGrafanaController: userGrafanaController,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.User{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch Users: %w", err)
	}
	return err
}

func (r *userGrafanaReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	user := &kubermaticv1.User{}
	if err := r.Get(ctx, request.NamespacedName, user); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !user.DeletionTimestamp.IsZero() {
		if err := r.userGrafanaController.handleDeletion(ctx, user); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if !kubernetes.HasFinalizer(user, mlaFinalizer) {
		kubernetes.AddFinalizer(user, mlaFinalizer)
		if err := r.Update(ctx, user); err != nil {
			return reconcile.Result{}, fmt.Errorf("updating finalizers: %w", err)
		}
	}

	if err := r.userGrafanaController.ensureGrafanaUser(ctx, user); err != nil {
		return reconcile.Result{}, fmt.Errorf("unable to add grafana user : %w", err)
	}
	return reconcile.Result{}, nil
}

type userGrafanaController struct {
	ctrlruntimeclient.Client
	grafanaClient *grafanasdk.Client
	httpClient    *http.Client

	log           *zap.SugaredLogger
	grafanaURL    string
	grafanaHeader string
}

func newUserGrafanaController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	grafanaClient *grafanasdk.Client,
	httpClient *http.Client,
	grafanaURL string,
	grafanaHeader string,
) *userGrafanaController {

	return &userGrafanaController{
		Client:        client,
		grafanaClient: grafanaClient,
		httpClient:    httpClient,

		log:           log,
		grafanaURL:    grafanaURL,
		grafanaHeader: grafanaHeader,
	}
}

func (r *userGrafanaController) cleanUp(ctx context.Context) error {
	userList := &kubermaticv1.UserList{}
	if err := r.List(ctx, userList); err != nil {
		return err
	}
	for _, user := range userList.Items {
		if err := r.handleDeletion(ctx, &user); err != nil {
			return err
		}
	}
	return nil
}

func (r *userGrafanaController) handleDeletion(ctx context.Context, user *kubermaticv1.User) error {
	grafanaUser, err := r.grafanaClient.LookupUser(ctx, user.Spec.Email)
	if err != nil && !errors.As(err, &grafanasdk.ErrNotFound{}) {
		return err
	}
	if err == nil {
		status, err := r.grafanaClient.DeleteGlobalUser(ctx, grafanaUser.ID)
		if err != nil {
			return fmt.Errorf("unable to delete user: %w (status: %s, message: %s)",
				err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
	}
	if kubernetes.HasFinalizer(user, mlaFinalizer) {
		kubernetes.RemoveFinalizer(user, mlaFinalizer)
		if err := r.Update(ctx, user); err != nil {
			return fmt.Errorf("updating User: %w", err)
		}
	}
	return nil
}

func (r *userGrafanaController) ensureGrafanaUser(ctx context.Context, user *kubermaticv1.User) error {
	req, err := http.NewRequest("GET", r.grafanaURL+"/api/user", nil)
	if err != nil {
		return err
	}
	req.Header.Add(r.grafanaHeader, user.Spec.Email)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	grafanaUser := &grafanasdk.User{}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(grafanaUser); err != nil || grafanaUser.ID == 0 {
		return fmt.Errorf("unable to decode response : %w", err)
	}

	// delete user from default org
	if status, err := r.grafanaClient.DeleteOrgUser(ctx, defaultOrgID, grafanaUser.ID); err != nil {
		return fmt.Errorf("failed to delete grafana user from default org: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
	}
	if grafanaUser.IsGrafanaAdmin != user.Spec.IsAdmin {
		grafanaUser.IsGrafanaAdmin = user.Spec.IsAdmin
		projectList := &kubermaticv1.ProjectList{}
		if err := r.List(ctx, projectList); err != nil {
			return err
		}
		// we also needs to remove user if IsAdmin is false, but keep in orgs with userprojectbingings
		for _, project := range projectList.Items {
			org, err := getOrgByProject(ctx, r.grafanaClient, &project)
			if err != nil {
				return err
			}
			if grafanaUser.IsGrafanaAdmin {
				if err := addUserToOrg(ctx, r.grafanaClient, org, grafanaUser, models.ROLE_ADMIN); err != nil {
					return err
				}
			} else {
				if err := removeUserFromOrg(ctx, r.grafanaClient, org, grafanaUser); err != nil {
					return err
				}
			}
		}
		if !grafanaUser.IsGrafanaAdmin {
			userProjectBindingList := &kubermaticv1.UserProjectBindingList{}
			if err := r.List(ctx, userProjectBindingList); err != nil {
				return err
			}
			for _, userProjectBinding := range userProjectBindingList.Items {
				if userProjectBinding.Spec.UserEmail != user.Spec.Email {
					continue
				}
				project := &kubermaticv1.Project{}
				if err := r.Get(ctx, types.NamespacedName{Name: userProjectBinding.Spec.ProjectID}, project); err != nil {
					return fmt.Errorf("failed to get project: %w", err)
				}
				if err := ensureOrgUser(ctx, r.grafanaClient, project, &userProjectBinding); err != nil {
					return err
				}
			}
		}
		status, err := r.grafanaClient.UpdateUserPermissions(ctx, grafanasdk.UserPermissions{IsGrafanaAdmin: user.Spec.IsAdmin}, grafanaUser.ID)
		if err != nil {
			return fmt.Errorf("failed to update user permissions: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
		}
	}
	return nil
}
