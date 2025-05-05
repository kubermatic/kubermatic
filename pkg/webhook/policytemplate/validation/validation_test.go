/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package validation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Minimal valid Kyverno policy spec for testing.
var validMinimalPolicySpec = map[string]interface{}{
	"validationFailureAction": "Audit",
	"background":              true,
	"rules": []map[string]interface{}{
		{
			"name": "sample-rule",
			"match": map[string]interface{}{
				"any": []map[string]interface{}{
					{
						"resources": map[string]interface{}{
							"kinds": []string{"Pod"},
						},
					},
				},
			},
			"validate": map[string]interface{}{
				"message": "Sample validation message.",
				"pattern": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]string{
							"app": "?*",
						},
					},
				},
			},
		},
	},
}

func mustMarshalJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func createTestPolicyTemplate(name string, mutators ...func(*kubermaticv1.PolicyTemplate)) *kubermaticv1.PolicyTemplate {
	base := &kubermaticv1.PolicyTemplate{
		TypeMeta:   metav1.TypeMeta{Kind: "PolicyTemplate", APIVersion: "kubermatic.k8c.io/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:       "Test Title - " + name,
			Description: "Test Description - " + name,
			Visibility:  kubermaticv1.PolicyTemplateVisibilityGlobal,
			PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
		},
	}
	for _, m := range mutators {
		m(base)
	}
	return base
}

// checkErrors asserts whether the validation result matches the expected error state
// and verifies the presence of expected error paths if errors were expected.
func checkErrors(t *testing.T, name string, err error, wantErrors bool, expectedPaths []string) {
	t.Helper()

	hasErrors := (err != nil)
	if hasErrors != wantErrors {
		t.Errorf("%s: error presence mismatch: got error = %v, wantErrors %v", name, err, wantErrors)
	}

	if wantErrors && err != nil && len(expectedPaths) > 0 {
		foundPaths := sets.NewString()
		fieldErrs := field.ErrorList{}

		var agg kerrors.Aggregate
		if errors.As(err, &agg) {
			for _, wrappedErr := range agg.Errors() {
				var fieldErr *field.Error
				if errors.As(wrappedErr, &fieldErr) {
					fieldErrs = append(fieldErrs, fieldErr)
				}
			}
		} else {
			var fieldErr *field.Error
			if errors.As(err, &fieldErr) {
				fieldErrs = append(fieldErrs, fieldErr)
			}
		}

		for _, fe := range fieldErrs {
			foundPaths.Insert(fe.Field)
		}

		for _, expectedPath := range expectedPaths {
			if !foundPaths.Has(expectedPath) {
				t.Errorf("%s: Expected error on path %q, but not found. Actual error: %v", name, expectedPath, err)
			}
		}
	} else if wantErrors && err == nil {
		t.Errorf("%s: Expected errors on paths %v but got no error", name, expectedPaths)
	}
}

func TestAdmissionValidationPolicyTemplate(t *testing.T) {
	validator := NewValidator(nil)

	tests := []struct {
		name       string
		template   *kubermaticv1.PolicyTemplate
		wantErrors bool
		errorPaths []string
	}{
		{
			name:       "Allow minimal Global template",
			template:   createTestPolicyTemplate("valid-global-min"),
			wantErrors: false,
		},
		{
			name: "Allow minimal Project template",
			template: createTestPolicyTemplate("valid-project-min", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityProject
				pt.Spec.ProjectID = "my-project-id"
			}),
			wantErrors: false,
		},
		{
			name: "Allow Global template with target selectors",
			template: createTestPolicyTemplate("valid-global-selectors", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{
					ProjectSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"env": "prod"}},
				}
			}),
			wantErrors: false,
		},
		{
			name: "Allow Project template with only Cluster selector",
			template: createTestPolicyTemplate("valid-project-cluster-selector", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityProject
				pt.Spec.ProjectID = "proj-abc"
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{
					ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"tier": "test"}},
				}
			}),
			wantErrors: false,
		},
		{
			name: "Allow template with optional fields filled",
			template: createTestPolicyTemplate("valid-optional-fields", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Category = "Security"
				pt.Spec.Severity = "High"
				pt.Spec.Enforced = true
			}),
			wantErrors: false,
		},
		{
			name: "Reject template missing Title",
			template: createTestPolicyTemplate("invalid-missing-title", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Title = ""
			}),
			wantErrors: true, errorPaths: []string{"spec.title"},
		},
		{
			name: "Reject template missing Description",
			template: createTestPolicyTemplate("invalid-missing-desc", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Description = ""
			}),
			wantErrors: true, errorPaths: []string{"spec.description"},
		},
		{
			name: "Reject template missing Visibility",
			template: createTestPolicyTemplate("invalid-missing-vis", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = ""
			}),
			wantErrors: true, errorPaths: []string{"spec.visibility"},
		},
		{
			name: "Reject template with bad Visibility Value",
			template: createTestPolicyTemplate("invalid-bad-vis", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = "Local" // Invalid enum
			}),
			wantErrors: true, errorPaths: []string{"spec.visibility"},
		},
		{
			name: "Reject Project template missing ProjectID",
			template: createTestPolicyTemplate("invalid-proj-no-id", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityProject
			}),
			wantErrors: true, errorPaths: []string{"spec.projectID"},
		},
		{
			name: "Reject Global template having ProjectID",
			template: createTestPolicyTemplate("invalid-global-with-id", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityGlobal
				pt.Spec.ProjectID = "should-be-empty"
			}),
			wantErrors: true, errorPaths: []string{"spec.projectID"},
		},
		{
			name: "Reject Project template having ProjectSelector",
			template: createTestPolicyTemplate("invalid-proj-has-projselector", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityProject
				pt.Spec.ProjectID = "proj-xyz"
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{
					ProjectSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "one"}},
				}
			}),
			wantErrors: true, errorPaths: []string{"spec.target.projectSelector"},
		},
		{
			name: "Reject template with bad ProjectSelector Syntax",
			template: createTestPolicyTemplate("invalid-bad-projselector", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityGlobal
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{
					ProjectSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test/": "two"}},
				}
			}),
			wantErrors: true, errorPaths: []string{"spec.target.projectSelector"},
		},
		{
			name: "Reject template with bad ClusterSelector Syntax",
			template: createTestPolicyTemplate("invalid-bad-clusterselector", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{
					ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"invalid/test/slash": "v"}},
				}
			}),
			wantErrors: true, errorPaths: []string{"spec.target.clusterSelector"},
		},
		{
			name: "Reject template missing PolicySpec",
			template: createTestPolicyTemplate("invalid-no-policyspec", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.PolicySpec = runtime.RawExtension{}
			}),
			wantErrors: true, errorPaths: []string{"spec.policySpec"},
		},
		{
			name: "Reject template with malformed PolicySpec JSON",
			template: createTestPolicyTemplate("invalid-policyspec-json", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.PolicySpec = runtime.RawExtension{Raw: []byte(`{"rules:[]}`)}
			}),
			wantErrors: true, errorPaths: []string{"spec.policySpec"},
		},
		{
			name: "Reject template where PolicySpec does not match Kyverno structure (missing rules)",
			template: createTestPolicyTemplate("invalid-policyspec-norules", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.PolicySpec = runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{"validationFailureAction": "Audit"})}
			}),
			wantErrors: true, errorPaths: []string{"spec.policySpec.rules"},
		},
		{
			name: "Reject template where PolicySpec rule missing name",
			template: createTestPolicyTemplate("invalid-policyspec-norulename", func(pt *kubermaticv1.PolicyTemplate) {
				spec := map[string]interface{}{
					"rules": []map[string]interface{}{
						{ /* "name": "missing", */ "match": validMinimalPolicySpec["rules"].([]map[string]interface{})[0]["match"], "validate": validMinimalPolicySpec["rules"].([]map[string]interface{})[0]["validate"]},
					},
				}
				pt.Spec.PolicySpec = runtime.RawExtension{Raw: mustMarshalJSON(spec)}
			}),
			wantErrors: true, errorPaths: []string{"spec.policySpec.rules[0].name"},
		},
		{
			name: "Reject template where PolicySpec rule missing action",
			template: createTestPolicyTemplate("invalid-policyspec-noaction", func(pt *kubermaticv1.PolicyTemplate) {
				spec := map[string]interface{}{
					"rules": []map[string]interface{}{
						{"name": "rule1", "match": validMinimalPolicySpec["rules"].([]map[string]interface{})[0]["match"]},
					},
				}
				pt.Spec.PolicySpec = runtime.RawExtension{Raw: mustMarshalJSON(spec)}
			}),
			wantErrors: true, errorPaths: []string{"spec.policySpec.rules[0].<action>"},
		},
	}

	t.Run("ValidateCreate", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := validator.ValidateCreate(context.Background(), tt.template.DeepCopy())
				checkErrors(t, tt.name+" [Create]", err, tt.wantErrors, tt.errorPaths)
			})
		}
	})

	t.Run("ValidateUpdate", func(t *testing.T) {
		testImmutability(t, validator)
	})

	t.Run("ValidateDelete", func(t *testing.T) {
		template := createTestPolicyTemplate("delete-test")
		_, err := validator.ValidateDelete(context.Background(), template)
		if err != nil {
			t.Errorf("ValidateDelete() returned error = %v, expected nil", err)
		}
	})
}

func testImmutability(t *testing.T, validator *validator) {
	oldGlobal := createTestPolicyTemplate("immutable-global")
	oldProject := createTestPolicyTemplate("immutable-project", func(pt *kubermaticv1.PolicyTemplate) {
		pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityProject
		pt.Spec.ProjectID = "proj-1"
	})

	tests := []struct {
		name        string
		oldTemplate *kubermaticv1.PolicyTemplate
		newTemplate *kubermaticv1.PolicyTemplate
		wantErrors  bool
		errorPaths  []string
	}{
		{
			name:        "Allow Update Description",
			oldTemplate: oldGlobal.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Description = "New Description"
			}),
			wantErrors: false,
		},
		{
			name:        "Allow Update Title",
			oldTemplate: oldGlobal.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Title = "New Title"
			}),
			wantErrors: false,
		},
		{
			name:        "Allow Update Category",
			oldTemplate: oldGlobal.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Category = "New Category"
			}),
			wantErrors: false,
		},
		{
			name:        "Allow Update Severity",
			oldTemplate: oldGlobal.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Severity = "Low"
			}),
			wantErrors: false,
		},
		{
			name:        "Allow Update Enforced flag",
			oldTemplate: oldGlobal.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Enforced = true
			}),
			wantErrors: false,
		},
		{
			name:        "Allow Update Adding Target",
			oldTemplate: oldGlobal.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "prod"}}}
			}),
			wantErrors: false,
		},
		{
			name: "Allow Update Modifying Target",
			oldTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "prod"}}}
			}),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "dev"}}}
			}),
			wantErrors: false,
		},
		{
			name: "Allow Update Removing Target",
			oldTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Target = &kubermaticv1.PolicyTemplateTarget{ClusterSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"test": "prod"}}}
			}),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Target = nil
			}),
			wantErrors: false,
		},
		{
			name:        "Reject Update Visibility Global to Project",
			oldTemplate: oldGlobal.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityProject
				pt.Spec.ProjectID = "new-proj"
			}),
			wantErrors: true, errorPaths: []string{"spec.visibility"},
		},
		{
			name:        "Reject Update Visibility Project to Global",
			oldTemplate: oldProject.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-project", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityGlobal
				pt.Spec.ProjectID = ""
			}),
			wantErrors: true, errorPaths: []string{"spec.visibility"},
		},
		{
			name:        "Reject Update ProjectID change",
			oldTemplate: oldProject.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-project", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.Visibility = kubermaticv1.PolicyTemplateVisibilityProject
				pt.Spec.ProjectID = "proj-2"
			}),
			wantErrors: true, errorPaths: []string{"spec.projectID"},
		},
		{
			name:        "Reject Update Adding ProjectID to Global",
			oldTemplate: oldGlobal.DeepCopy(),
			newTemplate: createTestPolicyTemplate("immutable-global", func(pt *kubermaticv1.PolicyTemplate) {
				pt.Spec.ProjectID = "should-be-empty"
			}),
			wantErrors: true, errorPaths: []string{"spec.projectID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := validator.ValidateUpdate(context.Background(), tt.oldTemplate.DeepCopyObject(), tt.newTemplate.DeepCopyObject())
			checkErrors(t, tt.name+" [Update]", err, tt.wantErrors, tt.errorPaths)
		})
	}
}
