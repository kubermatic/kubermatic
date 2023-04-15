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

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/api/v3/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/mla-controller/grafana"
	"k8c.io/kubermatic/v3/pkg/kubernetes"

	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Create/update/delete Grafana Users based on KKP Users.
type grafanaUserReconciler struct {
	seedClient     ctrlruntimeclient.Client
	log            *zap.SugaredLogger
	workerName     string
	recorder       record.EventRecorder
	clientProvider grafana.ClientProvider
}

var _ cleaner = &grafanaUserReconciler{}

func newGrafanaUserReconciler(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	clientProvider grafana.ClientProvider,
) *grafanaUserReconciler {
	return &grafanaUserReconciler{
		seedClient:     mgr.GetClient(),
		log:            log.Named("grafana-user"),
		recorder:       mgr.GetEventRecorderFor(ControllerName),
		clientProvider: clientProvider,
		workerName:     workerName,
	}
}

func (r *grafanaUserReconciler) Start(ctx context.Context, mgr manager.Manager, workers int) error {
	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: workers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	serviceAccountPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// We don't trigger reconciliation for service account.
		user := object.(*kubermaticv1.User)
		return !kubermaticv1helper.IsProjectServiceAccount(user.Spec.Email)
	})

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.User{}}, &handler.EnqueueRequestForObject{}, serviceAccountPredicate); err != nil {
		return fmt.Errorf("failed to watch Users: %w", err)
	}

	return err
}

func (r *grafanaUserReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("user", request.NamespacedName)
	log.Debug("Processing")

	user := &kubermaticv1.User{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, user); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	gClient, err := r.clientProvider(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	if !user.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, r.handleDeletion(ctx, user, gClient)
	}

	if gClient == nil {
		return reconcile.Result{}, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r.seedClient, user, mlaFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	if err := r.ensureGrafanaUser(ctx, log, user, gClient); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to ensure Grafana User: %w", err)
	}

	return reconcile.Result{}, nil
}

func (r *grafanaUserReconciler) Cleanup(ctx context.Context, log *zap.SugaredLogger) error {
	userList := &kubermaticv1.UserList{}
	if err := r.seedClient.List(ctx, userList); err != nil {
		return err
	}
	gClient, err := r.clientProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}
	for _, user := range userList.Items {
		if err := r.handleDeletion(ctx, &user, gClient); err != nil {
			return err
		}
	}
	return nil
}

func (r *grafanaUserReconciler) handleDeletion(ctx context.Context, user *kubermaticv1.User, gClient grafana.Client) error {
	if gClient != nil {
		grafanaUser, err := gClient.LookupUser(ctx, user.Spec.Email)
		if err != nil && !grafana.IsNotFoundErr(err) {
			return fmt.Errorf("failed to lookup user: %w", err)
		}

		if err == nil {
			_, err := gClient.DeleteGlobalUser(ctx, grafanaUser.ID)
			if err != nil && !grafana.IsNotFoundErr(err) {
				return fmt.Errorf("unable to delete user: %w", err)
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r.seedClient, user, mlaFinalizer)
}

func (r *grafanaUserReconciler) ensureGrafanaUser(ctx context.Context, log *zap.SugaredLogger, user *kubermaticv1.User, gClient grafana.Client) error {
	userLog := log.With("email", user.Spec.Email)

	// get user
	grafanaUser, err := gClient.LookupUser(ctx, user.Spec.Email)
	if err != nil {
		// user does not yet, create them
		userLog.Info("Creating Grafana OAuth user")
		newUser, err := gClient.CreateOAuthUser(ctx, user.Spec.Email)
		if err != nil {
			return fmt.Errorf("failed to create global Grafana user: %w", err)
		}

		grafanaUser = *newUser

		// remove user from default organization
		userLog.Info("Removing new Grafana user from default organization")
		if _, err := gClient.DeleteOrgUser(ctx, grafana.DefaultOrgID, newUser.ID); err != nil {
			return fmt.Errorf("failed to delete new user from default organization: %w", err)
		}
	}

	if grafanaUser.IsGrafanaAdmin != user.Spec.IsAdmin {
		userLog.Infow("Updating Grafana user permissions", "admin", user.Spec.IsAdmin)
		if _, err := gClient.UpdateUserPermissions(ctx, grafanasdk.UserPermissions{IsGrafanaAdmin: user.Spec.IsAdmin}, grafanaUser.ID); err != nil {
			return fmt.Errorf("failed to update user permissions: %w", err)
		}
	}

	// get org
	org, err := gClient.GetOrgByOrgName(ctx, GrafanaOrganization)
	if err != nil {
		if grafana.IsNotFoundErr(err) {
			log.Debug("Organization not found.")
			return nil
		}

		return fmt.Errorf("failed to get Grafana organization %q: %w", GrafanaOrganization, err)
	}

	// add user to org
	if err := ensureUserInOrgWithRole(ctx, userLog, gClient, org, &grafanaUser, getRoleForUser(user)); err != nil {
		return fmt.Errorf("failed to get Grafana organization %q: %w", GrafanaOrganization, err)
	}

	return nil
}

func getRoleForUser(user *kubermaticv1.User) grafanasdk.RoleType {
	if user.Spec.IsAdmin {
		return grafanasdk.ROLE_ADMIN
	}

	return grafanasdk.ROLE_EDITOR
}

func ensureUserInOrgWithRole(ctx context.Context, log *zap.SugaredLogger, gClient grafana.Client, org grafanasdk.Org, user *grafanasdk.User, role grafanasdk.RoleType) error {
	// check if user already exists in the corresponding organization
	orgUser, err := gClient.GetOrgUser(ctx, org.ID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// if there is no such user in organization, let's add one
	if orgUser == nil {
		userRole := grafanasdk.UserRole{
			LoginOrEmail: user.Email,
			Role:         string(role),
		}

		log.Infow("Adding Grafana user to organization", "role", role)
		if _, err := gClient.AddOrgUser(ctx, userRole, org.ID); err != nil {
			return fmt.Errorf("failed to add grafana user to org: %w", err)
		}

		return nil
	}

	if orgUser.Role != string(role) {
		userRole := grafanasdk.UserRole{
			LoginOrEmail: user.Email,
			Role:         string(role),
		}
		log.Infow("Updating Grafana user's role in organization", "role", role)
		if _, err := gClient.UpdateOrgUser(ctx, userRole, org.ID, orgUser.ID); err != nil {
			return fmt.Errorf("failed to update grafana user role: %w", err)
		}
	}

	return nil
}
