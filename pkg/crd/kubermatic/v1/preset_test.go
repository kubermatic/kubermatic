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

package v1

import (
	"testing"
)

func TestOpenstack_GetProjectIdOrDefaultToTenantId(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		tenantID  string
		want      string
	}{
		{
			name:      "projectID should be used if tenantID is not defined",
			projectID: "the_project_id",
			tenantID:  "",
			want:      "the_project_id",
		},
		{
			name:      "projectID should be used even if tenantID is defined",
			projectID: "the_project_id",
			tenantID:  "the_tenant_id",
			want:      "the_project_id",
		},
		{
			name:      "tenantID should be used if projectID is not defined",
			projectID: "",
			tenantID:  "the_tenant_id",
			want:      "the_tenant_id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Openstack{
				TenantID:  tt.tenantID,
				ProjectID: tt.projectID,
			}
			if got := s.GetProjectIdOrDefaultToTenantId(); got != tt.want {
				t.Errorf("GetProjectIdOrDefaultToTenantId() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenstack_GetProjectOrDefaultToTenant(t *testing.T) {
	tests := []struct {
		name    string
		project string
		tenant  string
		want    string
	}{
		{
			name:    "project should be used if tenant is not defined",
			project: "the_project",
			tenant:  "",
			want:    "the_project",
		},
		{
			name:    "project should be used even if tenant is defined",
			project: "the_project",
			tenant:  "the_tenant",
			want:    "the_project",
		},
		{
			name:    "tenant should be used if project is not defined",
			project: "",
			tenant:  "the_tenant",
			want:    "the_tenant",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Openstack{
				Tenant:  tt.tenant,
				Project: tt.project,
			}
			if got := s.GetProjectOrDefaultToTenant(); got != tt.want {
				t.Errorf("GetProjectOrDefaultToTenant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOpenstack_IsValid(t *testing.T) {
	type fields struct {
		UseToken                    bool
		ApplicationCredentialID     string
		ApplicationCredentialSecret string
		Username                    string
		Password                    string
		Domain                      string
		Tenant                      string
		TenantID                    string
		Project                     string
		ProjectID                   string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "should be valid if token is not empty",
			fields: fields{
				UseToken: true,
			},
			want: true,
		},
		{
			name: "should be valid ApplicationCredential are not empty",
			fields: fields{
				UseToken:                    false,
				ApplicationCredentialID:     "the_ApplicationCredentialID",
				ApplicationCredentialSecret: "ApplicationCredentialSecret",
			},
			want: true,
		},
		{
			name: "should be invalid ApplicationCredentialSecret is empty",
			fields: fields{
				UseToken:                    false,
				ApplicationCredentialID:     "the_ApplicationCredentialID",
				ApplicationCredentialSecret: "",
			},
			want: false,
		},
		{
			name: "should be valid is username, password, domain, project are defined",
			fields: fields{
				UseToken: false,
				Username: "user",
				Password: "pass",
				Domain:   "the_domain",
				Project:  "the_project",
			},
			want: true,
		},
		{
			name: "should be valid is username, password, domain, projectID are defined",
			fields: fields{
				UseToken:  false,
				Username:  "user",
				Password:  "pass",
				Domain:    "the_domain",
				ProjectID: "the_project_id",
			},
			want: true,
		},
		{
			name: "should be valid is username, password, domain, tenant are defined",
			fields: fields{
				UseToken: false,
				Username: "user",
				Password: "pass",
				Domain:   "the_domain",
				Tenant:   "the_tenant",
			},
			want: true,
		},
		{
			name: "should be valid is username, password, domain, tenantID are defined",
			fields: fields{
				UseToken: false,
				Username: "user",
				Password: "pass",
				Domain:   "the_domain",
				TenantID: "the_tenant_id",
			},
			want: true,
		},
		{
			name: "should be invalid if username is undefined",
			fields: fields{
				UseToken: false,
				Username: "",
				Password: "pass",
				Domain:   "the_domain",
				TenantID: "the_tenant_id",
			},
			want: false,
		},
		{
			name: "should be invalid if passwaord in undefined",
			fields: fields{
				UseToken: false,
				Username: "user",
				Password: "",
				Domain:   "the_domain",
				TenantID: "the_tenant_id",
			},
			want: false,
		},
		{
			name: "should be invalid if domain in undefined",
			fields: fields{
				UseToken: false,
				Username: "user",
				Password: "pass",
				Domain:   "",
				TenantID: "the_tenant_id",
			},
			want: false,
		},
		{
			name: "should be invalid if project, projectID, tenant and tenantID are undefined",
			fields: fields{
				UseToken:  false,
				Username:  "user",
				Password:  "pass",
				Domain:    "the_domain",
				Project:   "",
				ProjectID: "",
				Tenant:    "",
				TenantID:  "",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Openstack{
				UseToken:                    tt.fields.UseToken,
				ApplicationCredentialID:     tt.fields.ApplicationCredentialID,
				ApplicationCredentialSecret: tt.fields.ApplicationCredentialSecret,
				Username:                    tt.fields.Username,
				Password:                    tt.fields.Password,
				Domain:                      tt.fields.Domain,
				Tenant:                      tt.fields.Tenant,
				TenantID:                    tt.fields.TenantID,
				Project:                     tt.fields.Project,
				ProjectID:                   tt.fields.ProjectID,
			}
			if got := s.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
