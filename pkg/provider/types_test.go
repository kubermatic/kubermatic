/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package provider

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretKeySelectorValueFuncFactory(t *testing.T) {
	testCases := []struct {
		name      string
		configVar *kubermaticv1.GlobalSecretKeySelector
		secret    *corev1.Secret
		key       string

		expectedError  string
		expectedResult string
	}{
		{
			name:          "error on nil configVar",
			expectedError: "configVar is nil",
		},
		{
			name: "error on empty name",
			configVar: &kubermaticv1.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Namespace: "foo",
				},
			},
			key:           "hello",
			expectedError: "configVar.Name is empty",
		},
		{
			name: "error on empty namespace",
			configVar: &kubermaticv1.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name: "foo",
				},
			},
			key:           "bar",
			expectedError: "configVar.Namespace is empty",
		},
		{
			name: "error on empty key",
			configVar: &kubermaticv1.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Namespace: "default",
					Name:      "foo",
				},
			},
			expectedError: "key is empty",
		},
		{
			name: "happy path",
			configVar: &kubermaticv1.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Namespace: "default",
					Name:      "foo",
				},
			},
			key: "bar",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "foo",
				},
				Data: map[string][]byte{"bar": []byte("value")},
			},
			expectedResult: "value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientBuilder := fakectrlruntimeclient.NewClientBuilder()
			if tc.secret != nil {
				clientBuilder.WithObjects(tc.secret)
			}

			client := clientBuilder.Build()
			valueFunc := SecretKeySelectorValueFuncFactory(context.Background(), client)

			result, err := valueFunc(tc.configVar, tc.key)

			var actualErr string
			if err != nil {
				actualErr = err.Error()
			}

			if actualErr != tc.expectedError {
				t.Fatalf("actual err %q does not match expected err %q", actualErr, tc.expectedError)
			}

			if result != tc.expectedResult {
				t.Errorf("actual result %q does not match expected result %q", result, tc.expectedResult)
			}
		})
	}
}
