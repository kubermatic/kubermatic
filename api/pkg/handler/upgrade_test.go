package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/go-test/deep"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetClusterUpgradesV3(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cluster     *kubermaticv1.Cluster
		versions    []*version.MasterVersion
		updates     []*version.MasterUpdate
		wantUpdates []*apiv1.MasterVersion
	}{
		{
			name: "upgrade available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Spec: kubermaticv1.ClusterSpec{Version: "1.6.0"},
			},
			wantUpdates: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
			},
			versions: []*version.MasterVersion{
				{
					Version: semver.MustParse("1.6.0"),
				},
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
			},
			updates: []*version.MasterUpdate{
				{
					From:      "1.6.0",
					To:        "1.6.1",
					Automatic: false,
				},
				{
					From:      "1.6.x",
					To:        "1.7.0",
					Automatic: false,
				},
			},
		},
		{
			name: "no available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Spec: kubermaticv1.ClusterSpec{Version: "1.6.0"},
			},
			wantUpdates: []*apiv1.MasterVersion{},
			versions: []*version.MasterVersion{
				{
					Version: semver.MustParse("1.6.0"),
				},
			},
			updates: []*version.MasterUpdate{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v3/dc/us-central1/cluster/foo/upgrades", nil)
			res := httptest.NewRecorder()
			ep, err := createTestEndpoint(getUser(testUsername, false), []runtime.Object{}, []runtime.Object{test.cluster}, test.versions, test.updates)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Errorf("Expected status code to be 200, got %d", res.Code)
				t.Error(res.Body.String())
				return
			}

			var gotUpdates []*apiv1.MasterVersion
			err = json.Unmarshal(res.Body.Bytes(), &gotUpdates)
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(gotUpdates, test.wantUpdates); diff != nil {
				t.Errorf("got different upgrade response than expected. Diff: %v", diff)
			}
		})
	}
}

func TestGetClusterUpgradesV1(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cluster     *kubermaticv1.Cluster
		project     *kubermaticv1.Project
		user        *kubermaticv1.User
		apiUser     apiv1.User
		versions    []*version.MasterVersion
		updates     []*version.MasterUpdate
		wantUpdates []*apiv1.MasterVersion
	}{
		{
			name: "upgrade available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Spec: kubermaticv1.ClusterSpec{Version: "1.6.0"},
			},
			project: &kubermaticv1.Project{
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
			user: &kubermaticv1.User{
				Spec: kubermaticv1.UserSpec{
					Name:  "George",
					Email: testEmail,
					Projects: []kubermaticv1.ProjectGroup{
						{
							Group: "owners-myProjectInternalName",
							Name:  "myProjectInternalName",
						},
					},
				},
			},
			apiUser: apiv1.User{
				ID:    testUsername,
				Email: testEmail,
				Roles: map[string]struct{}{
					"user": struct{}{},
				},
			},
			wantUpdates: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
			},
			versions: []*version.MasterVersion{
				{
					Version: semver.MustParse("1.6.0"),
				},
				{
					Version: semver.MustParse("1.6.1"),
				},
				{
					Version: semver.MustParse("1.7.0"),
				},
			},
			updates: []*version.MasterUpdate{
				{
					From:      "1.6.0",
					To:        "1.6.1",
					Automatic: false,
				},
				{
					From:      "1.6.x",
					To:        "1.7.0",
					Automatic: false,
				},
			},
		},
		{
			name: "no available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUsername},
				},
				Spec: kubermaticv1.ClusterSpec{Version: "1.6.0"},
			},
			project: &kubermaticv1.Project{
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
			user: &kubermaticv1.User{
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
			apiUser: apiv1.User{
				ID:    testUsername,
				Email: testEmail,
				Roles: map[string]struct{}{
					"user": struct{}{},
				},
			},
			wantUpdates: []*apiv1.MasterVersion{},
			versions: []*version.MasterVersion{
				{
					Version: semver.MustParse("1.6.0"),
				},
			},
			updates: []*version.MasterUpdate{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/projects/myProjectInternalName/dc/us-central1/clusters/foo/upgrade", nil)
			res := httptest.NewRecorder()

			kubermaticObj := []runtime.Object{test.cluster}
			if test.project != nil {
				kubermaticObj = append(kubermaticObj, test.project)
			}
			if test.user != nil {
				kubermaticObj = append(kubermaticObj, test.user)
			}

			ep, err := createTestEndpoint(test.apiUser, []runtime.Object{}, kubermaticObj, test.versions, test.updates)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Errorf("Expected status code to be 200, got %d", res.Code)
				t.Error(res.Body.String())
				return
			}

			var gotUpdates []*apiv1.MasterVersion
			err = json.Unmarshal(res.Body.Bytes(), &gotUpdates)
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(gotUpdates, test.wantUpdates); diff != nil {
				t.Errorf("got different upgrade response than expected. Diff: %v", diff)
			}
		})
	}
}
