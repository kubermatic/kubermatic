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

package allowedregistrycontroller_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	constrainttemplatev1beta1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	allowedregistrycontroller "k8c.io/kubermatic/v2/pkg/ee/allowed-registry-controller"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const testNamespace = "kubermatic"

func TestReconcile(t *testing.T) {

	testCases := []struct {
		name               string
		allowedRegistry    *kubermaticv1.AllowedRegistry
		expectedCT         *kubermaticv1.ConstraintTemplate
		expectedConstraint *kubermaticv1.Constraint
		masterClient       ctrlruntimeclient.Client
	}{
		{
			name:               "scenario 1: sync ct to seed cluster",
			allowedRegistry:    genAllowedRegistry(false),
			expectedCT:         genConstraintTemplate(),
			expectedConstraint: genWRConstraint(sets.NewString("quay.io")),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(genAllowedRegistry(false)).
				Build(),
		},
		{
			name:               "scenario 2: cleanup ct on seed cluster when master ct is being terminated",
			allowedRegistry:    genAllowedRegistry(true),
			expectedCT:         genConstraintTemplate(),
			expectedConstraint: genWRConstraint(sets.NewString()),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(genAllowedRegistry(true),
					genConstraintTemplate(), genWRConstraint(sets.NewString("quay.io"))).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			r := allowedregistrycontroller.NewReconciler(
				kubermaticlog.Logger,
				&record.FakeRecorder{},
				tc.masterClient,
				testNamespace,
			)

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.allowedRegistry.Name}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			// check CT
			ct := &kubermaticv1.ConstraintTemplate{}
			err := tc.masterClient.Get(ctx, types.NamespacedName{Name: tc.expectedCT.Name}, ct)

			if err != nil {
				t.Fatalf("failed to get constraint template: %v", err)
			}

			if !reflect.DeepEqual(ct.Spec, tc.expectedCT.Spec) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(ct, tc.expectedCT))
			}

			if !reflect.DeepEqual(ct.Name, tc.expectedCT.Name) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(ct, tc.expectedCT))
			}

			// check Constraint
			constraint := &kubermaticv1.Constraint{}
			err = tc.masterClient.Get(ctx, types.NamespacedName{
				Namespace: testNamespace,
				Name:      tc.expectedConstraint.Name,
			}, constraint)
			if err != nil {
				t.Fatalf("failed to get constraint: %v", err)
			}

			if !equality.Semantic.DeepEqual(constraint.Spec, tc.expectedConstraint.Spec) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(constraint.Spec, tc.expectedConstraint.Spec))
			}

			if !reflect.DeepEqual(constraint.Name, tc.expectedConstraint.Name) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(constraint.Name, tc.expectedConstraint.Name))
			}
		})
	}
}

func genConstraintTemplate() *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}

	ct.Name = allowedregistrycontroller.AllowedRegistryCTName
	ct.Spec = kubermaticv1.ConstraintTemplateSpec{
		CRD: constrainttemplatev1beta1.CRD{
			Spec: constrainttemplatev1beta1.CRDSpec{
				Names: constrainttemplatev1beta1.Names{
					Kind: allowedregistrycontroller.AllowedRegistryCTName,
				},
				Validation: &constrainttemplatev1beta1.Validation{
					OpenAPIV3Schema: &apiextensionsv1beta1.JSONSchemaProps{
						Properties: map[string]apiextensionsv1beta1.JSONSchemaProps{
							allowedregistrycontroller.AllowedRegistryField: {
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
				Rego:   "package allowedregistry\n\nviolation[{\"msg\": msg}] {\n  container := input.review.object.spec.containers[_]\n  satisfied := [good | repo = input.parameters.allowed_registry[_] ; good = startswith(container.image, repo)]\n  not any(satisfied)\n  msg := sprintf(\"container <%v> has an invalid image registry <%v>, allowed image registries are %v\", [container.name, container.image, input.parameters.allowed_registry])\n}\nviolation[{\"msg\": msg}] {\n  container := input.review.object.spec.initContainers[_]\n  satisfied := [good | repo = input.parameters.allowed_registry[_] ; good = startswith(container.image, repo)]\n  not any(satisfied)\n  msg := sprintf(\"container <%v> has an invalid image registry <%v>, allowed image registries are %v\", [container.name, container.image, input.parameters.allowed_registry])\n}",
			},
		},
	}

	return ct
}

func genAllowedRegistry(deleted bool) *kubermaticv1.AllowedRegistry {
	wr := &kubermaticv1.AllowedRegistry{}
	wr.Name = "AllowedRegistry"
	wr.Spec = kubermaticv1.AllowedRegistrySpec{
		RegistryPrefix: "quay.io",
	}

	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		wr.DeletionTimestamp = &deleteTime
		wr.Finalizers = append(wr.Finalizers, v1.AllowedRegistryCleanupFinalizer)
	}

	return wr
}

func genWRConstraint(registrySet sets.String) *kubermaticv1.Constraint {
	ct := &kubermaticv1.Constraint{}
	ct.Name = allowedregistrycontroller.AllowedRegistryCTName
	ct.Namespace = testNamespace

	interfaceList := []interface{}{}
	for _, registry := range registrySet.List() {
		interfaceList = append(interfaceList, registry)
	}

	ct.Spec = kubermaticv1.ConstraintSpec{
		ConstraintType: allowedregistrycontroller.AllowedRegistryCTName,
		Match: kubermaticv1.Match{
			Kinds: []kubermaticv1.Kind{
				{
					APIGroups: []string{""},
					Kinds:     []string{"Pod"},
				},
			},
		},
		Parameters: kubermaticv1.Parameters{
			allowedregistrycontroller.AllowedRegistryField: interfaceList,
		},
		Disabled: registrySet.Len() == 0,
	}
	return ct
}
