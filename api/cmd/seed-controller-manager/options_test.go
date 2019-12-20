package main

import (
	"testing"

	"github.com/go-test/deep"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestParseAddons(t *testing.T) {
	testCases := []struct {
		name           string
		rawAddonString string
		expectedErr    string
		expectedAddons []kubermaticv1.Addon
	}{
		{
			name:           "No requires directive",
			rawAddonString: "my-addon;my-second-addon",
			expectedAddons: []kubermaticv1.Addon{
				{ObjectMeta: metav1.ObjectMeta{Name: "my-addon"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "my-second-addon"}},
			},
		},
		{
			name:           "Addon with requires directive",
			rawAddonString: `my-addon;my-second-addon=requires[{"kind":"Network","group":"config.openshift.io","version":"v1"}]`,
			expectedAddons: []kubermaticv1.Addon{
				{ObjectMeta: metav1.ObjectMeta{Name: "my-addon"}},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "my-second-addon"},
					Spec: kubermaticv1.AddonSpec{
						RequiredResourceTypes: []schema.GroupVersionKind{{
							Kind:    "Network",
							Group:   "config.openshift.io",
							Version: "v1",
						}},
					},
				},
			},
		},
		{
			name:           "Addon with defunct requires directive",
			rawAddonString: `my-addon;my-second-addon=requires[{"kind":false,"group":"config.openshift.io","version":"v1"}]`,
			expectedErr:    "failed to parse `=requires` directive of addon my-second-addon([{\"kind\":false,\"group\":\"config.openshift.io\",\"version\":\"v1\"}]): json: cannot unmarshal bool into Go struct field GroupVersionKind.Kind of type string",
		},
		{
			name:           "Addon with requires missing kind",
			rawAddonString: `my-addon;my-second-addon=requires[{"group":"config.openshift.io","version":"v1"}]`,
			expectedErr:    "requires directive for addon my-second-addon lacks required `kind` parameter",
		},
		{
			name:           "Addon with requires missing group",
			rawAddonString: `my-addon;my-second-addon=requires[{"kind":"config.openshift.io","version":"v1"}]`,
			expectedErr:    "requires directive for addon my-second-addon lacks required `group` parameter",
		},
		{
			name:           "Addon with requires missing version",
			rawAddonString: `my-addon;my-second-addon=requires[{"kind":"config.openshift.io","group":"v1"}]`,
			expectedErr:    "requires directive for addon my-second-addon lacks required `version` parameter",
		},
		{
			name:           "Coma-separated addons do not pass validation",
			rawAddonString: "my-addon,my-second-addon",
			expectedErr:    "addons must be semicolon-delimited",
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseAddons(tc.rawAddonString)

			var errString string
			if err != nil {
				errString = err.Error()
			}
			if tc.expectedErr != errString {
				t.Fatalf("Actual error %v does not match expected %q", err, tc.expectedErr)
			}

			if diff := deep.Equal(tc.expectedAddons, result); diff != nil {
				t.Errorf("resulting addons do not match expected, diff: %v", diff)
			}
		})
	}
}

func TestDefaultKubernetesAddonsParse(t *testing.T) {
	if _, err := parseAddons(defaultKubernetesAddons); err != nil {
		t.Errorf("defaultKubernetesAddons can not be parsed: %v", err)
	}
}

func TestDefaultOpenshiftAddonsParse(t *testing.T) {
	if _, err := parseAddons(defaultOpenshiftAddons); err != nil {
		t.Errorf("defaultOpenshiftAddons can not be parsed: %v", err)
	}
}
