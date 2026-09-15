//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package admingroupcontroller

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const annotationKey = kubermaticv1.AdminGrantedByGroupAnnotation

func TestMatchAdminGroup(t *testing.T) {
	testCases := []struct {
		name        string
		adminGroups []string
		userGroups  []string
		current     string
		expected    string
	}{
		{
			name:        "no admin groups configured",
			adminGroups: nil,
			userGroups:  []string{"dev"},
			expected:    "",
		},
		{
			name:        "user has no groups",
			adminGroups: []string{"dev"},
			userGroups:  nil,
			expected:    "",
		},
		{
			name:        "user in a configured group",
			adminGroups: []string{"dev", "admins"},
			userGroups:  []string{"admins"},
			expected:    "admins",
		},
		{
			name:        "matching is case-sensitive",
			adminGroups: []string{"Admins"},
			userGroups:  []string{"admins"},
			expected:    "",
		},
		{
			name:        "annotated group is preferred when still valid",
			adminGroups: []string{"dev", "admins"},
			userGroups:  []string{"dev", "admins"},
			current:     "admins",
			expected:    "admins",
		},
		{
			name:        "falls back to first configured group in list order when current is stale",
			adminGroups: []string{"dev", "admins"},
			userGroups:  []string{"dev", "admins"},
			current:     "gone",
			expected:    "dev",
		},
		{
			name:        "stale current, user no longer in it, another group matches",
			adminGroups: []string{"dev", "admins"},
			userGroups:  []string{"admins"},
			current:     "dev",
			expected:    "admins",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchAdminGroup(tc.adminGroups, tc.userGroups, tc.current); got != tc.expected {
				t.Errorf("matchAdminGroup(%v, %v, %q) = %q, want %q", tc.adminGroups, tc.userGroups, tc.current, got, tc.expected)
			}
		})
	}
}

func TestReconcile(t *testing.T) {
	const userName = "user-test"

	testCases := []struct {
		name string
		// omitSettings builds the fake client without a globalsettings object.
		omitSettings   bool
		adminGroups    []string
		user           *kubermaticv1.User
		wantAdmin      bool
		wantAnnotation string // "" means the annotation must be absent
	}{
		{
			name:           "promote a matched non-admin",
			adminGroups:    []string{"admins"},
			user:           testUser(userName, "bob@acme.com", []string{"admins"}, false, ""),
			wantAdmin:      true,
			wantAnnotation: "admins",
		},
		{
			name:           "demote when group removed from list",
			adminGroups:    []string{"other"},
			user:           testUser(userName, "bob@acme.com", []string{"admins"}, true, "admins"),
			wantAdmin:      false,
			wantAnnotation: "",
		},
		{
			name:           "demote when user left the group",
			adminGroups:    []string{"admins"},
			user:           testUser(userName, "bob@acme.com", []string{"dev"}, true, "admins"),
			wantAdmin:      false,
			wantAnnotation: "",
		},
		{
			name:           "re-stamp when a second configured group still matches",
			adminGroups:    []string{"admins", "platform"},
			user:           testUser(userName, "bob@acme.com", []string{"platform"}, true, "admins"),
			wantAdmin:      true,
			wantAnnotation: "platform",
		},
		{
			name:           "manual admin in a listed group is left untouched",
			adminGroups:    []string{"admins"},
			user:           testUser(userName, "bob@acme.com", []string{"admins"}, true, ""),
			wantAdmin:      true,
			wantAnnotation: "",
		},
		{
			name:           "manual admin outside any listed group is left untouched",
			adminGroups:    []string{"admins"},
			user:           testUser(userName, "bob@acme.com", []string{"dev"}, true, ""),
			wantAdmin:      true,
			wantAnnotation: "",
		},
		{
			name:           "service account is never escalated",
			adminGroups:    []string{"admins"},
			user:           testUser(userName, "serviceaccount-abc@sa.local", []string{"admins"}, false, ""),
			wantAdmin:      false,
			wantAnnotation: "",
		},
		{
			name:           "no globalsettings object demotes an annotated admin",
			omitSettings:   true,
			user:           testUser(userName, "bob@acme.com", []string{"admins"}, true, "admins"),
			wantAdmin:      false,
			wantAnnotation: "",
		},
		{
			name:           "no globalsettings object leaves a manual admin untouched",
			omitSettings:   true,
			user:           testUser(userName, "bob@acme.com", []string{"admins"}, true, ""),
			wantAdmin:      true,
			wantAnnotation: "",
		},
		{
			name:           "global viewer is not promoted",
			adminGroups:    []string{"admins"},
			user:           globalViewer(testUser(userName, "bob@acme.com", []string{"admins"}, false, "")),
			wantAdmin:      false,
			wantAnnotation: "",
		},
		{
			// A user promoted via a group and turned into a global viewer afterwards
			// keeps the annotation until we clear it, even though the group still matches.
			name:           "global viewer loses a leftover annotation",
			adminGroups:    []string{"admins"},
			user:           globalViewer(testUser(userName, "bob@acme.com", []string{"admins"}, false, "admins")),
			wantAdmin:      false,
			wantAnnotation: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			objects := []ctrlruntimeclient.Object{tc.user}
			if !tc.omitSettings {
				objects = append(objects, genSettings(tc.adminGroups))
			}
			client := fake.NewClientBuilder().WithObjects(objects...).Build()

			r := &reconciler{
				log:             kubermaticlog.Logger,
				recorder:        &events.FakeRecorder{},
				masterClient:    client,
				masterAPIReader: client,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: userName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}
			assertUser(t, ctx, client, userName, tc.wantAdmin, tc.wantAnnotation)

			// A second reconcile must be a no-op: the state already matches.
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("second reconcile failed: %v", err)
			}
			assertUser(t, ctx, client, userName, tc.wantAdmin, tc.wantAnnotation)
		})
	}
}

func assertUser(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client, name string, wantAdmin bool, wantAnnotation string) {
	t.Helper()

	user := &kubermaticv1.User{}
	if err := client.Get(ctx, types.NamespacedName{Name: name}, user); err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	if user.Spec.IsAdmin != wantAdmin {
		t.Errorf("IsAdmin = %v, want %v", user.Spec.IsAdmin, wantAdmin)
	}

	got, ok := user.Annotations[annotationKey]
	if wantAnnotation == "" {
		if ok {
			t.Errorf("annotation %q = %q, want absent", annotationKey, got)
		}
	} else if got != wantAnnotation {
		t.Errorf("annotation %q = %q, want %q", annotationKey, got, wantAnnotation)
	}
}

func testUser(name, email string, groups []string, admin bool, annotation string) *kubermaticv1.User {
	user := &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.UserSpec{
			Name:    name,
			Email:   email,
			Groups:  groups,
			IsAdmin: admin,
		},
	}
	if annotation != "" {
		user.Annotations = map[string]string{annotationKey: annotation}
	}
	return user
}

func globalViewer(user *kubermaticv1.User) *kubermaticv1.User {
	user.Spec.IsGlobalViewer = true
	return user
}

func genSettings(adminGroups []string) *kubermaticv1.KubermaticSetting {
	return &kubermaticv1.KubermaticSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubermaticv1.GlobalSettingsName,
		},
		Spec: kubermaticv1.SettingSpec{
			AdminGroups: adminGroups,
		},
	}
}
