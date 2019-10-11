package vsphere

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
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
			cloudspec:       testVsphereCloudSpec("user", "pass", "a", "b", true),
			expectedUser:    "user",
			expectedPasword: "pass",
		},
		{
			name:      "User from secret",
			cloudspec: testVsphereCloudSpec("", "", "a", "b", true),
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
			cloudspec: testVsphereCloudSpec("a", "b", "", "", true),
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
	}

	for idx := range tcs {
		t.Run(tcs[idx].name, func(t *testing.T) {
			t.Parallel()
			user, password, err := GetCredentialsForCluster(tcs[idx].cloudspec, tcs[idx].secretKeySelector, tcs[idx].dc)
			if (tcs[idx].expectedError == "" && err != nil) || (tcs[idx].expectedError != "" && err == nil) {
				t.Fatalf("Expected error %q, got error %v", tcs[idx].expectedError, err)
			}
			if user != tcs[idx].expectedUser {
				t.Errorf("expected user %q, got user %q", tcs[idx].expectedUser, user)
			}
			if password != tcs[idx].expectedPasword {
				t.Errorf("expected password %q, got password %q", tcs[idx].expectedPasword, password)
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
