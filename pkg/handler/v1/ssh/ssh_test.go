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

package ssh_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDeleteSSHKey(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		HTTPStatus             int
		SSHKeyToDelete         string
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingAPIUser        *apiv1.User
		ExistingSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name:           "scenario 1: delete an ssh-keyfrom from a project",
			HTTPStatus:     http.StatusOK,
			SSHKeyToDelete: "key-abc-second-key",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				test.GenDefaultCluster(),
				/*add ssh keys*/
				genSSHKey(test.DefaultCreationTimestamp(), "c08aa5c7abf34504f18552846485267d", "first-key", "my-first-project-ID", test.GenDefaultCluster().Name),
				genSSHKey(test.DefaultCreationTimestamp(), "abc", "second-key", "my-first-project-ID", "abcd-ID"),
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 2
		{
			Name:           "scenario 2: the admin user can delete SSH key from any project",
			HTTPStatus:     http.StatusOK,
			SSHKeyToDelete: "key-abc-second-key",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("admin", "admin@acme.com", true),
				/*add cluster*/
				test.GenDefaultCluster(),
				/*add ssh keys*/
				genSSHKey(test.DefaultCreationTimestamp(), "c08aa5c7abf34504f18552846485267d", "first-key", "my-first-project-ID", test.GenDefaultCluster().Name),
				genSSHKey(test.DefaultCreationTimestamp(), "abc", "second-key", "my-first-project-ID", "abcd-ID"),
			},
			ExistingAPIUser: test.GenAPIUser("admin", "admin@acme.com"),
		},
		// scenario 3
		{
			Name:           "scenario 3: the user who doesn't belong to the project can not delete SSH key from the project",
			HTTPStatus:     http.StatusForbidden,
			SSHKeyToDelete: "key-abc-second-key",
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("user", "user@acme.com", false),
				/*add cluster*/
				test.GenDefaultCluster(),
				/*add ssh keys*/
				genSSHKey(test.DefaultCreationTimestamp(), "c08aa5c7abf34504f18552846485267d", "first-key", "my-first-project-ID", test.GenDefaultCluster().Name),
				genSSHKey(test.DefaultCreationTimestamp(), "abc", "second-key", "my-first-project-ID", "abcd-ID"),
			},
			ExistingAPIUser: test.GenAPIUser("user", "user@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			sshKeyID := tc.SSHKeyToDelete
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/sshkeys/%s", "my-first-project-ID", sshKeyID), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []ctrlruntimeclient.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

		})
	}
}

func TestListSSHKeys(t *testing.T) {
	t.Parallel()
	creationTime := test.DefaultCreationTimestamp()

	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedKeys           []apiv1.SSHKey
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name: "scenario 1: gets a list of ssh keys assigned to cluster",
			Body: ``,
			ExpectedKeys: []apiv1.SSHKey{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "key-c08aa5c7abf34504f18552846485267d-first-key",
						Name:              "first-key",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "key-abc-second-key",
						Name:              "second-key",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 55, 0, 0, time.UTC),
					},
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				test.GenDefaultCluster(),
				/*add ssh keys*/
				genSSHKey(creationTime, "c08aa5c7abf34504f18552846485267d", "first-key", "my-first-project-ID", test.GenDefaultCluster().Name),
				genSSHKey(creationTime.Add(time.Minute), "abc", "second-key", "my-first-project-ID", "abcd-ID"),
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 2
		{
			Name: "scenario 2: the admin can gets a list of ssh keys assigned to cluster for any project",
			Body: ``,
			ExpectedKeys: []apiv1.SSHKey{
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "key-c08aa5c7abf34504f18552846485267d-first-key",
						Name:              "first-key",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
				},
				{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "key-abc-second-key",
						Name:              "second-key",
						CreationTimestamp: apiv1.Date(2013, 02, 03, 19, 55, 0, 0, time.UTC),
					},
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				test.GenDefaultCluster(),
				genUser("admin", "admin@acme.com", true),
				/*add ssh keys*/
				genSSHKey(creationTime, "c08aa5c7abf34504f18552846485267d", "first-key", "my-first-project-ID", test.GenDefaultCluster().Name),
				genSSHKey(creationTime.Add(time.Minute), "abc", "second-key", "my-first-project-ID", "abcd-ID"),
			},
			ExistingAPIUser: test.GenAPIUser("admin", "admin@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/sshkeys", "my-first-project-ID"), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []ctrlruntimeclient.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualKeys := test.NewSSHKeyV1SliceWrapper{}
			actualKeys.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedKeys := test.NewSSHKeyV1SliceWrapper(tc.ExpectedKeys)
			wrappedExpectedKeys.Sort()
			actualKeys.EqualOrDie(wrappedExpectedKeys, t)
		})
	}
}

func TestCreateSSHKeysEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		Body                   string
		RewriteSSHKeyID        bool
		ExpectedResponse       string
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingKubermaticObjs []ctrlruntimeclient.Object
		ExistingAPIUser        *apiv1.User
	}{
		// scenario 1
		{
			Name:             "scenario 1: a user can create ssh key that will be assigned to the given project",
			Body:             `{"name":"my-second-ssh-key","spec":{"publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com "}}`,
			RewriteSSHKeyID:  true,
			ExpectedResponse: `{"id":"%s","name":"my-second-ssh-key","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"fingerprint":"c0:8a:a5:c7:ab:f3:45:04:f1:85:52:84:64:85:26:7d","publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com "}}`,
			HTTPStatus:       http.StatusCreated,
			ExistingProject:  test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				test.GenDefaultCluster(),
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 2
		{
			Name:             "scenario 2: a user can't create ssh with already existing name",
			Body:             `{"name":"my-second-ssh-key","spec":{"publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com "}}`,
			ExpectedResponse: `{"error":{"code":409,"message":"ssh key \"my-second-ssh-key\" already exists"}}`,
			HTTPStatus:       http.StatusConflict,
			ExistingProject:  test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				/*add cluster*/
				test.GenDefaultCluster(),
				/*add sshkeys*/
				genSSHKey(test.DefaultCreationTimestamp(), "d08aa5d7bce34504f18552846485267c", "my-second-ssh-key", "my-first-project-ID", test.GenDefaultCluster().Name),
			},
			ExistingAPIUser: test.GenAPIUser("john", "john@acme.com"),
		},
		// scenario 3
		{
			Name:             "scenario 3: the admin can create ssh key that will be assigned to any project",
			Body:             `{"name":"my-second-ssh-key","spec":{"publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com "}}`,
			RewriteSSHKeyID:  true,
			ExpectedResponse: `{"id":"%s","name":"my-second-ssh-key","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"fingerprint":"c0:8a:a5:c7:ab:f3:45:04:f1:85:52:84:64:85:26:7d","publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== user@example.com "}}`,
			HTTPStatus:       http.StatusCreated,
			ExistingProject:  test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
			ExistingKubermaticObjs: []ctrlruntimeclient.Object{
				/*add projects*/
				test.GenProject("my-first-project", kubermaticv1.ProjectActive, test.DefaultCreationTimestamp()),
				/*add bindings*/
				test.GenBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				test.GenUser("", "john", "john@acme.com"),
				genUser("admin", "admin@acme.com", true),
				/*add cluster*/
				test.GenDefaultCluster(),
			},
			ExistingAPIUser: test.GenAPIUser("admin", "admin@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/sshkeys", "my-first-project-ID"), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []ctrlruntimeclient.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := test.CreateTestEndpoint(*tc.ExistingAPIUser, []ctrlruntimeclient.Object{}, kubermaticObj, nil, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			expectedResponse := tc.ExpectedResponse
			// since SSH Key ID is automatically generated by the system just rewrite it.
			if tc.RewriteSSHKeyID {
				actualSSHKey := &apiv1.SSHKey{}
				err = json.Unmarshal(res.Body.Bytes(), actualSSHKey)
				if err != nil {
					t.Fatal(err)
				}
				expectedResponse = fmt.Sprintf(tc.ExpectedResponse, actualSSHKey.ID)
			}

			test.CompareWithResult(t, res, expectedResponse)
		})
	}
}

func genSSHKey(creationTime time.Time, keyID string, keyName string, projectID string, clusters ...string) *kubermaticv1.UserSSHKey {
	return &kubermaticv1.UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("key-%s-%s", keyID, keyName),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "kubermatic.k8s.io/v1",
					Kind:       "Project",
					UID:        "",
					Name:       projectID,
				},
			},
			CreationTimestamp: metav1.NewTime(creationTime),
		},
		Spec: kubermaticv1.SSHKeySpec{
			Name:     keyName,
			Clusters: clusters,
		},
	}
}

func genUser(name, email string, isAdmin bool) *kubermaticv1.User {
	user := test.GenUser("", name, email)
	user.Spec.IsAdmin = isAdmin
	return user
}
