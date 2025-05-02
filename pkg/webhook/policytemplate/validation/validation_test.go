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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var testScheme = fake.NewScheme()

// Minimal valid Kyverno policy spec for testing
var validMinimalPolicySpec = map[string]interface{}{
	"validationFailureAction": "audit",
	"background":              true,
	"rules": []map[string]interface{}{
		{
			"name": "check-something",
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
				"message": "A message",
				"pattern": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]string{
							"some-label": "?*",
						},
					},
				},
			},
		},
	},
}

// Minimal spec needing only match and action
var minMatchActionSpec = map[string]interface{}{
	"rules": []map[string]interface{}{
		{
			"name": "check-something",
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
				"message": "A message",
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

func TestValidatePolicyTemplate(t *testing.T) {
	tests := []struct {
		name        string
		op          admissionv1.Operation
		template    kubermaticv1.PolicyTemplate
		wantAllowed bool
	}{
		{
			name: "Minimal valid global template",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "global-tpl"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Global Policy",
					Description: "A global policy template.",
					Visibility:  "global",
					PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Minimal valid project template",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "project-tpl"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Project Policy",
					Description: "A project policy template.",
					Visibility:  "project",
					ProjectID:   "my-project-id",
					PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Minimal valid cluster template",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster-tpl"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Cluster Policy",
					Description: "A cluster policy template.",
					Visibility:  "cluster",
					PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Invalid - Missing title",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "missing-title"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					// Title:       "Missing",
					Description: "Desc",
					Visibility:  "global",
					PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - Missing description",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "missing-desc"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title: "Title",
					// Description: "Missing",
					Visibility: "global",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - Invalid visibility",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "invalid-vis"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "namespace", // Invalid
					PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - Project visibility missing ProjectID",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "proj-missing-id"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "project",
					// ProjectID:   "missing",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - Global visibility has ProjectID",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "global-with-id"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "global",
					ProjectID:   "should-not-be-here",
					PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - Missing policySpec",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "missing-spec"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "global",
					// PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(validMinimalPolicySpec)},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - Malformed policySpec JSON",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-json"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "global",
					PolicySpec:  runtime.RawExtension{Raw: []byte(`{"rules": [{"name": "a"`)}, // Malformed
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - policySpec missing rules",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "no-rules"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "global",
					PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{"validationFailureAction": "audit"})},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - policySpec rule missing name",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "no-rule-name"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "global",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{
						"rules": []map[string]interface{}{
							{
								// "name": "missing",
								"match":    map[string]interface{}{ /*...*/ },
								"validate": map[string]interface{}{ /*...*/ },
							},
						},
					})},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - policySpec rule missing action",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "no-action"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "global",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{
						"rules": []map[string]interface{}{
							{
								"name":  "a-rule",
								"match": map[string]interface{}{ /*...*/ },
								// "validate": { ... },
							},
						},
					})},
				},
			},
			wantAllowed: false,
		},
		{
			name:        "Delete template allowed",
			op:          admissionv1.Delete,
			template:    kubermaticv1.PolicyTemplate{ObjectMeta: metav1.ObjectMeta{Name: "delete-me"}},
			wantAllowed: true,
		},
		{
			name: "Valid - Lowercase validationFailureAction 'audit'",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "lowercase-audit"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Lowercase",
					Description: "Lowercase action",
					Visibility:  "global",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{
						"validationFailureAction": "audit",
						"rules":                   validMinimalPolicySpec["rules"],
					})},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Valid - Lowercase validationFailureAction 'enforce'",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "lowercase-enforce"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Lowercase",
					Description: "Lowercase action",
					Visibility:  "global",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{
						"validationFailureAction": "enforce",
						"rules":                   validMinimalPolicySpec["rules"],
					})},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Invalid - Bad validationFailureAction",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "bad-action"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Bad Action",
					Description: "Desc",
					Visibility:  "global",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{
						"validationFailureAction": "deny", // Invalid value
						"rules":                   validMinimalPolicySpec["rules"],
					})},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - policySpec rule missing match",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "no-match"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "global",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{
						"rules": []map[string]interface{}{{
							"name": "no-match-rule",
							// "match": { ... }, // Missing
							"validate": map[string]interface{}{"message": "m"},
						}},
					})},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Invalid - policySpec rule empty match",
			op:   admissionv1.Create,
			template: kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "empty-match"},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:       "Title",
					Description: "Desc",
					Visibility:  "global",
					PolicySpec: runtime.RawExtension{Raw: mustMarshalJSON(map[string]interface{}{
						"rules": []map[string]interface{}{{
							"name":     "empty-match-rule",
							"match":    map[string]interface{}{ /* Empty */ },
							"validate": map[string]interface{}{"message": "m"},
						}},
					})},
				},
			},
			wantAllowed: false,
		},
	}

	// Shared validation logic tests
	runValidationTests(t, tests)

	// Update immutability tests
	testImmutability(t)
}

// Helper to run the main set of tests
func runValidationTests(t *testing.T, tests []struct {
	name        string
	op          admissionv1.Operation
	template    kubermaticv1.PolicyTemplate
	wantAllowed bool
}) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	validator := NewValidator(client)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			switch tt.op {
			case admissionv1.Create:
				_, err = validator.ValidateCreate(context.Background(), &tt.template)
			case admissionv1.Update:
				// Create a dummy old object for update validation
				oldTemplate := tt.template.DeepCopy()
				oldTemplate.Spec.Description = "Old description"
				_, err = validator.ValidateUpdate(context.Background(), oldTemplate, &tt.template)
			case admissionv1.Delete:
				_, err = validator.ValidateDelete(context.Background(), &tt.template)
			}

			allowed := err == nil
			if allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t. Error: %v", allowed, tt.wantAllowed, err)
			}
		})
	}
}

// Helper to test immutability checks during update
func testImmutability(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(testScheme).Build()
	validator := NewValidator(client)

	oldGlobalTemplate := kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "immutability-test"},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:       "Old Title",
			Description: "Old Desc",
			Visibility:  "global",
			PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(minMatchActionSpec)},
		},
	}
	oldProjectTemplate := kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "immutability-test-proj"},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:       "Old Title Proj",
			Description: "Old Desc Proj",
			Visibility:  "project",
			ProjectID:   "proj-1",
			PolicySpec:  runtime.RawExtension{Raw: mustMarshalJSON(minMatchActionSpec)},
		},
	}

	tests := []struct {
		name        string
		oldTemplate kubermaticv1.PolicyTemplate
		newTemplate kubermaticv1.PolicyTemplate
		wantAllowed bool
	}{
		{
			name:        "Update allowed field (Description)",
			oldTemplate: oldGlobalTemplate,
			newTemplate: func() kubermaticv1.PolicyTemplate {
				newT := oldGlobalTemplate.DeepCopy()
				newT.Spec.Description = "New Description"
				return *newT
			}(),
			wantAllowed: true,
		},
		{
			name:        "Update immutable field (Visibility)",
			oldTemplate: oldGlobalTemplate,
			newTemplate: func() kubermaticv1.PolicyTemplate {
				newT := oldGlobalTemplate.DeepCopy()
				newT.Spec.Visibility = "project"
				newT.Spec.ProjectID = "proj-x"
				return *newT
			}(),
			wantAllowed: false,
		},
		{
			name:        "Update immutable field (ProjectID)",
			oldTemplate: oldProjectTemplate,
			newTemplate: func() kubermaticv1.PolicyTemplate {
				newT := oldProjectTemplate.DeepCopy()
				newT.Spec.ProjectID = "proj-2"
				return *newT
			}(),
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidateUpdate(context.Background(), &tt.oldTemplate, &tt.newTemplate)
			allowed := err == nil
			if allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t. Error: %v", allowed, tt.wantAllowed, err)
			}
		})
	}
}
