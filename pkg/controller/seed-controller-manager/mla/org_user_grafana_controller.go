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
	"strings"
	"time"

	"go.uber.org/zap"

	grafanasdk "github.com/kubermatic/grafanasdk"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type orgUserGrafanaReconciler struct {
	ctrlruntimeclient.Client

	log                      *zap.SugaredLogger
	workerName               string
	recorder                 record.EventRecorder
	versions                 kubermatic.Versions
	orgUserGrafanaController *orgUserGrafanaController
}

func newOrgUserGrafanaReconciler(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
	orgUserGrafanaController *orgUserGrafanaController,
) error {
	client := mgr.GetClient()

	// Add index on email flag
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kubermaticv1.User{}, "spec.email", func(rawObj ctrlruntimeclient.Object) []string {
		a := rawObj.(*kubermaticv1.User)
		return []string{a.Spec.Email}
	}); err != nil {
		return fmt.Errorf("failed to add index on User Email parameter: %w", err)
	}

	reconciler := &orgUserGrafanaReconciler{
		Client: client,

		log:                      log.Named("grafana-org-user"),
		workerName:               workerName,
		recorder:                 mgr.GetEventRecorderFor(ControllerName),
		versions:                 versions,
		orgUserGrafanaController: orgUserGrafanaController,
	}

	ctrlOptions := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	serviceAccountPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// We don't trigger reconciliation for UserProjectBinding of service account.
		userProjectBinding := object.(*kubermaticv1.UserProjectBinding)
		return !kubermaticv1helper.IsProjectServiceAccount(userProjectBinding.Spec.UserEmail)
	})

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.UserProjectBinding{}}, &handler.EnqueueRequestForObject{}, serviceAccountPredicate); err != nil {
		return fmt.Errorf("failed to watch UserProjectBindings: %w", err)
	}
	return err
}

func (r *orgUserGrafanaReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	userProjectBinding := &kubermaticv1.UserProjectBinding{}
	if err := r.Get(ctx, request.NamespacedName, userProjectBinding); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	grafanaClient, err := r.orgUserGrafanaController.clientProvider(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create Grafana client: %w", err)
	}

	if !userProjectBinding.DeletionTimestamp.IsZero() {
		if err := r.orgUserGrafanaController.handleDeletion(ctx, userProjectBinding, grafanaClient); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if grafanaClient == nil {
		return reconcile.Result{}, nil
	}

	if err := kubernetes.TryAddFinalizer(ctx, r, userProjectBinding, mlaFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	project := &kubermaticv1.Project{}
	if err := r.Get(ctx, types.NamespacedName{Name: userProjectBinding.Spec.ProjectID}, project); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get project: %w", err)
	}

	userList := &kubermaticv1.UserList{}
	if err := r.List(ctx, userList, ctrlruntimeclient.MatchingFields{"spec.email": userProjectBinding.Spec.UserEmail}); err != nil {
		log.Errorf("Failed to check for existence of a user %s: %s", userProjectBinding.Spec.UserEmail, err.Error())
		return reconcile.Result{}, err
	} else if len(userList.Items) == 0 {
		log.Warnf("User %s does not exist, ignoring UserProjectBinding %s for 30 minutes.", userProjectBinding.Spec.UserEmail, userProjectBinding.Name)
		return reconcile.Result{RequeueAfter: time.Minute * 30}, nil
	}

	if err := ensureOrgUser(ctx, grafanaClient, project, userProjectBinding.Spec.UserEmail,
		rbac.ExtractGroupPrefix(userProjectBinding.Spec.Group)); err != nil {
		if strings.Contains(err.Error(), "project should have grafana org annotation set") {
			log.Warnf("unable to ensure Grafana Org/User, retrying in 30s: %s", err.Error())
			return reconcile.Result{RequeueAfter: time.Second * 30}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to ensure project: %w", err)
	}

	return reconcile.Result{}, nil
}

type orgUserGrafanaController struct {
	ctrlruntimeclient.Client
	clientProvider grafanaClientProvider
	log            *zap.SugaredLogger
}

func newOrgUserGrafanaController(client ctrlruntimeclient.Client, log *zap.SugaredLogger, clientProvider grafanaClientProvider,
) *orgUserGrafanaController {
	return &orgUserGrafanaController{
		Client:         client,
		clientProvider: clientProvider,
		log:            log,
	}
}

func (r *orgUserGrafanaController) CleanUp(ctx context.Context) error {
	userProjectBindingList := &kubermaticv1.UserProjectBindingList{}
	if err := r.List(ctx, userProjectBindingList); err != nil {
		return err
	}
	grafanaClient, err := r.clientProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Grafana client: %w", err)
	}
	for _, userProjectBinding := range userProjectBindingList.Items {
		if err := r.handleDeletion(ctx, &userProjectBinding, grafanaClient); err != nil {
			return err
		}
	}
	return nil
}

func (r *orgUserGrafanaController) handleDeletion(ctx context.Context, userProjectBinding *kubermaticv1.UserProjectBinding, grafanaClient *grafanasdk.Client) error {
	if grafanaClient != nil {
		project := &kubermaticv1.Project{}
		if err := r.Get(ctx, types.NamespacedName{Name: userProjectBinding.Spec.ProjectID}, project); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get project: %w", err)
		}
		org, err := getOrgByProject(ctx, grafanaClient, project)
		if err == nil {
			user, err := grafanaClient.LookupUser(ctx, userProjectBinding.Spec.UserEmail)
			if err != nil && !errors.As(err, &grafanasdk.ErrNotFound{}) {
				return err
			}
			if err == nil {
				status, err := grafanaClient.DeleteOrgUser(ctx, org.ID, user.ID)
				if err != nil {
					return fmt.Errorf("failed to delete org user: %w (status: %s, message: %s)", err, pointer.StringPtrDerefOr(status.Status, "no status"), pointer.StringPtrDerefOr(status.Message, "no message"))
				}
			}
		}
	}

	return kubernetes.TryRemoveFinalizer(ctx, r, userProjectBinding, mlaFinalizer)
}
