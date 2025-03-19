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

package policytemplate

import (
	"context"
	"testing"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	podsecurityapi "k8s.io/pod-security-admission/api"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	testScheme = fake.NewScheme()
)

func TestValidatingWebhook(t *testing.T) {
	tests := []struct {
		name        string
		template    *kubermaticv1.PolicyTemplate
		operation   admissionv1.Operation
		wantAllowed bool
		wantError   string
	}{
		{
			name: "valid global template without project ID",
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Test Template",
					Description:       "Test Description",
					Visibility:        "global",
					KyvernoPolicySpec: kyvernoPolicySpecTest(),
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: true,
		},
		{
			name: "invalid global template with project ID",
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Test Template",
					Description:       "Test Description",
					Visibility:        "global",
					ProjectID:         "test-project",
					KyvernoPolicySpec: kyvernoPolicySpecTest(),
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "global visibility policy templates must not specify a ProjectID",
		},
		{
			name: "invalid global template with no rules or empty kyvernoPolicySpec",
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Test Template",
					Description:       "Test Description",
					Visibility:        "global",
					KyvernoPolicySpec: kyvernov1.Spec{},
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "at least one rule must be specified in kyvernoPolicySpec.rules",
		},
		{
			name: "valid project template with project ID",
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Test Template",
					Description:       "Test Description",
					Visibility:        "project",
					ProjectID:         "test-project",
					KyvernoPolicySpec: kyvernoPolicySpecTest(),
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: true,
		},
		{
			name: "invalid project template without project ID",
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Test Template",
					Description:       "Test Description",
					Visibility:        "project",
					KyvernoPolicySpec: kyvernoPolicySpecTest(),
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "project visibility policy templates must specify a ProjectID",
		},
		{
			name: "invalid visibility value",
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Test Template",
					Description:       "Test Description",
					Visibility:        "invalid",
					KyvernoPolicySpec: kyvernoPolicySpecTest(),
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "visibility must be one of: global, project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			client := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				Build()

			log := zap.NewNop().Sugar()

			// Create handler
			handler := NewValidator(
				log,
				client,
				testScheme,
				func() (*kubermaticv1.Seed, error) {
					return &kubermaticv1.Seed{}, nil
				},
				func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
					return client, nil
				},
			)

			var err error
			switch tt.operation {
			case admissionv1.Create:
				_, err = handler.ValidateCreate(context.Background(), tt.template)
			case admissionv1.Update:
				_, err = handler.ValidateUpdate(context.Background(), nil, tt.template)
			case admissionv1.Delete:
				_, err = handler.ValidateDelete(context.Background(), tt.template)
			}

			if tt.wantAllowed {
				assert.NoError(t, err, "expected validation to pass")
			} else {
				assert.Error(t, err, "expected validation to fail")
				if tt.wantError != "" {
					assert.Contains(t, err.Error(), tt.wantError, "error message should match")
				}
			}
		})
	}
}

func TestValidatingWebhookUpdate(t *testing.T) {
	oldTemplate := &kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-template",
		},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:             "Test Template",
			Description:       "Test Description",
			Visibility:        "project",
			ProjectID:         "test-project",
			KyvernoPolicySpec: kyvernoPolicySpecTest(),
		},
	}

	tests := []struct {
		name        string
		newTemplate *kubermaticv1.PolicyTemplate
		wantAllowed bool
		wantError   string
	}{
		{
			name: "valid update - same visibility and project ID",
			newTemplate: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Updated Title",
					Description:       "Updated Description",
					Visibility:        "project",
					ProjectID:         "test-project",
					KyvernoPolicySpec: kyvernoPolicySpecTest(),
				},
			},
			wantAllowed: true,
		},
		{
			name: "invalid update - different visibility",
			newTemplate: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Updated Title",
					Description:       "Updated Description",
					Visibility:        "global",
					ProjectID:         "test-project",
					KyvernoPolicySpec: kyvernoPolicySpecTest(),
				},
			},
			wantAllowed: false,
			wantError:   "visibility is immutable",
		},
		{
			name: "invalid update - different project ID",
			newTemplate: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Title:             "Updated Title",
					Description:       "Updated Description",
					Visibility:        "project",
					ProjectID:         "different-project",
					KyvernoPolicySpec: kyvernoPolicySpecTest(),
				},
			},
			wantAllowed: false,
			wantError:   "projectID is immutable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			client := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				Build()

			log := zap.NewNop().Sugar()

			// Create handler
			handler := NewValidator(
				log,
				client,
				testScheme,
				func() (*kubermaticv1.Seed, error) {
					return &kubermaticv1.Seed{}, nil
				},
				func(seed *kubermaticv1.Seed) (ctrlruntimeclient.Client, error) {
					return client, nil
				},
			)

			_, err := handler.ValidateUpdate(context.Background(), oldTemplate, tt.newTemplate)

			if tt.wantAllowed {
				assert.NoError(t, err, "expected validation to pass")
			} else {
				assert.Error(t, err, "expected validation to fail")
				if tt.wantError != "" {
					assert.Contains(t, err.Error(), tt.wantError, "error message should match")
				}
			}
		})
	}
}

func kyvernoPolicySpecTest() kyvernov1.Spec {
	return kyvernov1.Spec{
		Rules: []kyvernov1.Rule{
			{
				Name: "test-rule",
				MatchResources: kyvernov1.MatchResources{
					Any: []kyvernov1.ResourceFilter{
						{
							ResourceDescription: kyvernov1.ResourceDescription{
								Kinds: []string{"v1/Pod"},
							},
						},
					},
				},
				Validation: &kyvernov1.Validation{
					Message: "test message",
					PodSecurity: &kyvernov1.PodSecurity{
						Level:   podsecurityapi.LevelBaseline,
						Version: "latest",
					},
				},
			},
		},
	}
}
