package version_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	"k8c.io/kubermatic/v2/pkg/version"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetClusterUpgrades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		existingKubermaticObjs []ctrlruntimeclient.Object
		apiUser                apiv1.User
		provider               kubermaticv1.ProviderType
		versions               []*version.Version
		updates                []*version.Update
		incompatibilities      []*version.ProviderIncompatibility
		wantVersions           []*apiv1.MasterVersion
	}{
		{
			name: "upgrade available and incompatibility with another provider",
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			apiUser:  *test.GenDefaultAPIUser(),
			provider: kubermaticv1.ProviderAWS,
			wantVersions: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.21.0"),
				},
				{
					Version: semver.MustParse("1.21.1"),
				},
				{
					Version: semver.MustParse("1.22.0"),
				},
				{
					Version: semver.MustParse("1.22.1"),
				},
			},
			versions: []*version.Version{
				{
					Version: semver.MustParse("1.21.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.21.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.1"),
					Type:    apiv1.KubernetesClusterType,
				},
			},
			updates: []*version.Update{
				{
					From:      "1.21.*",
					To:        "1.21.*",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "1.21.*",
					To:        "1.22.*",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "1.22.*",
					To:        "1.22.*",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
			},
			incompatibilities: []*version.ProviderIncompatibility{
				{
					Provider:  kubermaticv1.ProviderVSphere,
					Version:   "1.22.*",
					Condition: version.AlwaysCondition,
					Operation: version.CreateOperation,
					Type:      apiv1.KubernetesClusterType,
				},
			},
		},
		{
			name: "upgrade available but incompatible with the given provider",
			existingKubermaticObjs: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
			),
			apiUser:  *test.GenDefaultAPIUser(),
			provider: kubermaticv1.ProviderVSphere,
			wantVersions: []*apiv1.MasterVersion{
				{
					Version: semver.MustParse("1.21.0"),
				},
				{
					Version: semver.MustParse("1.21.1"),
				},
			},
			versions: []*version.Version{
				{
					Version: semver.MustParse("1.21.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.21.1"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.0"),
					Type:    apiv1.KubernetesClusterType,
				},
				{
					Version: semver.MustParse("1.22.1"),
					Type:    apiv1.KubernetesClusterType,
				},
			},
			updates: []*version.Update{
				{
					From:      "1.21.*",
					To:        "1.21.*",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "1.21.*",
					To:        "1.22.*",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
				{
					From:      "1.22.*",
					To:        "1.22.*",
					Automatic: false,
					Type:      apiv1.KubernetesClusterType,
				},
			},
			incompatibilities: []*version.ProviderIncompatibility{
				{
					Provider:  kubermaticv1.ProviderVSphere,
					Version:   "1.22.*",
					Condition: version.AlwaysCondition,
					Operation: version.CreateOperation,
					Type:      apiv1.KubernetesClusterType,
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", fmt.Sprintf("/api/v2/providers/%s/versions", tc.provider), nil)
			res := httptest.NewRecorder()
			var machineObj []ctrlruntimeclient.Object

			ep, _, err := test.CreateTestEndpointAndGetClients(tc.apiUser, nil, []ctrlruntimeclient.Object{}, machineObj, tc.existingKubermaticObjs, tc.versions, tc.updates, tc.incompatibilities, hack.NewTestRouting)
			if err != nil {
				t.Fatalf("failed to create tc endpoint due to %v", err)
			}
			ep.ServeHTTP(res, req)
			if res.Code != http.StatusOK {
				t.Fatalf("Expected status code to be 200, got %d\nResponse body: %q", res.Code, res.Body.String())
			}

			var gotVersions []*apiv1.MasterVersion
			err = json.Unmarshal(res.Body.Bytes(), &gotVersions)
			if err != nil {
				t.Fatal(err)
			}

			test.CompareVersions(t, gotVersions, tc.wantVersions)
		})
	}
}
