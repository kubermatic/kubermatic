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

package resources

import (
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/test"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
)

type FakeCredentialsData struct {
	KubermaticCluster                *kubermaticv1.Cluster
	GlobalSecretKeySelectorValueMock func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
}

func (f FakeCredentialsData) Cluster() *kubermaticv1.Cluster {
	return f.KubermaticCluster
}

func (f FakeCredentialsData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return f.GlobalSecretKeySelectorValueMock(configVar, key)
}

func TestGetOpenstackCredentials(t *testing.T) {
	tests := []struct {
		name    string
		spec    *kubermaticv1.OpenstackCloudSpec
		mock    func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
		want    OpenstackCredentials
		wantErr bool
	}{
		// there are 2 kinds of auth mode for openstack which are mutually exclusive
		//   * domain + ApplicationCredential (ApplicationCredentialID and ApplicationCredentialSecret)
		//   * domain + user (i.e. Username, Password, (Project or Tenant) and (ProjectID or tenantID))
		{
			name:    "valid spec with values - auth with user with project",
			spec:    &kubermaticv1.OpenstackCloudSpec{Domain: "domain", ApplicationCredentialID: "", Username: "user", Password: "pass", Project: "the_project", ProjectID: "the_project_id"},
			mock:    test.ShouldNotBeCalled,
			want:    OpenstackCredentials{Username: "user", Password: "pass", Project: "the_project", ProjectID: "the_project_id", Domain: "domain", ApplicationCredentialID: "", ApplicationCredentialSecret: ""},
			wantErr: false,
		},
		{
			name:    "valid spec with values - auth with applicationCredential",
			spec:    &kubermaticv1.OpenstackCloudSpec{Domain: "domain", ApplicationCredentialID: "app_id", ApplicationCredentialSecret: "app_secret"},
			mock:    test.ShouldNotBeCalled,
			want:    OpenstackCredentials{Domain: "domain", ApplicationCredentialID: "app_id", ApplicationCredentialSecret: "app_secret"},
			wantErr: false,
		},
		{
			name:    "valid spec with CredentialsReference - auth with user with project",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{OpenstackApplicationCredentialID: "", OpenstackProject: "the_project", OpenstackProjectID: "the_project_id"}),
			want:    OpenstackCredentials{Username: "username-value", Password: "password-value", Project: "the_project", ProjectID: "the_project_id", Domain: "domain-value", ApplicationCredentialID: "", ApplicationCredentialSecret: ""},
			wantErr: false,
		},
		{
			name:    "valid spec with CredentialsReference - auth with user with tenant( when project not defined it should fallback to tenant)",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{OpenstackApplicationCredentialID: "", OpenstackProject: test.MissingKeyErr(OpenstackProject), OpenstackProjectID: test.MissingKeyErr(OpenstackProjectID), OpenstackTenant: "the_tenant", OpenstackTenantID: "the_tenant_id"}),
			want:    OpenstackCredentials{Username: "username-value", Password: "password-value", Project: "the_tenant", ProjectID: "the_tenant_id", Domain: "domain-value", ApplicationCredentialID: "", ApplicationCredentialSecret: ""},
			wantErr: false,
		},
		{
			name:    "valid spec with CredentialsReference - auth with applicationCredential",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{}),
			want:    OpenstackCredentials{Domain: "domain-value", ApplicationCredentialID: "applicationCredentialID-value", ApplicationCredentialSecret: "applicationCredentialSecret-value"},
			wantErr: false,
		},

		{
			name:    "invalid spec CredentialsReference - missing Domain",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{OpenstackDomain: test.MissingKeyErr(OpenstackDomain)}),
			want:    OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing ApplicationCredentialSecret",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{OpenstackApplicationCredentialID: "applicationCredentialID-value", OpenstackApplicationCredentialSecret: test.MissingKeyErr(OpenstackApplicationCredentialSecret)}),
			want:    OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing username",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{OpenstackApplicationCredentialID: "", OpenstackUsername: test.MissingKeyErr(OpenstackUsername)}),
			want:    OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing password",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{OpenstackApplicationCredentialID: "", OpenstackPassword: test.MissingKeyErr(OpenstackPassword)}),
			want:    OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing Project and tenant",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{OpenstackApplicationCredentialID: "", OpenstackProject: test.MissingKeyErr(OpenstackProject), OpenstackTenant: test.MissingKeyErr(OpenstackTenant)}),
			want:    OpenstackCredentials{},
			wantErr: true,
		},
		{
			name:    "invalid spec CredentialsReference - missing ProjectID and tenantID",
			spec:    &kubermaticv1.OpenstackCloudSpec{CredentialsReference: &providerconfig.GlobalSecretKeySelector{ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"}, Key: "data"}},
			mock:    test.DefaultOrOverride(map[string]interface{}{OpenstackApplicationCredentialID: "", OpenstackProjectID: test.MissingKeyErr(OpenstackProjectID), OpenstackTenantID: test.MissingKeyErr(OpenstackTenantID)}),
			want:    OpenstackCredentials{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentialsData := FakeCredentialsData{
				KubermaticCluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							Openstack: tt.spec},
					},
				},
				GlobalSecretKeySelectorValueMock: tt.mock,
			}
			got, err := GetOpenstackCredentials(credentialsData)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetOpenstackCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOpenstackCredentials() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetVsphereCredentials(t *testing.T) {
	tests := []struct {
		name    string
		spec    *kubermaticv1.VSphereCloudSpec
		mock    func(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
		want    VSphereCredentials
		wantErr bool
	}{
		// vsphere auth can be split into using two sets of credentials or just using one.
		// See https://docs.kubermatic.com/kubermatic/latest/architecture/supported-providers/vsphere/vsphere#permissions for more details.
		// When just using one, username and password will be used (obviously).
		// When using two sets, infraManagementUser.Username and infraManagementUser.Password take precedence here.

		{
			name:    "valid spec with values - auth with only user",
			spec:    &kubermaticv1.VSphereCloudSpec{Username: "user", Password: "pass"},
			mock:    test.ShouldNotBeCalled,
			want:    VSphereCredentials{Username: "user", Password: "pass"},
			wantErr: false,
		},
		{
			name:    "valid spec with values - auth with infraManagementUser",
			spec:    &kubermaticv1.VSphereCloudSpec{Username: "user", Password: "pass", InfraManagementUser: kubermaticv1.VSphereCredentials{Username: "infraUser", Password: "infraPass"}},
			mock:    test.ShouldNotBeCalled,
			want:    VSphereCredentials{Username: "user", Password: "pass"},
			wantErr: false,
		},
		{
			name: "valid spec with credentialsReference - auth with only user",
			spec: &kubermaticv1.VSphereCloudSpec{
				CredentialsReference: &providerconfig.GlobalSecretKeySelector{
					ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"},
					Key:             "data",
				},
				Username: "user",
				Password: "pass",
			},
			mock:    test.DefaultOrOverride(map[string]interface{}{VsphereInfraManagementUserUsername: "", VsphereInfraManagementUserPassword: ""}),
			want:    VSphereCredentials{Username: "user", Password: "pass"},
			wantErr: false,
		},
		{
			name: "valid spec with credentialsReference - auth with infraManagementUser",
			spec: &kubermaticv1.VSphereCloudSpec{
				CredentialsReference: &providerconfig.GlobalSecretKeySelector{
					ObjectReference: corev1.ObjectReference{Name: "the-secret", Namespace: "default"},
					Key:             "data",
				},
				Username: "user",
				Password: "pass",
				InfraManagementUser: kubermaticv1.VSphereCredentials{
					Username: "infraUser",
					Password: "infraPass",
				},
			},
			mock:    test.DefaultOrOverride(map[string]interface{}{VsphereUsername: "user", VspherePassword: "pass", VsphereInfraManagementUserUsername: "infraUser", VsphereInfraManagementUserPassword: "infraPass"}),
			want:    VSphereCredentials{Username: "user", Password: "pass"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			credentialsData := FakeCredentialsData{
				KubermaticCluster: &kubermaticv1.Cluster{
					Spec: kubermaticv1.ClusterSpec{
						Cloud: kubermaticv1.CloudSpec{
							VSphere: tt.spec},
					},
				},
				GlobalSecretKeySelectorValueMock: tt.mock,
			}
			got, err := GetVSphereCredentials(credentialsData)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVSphereCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVSphereCredentials() got = %v, want %v", got, tt.want)
			}
		})
	}
}
