// +build e2e

package api

import (
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestGetDefaultGlobalSettings(t *testing.T) {
	tests := []struct {
		name             string
		expectedSettings *apiv1.GlobalSettings
	}{
		{
			name: "test, gets default settings",
			expectedSettings: &apiv1.GlobalSettings{

				CustomLinks: []kubermaticv1.CustomLink{},
				CleanupOptions: kubermaticv1.CleanupOptions{
					Enabled:  false,
					Enforced: false,
				},
				DefaultNodeCount:      10,
				ClusterTypeOptions:    kubermaticv1.ClusterTypeAll,
				DisplayDemoInfo:       false,
				DisplayAPIDocs:        false,
				DisplayTermsOfService: false,
				EnableDashboard:       true,
				EnableOIDCKubeconfig:  false,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var masterToken string

			masterToken, err := retrieveMasterToken()
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}
			apiRunner := createRunner(masterToken, t)

			settings, err := apiRunner.GetGlobalSettings()
			if err != nil {
				t.Fatalf("can not get global settings: %v", err)
			}
			if !equality.Semantic.DeepEqual(tc.expectedSettings, settings) {
				t.Fatalf("expected: %v, got %v", tc.expectedSettings, settings)
			}

		})
	}
}
