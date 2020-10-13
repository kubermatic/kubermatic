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

package constrainttemplatecontroller

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller syncs the kubermatic constraint templates to gatekeeper constraint templates on the user cluster.
	controllerName = "gatekeeper_constraint_template_controller"
)

type reconciler struct {
	ctx        context.Context
	log        *zap.SugaredLogger
	userClient ctrlruntimeclient.Client
	seedClient ctrlruntimeclient.Client
	recorder   record.EventRecorder
}

func Add(ctx context.Context, log *zap.SugaredLogger, userMgr, seedMgr manager.Manager) error {
	log = log.Named(controllerName)

	r := &reconciler{
		ctx:        ctx,
		log:        log,
		userClient: userMgr.GetClient(),
		seedClient: seedMgr.GetClient(),
		recorder:   userMgr.GetEventRecorderFor(controllerName),
	}
	c, err := controller.New(controllerName, userMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	ctSource := &source.Kind{Type: &v1.ConstraintTemplate{}}
	if err := ctSource.InjectCache(seedMgr.GetCache()); err != nil {
		return fmt.Errorf("failed to inject seed cache into watch: %v", err)
	}

	// Watch for changes to ConstraintTemplates
	if err = c.Watch(ctSource, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to establish watch for the ConstraintTemplates %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("ConstraintTemplate", request)
	log.Debug("Reconciling")

	constraintTemplate := &v1.ConstraintTemplate{}
	if err := r.seedClient.Get(r.ctx, request.NamespacedName, constraintTemplate); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debugw("constraint template not found, returning", "constraint-name", constraintTemplate.Name)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get constraint template %s: %v", constraintTemplate.Name, err)
	}

	err := r.reconcile(constraintTemplate)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(constraintTemplate, corev1.EventTypeWarning, "Reconciling failed", err.Error())
	}
	return reconcile.Result{}, err
}

// reconcile reconciles the kubermatic constraint template on the seed cluster to the gatekeeper one on the user cluster.
// For now without filters but can be added.
func (r *reconciler) reconcile(constraintTemplate *v1.ConstraintTemplate) error {

	if constraintTemplate.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperConstraintTemplateCleanupFinalizer) {
			return nil
		}
		if err := r.userClient.Delete(r.ctx, &v1beta1.ConstraintTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: constraintTemplate.Name,
			},
		}); err != nil {
			return fmt.Errorf("failed to delete constraint template %s: %v", constraintTemplate.Name, err)
		}

		kuberneteshelper.RemoveFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperConstraintTemplateCleanupFinalizer)
		if err := r.seedClient.Update(r.ctx, constraintTemplate); err != nil {
			return fmt.Errorf("failed to remove constraint template finalizer %s: %v", constraintTemplate.Name, err)
		}
		return nil
	}

	if !kuberneteshelper.HasFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperConstraintTemplateCleanupFinalizer) {
		kuberneteshelper.AddFinalizer(constraintTemplate, kubermaticapiv1.GatekeeperConstraintTemplateCleanupFinalizer)
		if err := r.seedClient.Update(r.ctx, constraintTemplate); err != nil {
			return fmt.Errorf("failed to set constraint template finalizer %s: %v", constraintTemplate.Name, err)
		}
	}
	ctCreatorGetters := []reconciling.NamedConstraintTemplateCreatorGetter{
		constraintTemplateCreatorGetter(constraintTemplate),
	}

	return reconciling.ReconcileConstraintTemplates(r.ctx, ctCreatorGetters, "", r.userClient)
}

func constraintTemplateCreatorGetter(kubeCT *v1.ConstraintTemplate) reconciling.NamedConstraintTemplateCreatorGetter {
	return func() (string, reconciling.ConstraintTemplateCreator) {
		return kubeCT.Name, func(ct *v1beta1.ConstraintTemplate) (*v1beta1.ConstraintTemplate, error) {
			ct.Name = kubeCT.Name
			ct.Spec = kubeCT.Spec

			return ct, nil
		}
	}
}
