package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

func TestCreateSSHKeysEndpoint(t *testing.T) {
	testcases := []struct {
		Name                   string
		Body                   string
		RewriteProjectID       bool
		ExpectedResponse       string
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
	}{
		// scenario 1
		{
			Name:             "scenario 1: a user can create ssh key that will be assigned to the given project",
			Body:             `{"metadata":{"displayName":"my-second-ssh-key"},"spec":{"publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com "}}`,
			ExpectedResponse: `{"metadata":{"name":"%s","displayName":"","creationTimestamp":"0001-01-01T00:00:00Z"},"spec":{"fingerprint":"c0:8a:a5:c7:ab:f3:45:04:f1:85:52:84:64:85:26:7d","publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com "}}`,
			HTTPStatus:       http.StatusCreated,
			ExistingProject: &kubermaticv1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name: "myProjectInternalName",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.io/v1",
							Kind:       "User",
							UID:        "",
							Name:       "my-first-project",
						},
					},
				},
				Spec: kubermaticv1.ProjectSpec{Name: "my-first-project"},
			},
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUsername,
				Email: testEmail,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/projects/myProjectInternalName/sshkeys", strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			expectedResponse := tc.ExpectedResponse
			{
				actualSSHKey := &apiv2.NewSSHKey{}
				err = json.Unmarshal(res.Body.Bytes(), actualSSHKey)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualSSHKey.Metadata.Name)
			}
			compareWithResult(t, res, expectedResponse)
		})
	}
}
func TestSSHKeysEndpoint(t *testing.T) {
	t.Parallel()
	keyList := []runtime.Object{
		&kubermaticv1.UserSSHKey{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user1-1",
			},
			Spec: kubermaticv1.SSHKeySpec{
				Owner:       "user1",
				PublicKey:   "AAAAAAAAAAAAAAA",
				Fingerprint: "BBBBBBBBBBBBBBB",
				Name:        "user1-1",
				Clusters:    []string{},
			},
		},
		&kubermaticv1.UserSSHKey{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user1-2",
			},
			Spec: kubermaticv1.SSHKeySpec{
				Owner:       "user1",
				PublicKey:   "CCCCCCCCCCCCCCC",
				Fingerprint: "DDDDDDDDDDDDDDD",
				Name:        "user1-2",
				Clusters:    []string{},
			},
		},
		&kubermaticv1.UserSSHKey{
			ObjectMeta: metav1.ObjectMeta{
				Name: "user2-1",
			},
			Spec: kubermaticv1.SSHKeySpec{
				Owner:       "user2",
				PublicKey:   "EEEEEEEEEEEEEEE",
				Fingerprint: "FFFFFFFFFFFFFFF",
				Name:        "user2-1",
				Clusters:    []string{},
			},
		},
	}

	tests := []struct {
		name         string
		wantKeyNames []string
		username     string
		admin        bool
	}{
		{
			name:         "got user1 keys",
			wantKeyNames: []string{"user1-1", "user1-2"},
			username:     testUsername,
			admin:        false,
		},
		{
			name:         "got user2 keys",
			wantKeyNames: []string{"user2-1"},
			username:     "user2",
			admin:        false,
		},
		{
			name:         "got no keys",
			wantKeyNames: []string{},
			username:     "does-not-exist",
			admin:        false,
		},
		{
			name:         "admin got all keys",
			wantKeyNames: []string{"user1-1", "user1-2", "user2-1"},
			username:     testUsername,
			admin:        true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/ssh-keys", nil)
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(getUser(test.username, test.admin), []runtime.Object{}, keyList, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			checkStatusCode(http.StatusOK, res, t)

			var gotKeys []kubermaticv1.UserSSHKey
			err = json.Unmarshal(res.Body.Bytes(), &gotKeys)
			if err != nil {
				t.Fatal(err, res.Body.String())
			}

			var gotKeyNames []string
			for _, k := range gotKeys {
				gotKeyNames = append(gotKeyNames, k.Name)
			}

			if len(gotKeyNames) != len(test.wantKeyNames) {
				t.Errorf("got more/less keys than expected. Got: %v Want: %v", gotKeyNames, test.wantKeyNames)
			}

			for _, w := range test.wantKeyNames {
				found := false
				for _, g := range gotKeyNames {
					if w == g {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("could not find key %s", w)
				}
			}
		})
	}
}
