// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Loodse GmbH

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

package whitelistedregistrycontroller

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
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	WhitelistedRegistryCTName = "whitelistedregistry"
	WhitelistedRegistryField  = "whitelisted_registry"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	masterClient ctrlruntimeclient.Client
	namespace    string
}

func NewReconciler(log *zap.SugaredLogger, recorder record.EventRecorder, masterClient ctrlruntimeclient.Client, namespace string) *reconciler {
	return &reconciler{
		log:          log,
		recorder:     recorder,
		masterClient: masterClient,
		namespace:    namespace,
	}
}

// Reconcile reconciles the whitelisted registry
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	whitelistedRegistry := &kubermaticv1.WhitelistedRegistry{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, whitelistedRegistry); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("whitelisted registry not found, returning")
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get whitelisted registry %s: %v", whitelistedRegistry.Name, err)
	}

	err := r.reconcile(ctx, whitelistedRegistry)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(whitelistedRegistry, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, whitelistedRegistry *kubermaticv1.WhitelistedRegistry) error {
	if whitelistedRegistry.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(whitelistedRegistry, kubermaticapiv1.WhitelistedRegistryCleanupFinalizer) {
			return nil
		}

		// Ensure Constraint with registry data
		constraintCreatorGetters := []reconciling.NamedKubermaticV1ConstraintCreatorGetter{
			whitelistedRegistryConstraintCreatorGetter(whitelistedRegistry),
		}

		err := reconciling.ReconcileKubermaticV1Constraints(ctx, constraintCreatorGetters, r.namespace, r.masterClient)
		if err != nil {
			return fmt.Errorf("error ensuring WhitelistedRegistry Constraint Template: %v", err)
		}

		oldWhitelistedRegistry := whitelistedRegistry.DeepCopy()
		kuberneteshelper.RemoveFinalizer(whitelistedRegistry, kubermaticapiv1.WhitelistedRegistryCleanupFinalizer)
		if err := r.masterClient.Patch(ctx, whitelistedRegistry, ctrlruntimeclient.MergeFrom(oldWhitelistedRegistry)); err != nil {
			return fmt.Errorf("failed to remove whitelisted registry finalizer %s: %v", whitelistedRegistry.Name, err)
		}
		return nil
	}

	if !kuberneteshelper.HasFinalizer(whitelistedRegistry, kubermaticapiv1.WhitelistedRegistryCleanupFinalizer) {
		oldWhitelistedRegistry := whitelistedRegistry.DeepCopy()
		kuberneteshelper.AddFinalizer(whitelistedRegistry, kubermaticapiv1.WhitelistedRegistryCleanupFinalizer)
		if err := r.masterClient.Patch(ctx, whitelistedRegistry, ctrlruntimeclient.MergeFrom(oldWhitelistedRegistry)); err != nil {
			return fmt.Errorf("failed to set whitelisted registry finalizer %s: %v", whitelistedRegistry.Name, err)
		}
	}

	// Ensure that the Constraint Template for WhitelistedRegistry exists
	ctCreatorGetters := []reconciling.NamedKubermaticV1ConstraintTemplateCreatorGetter{
		whitelistedRegistryCTCreatorGetter(),
	}
	err := reconciling.ReconcileKubermaticV1ConstraintTemplates(ctx, ctCreatorGetters, "", r.masterClient)
	if err != nil {
		return fmt.Errorf("error ensuring WhitelistedRegistry Constraint Template: %v", err)
	}

	// Ensure Constraint with registry data
	constraintCreatorGetters := []reconciling.NamedKubermaticV1ConstraintCreatorGetter{
		whitelistedRegistryConstraintCreatorGetter(whitelistedRegistry),
	}

	err = reconciling.ReconcileKubermaticV1Constraints(ctx, constraintCreatorGetters, r.namespace, r.masterClient)
	if err != nil {
		return fmt.Errorf("error ensuring WhitelistedRegistry Constraint Template: %v", err)
	}

	return nil
}

func whitelistedRegistryCTCreatorGetter() reconciling.NamedKubermaticV1ConstraintTemplateCreatorGetter {
	return func() (string, reconciling.KubermaticV1ConstraintTemplateCreator) {
		return WhitelistedRegistryCTName, func(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
			ct.Name = WhitelistedRegistryCTName
			ct.Spec = kubermaticv1.ConstraintTemplateSpec{
				CRD: constrainttemplatev1beta1.CRD{
					Spec: constrainttemplatev1beta1.CRDSpec{
						Names: constrainttemplatev1beta1.Names{
							Kind: WhitelistedRegistryCTName,
						},
						Validation: &constrainttemplatev1beta1.Validation{
							OpenAPIV3Schema: &apiextensionsv1beta1.JSONSchemaProps{
								Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
									WhitelistedRegistryField: {
										Type: "array",
										Items: &apiextensionsv1beta1.JSONSchemaPropsOrArray{
											Schema: &apiextensionsv1beta1.JSONSchemaProps{
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
						Rego:   "package whitelistedregistry\n\nviolation[{\"msg\": msg}] {\n  container := input.review.object.spec.containers[_]\n  satisfied := [good | repo = input.parameters.whitelisted_registry[_] ; good = startswith(container.image, repo)]\n  not any(satisfied)\n  msg := sprintf(\"container <%v> has an invalid image registry <%v>, allowed image registries are %v\", [container.name, container.image, input.parameters.whitelisted_registry])\n}\nviolation[{\"msg\": msg}] {\n  container := input.review.object.spec.initContainers[_]\n  satisfied := [good | repo = input.parameters.whitelisted_registry[_] ; good = startswith(container.image, repo)]\n  not any(satisfied)\n  msg := sprintf(\"container <%v> has an invalid image registry <%v>, allowed image registries are %v\", [container.name, container.image, input.parameters.whitelisted_registry])\n}",
					},
				},
			}

			return ct, nil
		}
	}
}

func whitelistedRegistryConstraintCreatorGetter(wr *kubermaticv1.WhitelistedRegistry) reconciling.NamedKubermaticV1ConstraintCreatorGetter {
	return func() (string, reconciling.KubermaticV1ConstraintCreator) {
		return WhitelistedRegistryCTName, func(ct *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
			regSet := getRegistrySet(ct)
			if wr.DeletionTimestamp != nil {
				regSet.Delete(wr.Spec.RegistryPrefix)
			} else {
				regSet.Insert(wr.Spec.RegistryPrefix)
			}

			ct.Name = WhitelistedRegistryCTName
			ct.Spec = kubermaticv1.ConstraintSpec{
				ConstraintType: WhitelistedRegistryCTName,
				Match: kubermaticv1.Match{
					Kinds: []kubermaticv1.Kind{
						{
							APIGroups: []string{""},
							Kinds:     []string{"Pod"},
						},
					},
					Namespaces: []string{"default"},
				},
				Parameters: kubermaticv1.Parameters{
					WhitelistedRegistryField: regSet.List(),
				},
				Disabled: regSet.Len() == 0,
			}

			return ct, nil
		}
	}
}

func getRegistrySet(constraint *kubermaticv1.Constraint) sets.String {
	rawRegList, ok := constraint.Spec.Parameters[WhitelistedRegistryField]
	if !ok {
		return sets.NewString()
	}

	regList, ok := rawRegList.([]string)
	if !ok {
		return sets.NewString()
	}

	return sets.NewString(regList...)
}
