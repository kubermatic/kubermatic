/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package usersshkeyprojectownershipcontroller

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v3/pkg/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kkp-usersshkey-project-ownership-controller"

// reconcileSyncProjectBinding reconciles UserSSHKey objects and ensures that
// they are owned by their parent project.
type reconcileSyncProjectBinding struct {
	ctrlruntimeclient.Client

	recorder record.EventRecorder
	log      *zap.SugaredLogger
}

func Add(mgr manager.Manager, log *zap.SugaredLogger) error {
	r := &reconcileSyncProjectBinding{
		Client: mgr.GetClient(),

		recorder: mgr.GetEventRecorderFor(ControllerName),
		log:      log,
	}

	// Create a new controller
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to UserSSHKey
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.UserSSHKey{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	// Notice when projects appear, then enqueue all keys that are in the project
	enqueueRelatedKeys := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		keyList := &kubermaticv1.UserSSHKeyList{}
		if err := mgr.GetClient().List(context.Background(), keyList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list UserSSHKeys: %w", err))
			return []reconcile.Request{}
		}

		requests := []reconcile.Request{}
		for _, key := range keyList.Items {
			if key.Spec.Project == a.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(&key),
				})
			}
		}

		return requests
	})

	// Only react to new projects
	onlyNewProjects := predicate.Funcs{
		CreateFunc: func(ce event.CreateEvent) bool {
			return true
		},
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, enqueueRelatedKeys, onlyNewProjects); err != nil {
		return err
	}

	return nil
}

func (r *reconcileSyncProjectBinding) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	sshKey := &kubermaticv1.UserSSHKey{}
	if err := r.Get(ctx, request.NamespacedName, sshKey); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log := r.log.With("usersshkey", sshKey.Name)
	log.Debug("Reconciling")

	err := r.reconcile(ctx, log, sshKey)
	if err != nil {
		r.recorder.Event(sshKey, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		log.Errorw("Reconciling failed", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconcileSyncProjectBinding) reconcile(ctx context.Context, log *zap.SugaredLogger, sshKey *kubermaticv1.UserSSHKey) error {
	if sshKey.DeletionTimestamp != nil {
		return nil
	}

	if sshKey.Spec.Project == "" {
		return errors.New("SSH key has no project name specified")
	}

	project := &kubermaticv1.Project{}
	err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Name: sshKey.Spec.Project}, project)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Debugw("Project does not exist", "project", sshKey.Spec.Project)
			return nil
		}

		return fmt.Errorf("failed to get project: %w", err)
	}

	oldKey := sshKey.DeepCopy()
	kuberneteshelper.EnsureUniqueOwnerReference(sshKey, metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       "Project",
		UID:        project.UID,
		Name:       project.Name,
	})
	kuberneteshelper.SortOwnerReferences(sshKey.OwnerReferences)

	if err := r.Patch(ctx, sshKey, ctrlruntimeclient.MergeFrom(oldKey)); err != nil {
		return fmt.Errorf("failed to ensure owner reference: %w", err)
	}

	return nil
}
