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
	const longForm = "Jan 2, 2006 at 3:04pm (MST)"
	creationTime, err := time.Parse(longForm, "Feb 3, 2013 at 7:54pm (PST)")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		Name                   string
		HTTPStatus             int
		SSHKeyToDelete         string
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name:            "scenario 1: delete a ssh-keyfrom from a specific project",
			HTTPStatus:      http.StatusOK,
			SSHKeyToDelete:  "key-abc-yafn",
			ExistingProject: createTestProject("my-first-project", kubermaticv1.ProjectActive),
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingSSHKeys: []*kubermaticv1.UserSSHKey{
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       testingProjectName,
							},
						},
						CreationTimestamp: metav1.NewTime(creationTime),
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "yafn",
						Clusters: []string{"abcd"},
					},
				},
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-abc-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       testingProjectName,
							},
						},
						CreationTimestamp: metav1.NewTime(creationTime.Add(time.Minute)),
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "abcd",
						Clusters: []string{"abcd"},
					},
				},
			},
			ExistingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       testingProjectName,
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			sshKeyID := tc.SSHKeyToDelete
			req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/projects/%s/sshkeys/%s", testingProjectName, sshKeyID), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingCluster != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingCluster)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			for _, existingKey := range tc.ExistingSSHKeys {
				kubermaticObj = append(kubermaticObj, existingKey)
			}
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
	const longForm = "Jan 2, 2006 at 3:04pm (MST)"
	creationTime, err := time.Parse(longForm, "Feb 3, 2013 at 7:54pm (PST)")
	if err != nil {
		t.Fatal(err)
	}

	testcases := []struct {
		Name                   string
		Body                   string
		ExpectedResponse       string
		HTTPStatus             int
		ExistingProject        *kubermaticv1.Project
		ExistingKubermaticUser *kubermaticv1.User
		ExistingAPIUser        *apiv1.User
		ExistingCluster        *kubermaticv1.Cluster
		ExistingSSHKeys        []*kubermaticv1.UserSSHKey
	}{
		// scenario 1
		{
			Name:             "scenario 1: gets a list of ssh keys assigned to cluster",
			Body:             ``,
			ExpectedResponse: `[{"id":"key-c08aa5c7abf34504f18552846485267d-yafn","name":"yafn","creationTimestamp":"2013-02-03T19:54:00Z","spec":{"fingerprint":"","publicKey":""}},{"id":"key-abc-yafn","name":"abcd","creationTimestamp":"2013-02-03T19:55:00Z","spec":{"fingerprint":"","publicKey":""}}]`,
			HTTPStatus:       http.StatusOK,
			ExistingProject:  createTestProject("my-first-project", kubermaticv1.ProjectActive),
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
			ExistingSSHKeys: []*kubermaticv1.UserSSHKey{
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-c08aa5c7abf34504f18552846485267d-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       testingProjectName,
							},
						},
						CreationTimestamp: metav1.NewTime(creationTime),
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "yafn",
						Clusters: []string{"abcd"},
					},
				},
				&kubermaticv1.UserSSHKey{
					ObjectMeta: metav1.ObjectMeta{
						Name: "key-abc-yafn",
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion: "kubermatic.k8s.io/v1",
								Kind:       "Project",
								UID:        "",
								Name:       testingProjectName,
							},
						},
						CreationTimestamp: metav1.NewTime(creationTime.Add(time.Minute)),
					},
					Spec: kubermaticv1.SSHKeySpec{
						Name:     "abcd",
						Clusters: []string{"abcd"},
					},
				},
			},
			ExistingCluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "abcd",
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "kubermatic.k8s.io/v1",
							Kind:       "Project",
							UID:        "",
							Name:       testingProjectName,
						},
					},
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/sshkeys", testingProjectName), strings.NewReader(tc.Body))
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{}
			if tc.ExistingProject != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingProject)
			}
			if tc.ExistingCluster != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingCluster)
			}
			if tc.ExistingKubermaticUser != nil {
				kubermaticObj = append(kubermaticObj, tc.ExistingKubermaticUser)
			}
			for _, existingKey := range tc.ExistingSSHKeys {
				kubermaticObj = append(kubermaticObj, existingKey)
			}
			ep, err := createTestEndpoint(*tc.ExistingAPIUser, []runtime.Object{}, kubermaticObj, nil, nil)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}

			ep.ServeHTTP(res, req)

			if res.Code != tc.HTTPStatus {
				t.Fatalf("Expected HTTP status code %d, got %d: %s", tc.HTTPStatus, res.Code, res.Body.String())
			}
			compareWithResult(t, res, tc.ExpectedResponse)
		})
	}
}

func TestCreateSSHKeysEndpoint(t *testing.T) {
	t.Parallel()
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
			Body:             `{"name":"my-second-ssh-key","spec":{"publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com "}}`,
			ExpectedResponse: `{"id":"%s","name":"my-second-ssh-key","creationTimestamp":"0001-01-01T00:00:00Z","spec":{"fingerprint":"c0:8a:a5:c7:ab:f3:45:04:f1:85:52:84:64:85:26:7d","publicKey":"ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQC8LlXSRW4HUYAjzx1+r5JzpjXIDDyFkWZzBQ8aU14J8LdMyQsU6/ZKuO5IKoWWVoPi0e63qSjkXPTjnUAwpE62hDm6uLaPgIlc3ND+8d9xbItS+gyXk9TSkC3emrsCWpS76W3KjLwyz5euIfnMCQZSASM7F5CrNg6XSppOgRWlyY09VEKi9PmvEDKCy5JNt6afcUzB3rAOK3SYZ0BYDyrVjuqTcMZwRodryxKb/jxDS+qQNplBNuUBqUzqjuKyI5oAk+aVTYIfTwgBTQyZT7So/u70gSDbRp9uHI05PkH60IftAHdYu4TJTmCwJxLW/suOEx3PPvIsUP14XQUZgmDJEuIuWDlsvfOo9DXZNnl832SGvTyhclBpsauWJ1OwOllT+hlM7u8dwcb70GD/OzCG7RSEatVoiNtg4XdeUf4kiqqzKZEqpopHQqwVKMhlhPKKulY0vrtetJxaLokEwPOYyycxlXsNBK2ei/IbGan+uI39v0s30ySWKzr+M9z0QlLAG7rjgCSWFSmy+Ez2fxU5HQQTNCep8+VjNeI79uO9VDJ8qvV/y6fDtrwgl67hUgDcHyv80TzVROTGFBMCP7hyswArT0GxpL9q7PjPU92D43UEDY5YNOZN2A976O5jd4bPrWp0mKsye1BhLrct16Xdn9x68D8nS2T1uSSWovFhkQ== lukasz@loodse.com "}}`,
			HTTPStatus:       http.StatusCreated,
			ExistingProject:  createTestProject("my-first-project", kubermaticv1.ProjectActive),
			ExistingKubermaticUser: &kubermaticv1.User{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: kubermaticv1.UserSpec{
					Name:  "John",
					Email: testUserEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-" + testingProjectName,
							Name:  testingProjectName,
						},
					},
				},
			},
			ExistingAPIUser: &apiv1.User{
				ID:    testUserName,
				Email: testUserEmail,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			req := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/projects/%s/sshkeys", testingProjectName), strings.NewReader(tc.Body))
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

			actualSSHKey := &apiv1.NewSSHKey{}
			err = json.Unmarshal(res.Body.Bytes(), actualSSHKey)
			if err != nil {
				t.Fatal(err)
			}
			expectedResponse := fmt.Sprintf(tc.ExpectedResponse, actualSSHKey.ID)

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
				Owner:       "1233",
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
				Owner:       "1233",
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
				Owner:       "222",
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
		useremail    string
		userid       string
		admin        bool
	}{
		{
			name:         "got user1 keys",
			wantKeyNames: []string{"user1-1", "user1-2"},
			username:     testUserName,
			useremail:    testUserEmail,
			userid:       testUserID,
			admin:        false,
		},
		{
			name:         "got user2 keys",
			wantKeyNames: []string{"user2-1"},
			username:     "user2",
			useremail:    "user2@user2.com",
			userid:       "222",
			admin:        false,
		},
		{
			name:         "got no keys",
			wantKeyNames: []string{},
			username:     "does-not-exist",
			useremail:    "does@not.exist",
			userid:       "222111",
			admin:        false,
		},
		{
			name:         "admin got all keys",
			wantKeyNames: []string{"user1-1", "user1-2", "user2-1"},
			username:     testUserName,
			useremail:    testUserEmail,
			userid:       testUserID,
			admin:        true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			apiUser := getUser(test.useremail, test.userid, test.username, test.admin)
			kubermaticObj := []runtime.Object{}
			kubermaticObj = append(kubermaticObj, keyList...)
			kubermaticObj = append(kubermaticObj, apiUserToKubermaticUser(apiUser))

			req := httptest.NewRequest("GET", "/api/v1/ssh-keys", nil)
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(apiUser, []runtime.Object{}, kubermaticObj, nil, nil)
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
