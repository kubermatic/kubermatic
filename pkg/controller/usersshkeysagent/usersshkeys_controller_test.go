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
	"os"
	"path/filepath"
	"strings"
	"testing"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileUserSSHKeys(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sshkeys")
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
			expectedSSHKey: "# kkp-managed: key-test\nssh-rsa test_user_ssh_key\n# kkp-managed: key-test-2\nssh-rsa test_user_ssh_key_2",
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
			expectedSSHKey:   "# kkp-managed: key-test\nssh-rsa test_user_ssh_key\n# kkp-managed: key-test-2\nssh-rsa test_user_ssh_key_2",
			sshDirPath:       sshPath,
			expectedFileMode: 384,
			expectedDirMode:  448,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.reconciler.Client = fake.NewClientBuilder().WithObjects(
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
				},
			).Build()

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
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

func cleanupFiles(tmpFiles []string) error {
	for _, tmpFile := range tmpFiles {
		if err := os.RemoveAll(tmpFile); err != nil {
			return err
		}
	}
	return nil
}

func changeFileModes(sshDir, authorizedKeysFile string) error {
	if err := os.Chmod(authorizedKeysFile, 0700); err != nil {
		return fmt.Errorf("error while changing file mode: %w", err)
	}

	if err := os.Chmod(sshDir, 0600); err != nil {
		return fmt.Errorf("error while changing file mode: %w", err)
	}
	return nil
}

func TestMergeAuthorizedKeys(t *testing.T) {
	testCases := []struct {
		name            string
		secretData      map[string][]byte
		existingFile    string
		expectedContent string
		expectNoChange  bool
	}{
		{
			name: "KKP keys written to empty file",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:    "",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a",
		},
		{
			name: "external keys (no marker) are preserved",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:    "ssh-rsa EXTERNAL md-key",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a\nssh-rsa EXTERNAL md-key",
		},
		{
			name:            "removed KKP key is cleaned up",
			secretData:      map[string][]byte{},
			existingFile:    "# kkp-managed\nssh-rsa AAA old-kkp-key\nssh-rsa EXTERNAL md-key",
			expectedContent: "ssh-rsa EXTERNAL md-key",
		},
		{
			name: "KKP key replaced while external key preserved",
			secretData: map[string][]byte{
				"key-b": []byte("ssh-rsa BBB key-b"),
			},
			existingFile:    "# kkp-managed\nssh-rsa AAA old-kkp-key\nssh-rsa EXTERNAL md-key",
			expectedContent: "# kkp-managed: key-b\nssh-rsa BBB key-b\nssh-rsa EXTERNAL md-key",
		},
		{
			name: "no change when file already matches",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:   "# kkp-managed: key-a\nssh-rsa AAA key-a",
			expectNoChange: true,
		},
		{
			name:            "empty secret removes all KKP markers and keys",
			secretData:      map[string][]byte{},
			existingFile:    "# kkp-managed\nssh-rsa AAA kkp-key",
			expectedContent: "",
		},
		{
			name:            "only external keys, empty secret - no change",
			secretData:      map[string][]byte{},
			existingFile:    "ssh-rsa EXTERNAL-1\nssh-rsa EXTERNAL-2",
			expectedContent: "ssh-rsa EXTERNAL-1\nssh-rsa EXTERNAL-2",
			expectNoChange:  true,
		},
		{
			name: "KKP keys are sorted alphabetically",
			secretData: map[string][]byte{
				"key-z": []byte("ssh-rsa ZZZ key-z"),
				"key-a": []byte("ssh-rsa AAA key-a"),
				"key-m": []byte("ssh-rsa MMM key-m"),
			},
			existingFile:    "",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a\n# kkp-managed: key-m\nssh-rsa MMM key-m\n# kkp-managed: key-z\nssh-rsa ZZZ key-z",
		},
		{
			name:            "multiple KKP keys removed, external preserved",
			secretData:      map[string][]byte{},
			existingFile:    "# kkp-managed\nssh-rsa AAA kkp-1\n# kkp-managed\nssh-rsa BBB kkp-2\nssh-rsa EXTERNAL",
			expectedContent: "ssh-rsa EXTERNAL",
		},
		{
			name: "empty line between marker and key",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:    "# kkp-managed\n\nssh-rsa AAA key-a",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a",
		},
		{
			name: "consecutive markers without intervening keys",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:    "# kkp-managed\n# kkp-managed\nssh-rsa AAA key-a",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a",
		},
		{
			name: "external key matching KKP secret is deduplicated",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:    "ssh-rsa AAA key-a",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a",
		},
		{
			name: "external key not matching any KKP secret is preserved",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:    "ssh-rsa AAA key-a\nssh-rsa EXTERNAL other",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a\nssh-rsa EXTERNAL other",
		},
		{
			name:            "MD-only keys with empty KKP secret",
			secretData:      map[string][]byte{},
			existingFile:    "ssh-rsa MD-KEY cloud-init",
			expectedContent: "ssh-rsa MD-KEY cloud-init",
			expectNoChange:  true,
		},
		{
			name: "common key between KKP and MD",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:    "ssh-rsa AAA key-a\nssh-rsa MD-KEY cloud-init",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a\nssh-rsa MD-KEY cloud-init",
		},
		{
			name: "upgrade simulation",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
				"key-b": []byte("ssh-rsa BBB key-b"),
			},
			existingFile:    "ssh-rsa AAA key-a\nssh-rsa BBB key-b",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a\n# kkp-managed: key-b\nssh-rsa BBB key-b",
		},
		{
			name: "upgrade with removed key",
			secretData: map[string][]byte{
				"key-a": []byte("ssh-rsa AAA key-a"),
			},
			existingFile:    "ssh-rsa AAA key-a\nssh-rsa BBB key-b",
			expectedContent: "# kkp-managed: key-a\nssh-rsa AAA key-a\nssh-rsa BBB key-b",
		},
		{
			name: "new marker format recognized as KKP-managed",
			secretData: map[string][]byte{
				"key-b": []byte("ssh-rsa BBB key-b"),
			},
			existingFile:    "# kkp-managed: key-a\nssh-rsa AAA key-a\nssh-rsa EXTERNAL md-key",
			expectedContent: "# kkp-managed: key-b\nssh-rsa BBB key-b\nssh-rsa EXTERNAL md-key",
		},
		{
			name:            "mixed old and new markers",
			secretData:      map[string][]byte{},
			existingFile:    "# kkp-managed\nssh-rsa AAA old-key\n# kkp-managed: key-b\nssh-rsa BBB new-key\nssh-rsa EXTERNAL",
			expectedContent: "ssh-rsa EXTERNAL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "sshkeys-merge")
			if err != nil {
				t.Fatalf("error creating temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			sshPath := filepath.Join(tmpDir, ".ssh")
			if err := os.Mkdir(sshPath, 0700); err != nil {
				t.Fatalf("error creating .ssh dir: %v", err)
			}

			authorizedKeysPath := filepath.Join(sshPath, "authorized_keys")
			if tc.existingFile != "" {
				if err := os.WriteFile(authorizedKeysPath, []byte(tc.existingFile+"\n"), 0600); err != nil {
					t.Fatalf("error writing initial file: %v", err)
				}
			} else {
				if f, err := os.Create(authorizedKeysPath); err != nil {
					t.Fatalf("error creating file: %v", err)
				} else {
					f.Close()
				}
			}

			if tc.secretData == nil {
				tc.secretData = map[string][]byte{}
			}

			reconciler := Reconciler{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      resources.UserSSHKeys,
							Namespace: metav1.NamespaceSystem,
						},
						Data: tc.secretData,
					},
				).Build(),
				log:                kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				authorizedKeysPath: []string{authorizedKeysPath},
			}

			_, err = reconciler.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{Name: resources.UserSSHKeys, Namespace: metav1.NamespaceSystem},
			})
			if err != nil {
				t.Fatalf("reconcile failed: %v", err)
			}

			content, err := readAuthorizedKeysFile(authorizedKeysPath)
			if err != nil {
				t.Fatalf("error reading file: %v", err)
			}

			if tc.expectNoChange {
				if content == tc.existingFile {
					return
				}
				t.Fatalf("expected no change but file was modified:\ngot:      %q\noriginal: %q", content, tc.existingFile)
			}

			if content != tc.expectedContent {
				t.Fatalf("content mismatch:\ngot:      %q\nexpected: %q", content, tc.expectedContent)
			}
		})
	}
}
