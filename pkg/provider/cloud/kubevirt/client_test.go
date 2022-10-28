/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubevirt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	testCases := []struct {
		name          string
		kubeconfig    string
		errorExpected bool
	}{
		{
			name: "base64 decoded kubeconfig",
			kubeconfig: `
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: dGVzdAo=
    server: http://localhost:6443
  name: test
contexts:
- context:
    cluster: test
    user: default
  name: default
current-context: default
kind: Config
preferences: {}
users:
- name: default
  user:
    token: test.test
`,
			errorExpected: false,
		},
		{
			name:          "base64 encoded kubeconfig",
			kubeconfig:    "YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBkR1Z6ZEFvPQogICAgc2VydmVyOiBodHRwOi8vbG9jYWxob3N0OjY0NDMKICBuYW1lOiB0ZXN0CmNvbnRleHRzOgotIGNvbnRleHQ6CiAgICBjbHVzdGVyOiB0ZXN0CiAgICB1c2VyOiBkZWZhdWx0CiAgbmFtZTogZGVmYXVsdApjdXJyZW50LWNvbnRleHQ6IGRlZmF1bHQKa2luZDogQ29uZmlnCnByZWZlcmVuY2VzOiB7fQp1c2VyczoKLSBuYW1lOiBkZWZhdWx0CiAgdXNlcjoKICAgIHRva2VuOiB0ZXN0LnRlc3QK",
			errorExpected: false,
		},
		{
			name:          "invalid kubeconfig",
			kubeconfig:    "invalidkubeconfig",
			errorExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := NewFakeClient(tc.kubeconfig, ClientOptions{})

			assert.EqualValues(t, err != nil, tc.errorExpected)
			if err != nil {
				return
			}
			assert.NotEmpty(t, client.Client)
			assert.NotNil(t, client.RestConfig)
		})
	}
}
