/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

func TestValidateGatewayAPIMigrationFlags(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		cliFlag   bool
		helmYAML  string
		wantError bool
	}{
		{
			name:      "flag disabled and value missing",
			cliFlag:   false,
			helmYAML:  "---\n",
			wantError: false,
		},
		{
			name:      "flag enabled and value set to true",
			cliFlag:   true,
			helmYAML:  "migrateGatewayAPI: true\n",
			wantError: false,
		},
		{
			name:      "flag enabled and value missing",
			cliFlag:   true,
			helmYAML:  "---\n",
			wantError: true,
		},
		{
			name:      "flag enabled and value false",
			cliFlag:   true,
			helmYAML:  "migrateGatewayAPI: false\n",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			helmValues, err := yamled.Load(bytes.NewReader([]byte(tc.helmYAML)))
			require.NoError(t, err)

			err = validateGatewayAPIMigrationFlags(&DeployOptions{MigrateGatewayAPI: tc.cliFlag}, helmValues)

			if tc.wantError {
				require.ErrorContains(t, err, "--migrate-gateway-api requires the migrateGatewayAPI Helm value to be set to true")
				return
			}

			require.NoError(t, err)
		})
	}
}
