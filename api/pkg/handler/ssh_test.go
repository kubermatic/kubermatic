package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clienttesting "k8s.io/client-go/testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

func TestDeleteSSHKey(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		Name                   string
		HTTPStatus             int
		SSHKeyToDelete         string
		ExistingKubermaticObjs []runtime.Object
		ExistingAPIUser        *apiv1.LegacyUser
		ExistingSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name:           "scenario 1: delete an ssh-keyfrom from a project",
			HTTPStatus:     http.StatusOK,
			SSHKeyToDelete: "key-abc-second-key",
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
				/*add bindings*/
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				genUser("", "john", "john@acme.com"),
				/*add cluster*/
				genDefaultCluster(),
				/*add ssh keys*/
				genSSHKey(defaultCreationTimestamp(), "c08aa5c7abf34504f18552846485267d", "first-key", "my-first-project-ID", genDefaultCluster().Name),
				genSSHKey(defaultCreationTimestamp(), "abc", "second-key", "my-first-project-ID", "abcd-ID"),
			},
			ExistingAPIUser: genAPIUser("john", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			sshKeyID := tc.SSHKeyToDelete
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/sshkeys/%s", testingProjectName, sshKeyID), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, clients, err := createTestEndpointAndGetClients(*tc.ExistingAPIUser, nil, []runtime.Object{}, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			kubermaticFakeClient := clients.fakeKubermaticClient
			{
				// check only if ssh key was delteted
				if tc.HTTPStatus == http.StatusOK {
					actionWasValidated := false
					for _, action := range kubermaticFakeClient.Actions() {
						if action.Matches("delete", "usersshkeies") {
							deleteAction, ok := action.(clienttesting.DeleteAction)
							if !ok {
								t.Fatalf("unexpected action %#v", action)
							}
							if deleteAction.GetName() != tc.SSHKeyToDelete {
								t.Fatalf("wrong ssh-key removed, wanted = %s, actual = %s", tc.SSHKeyToDelete, deleteAction.GetName())
							}
							actionWasValidated = true
							break
						}
					}
					if !actionWasValidated {
						t.Fatal("create action was not validated, a binding for a user was not updated ?")
					}
				}
			}
		})
	}
}

func TestListSSHKeys(t *testing.T) {
	t.Parallel()
	creationTime := defaultCreationTimestamp()

	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedKeys           []apiv1.SSHKey
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingKubermaticObjs []runtime.Object
		ExistingAPIUser        *apiv1.LegacyUser
		ExistingCluster        *kubermaticv1.Cluster
		ExistingSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name: "scenario 1: gets a list of ssh keys assigned to cluster",
			Body: ``,
			ExpectedKeys: []apiv1.SSHKey{
				apiv1.SSHKey{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "key-c08aa5c7abf34504f18552846485267d-first-key",
						Name:              "first-key",
						CreationTimestamp: time.Date(2013, 02, 03, 19, 54, 0, 0, time.UTC),
					},
				},
				apiv1.SSHKey{
					ObjectMeta: apiv1.ObjectMeta{
						ID:                "key-abc-second-key",
						Name:              "second-key",
						CreationTimestamp: time.Date(2013, 02, 03, 19, 55, 0, 0, time.UTC),
					},
				},
			},
			HTTPStatus: http.StatusOK,
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
				/*add bindings*/
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				genUser("", "john", "john@acme.com"),
				/*add cluster*/
				genDefaultCluster(),
				/*add ssh keys*/
				genSSHKey(creationTime, "c08aa5c7abf34504f18552846485267d", "first-key", "my-first-project-ID", genDefaultCluster().Name),
				genSSHKey(creationTime.Add(time.Minute), "abc", "second-key", "my-first-project-ID", "abcd-ID"),
			},
			ExistingAPIUser: genAPIUser("john", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/sshkeys", testingProjectName), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}

			actualKeys := newSSHKeyV1SliceWrapper{}
			actualKeys.DecodeOrDie(res.Body, t).Sort()

			wrappedExpectedKeys := newSSHKeyV1SliceWrapper(tc.ExpectedKeys)
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
		ExistingKubermaticObjs []runtime.Object
		ExistingAPIUser        *apiv1.LegacyUser
	}{
		// scenario 1
		{
			Name:             "scenario 1: a user can create ssh key that will be assigned to the given project",
			Body:             `{"name":"my-second-ssh-key","spec":{"publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com "}}`,
			RewriteSSHKeyID:  true,
			ExpectedResponse: `{"id":"%s","name":"my-second-ssh-key","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"fingerprint":"c0:8a:a5:c7:ab:f3:45:04:f1:85:52:84:64:85:26:7d","publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com "}}`,
			HTTPStatus:       http.StatusCreated,
			ExistingProject:  genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
				/*add bindings*/
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				genUser("", "john", "john@acme.com"),
				/*add cluster*/
				genDefaultCluster(),
			},
			ExistingAPIUser: genAPIUser("john", "john@acme.com"),
		},
		// scenario 2
		{
			Name:             "scenario 2: a user can't create ssh with already existing name",
			Body:             `{"name":"my-second-ssh-key","spec":{"publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com "}}`,
			ExpectedResponse: `{"error":{"code":409,"message":"ssh key \"my-second-ssh-key\" already exists"}}`,
			HTTPStatus:       http.StatusConflict,
			ExistingProject:  genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
			ExistingKubermaticObjs: []runtime.Object{
				/*add projects*/
				genProject("my-first-project", kubermaticv1.ProjectActive, defaultCreationTimestamp()),
				/*add bindings*/
				genBinding("my-first-project-ID", "john@acme.com", "owners"),
				/*add users*/
				genUser("", "john", "john@acme.com"),
				/*add cluster*/
				genDefaultCluster(),
				/*add sshkeys*/
				genSSHKey(defaultCreationTimestamp(), "d08aa5d7bce34504f18552846485267c", "my-second-ssh-key", "my-first-project-ID", genDefaultCluster().Name),
			},
			ExistingAPIUser: genAPIUser("john", "john@acme.com"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/sshkeys", testingProjectName), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticObjs...)
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
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

			compareWithResult(t, res, expectedResponse)
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
