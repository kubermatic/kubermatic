package usersshkeysagent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	tmpFiles []string
	log      = kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
)

func TestMain(m *testing.M) {
	m.Run()
	if err := cleanupFiles(); err != nil {
		log.Errorw("Failed cleaning up files", zap.Error(err))
		os.Exit(1)
	}
}

func TestReconcileUserSSHKeys(t *testing.T) {
	testCases := []struct {
		name           string
		reconciler     Reconciler
		expectedSSHKey string
		modifier       func(path string) error
	}{
		{
			name: "Test updating authorized_keys file from reconcile",
			reconciler: Reconciler{
				log: log,
				authorizedKeysPath: []string{
					fmt.Sprintf("%v%v", os.TempDir(), time.Now().Nanosecond()),
				},
			},
			expectedSSHKey: "ssh-rsa test_user_ssh_key",
		},
		{
			name: "Test updating authorized_keys file manually",
			reconciler: Reconciler{
				log: log,
				authorizedKeysPath: []string{
					fmt.Sprintf("%v%v", os.TempDir(), time.Now().Nanosecond()),
				},
			},
			expectedSSHKey: "ssh-rsa test_user_ssh_key",
			modifier:       updateAuthorizedKeysFile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFiles = append(tmpFiles, tc.reconciler.authorizedKeysPath...)
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

			if err := tc.reconciler.watchAuthorizedKeys(context.Background(), tc.reconciler.authorizedKeysPath); err != nil {
				t.Fatal(err)
			}

			if tc.modifier != nil {
				if err := tc.modifier(tc.reconciler.authorizedKeysPath[0]); err != nil {
					t.Fatal(err)
				}
				time.Sleep(2 * time.Second)
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

func updateAuthorizedKeysFile(path string) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	defer file.Close()
	if _, err := file.WriteString("test_update_ssh_keys"); err != nil {
		return err
	}
	return nil
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

func cleanupFiles() error {
	for _, tmpFile := range tmpFiles {
		if _, err := os.Stat(tmpFile); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := os.Remove(tmpFile); err == nil {
			return err
		}
	}

	return nil
}
