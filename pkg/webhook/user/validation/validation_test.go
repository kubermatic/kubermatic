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
	"errors"
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/validation"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestValidateUserEmailUniqueness(t *testing.T) {
	testCases := []struct {
		name            string
		email           string
		currentUserName string
		existingUsers   []ctrlruntimeclient.Object
		expectError     bool
	}{
		{
			name:            "unique email - no existing users",
			email:           "user@example.com",
			currentUserName: "",
			existingUsers:   []ctrlruntimeclient.Object{},
			expectError:     false,
		},
		{
			name:            "unique email - different existing users",
			email:           "newuser@example.com",
			currentUserName: "",
			existingUsers: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "user1",
					},
					Spec: kubermaticv1.UserSpec{
						Email: "user1@example.com",
						Name:  "User One",
					},
				},
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "user2",
					},
					Spec: kubermaticv1.UserSpec{
						Email: "user2@example.com",
						Name:  "User Two",
					},
				},
			},
			expectError: false,
		},
		{
			name:            "duplicate email - should fail",
			email:           "duplicate@example.com",
			currentUserName: "",
			existingUsers: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-user",
					},
					Spec: kubermaticv1.UserSpec{
						Email: "duplicate@example.com",
						Name:  "Existing User",
					},
				},
			},
			expectError: true,
		},
		{
			name:            "same email but for same user (update scenario)",
			email:           "user@example.com",
			currentUserName: "user1",
			existingUsers: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "user1",
					},
					Spec: kubermaticv1.UserSpec{
						Email: "user@example.com",
						Name:  "User One",
					},
				},
			},
			expectError: false,
		},
		{
			name:            "empty email - should pass (other validators handle this)",
			email:           "",
			currentUserName: "",
			existingUsers:   []ctrlruntimeclient.Object{},
			expectError:     false,
		},
		{
			name:            "service account email - should check uniqueness",
			email:           "serviceaccount-test@test.kubermatic.io",
			currentUserName: "",
			existingUsers: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "serviceaccount-existing",
					},
					Spec: kubermaticv1.UserSpec{
						Email:   "serviceaccount-test@test.kubermatic.io",
						Name:    "Service Account",
						Project: "test-project",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithObjects(tc.existingUsers...).Build()
			ctx := context.Background()

			err := validation.ValidateUserEmailUniqueness(ctx, client, tc.email, tc.currentUserName)

			if tc.expectError && err == nil {
				t.Error("expected validation error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}

			if err != nil {
				var fieldErr *field.Error
				if errors.As(err, &fieldErr) {
					if fieldErr.Type != field.ErrorTypeDuplicate {
						t.Errorf("expected error type 'duplicate' but got '%s'", fieldErr.Type)
					}
					if fieldErr.Field != "spec.email" {
						t.Errorf("expected error field 'spec.email' but got '%s'", fieldErr.Field)
					}
				}
			}
		})
	}
}

func TestValidatorCreate(t *testing.T) {
	testCases := []struct {
		name          string
		user          *kubermaticv1.User
		existingUsers []ctrlruntimeclient.Object
		expectError   bool
		errorContains string
	}{
		{
			name: "valid new user",
			user: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-user",
				},
				Spec: kubermaticv1.UserSpec{
					Email: "newuser@example.com",
					Name:  "New User",
				},
			},
			existingUsers: []ctrlruntimeclient.Object{},
			expectError:   false,
		},
		{
			name: "duplicate email on create",
			user: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-user",
				},
				Spec: kubermaticv1.UserSpec{
					Email: "existing@example.com",
					Name:  "New User",
				},
			},
			existingUsers: []ctrlruntimeclient.Object{
				&kubermaticv1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-user",
					},
					Spec: kubermaticv1.UserSpec{
						Email: "existing@example.com",
						Name:  "Existing User",
					},
				},
			},
			expectError:   true,
			errorContains: "duplicate",
		},
		{
			name: "missing email field",
			user: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-email-user",
				},
				Spec: kubermaticv1.UserSpec{
					Email: "",
					Name:  "User Without Email",
				},
			},
			existingUsers: []ctrlruntimeclient.Object{},
			expectError:   true,
			errorContains: "required",
		},
		{
			name: "missing name field",
			user: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{
					Name: "no-name-user",
				},
				Spec: kubermaticv1.UserSpec{
					Email: "noname@example.com",
					Name:  "",
				},
			},
			existingUsers: []ctrlruntimeclient.Object{},
			expectError:   true,
			errorContains: "required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewClientBuilder().WithObjects(tc.existingUsers...).Build()
			validator := NewValidator(client, nil, nil)
			ctx := context.Background()

			_, err := validator.ValidateCreate(ctx, tc.user)

			if tc.expectError && err == nil {
				t.Error("expected validation error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}

			if tc.expectError && err != nil && tc.errorContains != "" {
				errStr := strings.ToLower(err.Error())
				expectedStr := strings.ToLower(tc.errorContains)
				if !strings.Contains(errStr, expectedStr) {
					t.Errorf("expected error to contain '%s' but got '%s'", tc.errorContains, err.Error())
				}
			}
		})
	}
}
