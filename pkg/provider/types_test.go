package provider

import (
	"context"
	"testing"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSecretKeySelectorValueFuncFactory(t *testing.T) {
	testCases := []struct {
		name      string
		configVar *providerconfig.GlobalSecretKeySelector
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
			configVar: &providerconfig.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Namespace: "foo",
				},
			},
			key:           "hello",
			expectedError: "configVar.Name is empty",
		},
		{
			name: "error on empty namespace",
			configVar: &providerconfig.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Name: "foo",
				},
			},
			key:           "bar",
			expectedError: "configVar.Namspace is empty",
		},
		{
			name: "error on empty key",
			configVar: &providerconfig.GlobalSecretKeySelector{
				ObjectReference: corev1.ObjectReference{
					Namespace: "default",
					Name:      "foo",
				},
			},
			expectedError: "key is empty",
		},
		{
			name: "happy path",
			configVar: &providerconfig.GlobalSecretKeySelector{
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
			var client ctrlruntimeclient.Client
			if tc.secret != nil {
				client = fakectrlruntimeclient.NewFakeClient(tc.secret)
			} else {
				client = fakectrlruntimeclient.NewFakeClient()
			}

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
