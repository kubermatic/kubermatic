package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/ssh"
)

func TestSSHKeysEndpoint(t *testing.T) {
	t.Parallel()
	keyList := []runtime.Object{
		&kubermaticv1.UserSSHKey{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "user1-1",
				Labels: map[string]string{ssh.DefaultUserLabel: ssh.UserToLabel("user1")},
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
				Name:   "user1-2",
				Labels: map[string]string{ssh.DefaultUserLabel: ssh.UserToLabel("user1")},
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
				Name:   "user2-1",
				Labels: map[string]string{ssh.DefaultUserLabel: ssh.UserToLabel("user2")},
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
			e := createTestEndpoint(getUser(test.username, test.admin), keyList, nil, nil)
			e.ServeHTTP(res, req)
			checkStatusCode(http.StatusOK, res, t)

			gotKeys := []kubermaticv1.UserSSHKey{}
			err := json.Unmarshal(res.Body.Bytes(), &gotKeys)
			if err != nil {
				t.Fatal(err, res.Body.String())
			}

			gotKeyNames := []string{}
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
