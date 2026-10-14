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
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestPrintImages(t *testing.T) {
	testCases := []struct {
		name     string
		images   []string
		expected string
	}{
		{
			name:   "sorted and deduplicated, one per line",
			images: []string{"quay.io/kubermatic/kubermatic:v2.31.0", "docker.io/library/alpine:3.20", "quay.io/kubermatic/kubermatic:v2.31.0", "docker.io/library/busybox:1.36"},
			expected: "docker.io/library/alpine:3.20\n" +
				"docker.io/library/busybox:1.36\n" +
				"quay.io/kubermatic/kubermatic:v2.31.0\n",
		},
		{
			name:     "empty set prints nothing",
			images:   nil,
			expected: "",
		},
		{
			name:     "single image has trailing newline",
			images:   []string{"quay.io/kubermatic/addons:v2.31.0"},
			expected: "quay.io/kubermatic/addons:v2.31.0\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			printImages(&buf, sets.New(tc.images...))
			assert.Equal(t, tc.expected, buf.String())
		})
	}
}

func TestListImagesCommandFlagSurface(t *testing.T) {
	logger := logrus.New()
	cmd := ListImagesCommand(logger, kubermaticversion.GetVersions())

	expectedFlags := map[string]string{
		"config":                      "",
		"version-filter":              "",
		"registry-prefix":             "",
		"ignore-repository-overrides": "true",
		"addons-path":                 "",
		"addons-image":                "",
		"helm-timeout":                "5m0s",
		"helm-values":                 "",
		"helm-binary":                 "helm",
	}

	for name, defaultValue := range expectedFlags {
		flag := cmd.PersistentFlags().Lookup(name)
		if !assert.NotNil(t, flag, "expected flag --%s", name) {
			continue
		}
		assert.Equal(t, defaultValue, flag.DefValue, "unexpected default for --%s", name)
	}

	assert.NotNil(t, cmd.PersistentFlags().Lookup("provider-filter"))
	assert.Equal(t, "stringArray", cmd.PersistentFlags().Lookup("provider-filter").Value.Type())

	for _, name := range []string{"dry-run", "load-from", "insecure"} {
		assert.Nil(t, cmd.Flags().Lookup(name), "list-images must not expose --%s", name)
		assert.Nil(t, cmd.PersistentFlags().Lookup(name), "list-images must not expose --%s", name)
	}
}

func TestMirrorAndListImagesSharedFlagDefaults(t *testing.T) {
	logger := logrus.New()
	versions := kubermaticversion.GetVersions()

	mirror := MirrorImagesCommand(logger, versions)
	list := ListImagesCommand(logger, versions)

	sharedFlags := []string{
		"config", "version-filter", "provider-filter", "registry-prefix",
		"ignore-repository-overrides", "addons-path", "addons-image",
		"helm-timeout", "helm-values", "helm-binary",
	}

	for _, name := range sharedFlags {
		mirrorFlag := mirror.PersistentFlags().Lookup(name)
		listFlag := list.PersistentFlags().Lookup(name)
		if !assert.NotNil(t, mirrorFlag, "mirror-images must expose --%s", name) || !assert.NotNil(t, listFlag, "list-images must expose --%s", name) {
			continue
		}
		assert.Equal(t, mirrorFlag.DefValue, listFlag.DefValue, "--%s default differs between mirror-images and list-images", name)
	}

	for _, name := range []string{"load-from", "dry-run", "insecure"} {
		assert.Nil(t, list.PersistentFlags().Lookup(name), "list-images must not expose --%s", name)
		assert.Nil(t, list.Flags().Lookup(name), "list-images must not expose --%s", name)
	}
}

func TestListImagesCommandRejectsPositionalArgs(t *testing.T) {
	logger := logrus.New()
	cmd := ListImagesCommand(logger, kubermaticversion.GetVersions())

	if !assert.NotNil(t, cmd.Args) {
		return
	}
	assert.NoError(t, cmd.Args(cmd, nil))
	assert.Error(t, cmd.Args(cmd, []string{"registry.example.com"}))
}

func TestLoadAndDefaultKubermaticConfigurationRequiresConfigNotRegistry(t *testing.T) {
	options := &ImageCollectionOptions{}

	_, err := loadAndDefaultKubermaticConfiguration(options)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "no target registry was passed")
	assert.Contains(t, err.Error(), "please specify your KubermaticConfiguration")
}

func TestLoadAndDefaultKubermaticConfigurationRejectsAddonsImageAndPath(t *testing.T) {
	options := &ImageCollectionOptions{
		AddonsImage: "quay.io/kubermatic/addons:v2.31.0",
		AddonsPath:  "/tmp/addons",
	}

	_, err := loadAndDefaultKubermaticConfiguration(options)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--addons-image and --addons-path must not be set at the same time")
}

func TestGetKubermaticConfigurationRequiresRegistry(t *testing.T) {
	options := &MirrorImagesOptions{}

	_, err := getKubermaticConfiguration(options)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no target registry was passed")

	options.Registry = "registry.example.com"
	options.Config = "/does/not/exist.yaml"
	_, err = getKubermaticConfiguration(options)
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "no target registry was passed")
}
