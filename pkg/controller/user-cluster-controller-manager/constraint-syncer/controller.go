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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	userclustercontrollermanager "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	controllerName       = "kkp-constraint-synchronizer"
	constraintAPIVersion = "constraints.gatekeeper.sh/v1beta1"
	spec                 = "spec"
	parametersField      = "parameters"
	matchField           = "match"
	rawJSONField         = "rawJSON"
	enforcementAction    = "enforcementAction"
)

type reconciler struct {
	log             *zap.SugaredLogger
	seedClient      ctrlruntimeclient.Client
	userClient      ctrlruntimeclient.Client
	recorder        events.EventRecorder
	clusterIsPaused userclustercontrollermanager.IsPausedChecker
}

func Add(ctx context.Context, log *zap.SugaredLogger, seedMgr, userMgr manager.Manager, namespace string, clusterIsPaused userclustercontrollermanager.IsPausedChecker) error {
	log = log.Named(controllerName)

	r := &reconciler{
		log:             log,
		seedClient:      seedMgr.GetClient(),
		userClient:      userMgr.GetClient(),
		recorder:        userMgr.GetEventRecorder(controllerName),
		clusterIsPaused: clusterIsPaused,
	}

	_, err := builder.ControllerManagedBy(seedMgr).
		Named(controllerName).
		For(&kubermaticv1.Constraint{}, builder.WithPredicates(predicate.ByNamespace(namespace))).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("constraint", request)
	log.Debug("Reconciling")

	paused, err := r.clusterIsPaused(ctx)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check cluster pause status: %w", err)
	}
	if paused {
		return reconcile.Result{}, nil
	}

	constraint := &kubermaticv1.Constraint{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, constraint); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("constraint not found, returning")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get constraint: %w", err)
	}

	err = r.reconcile(ctx, constraint, log)
	if err != nil {
		r.recorder.Eventf(constraint, nil, corev1.EventTypeWarning, "ConstraintReconcileFailed", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) createConstraint(ctx context.Context, constraint *kubermaticv1.Constraint) error {
	constraintReconcilerFactories := []reconciling.NamedUnstructuredReconcilerFactory{
		constraintReconcilerFactory(constraint),
	}

	if err := reconciling.ReconcileUnstructureds(ctx, constraintReconcilerFactories, "", r.userClient); err != nil {
		return fmt.Errorf("failed to reconcile constraint: %w", err)
	}

	return nil
}

func (r *reconciler) cleanupConstraint(ctx context.Context, constraint *kubermaticv1.Constraint, log *zap.SugaredLogger) error {
	toDelete := &unstructured.Unstructured{}
	toDelete.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "constraints.gatekeeper.sh",
		Version: "v1beta1",
		Kind:    constraint.Spec.ConstraintType,
	})
	toDelete.SetName(constraint.Name)

	log.Info("Deleting Constraint")
	if err := r.userClient.Delete(ctx, toDelete); ctrlruntimeclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete constraint: %w", err)
	}

	return nil
}

// constraintReconcilerFactory returns the unstructured gatekeeper Constraint object.
func constraintReconcilerFactory(constraint *kubermaticv1.Constraint) reconciling.NamedUnstructuredReconcilerFactory {
	return func() (string, string, string, reconciling.UnstructuredReconciler) {
		return constraint.Name, constraint.Spec.ConstraintType, constraintAPIVersion, func(u *unstructured.Unstructured) (*unstructured.Unstructured, error) {
			if len(constraint.Spec.Parameters) > 0 {
				var params map[string]interface{}

				rawParams, err := json.Marshal(constraint.Spec.Parameters)
				if err != nil {
					return nil, fmt.Errorf("error marshalling constraint parameters: %w", err)
				}

				if err := json.Unmarshal(rawParams, &params); err != nil {
					return nil, fmt.Errorf("error unmarshalling constraint parameters: %w", err)
				}

				// To keep backwards compatibility for Constraints that still use rawJSON. Support for this should be removed for 2.19
				if rawJSON, ok := params[rawJSONField]; ok {
					var rawJSONParams map[string]interface{}
					rawJSON, ok := rawJSON.(string)
					if !ok {
						return nil, fmt.Errorf("error converting raw json parameters")
					}
					err := json.Unmarshal([]byte(rawJSON), &rawJSONParams)
					if err != nil {
						return nil, fmt.Errorf("error unmarshalling raw json parameters: %w", err)
					}
					params = rawJSONParams
				}

				if err := unstructured.SetNestedField(u.Object, params, spec, parametersField); err != nil {
					return nil, fmt.Errorf("error setting constraint nested parameters: %w", err)
				}
			}

			// set Match
			matchMap, err := unmarshalToJSONMap(&constraint.Spec.Match)
			if err != nil {
				return nil, err
			}

			err = unstructured.SetNestedField(u.Object, matchMap, spec, matchField)
			if err != nil {
				return nil, fmt.Errorf("error setting constraint nested spec: %w", err)
			}

			// set EnforcementAction
			if len(constraint.Spec.EnforcementAction) > 0 {
				if err := unstructured.SetNestedField(u.Object, constraint.Spec.EnforcementAction, spec, enforcementAction); err != nil {
					return nil, fmt.Errorf("error setting constraint nested EnforcementAction: %w", err)
				}
			} else {
				unstructured.RemoveNestedField(u.Object, spec, enforcementAction)
			}

			return u, nil
		}
	}
}

func unmarshalToJSONMap(object interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("error marshalling: %w", err)
	}
	result := make(map[string]interface{})
	err = json.Unmarshal(raw, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling: %w", err)
	}

	return result, nil
}
