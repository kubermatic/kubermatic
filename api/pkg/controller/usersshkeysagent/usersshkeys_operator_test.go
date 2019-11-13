package usersshkeysagent

import (
	"bufio"
	"fmt"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"testing"
	"time"
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
				log: zap.NewNop().Sugar(),
				authorizedKeysPath: []string{
					fmt.Sprintf("%v%v", os.TempDir(), time.Now().Nanosecond()),
				},
			},
			expectedSSHKey: "ssh-rsa test_user_ssh_key",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrlruntimelog.SetLogger(ctrlruntimelog.ZapLogger(true))
			tc.reconciler.Client = fake.NewFakeClient(
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "123456",
						Name:            resources.UserSSHKeys,
						Namespace:       metav1.NamespaceSystem,
					},
					Data: map[string][]byte{
						"key-test": []byte("ssh-rsa test_user_ssh_key"),
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
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var text string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text += scanner.Text()
	}
	return text, nil
}
