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
	"fmt"

	constrainttemplatev1beta1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
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

// Reconcile reconciles the allowed registry
func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	allowedRegistry := &kubermaticv1.AllowedRegistry{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, allowedRegistry); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("allowed registry not found, returning")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get allowed registry %s: %v", allowedRegistry.Name, err)
	}

	err := r.reconcile(ctx, allowedRegistry)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(allowedRegistry, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, allowedRegistry *kubermaticv1.AllowedRegistry) error {
	regSet, err := r.getRegistrySet()
	if err != nil {
		return fmt.Errorf("error getting registry set from AllowedRegistries: %v", err)
	}

	if allowedRegistry.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(allowedRegistry, kubermaticapiv1.AllowedRegistryCleanupFinalizer) {
			return nil
		}

		// Ensure Constraint with registry data
		constraintCreatorGetters := []reconciling.NamedKubermaticV1ConstraintCreatorGetter{
			allowedRegistryConstraintCreatorGetter(regSet),
		}

		err := reconciling.ReconcileKubermaticV1Constraints(ctx, constraintCreatorGetters, r.namespace, r.masterClient)
		if err != nil {
			return fmt.Errorf("error ensuring AllowedRegistry Constraint Template: %v", err)
		}

		oldAllowedRegistry := allowedRegistry.DeepCopy()
		kuberneteshelper.RemoveFinalizer(allowedRegistry, kubermaticapiv1.AllowedRegistryCleanupFinalizer)
		if err := r.masterClient.Patch(ctx, allowedRegistry, ctrlruntimeclient.MergeFrom(oldAllowedRegistry)); err != nil {
			return fmt.Errorf("failed to remove allowed registry finalizer %s: %v", allowedRegistry.Name, err)
		}
		return nil
	}

	if !kuberneteshelper.HasFinalizer(allowedRegistry, kubermaticapiv1.AllowedRegistryCleanupFinalizer) {
		oldAllowedRegistry := allowedRegistry.DeepCopy()
		kuberneteshelper.AddFinalizer(allowedRegistry, kubermaticapiv1.AllowedRegistryCleanupFinalizer)
		if err := r.masterClient.Patch(ctx, allowedRegistry, ctrlruntimeclient.MergeFrom(oldAllowedRegistry)); err != nil {
			return fmt.Errorf("failed to set allowed registry finalizer %s: %v", allowedRegistry.Name, err)
		}
	}

	// Ensure that the Constraint Template for AllowedRegistry exists
	ctCreatorGetters := []reconciling.NamedKubermaticV1ConstraintTemplateCreatorGetter{
		allowedRegistryCTCreatorGetter(),
	}
	err = reconciling.ReconcileKubermaticV1ConstraintTemplates(ctx, ctCreatorGetters, "", r.masterClient)
	if err != nil {
		return fmt.Errorf("error ensuring AllowedRegistry Constraint Template: %v", err)
	}

	// Ensure Constraint with registry data
	constraintCreatorGetters := []reconciling.NamedKubermaticV1ConstraintCreatorGetter{
		allowedRegistryConstraintCreatorGetter(regSet),
	}

	err = reconciling.ReconcileKubermaticV1Constraints(ctx, constraintCreatorGetters, r.namespace, r.masterClient)
	if err != nil {
		return fmt.Errorf("error ensuring AllowedRegistry Constraint Template: %v", err)
	}

	return nil
}

func allowedRegistryCTCreatorGetter() reconciling.NamedKubermaticV1ConstraintTemplateCreatorGetter {
	return func() (string, reconciling.KubermaticV1ConstraintTemplateCreator) {
		return AllowedRegistryCTName, func(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
			ct.Name = AllowedRegistryCTName
			ct.Spec = kubermaticv1.ConstraintTemplateSpec{
				CRD: constrainttemplatev1beta1.CRD{
					Spec: constrainttemplatev1beta1.CRDSpec{
						Names: constrainttemplatev1beta1.Names{
							Kind: AllowedRegistryCTName,
						},
						Validation: &constrainttemplatev1beta1.Validation{
							OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
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
				Targets: []constrainttemplatev1beta1.Target{
					{
						Target: "admission.k8s.gatekeeper.sh",
						Rego:   "package allowedregistry\n\nviolation[{\"msg\": msg}] {\n  container := input.review.object.spec.containers[_]\n  satisfied := [good | repo = input.parameters.allowed_registry[_] ; good = startswith(container.image, repo)]\n  not any(satisfied)\n  msg := sprintf(\"container <%v> has an invalid image registry <%v>, allowed image registries are %v\", [container.name, container.image, input.parameters.allowed_registry])\n}\nviolation[{\"msg\": msg}] {\n  container := input.review.object.spec.initContainers[_]\n  satisfied := [good | repo = input.parameters.allowed_registry[_] ; good = startswith(container.image, repo)]\n  not any(satisfied)\n  msg := sprintf(\"container <%v> has an invalid image registry <%v>, allowed image registries are %v\", [container.name, container.image, input.parameters.allowed_registry])\n}",
					},
				},
			}

			return ct, nil
		}
	}
}

func allowedRegistryConstraintCreatorGetter(regSet sets.String) reconciling.NamedKubermaticV1ConstraintCreatorGetter {
	return func() (string, reconciling.KubermaticV1ConstraintCreator) {
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
			ct.Spec.Parameters = kubermaticv1.Parameters{
				AllowedRegistryField: regSet.List(),
			}

			return ct, nil
		}
	}
}

func (r *Reconciler) getRegistrySet() (sets.String, error) {
	var arList kubermaticv1.AllowedRegistryList
	if err := r.masterClient.List(context.Background(), &arList); err != nil {
		return nil, err
	}

	regSet := sets.NewString()

	for _, ar := range arList.Items {
		if ar.DeletionTimestamp == nil {
			regSet.Insert(ar.Spec.RegistryPrefix)
		}
	}
	return regSet, nil
}
