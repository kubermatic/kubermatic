/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package ownerbindingcreator

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller creates cluster role bindings for the API roles.
	controllerName = "kkp-owner-binding-creator"
)

type reconciler struct {
	log             *zap.SugaredLogger
	client          ctrlruntimeclient.Client
	recorder        events.EventRecorder
	ownerEmail      string
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(log *zap.SugaredLogger, mgr manager.Manager, ownerEmail string, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		client:          mgr.GetClient(),
		recorder:        mgr.GetEventRecorder(controllerName),
		ownerEmail:      ownerEmail,
		clusterIsPaused: clusterIsPaused,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName).
		For(&rbacv1.ClusterRole{}, builder.WithPredicates(predicateutil.ByLabel(userclustercontrollermanager.UserClusterComponentKey, userclustercontrollermanager.UserClusterRoleComponentValue))).
		Watches(&rbacv1.ClusterRoleBinding{}, enqueueAPIBindings(mgr.GetClient()), builder.WithPredicates(predicateutil.ByLabel(userclustercontrollermanager.UserClusterComponentKey, userclustercontrollermanager.UserClusterBindingComponentValue))).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("clusterrole", request.Name)
	log.Debug("Reconciling")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	err = r.reconcile(ctx, log, request.Name)
	if err != nil {
		r.recorder.Eventf(&rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: request.Name}}, nil, corev1.EventTypeWarning, "AddBindingFailed", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, clusterRoleName string) error {
	labels := map[string]string{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterBindingComponentValue}

	clusterRoleBindingList := &rbacv1.ClusterRoleBindingList{}
	if err := r.client.List(ctx, clusterRoleBindingList, ctrlruntimeclient.MatchingLabels(labels)); err != nil {
		return fmt.Errorf("failed to list ClusterRoleBindings: %w", err)
	}

	var existingClusterRoleBinding *rbacv1.ClusterRoleBinding
	for _, clusterRoleBinding := range clusterRoleBindingList.Items {
		if clusterRoleBinding.RoleRef.Name == clusterRoleName {
			existingClusterRoleBinding = clusterRoleBinding.DeepCopy()
			break
		}
	}

	// Create Cluster Role Binding if doesn't exist
	if existingClusterRoleBinding == nil {
		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:   fmt.Sprintf("%s:%s", rand.String(10), clusterRoleName),
				Labels: labels,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "ClusterRole",
				Name:     clusterRoleName,
			},
		}

		// Bind user who created the cluster to the `cluster-admin` ClusterRole.
		// Add cluster owner only once when binding doesn't exist yet.
		// Later the user can remove/add subjects from the binding using the API.
		if clusterRoleName == "cluster-admin" {
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					APIGroup: rbacv1.GroupName,
					Name:     r.ownerEmail,
				},
			}
		}

		log.Infow("Creating ClusterRoleBinding", "name", crb.Name)
		if err := r.client.Create(ctx, crb); err != nil {
			return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
		}
	}

	return nil
}

// enqueueAPIBindings enqueues the ClusterRoleBindings with a special label component=userClusterRole.
func enqueueAPIBindings(client ctrlruntimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, _ ctrlruntimeclient.Object) []reconcile.Request {
		clusterRoleList := &rbacv1.ClusterRoleList{}
		if err := client.List(ctx, clusterRoleList, ctrlruntimeclient.MatchingLabels{userclustercontrollermanager.UserClusterComponentKey: userclustercontrollermanager.UserClusterRoleComponentValue}); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list ClusterRoles: %w", err))
			return []reconcile.Request{}
		}

		var requests []reconcile.Request
		for _, clusterRole := range clusterRoleList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: clusterRole.Name}})
		}
		return requests
	})
}
