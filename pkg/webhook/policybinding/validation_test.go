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

package policybinding

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	testScheme = fake.NewScheme()
)

func TestValidatingWebhook(t *testing.T) {
	tests := []struct {
		name        string
		binding     *kubermaticv1.PolicyBinding
		template    *kubermaticv1.PolicyTemplate
		operation   admissionv1.Operation
		wantAllowed bool
		wantError   string
	}{
		{
			name: "valid global binding with global template",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "global",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{
						Projects: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "global",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: true,
		},
		{
			name: "invalid global binding - project template",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "global",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{
						Projects: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "project",
					ProjectID:  "test-project",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "global scope policy bindings can only reference policy templates with global visibility",
		},
		{
			name: "invalid global binding - missing projects selector",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "global",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "global",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "global scope policy bindings must specify target projects",
		},
		{
			name: "invalid global binding - with clusters selector",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "global",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{
						Projects: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
						Clusters: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "global",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "global scope policy bindings must not specify target clusters",
		},
		{
			name: "valid project binding with project template",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "project",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{
						Clusters: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "project",
					ProjectID:  "test-project",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: true,
		},
		{
			name: "invalid project binding - global template",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "project",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{
						Clusters: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "global",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "project scope policy bindings can only reference policy templates with project visibility",
		},
		{
			name: "invalid project binding - missing project ID",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "project",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{
						Clusters: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "project",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "project-visible policy templates must specify a ProjectID",
		},
		{
			name: "invalid project binding - with projects selector",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "project",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{
						Projects: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
						Clusters: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "project",
					ProjectID:  "test-project",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "project scope policy bindings must not specify target projects",
		},
		{
			name: "invalid project binding - missing clusters selector",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "project",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "project",
					ProjectID:  "test-project",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "project scope policy bindings must specify target clusters",
		},
		{
			name: "invalid scope value",
			binding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "invalid",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "project",
					ProjectID:  "test-project",
				},
			},
			operation:   admissionv1.Create,
			wantAllowed: false,
			wantError:   "scope must be one of: global, project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with template
			client := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(tt.template).
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
				_, err = handler.ValidateCreate(context.Background(), tt.binding)
			case admissionv1.Update:
				_, err = handler.ValidateUpdate(context.Background(), nil, tt.binding)
			case admissionv1.Delete:
				_, err = handler.ValidateDelete(context.Background(), tt.binding)
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
	oldBinding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-binding",
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			Scope: "project",
			PolicyTemplateRef: corev1.ObjectReference{
				Name: "test-template",
			},
		},
	}

	tests := []struct {
		name        string
		newBinding  *kubermaticv1.PolicyBinding
		template    *kubermaticv1.PolicyTemplate
		wantAllowed bool
	}{
		{
			name: "valid update - same scope",
			newBinding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "project",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
					Target: kubermaticv1.PolicyTargetSpec{
						Clusters: kubermaticv1.ResourceSelector{
							SelectAll: true,
						},
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "project",
					ProjectID:  "test-project",
				},
			},
			wantAllowed: true,
		},
		{
			name: "invalid update - different scope",
			newBinding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					Scope: "global",
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-template",
					},
				},
			},
			template: &kubermaticv1.PolicyTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-template",
				},
				Spec: kubermaticv1.PolicyTemplateSpec{
					Visibility: "global",
				},
			},
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client with template
			client := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(tt.template).
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

			_, err := handler.ValidateUpdate(context.Background(), oldBinding, tt.newBinding)

			if tt.wantAllowed {
				assert.NoError(t, err, "expected validation to pass")
			} else {
				assert.Error(t, err, "expected validation to fail")
			}
		})
	}
}
