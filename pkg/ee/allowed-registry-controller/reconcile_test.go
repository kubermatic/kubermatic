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
	"testing"
	"time"

	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	regoschema "github.com/open-policy-agent/frameworks/constraint/pkg/client/drivers/rego/schema"
	"github.com/open-policy-agent/frameworks/constraint/pkg/core/templates"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const testNamespace = "kubermatic"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                  string
		allowedRegistry       []*kubermaticv1.AllowedRegistry
		allowedRegistryUpdate *kubermaticv1.AllowedRegistry
		expectedCT            *kubermaticv1.ConstraintTemplate
		expectedConstraint    *kubermaticv1.Constraint
		masterClient          ctrlruntimeclient.Client
	}{
		{
			name:               "scenario 1: sync allowedlist to seed cluster",
			allowedRegistry:    []*kubermaticv1.AllowedRegistry{genAllowedRegistry("quay", "quay.io", false)},
			expectedCT:         genConstraintTemplate(),
			expectedConstraint: genWRConstraint(sets.New("quay.io")),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(genAllowedRegistry("quay", "quay.io", false)).
				Build(),
		},
		{
			name:               "scenario 2: cleanup allowedlist on seed cluster when master ct is being terminated",
			allowedRegistry:    []*kubermaticv1.AllowedRegistry{genAllowedRegistry("quay", "quay.io", true)},
			expectedCT:         genConstraintTemplate(),
			expectedConstraint: genWRConstraint(sets.New[string]()),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(genAllowedRegistry("quay", "quay.io", true),
					genConstraintTemplate(), genWRConstraint(sets.New("quay.io"))).
				Build(),
		},
		{
			name: "scenario 3: sync multiple allowedlists to seed cluster",
			allowedRegistry: []*kubermaticv1.AllowedRegistry{
				genAllowedRegistry("quay", "quay.io", false),
				genAllowedRegistry("myreg", "https://myregistry.com", false),
			},
			expectedCT:         genConstraintTemplate(),
			expectedConstraint: genWRConstraint(sets.New("quay.io", "https://myregistry.com")),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(
					genAllowedRegistry("quay", "quay.io", false),
					genAllowedRegistry("myreg", "https://myregistry.com", false)).
				Build(),
		},
		{
			name: "scenario 4: update a allowedlist",
			allowedRegistry: []*kubermaticv1.AllowedRegistry{
				genAllowedRegistry("quay", "quay.io", false),
				genAllowedRegistry("myreg", "https://myregistry.com", false),
			},
			allowedRegistryUpdate: genAllowedRegistry("quay", "quay.io-edited", false),
			expectedCT:            genConstraintTemplate(),
			expectedConstraint:    genWRConstraint(sets.New("quay.io-edited", "https://myregistry.com")),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(
					genAllowedRegistry("quay", "quay.io", false),
					genAllowedRegistry("myreg", "https://myregistry.com", false)).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			r := NewReconciler(
				kubermaticlog.Logger,
				&record.FakeRecorder{},
				tc.masterClient,
				testNamespace,
			)

			for _, ar := range tc.allowedRegistry {
				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: ar.Name}}
				if _, err := r.Reconcile(ctx, request); err != nil {
					t.Fatalf("reconciling failed: %v", err)
				}
			}

			if tc.allowedRegistryUpdate != nil {
				var ar kubermaticv1.AllowedRegistry
				if err := tc.masterClient.Get(ctx, types.NamespacedName{Name: tc.allowedRegistryUpdate.Name}, &ar); err != nil {
					t.Fatal(err)
				}

				ar.Spec = tc.allowedRegistryUpdate.Spec
				if err := tc.masterClient.Update(ctx, &ar); err != nil {
					t.Fatal(err)
				}

				request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.allowedRegistryUpdate.Name}}
				if _, err := r.Reconcile(ctx, request); err != nil {
					t.Fatalf("reconciling failed: %v", err)
				}
			}

			// check CT
			ct := &kubermaticv1.ConstraintTemplate{}
			err := tc.masterClient.Get(ctx, types.NamespacedName{Name: tc.expectedCT.Name}, ct)

			if err != nil {
				t.Fatalf("failed to get constraint template: %v", err)
			}

			ct.ResourceVersion = ""
			ct.APIVersion = ""
			ct.Kind = ""

			if !diff.SemanticallyEqual(tc.expectedCT, ct) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedCT, ct))
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

			constraint.ResourceVersion = ""
			constraint.APIVersion = ""
			constraint.Kind = ""

			if !diff.SemanticallyEqual(tc.expectedConstraint, constraint) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedConstraint, constraint))
			}
		})
	}
}

func genConstraintTemplate() *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}

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

	return ct
}

func genAllowedRegistry(name, registry string, deleted bool) *kubermaticv1.AllowedRegistry {
	wr := &kubermaticv1.AllowedRegistry{}
	wr.Name = name
	wr.Spec = kubermaticv1.AllowedRegistrySpec{
		RegistryPrefix: registry,
	}

	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		wr.DeletionTimestamp = &deleteTime
		wr.Finalizers = append(wr.Finalizers, cleanupFinalizer)
	}

	return wr
}

func genWRConstraint(registrySet sets.Set[string]) *kubermaticv1.Constraint {
	ct := &kubermaticv1.Constraint{}
	ct.Name = AllowedRegistryCTName
	ct.Namespace = testNamespace

	jsonRegSet, _ := json.Marshal(sets.List(registrySet))

	ct.Spec = kubermaticv1.ConstraintSpec{
		ConstraintType: AllowedRegistryCTName,
		Match: kubermaticv1.Match{
			Kinds: []kubermaticv1.Kind{
				{
					APIGroups: []string{""},
					Kinds:     []string{"Pod"},
				},
			},
		},
		Parameters: map[string]json.RawMessage{
			AllowedRegistryField: jsonRegSet,
		},
		Disabled: registrySet.Len() == 0,
	}
	return ct
}
