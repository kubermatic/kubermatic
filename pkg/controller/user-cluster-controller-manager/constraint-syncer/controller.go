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

package constraintsyncer

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	constrainthandler "k8c.io/kubermatic/v2/pkg/handler/v2/constraint"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName       = "constraint_controller"
	constraintAPIVersion = "constraints.gatekeeper.sh/v1beta1"
	spec                 = "spec"
	parametersField      = "parameters"
	matchField           = "match"
)

type reconciler struct {
	log        *zap.SugaredLogger
	seedClient ctrlruntimeclient.Client
	userClient ctrlruntimeclient.Client
	recorder   record.EventRecorder
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, namespace string) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:        log,
		seedClient: seedMgr.GetClient(),
		userClient: userMgr.GetClient(),
		recorder:   userMgr.GetEventRecorderFor(controllerName),
	}
	c, err := controller.New(controllerName, seedMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	// Watch for changes to Constraints
	if err = c.Watch(
		&source.Kind{Type: &kubermaticv1.Constraint{}}, &handler.EnqueueRequestForObject{}, predicate.ByNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to establish watch for the Constraints %v", err)
	}

	return nil
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request)
	log.Debug("Reconciling")

	constraint := &kubermaticv1.Constraint{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, constraint); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("constraint not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get constraint: %v", err)
	}

	err := r.reconcile(ctx, constraint, log)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(constraint, corev1.EventTypeWarning, "ConstraintReconcileFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) createConstraint(ctx context.Context, constraint *kubermaticv1.Constraint, log *zap.SugaredLogger) error {
	log = log.With("constraint", constraint)

	constraintCreatorGetters := []reconciling.NamedUnstructuredCreatorGetter{
		constraintCreatorGetter(constraint),
	}

	if err := reconciling.ReconcileUnstructureds(ctx, constraintCreatorGetters, "", r.userClient); err != nil {
		return fmt.Errorf("failed to reconcile constraint: %v", err)
	}
	log.Debugw("constraint created")
	return nil
}

func (r *reconciler) cleanupConstraint(ctx context.Context, constraint *kubermaticv1.Constraint, log *zap.SugaredLogger) error {
	log = log.With("constraint", constraint)
	log.Debugw("cleanup processing:")

	toDelete := &unstructured.Unstructured{}
	toDelete.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   constrainthandler.ConstraintsGroup,
		Version: constrainthandler.ConstraintsVersion,
		Kind:    constraint.Spec.ConstraintType,
	})
	toDelete.SetName(constraint.Name)

	if err := r.userClient.Delete(ctx, toDelete); err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete constraint: %v", err)
	}
	log.Debugw("constraint deleted")
	return nil
}

// constraintCreatorGetter returns the unstructured gatekeeper Constraint object.
func constraintCreatorGetter(constraint *kubermaticv1.Constraint) reconciling.NamedUnstructuredCreatorGetter {
	return func() (string, string, string, reconciling.UnstructuredCreator) {
		return constraint.Name, constraint.Spec.ConstraintType, constraintAPIVersion, func(u *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			var params map[string]interface{}

			// first check if the Constraint is using the deprecated rawJSON parameters, if yes, we should use them
			rawJSON, ok, err := unstructured.NestedString(constraint.Spec.Parameters, "rawJSON")
			if err != nil {
				return nil, fmt.Errorf("error getting constraint rawJSON parameters %s", err)
			}
			if ok {
				err = json.Unmarshal([]byte(rawJSON), &params)
				if err != nil {
					return nil, fmt.Errorf("error unmarshalling constraint params: %v", err)
				}
			} else {
				params = *constraint.Spec.Parameters.DeepCopy()
			}

			if err = unstructured.SetNestedField(u.Object, params, spec, parametersField); err != nil {
				return nil, fmt.Errorf("error setting constraint nested parameters: %v", err)
			}

			// set Match
			matchMap, err := unmarshallToJSONMap(&constraint.Spec.Match)
			if err != nil {
				return nil, err
			}

			err = unstructured.SetNestedField(u.Object, matchMap, spec, matchField)
			if err != nil {
				return nil, fmt.Errorf("error setting constraint nested spec: %v", err)
			}

			return u, nil
		}
	}
}

func unmarshallToJSONMap(object interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("error marshalling: %v", err)
	}
	result := make(map[string]interface{})
	err = json.Unmarshal(raw, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling: %v", err)
	}

	return result, nil
}
