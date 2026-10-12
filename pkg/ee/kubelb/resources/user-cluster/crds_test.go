//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

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

package resources

import (
	"reflect"
	"testing"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestKubeLBCRDReconcilers(t *testing.T) {
	tests := []struct {
		name       string
		kind       string
		definition string
		factory    reconciling.NamedCustomResourceDefinitionReconcilerFactory
	}{
		{
			name:       "syncsecrets.kubelb.k8c.io",
			kind:       "SyncSecret",
			definition: syncSecretYAML,
			factory:    SyncSecretCRDReconciler(),
		},
		{
			name:       "tenantwafpolicies.kubelb.k8c.io",
			kind:       "TenantWAFPolicy",
			definition: tenantWAFPolicyYAML,
			factory:    TenantWAFPolicyCRDReconciler(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var embedded apiextensionsv1.CustomResourceDefinition
			if err := yaml.UnmarshalStrict([]byte(tt.definition), &embedded); err != nil {
				t.Fatalf("invalid embedded CRD: %v", err)
			}
			if embedded.APIVersion != apiextensionsv1.SchemeGroupVersion.String() || embedded.Kind != "CustomResourceDefinition" || embedded.Name != tt.name {
				t.Fatalf("unexpected embedded CRD identity: %s %s %s", embedded.APIVersion, embedded.Kind, embedded.Name)
			}

			name, reconcile := tt.factory()
			if name != tt.name {
				t.Fatalf("reconciler name = %q, want %q", name, tt.name)
			}

			// Updating the definition must preserve server-owned identity, finalizers and status.
			existing := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:            name,
					UID:             "existing-crd",
					ResourceVersion: "42",
					Finalizers:      []string{"customresourcecleanup.apiextensions.k8s.io"},
				},
				Status: apiextensionsv1.CustomResourceDefinitionStatus{
					StoredVersions: []string{"v1alpha1"},
					Conditions: []apiextensionsv1.CustomResourceDefinitionCondition{{
						Type:   apiextensionsv1.Established,
						Status: apiextensionsv1.ConditionTrue,
					}},
				},
			}
			previous := existing.DeepCopy()
			got, err := reconcile(existing)
			if err != nil {
				t.Fatalf("reconcile CRD: %v", err)
			}
			if got.Name != previous.Name || got.UID != previous.UID || got.ResourceVersion != previous.ResourceVersion || !reflect.DeepEqual(got.Finalizers, previous.Finalizers) || !reflect.DeepEqual(got.Status, previous.Status) {
				t.Error("reconciliation changed existing identity, finalizers or status")
			}

			// Conversion is defaulted by the API server; setting None avoids reconciliation churn.
			embedded.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{Strategy: apiextensionsv1.NoneConverter}
			if !reflect.DeepEqual(got.Spec, embedded.Spec) || !reflect.DeepEqual(got.Labels, embedded.Labels) || !reflect.DeepEqual(got.Annotations, embedded.Annotations) {
				t.Error("reconciliation did not retain the complete embedded definition")
			}
			if got.Spec.Group != "kubelb.k8c.io" || got.Spec.Scope != apiextensionsv1.NamespaceScoped || got.Spec.Names.Kind != tt.kind {
				t.Errorf("unexpected API registration: %+v", got.Spec)
			}
			if len(got.Spec.Versions) != 1 {
				t.Fatalf("got %d versions, want one", len(got.Spec.Versions))
			}
			version := got.Spec.Versions[0]
			if version.Name != "v1alpha1" || !version.Served || !version.Storage {
				t.Errorf("unexpected served/storage version: %+v", version)
			}
			if version.Subresources == nil || version.Subresources.Status == nil {
				t.Error("missing status subresource required by the CCM")
			}
			assertKubeLBCRDStructuralSchema(t, version.Schema)

			once := got.DeepCopy()
			twice, err := reconcile(got)
			if err != nil {
				t.Fatalf("reconcile CRD again: %v", err)
			}
			if !reflect.DeepEqual(once, twice) {
				t.Error("reconciliation is not idempotent")
			}
		})
	}
}

func assertKubeLBCRDStructuralSchema(t *testing.T, schema *apiextensionsv1.CustomResourceValidation) {
	t.Helper()

	if schema == nil || schema.OpenAPIV3Schema == nil {
		t.Fatal("missing OpenAPI schema")
	}
	internalSchema := &apiextensions.JSONSchemaProps{}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(schema.OpenAPIV3Schema, internalSchema, nil); err != nil {
		t.Fatalf("convert schema: %v", err)
	}
	structural, err := structuralschema.NewStructural(internalSchema)
	if err != nil {
		t.Fatalf("build structural schema: %v", err)
	}
	if errs := structuralschema.ValidateStructural(field.NewPath("schema"), structural); len(errs) != 0 {
		t.Fatalf("invalid structural schema: %v", errs)
	}
}

func TestResourcesForDeletionIncludesKubeLBCRDs(t *testing.T) {
	crds := map[string]int{}
	for _, resource := range ResourcesForDeletion() {
		if crd, ok := resource.(*apiextensionsv1.CustomResourceDefinition); ok {
			crds[crd.Name]++
			if crd.Namespace != "" {
				t.Errorf("CRD %s must be cluster-scoped", crd.Name)
			}
		}
	}
	want := map[string]int{
		"syncsecrets.kubelb.k8c.io":       1,
		"tenantwafpolicies.kubelb.k8c.io": 1,
	}
	if !reflect.DeepEqual(crds, want) {
		t.Errorf("CRDs for deletion = %v, want %v", crds, want)
	}
}
