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
	"github.com/spf13/cobra"
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

func TestWriteImagesDefaultModeGoldenBytes(t *testing.T) {
	collected := newImageCollection()
	collected.record("quay.io/kubermatic/kubermatic:v2.31.0", ImageOrigin{Kind: originReconciler, Version: "v2.31.0"})
	collected.record("quay.io/kubermatic/kubermatic:v2.31.0", ImageOrigin{Kind: originMirrorImages})
	collected.record("docker.io/library/alpine:3.20", ImageOrigin{Kind: originAddon, Name: "canal"})

	var buf bytes.Buffer
	assert.NoError(t, collected.writeImages(&buf, &ListImagesOptions{}))

	expected := "docker.io/library/alpine:3.20\n" +
		"quay.io/kubermatic/kubermatic:v2.31.0\n"
	assert.Equal(t, expected, buf.String())

	var plain bytes.Buffer
	printImages(&plain, collected.flatImageSet())
	assert.Equal(t, expected, plain.String())
}

func TestWriteImagesShowSource(t *testing.T) {
	collected := newImageCollection()
	collected.record("quay.io/kubermatic/http-prober:v0.5.1", ImageOrigin{Kind: originReconciler, Version: "v2.31"})
	collected.record("quay.io/kubermatic/http-prober:v0.5.1", ImageOrigin{Kind: originMirrorImages})
	collected.record("docker.io/library/alpine:3.20", ImageOrigin{Kind: originAddon, Name: "kubectl-env"})
	collected.recordApplicationChart("cilium", "1.13.3", "oci://quay.io/kubermatic-mirror/helm-charts", []string{"quay.io/cilium/cilium:v1.13.3"})
	collected.record("registry.k8s.io/pause:3.10", ImageOrigin{Kind: originInstallerChart})
	collected.record("registry.k8s.io/pause:3.10", ImageOrigin{Kind: originStatic})

	var buf bytes.Buffer
	assert.NoError(t, collected.writeImages(&buf, &ListImagesOptions{ShowSource: true}))

	expected := "docker.io/library/alpine:3.20\taddon/kubectl-env\n" +
		"quay.io/cilium/cilium:v1.13.3\tapplication-definition/cilium\n" +
		"quay.io/kubermatic-mirror/helm-charts/cilium:1.13.3\tapplication-definition/cilium\n" +
		"quay.io/kubermatic/http-prober:v0.5.1\treconciler@v2.31,mirror-images\n" +
		"registry.k8s.io/pause:3.10\tinstaller-chart,static\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriteImagesChartsOnlyPlain(t *testing.T) {
	collected := newImageCollection()
	collected.recordApplicationChart("cilium", "1.13.3", "oci://quay.io/kubermatic-mirror/helm-charts", []string{"quay.io/cilium/cilium:v1.13.3"})
	collected.recordApplicationChart("argocd", "3.0.0", "https://charts.example.com", []string{"quay.io/argocd/argocd:v3.0.0"})
	collected.record("quay.io/kubermatic/http-prober:v0.5.1", ImageOrigin{Kind: originReconciler, Version: "v2.31"})

	var plain bytes.Buffer
	assert.NoError(t, collected.writeImages(&plain, &ListImagesOptions{ChartsOnly: true}))
	assert.Equal(t, "quay.io/kubermatic-mirror/helm-charts/cilium:1.13.3\n", plain.String())

	var withSources bytes.Buffer
	assert.NoError(t, collected.writeImages(&withSources, &ListImagesOptions{ChartsOnly: true, ShowSource: true}))
	assert.Equal(t, "quay.io/kubermatic-mirror/helm-charts/cilium:1.13.3\tapplication-definition/cilium\n", withSources.String())
}

func TestWriteImagesJSONUnifiedStream(t *testing.T) {
	collected := newImageCollection()
	collected.recordApplicationChart("cilium", "1.13.3", "oci://quay.io/kubermatic-mirror/helm-charts", []string{"quay.io/cilium/cilium:v1.13.3"})
	collected.record("quay.io/kubermatic/http-prober:v0.5.1", ImageOrigin{Kind: originReconciler, Version: "v2.31"})
	collected.record("quay.io/kubermatic/http-prober:v0.5.1", ImageOrigin{Kind: originMirrorImages})

	var buf bytes.Buffer
	assert.NoError(t, collected.writeImages(&buf, &ListImagesOptions{OutputFormat: "json"}))

	expected := `{"kind":"chart","name":"cilium","chartVersion":"1.13.3","origin":"application-definition","source":"oci://quay.io/kubermatic-mirror/helm-charts"}` + "\n" +
		`{"kind":"image","image":"quay.io/cilium/cilium:v1.13.3","origins":[{"origin":"application-definition","name":"cilium"}]}` + "\n" +
		`{"kind":"image","image":"quay.io/kubermatic-mirror/helm-charts/cilium:1.13.3","origins":[{"origin":"application-definition","name":"cilium"}]}` + "\n" +
		`{"kind":"image","image":"quay.io/kubermatic/http-prober:v0.5.1","origins":[{"origin":"reconciler","version":"v2.31"},{"origin":"mirror-images"}]}` + "\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriteImagesChartsOnlyJSON(t *testing.T) {
	collected := newImageCollection()
	collected.recordApplicationChart("cilium", "1.13.3", "oci://quay.io/kubermatic-mirror/helm-charts", []string{"quay.io/cilium/cilium:v1.13.3"})
	collected.recordApplicationChart("argocd", "3.0.0", "https://charts.example.com", []string{"quay.io/argocd/argocd:v3.0.0"})
	collected.record("quay.io/kubermatic/http-prober:v0.5.1", ImageOrigin{Kind: originReconciler, Version: "v2.31"})

	var buf bytes.Buffer
	assert.NoError(t, collected.writeImages(&buf, &ListImagesOptions{ChartsOnly: true, OutputFormat: "json"}))

	expected := `{"kind":"chart","name":"argocd","chartVersion":"3.0.0","origin":"application-definition","source":"https://charts.example.com"}` + "\n" +
		`{"kind":"chart","name":"cilium","chartVersion":"1.13.3","origin":"application-definition","source":"oci://quay.io/kubermatic-mirror/helm-charts"}` + "\n"
	assert.Equal(t, expected, buf.String())
}

func TestWriteImagesJSONTakesPrecedenceOverShowSource(t *testing.T) {
	collected := newImageCollection()
	collected.record("quay.io/kubermatic/http-prober:v0.5.1", ImageOrigin{Kind: originReconciler, Version: "v2.31"})

	var buf bytes.Buffer
	assert.NoError(t, collected.writeImages(&buf, &ListImagesOptions{OutputFormat: "json", ShowSource: true}))

	expected := `{"kind":"image","image":"quay.io/kubermatic/http-prober:v0.5.1","origins":[{"origin":"reconciler","version":"v2.31"}]}` + "\n"
	assert.Equal(t, expected, buf.String())
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

	expectedLocalFlags := map[string]string{
		"show-source": "false",
		"charts":      "false",
		"output":      "",
	}

	for name, defaultValue := range expectedLocalFlags {
		flag := cmd.Flags().Lookup(name)
		if !assert.NotNil(t, flag, "expected local flag --%s", name) {
			continue
		}
		assert.Equal(t, defaultValue, flag.DefValue, "unexpected default for local --%s", name)
	}

	assert.Equal(t, "o", cmd.Flags().Lookup("output").Shorthand, "list-images --output must bind the -o shorthand")

	for _, name := range []string{"dry-run", "load-from", "insecure"} {
		assert.Nil(t, cmd.Flags().Lookup(name), "list-images must not expose --%s", name)
		assert.Nil(t, cmd.PersistentFlags().Lookup(name), "list-images must not expose --%s", name)
	}
}

func TestListImagesOutputFlagShadowsRootLogFormatFlag(t *testing.T) {
	logger := logrus.New()
	cmd := ListImagesCommand(logger, kubermaticversion.GetVersions())

	root := &cobra.Command{Use: "installer"}
	root.PersistentFlags().StringP("output", "o", "console", "write logs in a specific output format")
	root.AddCommand(cmd)

	assert.Nil(t, cmd.InheritedFlags().Lookup("output"), "the local list-images --output flag must shadow the root persistent --output flag")

	assert.NoError(t, cmd.Flags().Parse([]string{"-o", "json"}))
	value, err := cmd.Flags().GetString("output")
	assert.NoError(t, err)
	assert.Equal(t, "json", value, "-o json must parse into the list-images-local output flag")
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

	for _, name := range []string{"show-source", "charts", "output"} {
		assert.Nil(t, mirror.Flags().Lookup(name), "mirror-images must not expose --%s", name)
		assert.Nil(t, mirror.PersistentFlags().Lookup(name), "mirror-images must not expose --%s", name)
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

func TestListImagesFuncRejectsInvalidOutputFormat(t *testing.T) {
	logger := logrus.New()
	cmd := ListImagesCommand(logger, kubermaticversion.GetVersions())

	var out bytes.Buffer
	cmd.SetOut(&out)

	options := &ListImagesOptions{OutputFormat: "yaml"}
	err := ListImagesFunc(logger, kubermaticversion.GetVersions(), options)(cmd, nil)

	assert.Error(t, err)
	assert.Equal(t, `invalid output format "yaml", supported formats: json`, err.Error())
	assert.Empty(t, out.String())
}
