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
	"strings"
	"time"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
	subname := "grafana-user"

	reconciler := &userGrafanaReconciler{
		Client: client,

		log:                   log.Named(subname),
		workerName:            workerName,
		recorder:              mgr.GetEventRecorderFor(controllerName(subname)),
		versions:              versions,
		userGrafanaController: userGrafanaController,
	}

	serviceAccountPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// We don't trigger reconciliation for service account.
		user := object.(*kubermaticv1.User)
		return !kubermaticv1helper.IsProjectServiceAccount(user.Name)
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName(subname)).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.User{}, builder.WithPredicates(serviceAccountPredicate)).
		Watches(&kubermaticv1.UserProjectBinding{}, handler.EnqueueRequestsFromMapFunc(enqueueUserForUserProjectBinding(reconciler))).
		Watches(&kubermaticv1.GroupProjectBinding{}, handler.EnqueueRequestsFromMapFunc(enqueueUserForGroupProjectBinding(reconciler))).
		Build(reconciler)

	return err
}

// enqueueUserForUserProjectBinding enqueues users connected with the userprojectbinding.
func enqueueUserForUserProjectBinding(c ctrlruntimeclient.Client) func(context.Context, ctrlruntimeclient.Object) []reconcile.Request {
	return func(ctx context.Context, o ctrlruntimeclient.Object) []reconcile.Request {
		var res []reconcile.Request
		upb, ok := o.(*kubermaticv1.UserProjectBinding)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("object is not an UserProjectBinding: %T", o))
			return res
		}

		userList := &kubermaticv1.UserList{}
		if err := c.List(ctx, userList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list users: %w", err))
			return res
		}
		for _, user := range userList.Items {
			// Skip service accounts
			if kubermaticv1helper.IsProjectServiceAccount(user.Name) {
				continue
			}
			if upb.Spec.UserEmail == user.Spec.Email {
				res = append(res, reconcile.Request{NamespacedName: types.NamespacedName{Name: user.Name, Namespace: user.Namespace}})
			}
		}
		return res
	}
}

// enqueueUserForGroupProjectBinding enqueues users connected with the groupprojectbinding.
func enqueueUserForGroupProjectBinding(c ctrlruntimeclient.Client) func(context.Context, ctrlruntimeclient.Object) []reconcile.Request {
	return func(ctx context.Context, o ctrlruntimeclient.Object) []reconcile.Request {
		var res []reconcile.Request
		gpb, ok := o.(*kubermaticv1.GroupProjectBinding)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("object is not an GroupProjectBinding: %T", o))
			return res
		}

		userList := &kubermaticv1.UserList{}
		if err := c.List(ctx, userList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list users: %w", err))
			return res
		}
		for _, user := range userList.Items {
			// Skip service accounts
			if kubermaticv1helper.IsProjectServiceAccount(user.Name) {
				continue
			}
			if slices.Contains(user.Spec.Groups, gpb.Spec.Group) {
				res = append(res, reconcile.Request{NamespacedName: types.NamespacedName{Name: user.Name, Namespace: user.Namespace}})
			}
		}
		return res
	}
}

func (r *userGrafanaReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	user := &kubermaticv1.User{}
	if err := r.Get(ctx, request.NamespacedName, user); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	grafanaClient, err := r.userGrafanaController.clientProvider(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	if !user.DeletionTimestamp.IsZero() {
		if err := r.userGrafanaController.handleDeletion(ctx, user, grafanaClient); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if grafanaClient == nil {
		return reconcile.Result{}, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r, user, mlaFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.userGrafanaController.ensureGrafanaUser(ctx, user, grafanaClient); err != nil {
		if strings.Contains(err.Error(), "project should have grafana org annotation set") {
			log.Warnf("unable to ensure Grafana User, retrying in 30s: %s", err.Error())
			return reconcile.Result{RequeueAfter: time.Second * 30}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana User: %w", err)
	}

	return reconcile.Result{}, nil
}

type userGrafanaController struct {
	ctrlruntimeclient.Client
	clientProvider grafanaClientProvider
	httpClient     *http.Client

	log           *zap.SugaredLogger
	grafanaURL    string
	grafanaHeader string
}

func newUserGrafanaController(
	client ctrlruntimeclient.Client,
	log *zap.SugaredLogger,
	clientProvider grafanaClientProvider,
	httpClient *http.Client,
	grafanaURL string,
	grafanaHeader string,
) *userGrafanaController {
	return &userGrafanaController{
		Client:         client,
		clientProvider: clientProvider,
		httpClient:     httpClient,

		log:           log,
		grafanaURL:    grafanaURL,
		grafanaHeader: grafanaHeader,
	}
}

func (r *userGrafanaController) CleanUp(ctx context.Context) error {
	userList := &kubermaticv1.UserList{}
	if err := r.List(ctx, userList); err != nil {
		return err
	}
	grafanaClient, err := r.clientProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}
	for _, user := range userList.Items {
		if err := r.handleDeletion(ctx, &user, grafanaClient); err != nil {
			return fmt.Errorf("failed to handle Grafana user cleanup for User %s: %w", user.Name, err)
		}
	}
	return nil
}

func (r *userGrafanaController) handleDeletion(ctx context.Context, user *kubermaticv1.User, grafanaClient *grafanasdk.Client) error {
	if grafanaClient != nil {
		grafanaUser, err := grafanaClient.LookupUser(ctx, user.Spec.Email)
		if err != nil && !errors.As(err, &grafanasdk.ErrNotFound{}) {
			return err
		}
		if err == nil {
			status, err := grafanaClient.DeleteUser(ctx, grafanaUser.ID)
			if err != nil {
				return fmt.Errorf("unable to delete user: %w (status: %s, message: %s)",
					err, ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r, user, mlaFinalizer)
}

func (r *userGrafanaController) ensureGrafanaUser(ctx context.Context, user *kubermaticv1.User, grafanaClient *grafanasdk.Client) error {
	// get user
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.grafanaURL+"/api/user", nil)
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
	if err := decoder.Decode(grafanaUser); err != nil {
		return fmt.Errorf("unable to decode response: %w", err)
	}
	if grafanaUser.ID == 0 {
		return fmt.Errorf("user %q was not found", user.Spec.Email)
	}

	// delete user from default org
	if status, err := grafanaClient.DeleteOrgUser(ctx, defaultOrgID, grafanaUser.ID); err != nil {
		return fmt.Errorf("failed to delete grafana user from default org: %w (status: %s, message: %s)", err,
			ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
	}

	projectList := &kubermaticv1.ProjectList{}
	if err := r.List(ctx, projectList); err != nil {
		return err
	}

	// if admin flipped, give/remove user from all orgs and update grafana admin
	if grafanaUser.IsGrafanaAdmin != user.Spec.IsAdmin {
		grafanaUser.IsGrafanaAdmin = user.Spec.IsAdmin

		// we also needs to remove user if IsAdmin is false, but keep in orgs with userprojectbingings
		for _, project := range projectList.Items {
			org, err := getOrgByProject(ctx, grafanaClient, &project)
			if err != nil {
				return err
			}
			if grafanaUser.IsGrafanaAdmin {
				if err := addUserToOrg(ctx, grafanaClient, org, grafanaUser, grafanasdk.ROLE_EDITOR); err != nil {
					return err
				}
			} else {
				if err := removeUserFromOrg(ctx, grafanaClient, org, grafanaUser); err != nil {
					return err
				}
			}
		}
		status, err := grafanaClient.UpdateUserPermissions(ctx, grafanasdk.UserPermissions{IsGrafanaAdmin: user.Spec.IsAdmin}, grafanaUser.ID)
		if err != nil {
			return fmt.Errorf("failed to update user permissions: %w (status: %s, message: %s)", err,
				ptr.Deref(status.Status, "no status"), ptr.Deref(status.Message, "no message"))
		}
	}

	// handle regular user
	projectMap := map[string]*kubermaticv1.Project{}
	for _, project := range projectList.Items {
		projectMap[project.Name] = project.DeepCopy()
	}
	if !user.Spec.IsAdmin {
		projectRoles, err := getProjectRolesForUser(ctx, r, user)
		if err != nil {
			return fmt.Errorf("error getting project roles for user %q: %w", user.Name, err)
		}

		// add to project orgs
		for projectName, role := range projectRoles {
			project, ok := projectMap[projectName]
			if !ok {
				// don't stop reconciling if there is an upb/gbp which was not cleaned up properly
				r.log.Warnw("user-bound project not found", "user", user.Name, "project", projectName)
				continue
			}
			if err := ensureOrgUser(ctx, grafanaClient, project, user.Spec.Email, role); err != nil {
				return err
			}
			// handled, remove the key for pruning phase
			delete(projectMap, projectName)
		}

		// Prune from project orgs user does not belong to.
		// This is not very effective as it goes through all orgs and deletes the user which may not be present
		// from them. Unfortunately there is no API to get all orgs for an user so that we could compare it.
		for _, project := range projectMap {
			org, err := getOrgByProject(ctx, grafanaClient, project)
			if err != nil {
				return err
			}
			if err := removeUserFromOrg(ctx, grafanaClient, org, grafanaUser); err != nil {
				return err
			}
		}
	}

	return nil
}

func getProjectRolesForUser(ctx context.Context, client ctrlruntimeclient.Client, user *kubermaticv1.User) (map[string]grafanasdk.RoleType, error) {
	projectMap := make(map[string]grafanasdk.RoleType)

	// get projects/roles by userProjectBinding
	upbList := &kubermaticv1.UserProjectBindingList{}
	if err := client.List(ctx, upbList); err != nil {
		return projectMap, err
	}
	for _, upb := range upbList.Items {
		if upb.Spec.UserEmail == user.Spec.Email {
			projectMap[upb.Spec.ProjectID] = groupToRole[rbac.ExtractGroupPrefix(upb.Spec.Group)]
		}
	}

	// get projects/roles by groupProjectBinding
	gpbList := &kubermaticv1.GroupProjectBindingList{}
	if err := client.List(ctx, gpbList); err != nil {
		return projectMap, err
	}
	userGroups := sets.New(user.Spec.Groups...)
	for _, gpb := range gpbList.Items {
		if userGroups.Has(gpb.Spec.Group) {
			role := groupToRole[gpb.Spec.Role]

			if upbRole, ok := projectMap[gpb.Spec.ProjectID]; ok && role != upbRole {
				// we use 2 roles in grafana viewer and editor, so if the roles are different,
				// means they are not both viewers, so we can set editor role here.
				role = grafanasdk.ROLE_EDITOR
			}
			projectMap[gpb.Spec.ProjectID] = role
		}
	}

	return projectMap, nil
}
