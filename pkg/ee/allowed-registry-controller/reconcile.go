//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package allowedregistrycontroller

import (
	"context"
	"encoding/json"
	"fmt"

	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	regoschema "github.com/open-policy-agent/frameworks/constraint/pkg/client/drivers/rego/schema"
	"github.com/open-policy-agent/frameworks/constraint/pkg/core/templates"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	AllowedRegistryCTName = "allowedregistry"
	AllowedRegistryField  = "allowed_registry"
)

type Reconciler struct {
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	masterClient ctrlruntimeclient.Client
	namespace    string
}

func NewReconciler(log *zap.SugaredLogger, recorder record.EventRecorder, masterClient ctrlruntimeclient.Client, namespace string) *Reconciler {
	return &Reconciler{
		log:          log,
		recorder:     recorder,
		masterClient: masterClient,
		namespace:    namespace,
	}
}

// Reconcile reconciles the allowed registry.
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	allowedRegistry := &kubermaticv1.AllowedRegistry{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, allowedRegistry); err != nil {
		if apierrors.IsNotFound(err) {
			log.Debug("allowed registry not found, returning")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get allowed registry %s: %w", allowedRegistry.Name, err)
	}

	err := r.reconcile(ctx, allowedRegistry)
	if err != nil {
		r.recorder.Event(allowedRegistry, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, allowedRegistry *kubermaticv1.AllowedRegistry) error {
	finalizer := cleanupFinalizer

	regSet, err := r.getRegistrySet(ctx)
	if err != nil {
		return fmt.Errorf("error getting registry set from AllowedRegistries: %w", err)
	}

	if allowedRegistry.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(allowedRegistry, finalizer) {
			return nil
		}

		// Ensure Constraint with registry data
		constraintReconcilerFactories := []reconciling.NamedConstraintReconcilerFactory{
			allowedRegistryConstraintReconcilerFactory(regSet),
		}

		err := reconciling.ReconcileConstraints(ctx, constraintReconcilerFactories, r.namespace, r.masterClient)
		if err != nil {
			return fmt.Errorf("error ensuring AllowedRegistry Constraint Template: %w", err)
		}

		return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, allowedRegistry, finalizer)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, allowedRegistry, finalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	// Ensure that the Constraint Template for AllowedRegistry exists
	ctReconcilerFactories := []reconciling.NamedConstraintTemplateReconcilerFactory{
		allowedRegistryCTReconcilerFactory(),
	}
	err = reconciling.ReconcileConstraintTemplates(ctx, ctReconcilerFactories, "", r.masterClient)
	if err != nil {
		return fmt.Errorf("error ensuring AllowedRegistry Constraint Template: %w", err)
	}

	// Ensure Constraint with registry data
	constraintReconcilerFactories := []reconciling.NamedConstraintReconcilerFactory{
		allowedRegistryConstraintReconcilerFactory(regSet),
	}

	err = reconciling.ReconcileConstraints(ctx, constraintReconcilerFactories, r.namespace, r.masterClient)
	if err != nil {
		return fmt.Errorf("error ensuring AllowedRegistry Constraint Template: %w", err)
	}

	return nil
}

const regoSource = `package allowedregistry

violation[{"msg": msg}] {
  container := input.review.object.spec.containers[_]
  satisfied := [good | repo = input.parameters.allowed_registry[_] ; good = startswith(container.image, repo)]
  not any(satisfied)
  msg := sprintf("container <%v> has an invalid image registry <%v>, allowed image registries are %v", [container.name, container.image, input.parameters.allowed_registry])
}

violation[{"msg": msg}] {
  container := input.review.object.spec.initContainers[_]
  satisfied := [good | repo = input.parameters.allowed_registry[_] ; good = startswith(container.image, repo)]
  not any(satisfied)
  msg := sprintf("init container <%v> has an invalid image registry <%v>, allowed image registries are %v", [container.name, container.image, input.parameters.allowed_registry])
}`

func allowedRegistryCTReconcilerFactory() reconciling.NamedConstraintTemplateReconcilerFactory {
	return func() (string, reconciling.ConstraintTemplateReconciler) {
		return AllowedRegistryCTName, func(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
			ct.Name = AllowedRegistryCTName
			ct.Spec = kubermaticv1.ConstraintTemplateSpec{
				CRD: constrainttemplatev1.CRD{
					Spec: constrainttemplatev1.CRDSpec{
						Names: constrainttemplatev1.Names{
							Kind: AllowedRegistryCTName,
						},
						Validation: &constrainttemplatev1.Validation{
							LegacySchema: ptr.To(false),
							OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]apiextensionsv1.JSONSchemaProps{
									AllowedRegistryField: {
										Type: "array",
										Items: &apiextensionsv1.JSONSchemaPropsOrArray{
											Schema: &apiextensionsv1.JSONSchemaProps{
												Type: "string",
											},
										},
									},
								},
							},
						},
					},
				},
				Targets: []constrainttemplatev1.Target{
					{
						Target: "admission.k8s.gatekeeper.sh",
						Code: []constrainttemplatev1.Code{
							{
								Engine: regoschema.Name,
								Source: &templates.Anything{
									Value: (&regoschema.Source{
										Rego: regoSource,
									}).ToUnstructured(),
								},
							},
						},
					},
				},
			}

			return ct, nil
		}
	}
}

func allowedRegistryConstraintReconcilerFactory(regSet sets.Set[string]) reconciling.NamedConstraintReconcilerFactory {
	return func() (string, reconciling.ConstraintReconciler) {
		return AllowedRegistryCTName, func(ct *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
			ct.Name = AllowedRegistryCTName
			ct.Spec.Match.Kinds = []kubermaticv1.Kind{
				{
					APIGroups: []string{""},
					Kinds:     []string{"Pod"},
				},
			}
			ct.Spec.ConstraintType = AllowedRegistryCTName
			ct.Spec.Disabled = regSet.Len() == 0

			jsonRegSet, err := json.Marshal(sets.List(regSet))
			if err != nil {
				return nil, fmt.Errorf("error marshalling registry set: %w", err)
			}

			ct.Spec.Parameters = map[string]json.RawMessage{
				AllowedRegistryField: jsonRegSet,
			}

			return ct, nil
		}
	}
}

func (r *Reconciler) getRegistrySet(ctx context.Context) (sets.Set[string], error) {
	var arList kubermaticv1.AllowedRegistryList
	if err := r.masterClient.List(ctx, &arList); err != nil {
		return nil, err
	}

	regSet := sets.New[string]()

	for _, ar := range arList.Items {
		if ar.DeletionTimestamp == nil {
			regSet.Insert(ar.Spec.RegistryPrefix)
		}
	}
	return regSet, nil
}
