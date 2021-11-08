package resources

import (
	"reflect"
	"testing"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/test"

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
		// there are 2 kinds of auth mode for openstack which are mutualy exclusive
		//   * domain + ApplicationCredential (ApplicationCredentialID and ApplicationCredentialSecret)
		//   * domain + user (ie  Username, Password, (Project or Tenant) and (ProjectID or tenantID))
		{
			name:    "valid spec with values - auth with user with project",
			spec:    &kubermaticv1.OpenstackCloudSpec{Domain: "domain", ApplicationCredentialID: "", Username: "user", Password: "pass", Project: "the_project", ProjectID: "the_project_id"},
			mock:    test.ShouldNotBeCalled,
			want:    OpenstackCredentials{Username: "user", Password: "pass", Project: "the_project", ProjectID: "the_project_id", Domain: "domain", ApplicationCredentialID: "", ApplicationCredentialSecret: ""},
			wantErr: false,
		},
		{
			name:    "valid spec with values - auth with user with tenant( when project not defined it should fallback to tenant)",
			spec:    &kubermaticv1.OpenstackCloudSpec{Domain: "domain", ApplicationCredentialID: "", Username: "user", Password: "pass", Tenant: "the_tenant", TenantID: "the_tenant_id"},
			mock:    test.ShouldNotBeCalled,
			want:    OpenstackCredentials{Username: "user", Password: "pass", Project: "the_tenant", ProjectID: "the_tenant_id", Domain: "domain", ApplicationCredentialID: "", ApplicationCredentialSecret: ""},
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
