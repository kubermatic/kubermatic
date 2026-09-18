/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/util/yamled"

	"k8s.io/apimachinery/pkg/util/sets"
)

// testLogger returns a logger that does not pollute the test output.
func testLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	return logger
}

// testCertificate creates a self-signed certificate and returns it PEM-encoded.
func testCertificate(t *testing.T, commonName string) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func writeFile(t *testing.T, dir string, name string, content string) string {
	t.Helper()

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create %s: %v", dir, err)
	}

	filename := filepath.Join(dir, name)
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", filename, err)
	}

	return filename
}

// chartsDirectory creates a minimal charts directory containing the shipped CA bundle and
// whatever drop-in files the test needs.
func chartsDirectory(t *testing.T, shipped string, dropIns map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	static := filepath.Join(dir, caBundleChartPath, "static")

	writeFile(t, static, "ca-bundle.pem", shipped)
	for name, content := range dropIns {
		writeFile(t, filepath.Join(dir, caBundleChartPath, caBundleExtraDir), name, content)
	}

	return dir
}

func TestReadCertificateFiles(t *testing.T) {
	dir := t.TempDir()
	first := testCertificate(t, "first")
	second := testCertificate(t, "second")

	firstFile := writeFile(t, dir, "first.pem", first)
	secondFile := writeFile(t, dir, "second.pem", second)
	garbageFile := writeFile(t, dir, "garbage.pem", "this is not a certificate")

	testcases := []struct {
		name      string
		filenames []string
		expected  []string
		expectErr bool
	}{
		{
			name:      "single file",
			filenames: []string{firstFile},
			expected:  []string{first},
		},
		{
			name:      "multiple files are concatenated in order",
			filenames: []string{secondFile, firstFile},
			expected:  []string{second, first},
		},
		{
			name:      "no files yields no certificates",
			filenames: nil,
		},
		{
			name:      "garbage is rejected",
			filenames: []string{garbageFile},
			expectErr: true,
		},
		{
			name:      "missing file is rejected",
			filenames: []string{filepath.Join(dir, "does-not-exist.pem")},
			expectErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := readCertificateFiles(tc.filenames)
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected an error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			if count := strings.Count(result, "BEGIN CERTIFICATE"); count != len(tc.expected) {
				t.Fatalf("expected %d certificates, but got %d", len(tc.expected), count)
			}

			offset := 0
			for _, cert := range tc.expected {
				index := strings.Index(result[offset:], strings.TrimSpace(cert))
				if index < 0 {
					t.Fatalf("expected certificate is missing or out of order in:\n%s", result)
				}
				offset += index
			}
		})
	}
}

func TestEncodeCABundleValues(t *testing.T) {
	replacement := testCertificate(t, "corp")
	additional := testCertificate(t, "corp-additional")

	testcases := []struct {
		name         string
		certificates map[string]string
		expectedKeys []string
	}{
		{
			name:         "replacement only",
			certificates: map[string]string{"certificates": replacement},
			expectedKeys: []string{"certificates"},
		},
		{
			name:         "additional only",
			certificates: map[string]string{"additionalCertificates": additional},
			expectedKeys: []string{"additionalCertificates"},
		},
		{
			name: "both, in the order the chart applies them",
			certificates: map[string]string{
				"additionalCertificates": additional,
				"certificates":           replacement,
			},
			expectedKeys: []string{"certificates", "additionalCertificates"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := encodeCABundleValues(tc.certificates)
			if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			// the PEM must be written as a literal block, not as an escaped one-liner
			if strings.Contains(encoded, "\\n") {
				t.Fatalf("expected no escaped newlines, but got:\n%s", encoded)
			}

			offset := 0
			for _, key := range tc.expectedKeys {
				index := strings.Index(encoded[offset:], key+": |")
				if index < 0 {
					t.Fatalf("expected a literal block for %s in the expected order, but got:\n%s", key, encoded)
				}
				offset += index + 1
			}

			// the result must be loadable as Helm values again
			filename := writeFile(t, t.TempDir(), "values.yaml", encoded)

			values, err := loadHelmValues([]string{filename})
			if err != nil {
				t.Fatalf("failed to load the generated values: %v", err)
			}

			for key, certificate := range tc.certificates {
				value, ok := values.GetString(yamled.Path{"caBundle", key})
				if !ok {
					t.Fatalf("generated values do not contain caBundle.%s:\n%s", key, encoded)
				}

				if strings.TrimSpace(value) != strings.TrimSpace(certificate) {
					t.Fatalf("expected the certificate to survive the roundtrip, but got:\n%s", value)
				}
			}
		})
	}
}

func TestExtraCABundleFiles(t *testing.T) {
	shipped := testCertificate(t, "shipped")
	dir := chartsDirectory(t, shipped, map[string]string{
		"second.pem": testCertificate(t, "second"),
		"first.pem":  testCertificate(t, "first"),
		"README.md":  "not a certificate",
	})

	files, err := extraCABundleFiles(dir)
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}

	expected := []string{"first.pem", "second.pem"}
	if len(files) != len(expected) {
		t.Fatalf("expected %v, but got %v", expected, files)
	}

	for i, name := range expected {
		if filepath.Base(files[i]) != name {
			t.Fatalf("expected %v in sorted order, but got %v", expected, files)
		}
	}
}

func TestLocalCABundle(t *testing.T) {
	shipped := testCertificate(t, "shipped")

	t.Run("without drop-ins", func(t *testing.T) {
		bundle, err := localCABundle(chartsDirectory(t, shipped, nil))
		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}

		if count := strings.Count(bundle.String(), "BEGIN CERTIFICATE"); count != 1 {
			t.Fatalf("expected 1 certificate, but got %d", count)
		}
	})

	t.Run("with drop-ins", func(t *testing.T) {
		dir := chartsDirectory(t, shipped, map[string]string{
			"corp.pem": testCertificate(t, "corp"),
		})

		bundle, err := localCABundle(dir)
		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}

		if count := strings.Count(bundle.String(), "BEGIN CERTIFICATE"); count != 2 {
			t.Fatalf("expected 2 certificates, but got %d", count)
		}
	})

	t.Run("broken drop-in is rejected", func(t *testing.T) {
		dir := chartsDirectory(t, shipped, map[string]string{
			"broken.pem": "not a certificate",
		})

		if _, err := localCABundle(dir); err == nil {
			t.Fatal("expected an error, but got none")
		}
	})
}

func TestFindDuplicateDropIn(t *testing.T) {
	shipped := testCertificate(t, "shipped")
	corp := testCertificate(t, "corp")
	other := testCertificate(t, "other")

	dir := chartsDirectory(t, shipped, map[string]string{
		"corp.pem": "# our corporate CA\n\n" + corp,
	})

	t.Run("duplicate is found despite different comments", func(t *testing.T) {
		duplicate, err := findDuplicateDropIn(dir, corp)
		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}

		if filepath.Base(duplicate) != "corp.pem" {
			t.Fatalf("expected corp.pem, but got %q", duplicate)
		}
	})

	t.Run("unrelated certificate is not a duplicate", func(t *testing.T) {
		duplicate, err := findDuplicateDropIn(dir, other)
		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}

		if duplicate != "" {
			t.Fatalf("expected no duplicate, but got %q", duplicate)
		}
	})

	t.Run("chart without drop-ins", func(t *testing.T) {
		duplicate, err := findDuplicateDropIn(chartsDirectory(t, shipped, nil), corp)
		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}

		if duplicate != "" {
			t.Fatalf("expected no duplicate, but got %q", duplicate)
		}
	})
}

func TestApplyCABundleFlags(t *testing.T) {
	dir := t.TempDir()
	corp := testCertificate(t, "corp")
	other := testCertificate(t, "other")
	corpFile := writeFile(t, dir, "corp.pem", corp)
	otherFile := writeFile(t, dir, "other.pem", other)

	testcases := []struct {
		name         string
		caBundle     string
		additional   []string
		expectedKeys sets.Set[string]
		expectErr    bool
	}{
		{
			name:         "no flags leaves the values untouched",
			expectedKeys: sets.New[string](),
		},
		{
			name:         "replacement only",
			caBundle:     corpFile,
			expectedKeys: sets.New("certificates"),
		},
		{
			name:         "additional only",
			additional:   []string{corpFile, otherFile},
			expectedKeys: sets.New("additionalCertificates"),
		},
		{
			name:         "both",
			caBundle:     corpFile,
			additional:   []string{otherFile},
			expectedKeys: sets.New("certificates", "additionalCertificates"),
		},
		{
			name:      "invalid file is rejected",
			caBundle:  writeFile(t, dir, "broken.pem", "not a certificate"),
			expectErr: true,
		},
	}

	shipped := testCertificate(t, "shipped")

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// applyCABundleFlags writes its artifact into the working directory
			t.Chdir(t.TempDir())

			values, err := loadHelmValues(nil)
			if err != nil {
				t.Fatalf("failed to create Helm values: %v", err)
			}

			err = applyCABundleFlags(testLogger(), values, chartsDirectory(t, shipped, nil), tc.caBundle, tc.additional)
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected an error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}

			for _, key := range []string{"certificates", "additionalCertificates"} {
				value, exists := values.GetString(yamled.Path{"caBundle", key})

				if want := tc.expectedKeys.Has(key); exists != want {
					t.Fatalf("expected caBundle.%s to exist=%v, but got %v", key, want, exists)
				}

				if exists && !strings.Contains(value, "BEGIN CERTIFICATE") {
					t.Fatalf("expected caBundle.%s to contain a certificate, but got %q", key, value)
				}
			}

			if tc.expectedKeys.Has("additionalCertificates") && len(tc.additional) > 1 {
				value, _ := values.GetString(yamled.Path{"caBundle", "additionalCertificates"})
				if count := strings.Count(value, "BEGIN CERTIFICATE"); count != len(tc.additional) {
					t.Fatalf("expected %d certificates, but got %d", len(tc.additional), count)
				}
			}

			// the artifact is only written when certificates were actually configured
			recorded, err := os.ReadFile(CABundleValuesFile)
			if want := tc.expectedKeys.Len() > 0; want != (err == nil) {
				t.Fatalf("expected %s to exist=%v, but got %v", CABundleValuesFile, want, err)
			}

			for _, key := range tc.expectedKeys.UnsortedList() {
				if !strings.Contains(string(recorded), key+": |") {
					t.Fatalf("expected %s to record caBundle.%s, but got:\n%s", CABundleValuesFile, key, recorded)
				}
			}
		})
	}
}

// TestApplyCABundleFlagsAppliesCertificatesWhenRecordingFails guards the property that makes
// the recorded file safe to lose: the certificates reach the Helm values that drive the
// installation, and writing ca-bundle.values.yaml is only a convenience afterwards. An
// administrator installing from a read-only working directory must still get their CA bundle.
func TestApplyCABundleFlagsAppliesCertificatesWhenRecordingFails(t *testing.T) {
	t.Chdir(t.TempDir())

	// make the recording fail for any user, including root
	if err := os.Mkdir(CABundleValuesFile, 0755); err != nil {
		t.Fatalf("failed to create %s: %v", CABundleValuesFile, err)
	}

	dir := t.TempDir()
	certificate := testCertificate(t, "corp")
	pemFile := writeFile(t, dir, "corp.pem", certificate)

	helmValues, err := loadHelmValues(nil)
	if err != nil {
		t.Fatalf("failed to create Helm values: %v", err)
	}

	if err := applyCABundleFlags(testLogger(), helmValues, chartsDirectory(t, testCertificate(t, "shipped"), nil), "", []string{pemFile}); err != nil {
		t.Fatalf("a failed recording must not fail the installation, but got: %v", err)
	}

	value, ok := helmValues.GetString(yamled.Path{"caBundle", "additionalCertificates"})
	if !ok {
		t.Fatal("expected caBundle.additionalCertificates to be set even though the file could not be written")
	}

	if strings.TrimSpace(value) != strings.TrimSpace(certificate) {
		t.Fatalf("expected the certificate to be applied to the Helm values, but got:\n%s", value)
	}
}

func TestRecordCABundleValuesSurvivesFailedWrite(t *testing.T) {
	t.Chdir(t.TempDir())

	// Occupy the target path with a directory, so that writing the file fails no matter
	// which user the test runs as; permission bits would not stop root, which is how the
	// CI containers execute.
	if err := os.Mkdir(CABundleValuesFile, 0755); err != nil {
		t.Fatalf("failed to create %s: %v", CABundleValuesFile, err)
	}

	// a failed write must not panic or be reported as an error, as the file is only a
	// record for the administrator and not an input to the installation
	recordCABundleValues(testLogger(), map[string]string{"certificates": testCertificate(t, "corp")})

	info, err := os.Stat(CABundleValuesFile)
	if err != nil {
		t.Fatalf("expected %s to still exist: %v", CABundleValuesFile, err)
	}

	if !info.IsDir() {
		t.Fatalf("expected %s to still be the directory we created", CABundleValuesFile)
	}
}
