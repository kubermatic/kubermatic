package handler

import (
	"encoding/json"
	"fmt"
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

func TestGetClusterUpgradesV1(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		cluster                *kubermaticv1.Cluster
		existingKubermaticObjs []runtime.Object
		apiUser                apiv1.User
		versions               []*version.MasterVersion
		updates                []*version.MasterUpdate
		wantUpdates            []*apiv1.MasterVersion
	}{
		{
			name: "upgrade available",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"user": testUserName},
				},
				Spec: kubermaticv1.ClusterSpec{Version: "1.6.0"},
			},
			existingKubermaticObjs: genDefaultKubermaticObjects(),
			apiUser:                *genDefaultAPIUser(),
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
					Labels: map[string]string{"user": testUserName},
				},
				Spec: kubermaticv1.ClusterSpec{Version: "1.6.0"},
			},
			existingKubermaticObjs: genDefaultKubermaticObjects(),
			apiUser:                *genDefaultAPIUser(),
			wantUpdates:            []*apiv1.MasterVersion{},
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
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/projects/%s/dc/us-central1/clusters/foo/upgrades", testingProjectName), nil)
			res := httptest.NewRecorder()
			kubermaticObj := []runtime.Object{test.cluster}
			kubermaticObj = append(kubermaticObj, test.existingKubermaticObjs...)
			ep, err := createTestEndpoint(test.apiUser, []runtime.Object{}, kubermaticObj, test.versions, test.updates)
			if err != nil {
				t.Fatalf("failed to create test endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("Expected status code to be 200, got %d\nResponse body: %q", res.Code, res.Body.String())
			}

			var gotUpdates []*apiv1.MasterVersion
			err = json.Unmarshal(res.Body.Bytes(), &gotUpdates)
			if err != nil {
				t.Fatal(err)
			}

			if diff := deep.Equal(gotUpdates, test.wantUpdates); diff != nil {
				t.Fatalf("got different upgrade response than expected. Diff: %v", diff)
			}
		})
	}
}
