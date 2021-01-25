/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package usersshkeysagent

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReconcileUserSSHKeys(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "sshkeys")
	if err != nil {
		t.Fatalf("error while creating test base dir: %v", err)
	}
	sshPath := filepath.Join(tmpDir, ".ssh")
	if err := os.Mkdir(sshPath, 0700); err != nil {
		t.Fatalf("error while creating .ssh dir: %v", err)
	}

	defer func() {
		if err := cleanupFiles([]string{tmpDir}); err != nil {
			t.Fatalf("failed to cleanup test files: %v", err)
		}
	}()

	authorizedKeysPath := filepath.Join(sshPath, "authorized_keys")
	_, err = os.Create(authorizedKeysPath)
	if err != nil {
		t.Fatalf("error while creating authorized_keys file: %v", err)
	}

	if err := os.Chmod(authorizedKeysPath, os.FileMode(0600)); err != nil {
		t.Fatalf("error while changing file mode: %v", err)
	}

	testCases := []struct {
		name             string
		reconciler       Reconciler
		modifier         func(string, string) error
		sshDirPath       string
		expectedSSHKey   string
		expectedFileMode int16
		expectedDirMode  int16
	}{
		{
			name: "Test updating authorized_keys file from reconcile",
			reconciler: Reconciler{
				log: kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				authorizedKeysPath: []string{
					authorizedKeysPath,
				},
			},
			expectedSSHKey: "ssh-rsa test_user_ssh_key\nssh-rsa test_user_ssh_key_2",
		},
		{
			name: "Test ssh dir file mode",
			reconciler: Reconciler{
				log: kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				authorizedKeysPath: []string{
					authorizedKeysPath,
				},
			},
			modifier:         changeFileModes,
			expectedSSHKey:   "ssh-rsa test_user_ssh_key\nssh-rsa test_user_ssh_key_2",
			sshDirPath:       sshPath,
			expectedFileMode: 384,
			expectedDirMode:  448,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			if tc.modifier != nil {
				if err := tc.modifier(sshPath, authorizedKeysPath); err != nil {
					t.Fatalf("error while executing test modifier: %v", err)
				}
			}

			if _, err := tc.reconciler.Reconcile(context.Background(), reconcile.Request{
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

				if tc.modifier != nil {
					authorizedKeysInfo, err := os.Stat(path)
					if err != nil {
						t.Fatalf("failed describing file %s: %v", path, err)
					}

					if int16(authorizedKeysInfo.Mode()) != tc.expectedFileMode {
						t.Fatal("authorized_keys file mode and its expected file mode don't match")
					}

					sshDirInfo, err := os.Stat(tc.sshDirPath)
					if err != nil {
						t.Fatalf("failed describing file %s: %v", path, err)
					}

					if int16(sshDirInfo.Mode()) != tc.expectedDirMode {
						t.Fatal(".ssh dir mode and its expected file mode don't match")
					}

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

func changeFileModes(sshDir, authorizedKeysFile string) error {
	if err := os.Chmod(authorizedKeysFile, 0700); err != nil {
		return fmt.Errorf("error while changing file mode: %v", err)
	}

	if err := os.Chmod(sshDir, 0600); err != nil {
		return fmt.Errorf("error while changing file mode: %v", err)
	}
	return nil
}
