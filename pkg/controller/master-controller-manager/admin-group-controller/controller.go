/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package admingroupcontroller

import (
	"context"
	"fmt"
	"reflect"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	utilpredicate "k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const ControllerName = "kkp-admin-group-controller"

type reconciler struct {
	log             *zap.SugaredLogger
	recorder        events.EventRecorder
	masterClient    ctrlruntimeclient.Client
	masterAPIReader ctrlruntimeclient.Reader
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	r := &reconciler{
		log:             log.Named(ControllerName),
		recorder:        mgr.GetEventRecorder(ControllerName),
		masterClient:    mgr.GetClient(),
		masterAPIReader: mgr.GetAPIReader(),
	}

	serviceAccountPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// Service accounts have no OIDC login and must never be escalated.
		user := object.(*kubermaticv1.User)
		return !kubermaticv1helper.IsProjectServiceAccount(user.Spec.Email)
	})

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.User{}, builder.WithPredicates(serviceAccountPredicate, withUserEventFilter())).
		Watches(&kubermaticv1.KubermaticSetting{}, enqueueAllUsers(r.masterClient), builder.WithPredicates(utilpredicate.ByName(kubermaticv1.GlobalSettingsName), withSettingsEventFilter())).
		Build(r)

	return err
}

// Reconcile keeps a single User's admin status in sync with the configured admin groups.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	user := &kubermaticv1.User{}
	// Bypass the cache: the controller mutates the same User objects it watches, so a
	// stale cache read could overwrite a just-written change.
	if err := r.masterAPIReader.Get(ctx, request.NamespacedName, user); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	// Never touch service accounts or users being deleted.
	if kubermaticv1helper.IsProjectServiceAccount(user.Spec.Email) || !user.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	if err := r.reconcile(ctx, log, user); err != nil {
		r.recorder.Eventf(user, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconciled user %s: %w", user.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, user *kubermaticv1.User) error {
	adminGroups, err := r.adminGroups(ctx)
	if err != nil {
		return err
	}

	current, annotated := user.Annotations[kubermaticv1.AdminGrantedByGroupAnnotation]
	matched := matchAdminGroup(adminGroups, user.Spec.Groups, current)

	switch {
	case matched != "":
		// A configured group grants admin to this user.
		if !annotated {
			// The annotation is absent: this is a manual admin (or a not-yet-promoted
			// user). Only ever promote a user we have not touched if they are not
			// already admin; a manual admin who also sits in a listed group is left
			// alone and never annotated (hands-off contract).
			if user.Spec.IsAdmin {
				return nil
			}
		} else if current == matched && user.Spec.IsAdmin {
			// Already promoted via this exact group; nothing to do.
			return nil
		}

		// admin and globalViewer are mutually exclusive (rejected by the User webhook),
		// so a global viewer is never promoted.
		if user.Spec.IsGlobalViewer {
			log.Infow("skipping admin promotion: user is a global viewer", "user", user.Name)
			r.recorder.Eventf(user, nil, corev1.EventTypeWarning, "AdminSkipped", "Skipping",
				"not granting admin via group %q: user is a global viewer", matched)
			return nil
		}

		return r.setAdmin(ctx, user, true, matched)

	case annotated:
		// No configured group matches, but the user carries our annotation: demote.
		return r.setAdmin(ctx, user, false, "")

	default:
		// No match and no annotation: hands off (manual admins, first-user auto-admin).
		return nil
	}
}

// adminGroups returns the configured admin group names, or an empty list when the
// globalsettings object does not exist.
func (r *reconciler) adminGroups(ctx context.Context) ([]string, error) {
	settings := &kubermaticv1.KubermaticSetting{}
	if err := r.masterClient.Get(ctx, types.NamespacedName{Name: kubermaticv1.GlobalSettingsName}, settings); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get global settings: %w", err)
	}
	return settings.Spec.AdminGroups, nil
}

func (r *reconciler) setAdmin(ctx context.Context, user *kubermaticv1.User, admin bool, group string) error {
	oldUser := user.DeepCopy()

	user.Spec.IsAdmin = admin
	if admin {
		if user.Annotations == nil {
			user.Annotations = map[string]string{}
		}
		user.Annotations[kubermaticv1.AdminGrantedByGroupAnnotation] = group
	} else {
		delete(user.Annotations, kubermaticv1.AdminGrantedByGroupAnnotation)
	}

	if err := r.masterClient.Patch(ctx, user, ctrlruntimeclient.MergeFrom(oldUser)); err != nil {
		return fmt.Errorf("failed to update user admin status: %w", err)
	}

	if admin {
		r.recorder.Eventf(user, nil, corev1.EventTypeNormal, "AdminGranted", "Granting", "granted admin via group %q", group)
	} else {
		r.recorder.Eventf(user, nil, corev1.EventTypeNormal, "AdminDemoted", "Demoting", "revoked group-granted admin")
	}

	return nil
}

// matchAdminGroup returns the admin group that should grant this user admin, or ""
// if none does. It prefers the currently-annotated group when it is still both
// configured and carried by the user (avoiding a needless re-stamp), otherwise the
// first configured group the user belongs to. Matching is exact and case-sensitive.
func matchAdminGroup(adminGroups, userGroups []string, current string) string {
	if current != "" && containsString(adminGroups, current) && containsString(userGroups, current) {
		return current
	}
	for _, g := range adminGroups {
		if containsString(userGroups, g) {
			return g
		}
	}
	return ""
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// enqueueAllUsers requeues every User whenever the admin groups change; the settings
// object carries no per-user identity, so all users must be re-evaluated.
func enqueueAllUsers(client ctrlruntimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, _ ctrlruntimeclient.Object) []reconcile.Request {
		users := &kubermaticv1.UserList{}
		if err := client.List(ctx, users); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list users: %w", err))
			return nil
		}

		requests := make([]reconcile.Request, 0, len(users.Items))
		for _, user := range users.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{Name: user.Name}})
		}
		return requests
	})
}

// withUserEventFilter requeues a User only when a field the controller reacts to or
// writes has changed, so the controller does not loop on its own updates.
func withUserEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldUser, ok := e.ObjectOld.(*kubermaticv1.User)
			if !ok {
				return false
			}
			newUser, ok := e.ObjectNew.(*kubermaticv1.User)
			if !ok {
				return false
			}
			return !reflect.DeepEqual(oldUser.Spec.Groups, newUser.Spec.Groups) ||
				oldUser.Spec.IsAdmin != newUser.Spec.IsAdmin ||
				oldUser.Annotations[kubermaticv1.AdminGrantedByGroupAnnotation] != newUser.Annotations[kubermaticv1.AdminGrantedByGroupAnnotation]
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

// withSettingsEventFilter requeues only when the admin groups list changes; a
// settings deletion counts as clearing the list (annotated admins get demoted).
func withSettingsEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSettings, ok := e.ObjectOld.(*kubermaticv1.KubermaticSetting)
			if !ok {
				return false
			}
			newSettings, ok := e.ObjectNew.(*kubermaticv1.KubermaticSetting)
			if !ok {
				return false
			}
			return !reflect.DeepEqual(oldSettings.Spec.AdminGroups, newSettings.Spec.AdminGroups)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}
