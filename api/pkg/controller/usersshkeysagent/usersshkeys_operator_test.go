package usersshkeysagent

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReconcileUserSSHKeys(t *testing.T) {
	testCases := []struct {
		name           string
		reconciler     Reconciler
		expectedSSHKey string
	}{
		{
			name: "Test updating authorized_keys file from reconcile",
			reconciler: Reconciler{
				log: kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				authorizedKeysPath: []string{
					fmt.Sprintf("%v%v", os.TempDir(), time.Now().Nanosecond()),
				},
			},
			expectedSSHKey: "ssh-rsa test_user_ssh_key\nssh-rsa test_user_ssh_key_2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if err := cleanupFiles(tc.reconciler.authorizedKeysPath); err != nil {
					t.Fatal(err)
				}
			}()
			tc.reconciler.Client = fake.NewFakeClient(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "123456",
						Name:            resources.UserSSHKeys,
						Namespace:       metav1.NamespaceSystem,
					},
					Data: map[string][]byte{
						"key-test":   []byte("ssh-rsa test_user_ssh_key"),
						"key-test-2": []byte("ssh-rsa test_user_ssh_key_2"),
					},
				})
			if _, err := tc.reconciler.Reconcile(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resources.UserSSHKeys, Namespace: metav1.NamespaceSystem}}); err != nil {
				t.Fatalf("failed to run reconcile: %v", err)
			}

			for _, path := range tc.reconciler.authorizedKeysPath {
				key, err := readAuthorizedKeysFile(path)
				if err != nil {
					t.Fatal(err)
				}

				if key != tc.expectedSSHKey {
					t.Fatal("usersshkey secret and authorized_keys file don't match")
				}
			}
		})
	}
}

func readAuthorizedKeysFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

func cleanupFiles(tmpFiles []string) error {
	for _, tmpFile := range tmpFiles {
		if err := os.RemoveAll(tmpFile); err == nil {
			return err
		}
	}
	return nil
}
