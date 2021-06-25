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
	"reflect"
	"testing"
	"time"

	constrainttemplatev1beta1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
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
		name                string
		whitelistedRegistry *kubermaticv1.WhitelistedRegistry
		expectedCT          *kubermaticv1.ConstraintTemplate
		expectedConstraint  *kubermaticv1.Constraint
		masterClient        ctrlruntimeclient.Client
	}{
		{
			name:                "scenario 1: sync ct to seed cluster",
			whitelistedRegistry: genWhitelistedRegistry(false),
			expectedCT:          genConstraintTemplate(),
			expectedConstraint:  genWRConstraint(sets.NewString("quay.io")),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(genWhitelistedRegistry(false)).
				Build(),
		},
		{
			name:                "scenario 2: cleanup ct on seed cluster when master ct is being terminated",
			whitelistedRegistry: genWhitelistedRegistry(true),
			expectedCT:          genConstraintTemplate(),
			expectedConstraint:  genWRConstraint(sets.NewString()),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(genWhitelistedRegistry(true),
					genConstraintTemplate(), genWRConstraint(sets.NewString("quay.io"))).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			r := reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: tc.masterClient,
				namespace:    testNamespace,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.whitelistedRegistry.Name}}
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
				Rego:   "violation[{\"msg\": msg}] {\n\tcontainer := input.request.object.spec.containers[_].image\n\tsatisfied := [good | repo = input.parameters.repos[_]; good = startswith(container, repo)]\n\tnot any(satisfied)\n\tmsg := sprintf(\"container <%v> has an invalid image repo <%v>, allowed repos are %v\", [container.name, container.image, input.parameters.repos])\n}",
			},
		},
	}

	return ct
}

func genWhitelistedRegistry(deleted bool) *kubermaticv1.WhitelistedRegistry {
	wr := &kubermaticv1.WhitelistedRegistry{}
	wr.Name = "WhitelistedRegistry"
	wr.Spec = kubermaticv1.WhitelistedRegistrySpec{
		RegistryPrefix: "quay.io",
	}

	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		wr.DeletionTimestamp = &deleteTime
		wr.Finalizers = append(wr.Finalizers, v1.WhitelistedRegistryCleanupFinalizer)
	}

	return wr
}

func genWRConstraint(registrySet sets.String) *kubermaticv1.Constraint {
	ct := &kubermaticv1.Constraint{}
	ct.Name = WhitelistedRegistryCTName
	ct.Namespace = testNamespace

	interfaceList := []interface{}{}
	for _, registry := range registrySet.List() {
		interfaceList = append(interfaceList, registry)
	}

	ct.Spec = kubermaticv1.ConstraintSpec{
		ConstraintType: WhitelistedRegistryCTName,
		Match: kubermaticv1.Match{
			Kinds: []kubermaticv1.Kind{
				{
					APIGroups: []string{""},
					Kinds:     []string{"Pod"},
				},
				{
					APIGroups: []string{"apps"},
					Kinds:     []string{"Deployment", "StatefulSet", "DaemonSet", "ReplicaSet"},
				},
			},
		},
		Parameters: kubermaticv1.Parameters{
			WhitelistedRegistryField: interfaceList,
		},
		Disabled: registrySet.Len() == 0,
	}
	return ct
}
