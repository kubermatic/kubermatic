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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sha256Hex(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return logger
}

// newReleaseServer serves the given files below their paths, like a GitHub release download URL.
func newReleaseServer(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, content)
	}))
	t.Cleanup(server.Close)

	return server
}

func TestGetChecksumForFile(t *testing.T) {
	checksumFile := "" +
		"1111111111111111111111111111111111111111111111111111111111111111  runc.386\n" +
		"2222222222222222222222222222222222222222222222222222222222222222  runc.amd64.asc\n" +
		"3333333333333333333333333333333333333333333333333333333333333333  runc.arm64\n" +
		"4444444444444444444444444444444444444444444444444444444444444444 *runc.amd64\n"

	path := filepath.Join(t.TempDir(), "runc.sha256sum")
	require.NoError(t, os.WriteFile(path, []byte(checksumFile), 0o644))

	testCases := []struct {
		name     string
		fileName string
		expected string
		wantErr  bool
	}{
		{
			name:     "entry in text mode",
			fileName: "runc.arm64",
			expected: "3333333333333333333333333333333333333333333333333333333333333333",
		},
		{
			name:     "entry in binary mode, not confused with a similar name",
			fileName: "runc.amd64",
			expected: "4444444444444444444444444444444444444444444444444444444444444444",
		},
		{
			name:     "missing entry",
			fileName: "runc.s390x",
			wantErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checksum, err := getChecksumForFile(path, tc.fileName)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, checksum)
		})
	}
}

func TestDownloadContainerd(t *testing.T) {
	const tarball = "containerd static tarball"
	fileName := "containerd-static-2.2.6-linux-amd64.tar.gz"
	server := newReleaseServer(t, map[string]string{
		"/v2.2.6/" + fileName:                fileName + " content: " + tarball,
		"/v2.2.6/" + fileName + ".sha256sum": fmt.Sprintf("%s  %s\n", sha256Hex(fileName+" content: "+tarball), fileName),
	})

	// The version can be given with or without the "v" prefix.
	t.Setenv("CONTAINERD_VERSION", "2.2.6")
	binPath := t.TempDir()

	require.NoError(t, downloadContainerd(context.Background(), newTestLogger(), server.URL, binPath, "amd64"))

	dir := filepath.Join(binPath, "containerd", "containerd", "releases", "download", "v2.2.6")
	assert.FileExists(t, filepath.Join(dir, fileName))
	assert.FileExists(t, filepath.Join(dir, fileName+".sha256sum"))
}

func TestDownloadContainerdChecksumMismatch(t *testing.T) {
	fileName := "containerd-static-2.2.6-linux-amd64.tar.gz"
	server := newReleaseServer(t, map[string]string{
		"/v2.2.6/" + fileName:                "tampered",
		"/v2.2.6/" + fileName + ".sha256sum": fmt.Sprintf("%s  %s\n", sha256Hex("original"), fileName),
	})

	t.Setenv("CONTAINERD_VERSION", "v2.2.6")

	err := downloadContainerd(context.Background(), newTestLogger(), server.URL, t.TempDir(), "amd64")
	assert.ErrorContains(t, err, "checksum verification failed")
}

func TestDownloadRunc(t *testing.T) {
	// The amd64 entry is deliberately not the first line of the shared checksum file.
	checksums := fmt.Sprintf("%s  runc.arm64\n%s  runc.amd64\n", sha256Hex("runc arm64"), sha256Hex("runc amd64"))

	testCases := []struct {
		name    string
		files   map[string]string
		wantErr string
	}{
		{
			name: "binary matches its entry in the shared checksum file",
			files: map[string]string{
				"/v1.3.6/runc.amd64":     "runc amd64",
				"/v1.3.6/runc.sha256sum": checksums,
			},
		},
		{
			name: "binary does not match its entry",
			files: map[string]string{
				"/v1.3.6/runc.amd64":     "tampered",
				"/v1.3.6/runc.sha256sum": checksums,
			},
			wantErr: "checksum verification failed",
		},
		{
			name: "checksum file has no entry for the architecture",
			files: map[string]string{
				"/v1.3.6/runc.amd64":     "runc amd64",
				"/v1.3.6/runc.sha256sum": fmt.Sprintf("%s  runc.arm64\n", sha256Hex("runc arm64")),
			},
			wantErr: "no checksum for runc.amd64",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := newReleaseServer(t, tc.files)
			t.Setenv("RUNC_VERSION", "")
			binPath := t.TempDir()

			err := downloadRunc(context.Background(), newTestLogger(), server.URL, binPath, "amd64")
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)

			dir := filepath.Join(binPath, "opencontainers", "runc", "releases", "download", DefaultRuncVersion)
			assert.FileExists(t, filepath.Join(dir, "runc.amd64"))
			assert.FileExists(t, filepath.Join(dir, "runc.sha256sum"))
		})
	}
}
