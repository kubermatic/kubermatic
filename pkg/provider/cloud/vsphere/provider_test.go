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

package vsphere

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
)

func TestGetCredentialsForCluster(t *testing.T) {
	tcs := []struct {
		name              string
		cloudspec         kubermaticv1.CloudSpec
		secretKeySelector provider.SecretKeySelectorValueFunc
		dc                *kubermaticv1.DatacenterSpecVSphere
		expectedUser      string
		expectedPasword   string
		expectedError     string
	}{
		{
			name:            "User from cluster",
			cloudspec:       testVsphereCloudSpec("user", "pass", "", "", true),
			expectedUser:    "user",
			expectedPasword: "pass",
		},
		{
			name:      "User from secret",
			cloudspec: testVsphereCloudSpec("", "", "", "", true),
			secretKeySelector: testSecretKeySelectorValueFuncFactory(map[string]string{
				resources.VsphereUsername: "user",
				resources.VspherePassword: "pass",
			}),
			expectedUser:    "user",
			expectedPasword: "pass",
		},
		{
			name:            "InfraManagementUser from clusters InfraManagementUser field",
			cloudspec:       testVsphereCloudSpec("a", "b", "infraManagementUser", "infraManagementUserPass", true),
			expectedUser:    "infraManagementUser",
			expectedPasword: "infraManagementUserPass",
		},
		{
			name:            "InfraManagementUser from clusters user field",
			cloudspec:       testVsphereCloudSpec("user", "pass", "", "", false),
			expectedUser:    "user",
			expectedPasword: "pass",
		},
		{
			name:      "InfraManagementUser from secrets InfraManagementUser field",
			cloudspec: testVsphereCloudSpec("", "", "", "", true),
			secretKeySelector: testSecretKeySelectorValueFuncFactory(map[string]string{
				resources.VsphereInfraManagementUserUsername: "user",
				resources.VsphereInfraManagementUserPassword: "pass",
			}),
			expectedUser:    "user",
			expectedPasword: "pass",
		},
		{
			name:      "InfraManagementUser from secrets User field",
			cloudspec: testVsphereCloudSpec("", "", "", "", true),
			secretKeySelector: testSecretKeySelectorValueFuncFactory(map[string]string{
				resources.VsphereUsername: "user",
				resources.VspherePassword: "pass",
			}),
			expectedUser:    "user",
			expectedPasword: "pass",
		},
		{
			name:      "InfraManagementUser from DC takes precedence",
			cloudspec: testVsphereCloudSpec("", "", "", "", true),
			secretKeySelector: testSecretKeySelectorValueFuncFactory(map[string]string{
				resources.VsphereUsername: "user",
				resources.VspherePassword: "pass",
			}),
			dc: &kubermaticv1.DatacenterSpecVSphere{
				InfraManagementUser: &kubermaticv1.VSphereCredentials{
					Username: "dc-user",
					Password: "dc-pass",
				},
			},
			expectedUser:    "dc-user",
			expectedPasword: "dc-pass",
		},
		{
			name:      "InfraManagementUser from DC takes precedence over InfraMangementUser from cluster",
			cloudspec: testVsphereCloudSpec("", "", "cluster-infra-user", "cluster-infra-pass", true),
			dc: &kubermaticv1.DatacenterSpecVSphere{
				InfraManagementUser: &kubermaticv1.VSphereCredentials{
					Username: "dc-user",
					Password: "dc-pass",
				},
			},
			expectedUser:    "dc-user",
			expectedPasword: "dc-pass",
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic in test %q: %v", tc.name, r)
				}
			}()
			user, password, err := GetCredentialsForCluster(tc.cloudspec, tc.secretKeySelector, tc.dc)
			if (tc.expectedError == "" && err != nil) || (tc.expectedError != "" && err == nil) {
				t.Fatalf("Expected error %q, got error %v", tc.expectedError, err)
			}
			if user != tc.expectedUser {
				t.Errorf("expected user %q, got user %q", tc.expectedUser, user)
			}
			if password != tc.expectedPasword {
				t.Errorf("expected password %q, got password %q", tc.expectedPasword, password)
			}
		})
	}
}

func testSecretKeySelectorValueFuncFactory(values map[string]string) provider.SecretKeySelectorValueFunc {
	return func(_ *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
		if val, ok := values[key]; ok {
			return val, nil
		}
		return "", nil
	}
}

func testVsphereCloudSpec(user, password, infraManagementUser, infraManagementUserPass string, credRefSet bool) kubermaticv1.CloudSpec {
	var credRef *providerconfig.GlobalSecretKeySelector
	if credRefSet {
		credRef = &providerconfig.GlobalSecretKeySelector{}
	}
	return kubermaticv1.CloudSpec{
		VSphere: &kubermaticv1.VSphereCloudSpec{
			CredentialsReference: credRef,
			Username:             user,
			Password:             password,
			InfraManagementUser: kubermaticv1.VSphereCredentials{
				Username: infraManagementUser,
				Password: infraManagementUserPass,
			},
		},
	}
}
