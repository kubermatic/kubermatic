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

package test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/test/diff"
)

func CompareOutput(t *testing.T, name, output string, update bool, suffix string) {
	filename := name + ".golden"
	if suffix != "" {
		filename += suffix
	}
	golden, err := filepath.Abs(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("failed to get absolute path to goldan file: %v", err)
	}
	if update {
		if err := os.WriteFile(golden, []byte(output), 0644); err != nil {
			t.Fatalf("failed to write updated fixture: %v", err)
		}
	}
	expected, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("failed to read .golden file: %v", err)
	}

	if d := diff.StringDiff(string(expected), output); d != "" {
		t.Fatalf("got diff between expected and actual result:\n%v", d)
	}
}

func NewSeedGetter(seed *kubermaticv1.Seed) provider.SeedGetter {
	return func() (*kubermaticv1.Seed, error) {
		return seed, nil
	}
}

func NewConfigGetter(config *kubermaticv1.KubermaticConfiguration) provider.KubermaticConfigurationGetter {
	defaulted, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	return func(_ context.Context) (*kubermaticv1.KubermaticConfiguration, error) {
		return defaulted, err
	}
}

func NewSeedsGetter(seeds ...*kubermaticv1.Seed) provider.SeedsGetter {
	result := map[string]*kubermaticv1.Seed{}

	for i, seed := range seeds {
		result[seed.Name] = seeds[i]
	}

	return func() (map[string]*kubermaticv1.Seed, error) {
		return result, nil
	}
}

func ObjectYAMLDiff(t *testing.T, expectedObj, actualObj interface{}) error {
	t.Helper()

	if d := diff.ObjectDiff(expectedObj, actualObj); d != "" {
		return errors.New(d)
	}

	return nil
}

func kubernetesVersions(cfg *kubermaticv1.KubermaticConfiguration) []semver.Semver {
	if cfg == nil {
		return defaulting.DefaultKubernetesVersioning.Versions
	}

	return cfg.Spec.Versions.Versions
}

const (
	versionLatest = "latest"
	versionStable = "stable"
)

var releaseOnly = regexp.MustCompile(`^v?[0-9]+\.[0-9]+$`)

// ParseVersionOrRelease returns the most recent supported patch release
// for a given release branch (i.e. release="1.24" might return "1.24.7"). Passing nil for the
// KubermaticConfiguration is fine and in this case the compiled-in defaults will be used.
// If the release is empty, the default version is returned.
func ParseVersionOrRelease(release string, cfg *kubermaticv1.KubermaticConfiguration) *semver.Semver {
	switch {
	case strings.ToLower(release) == versionLatest:
		return LatestKubernetesVersion(cfg)

	case strings.ToLower(release) == versionStable:
		return LatestStableKubernetesVersion(cfg)

	case release == "":
		if cfg == nil {
			return defaulting.DefaultKubernetesVersioning.Default
		}

		return cfg.Spec.Versions.Default

	// was only "1.23" or "v1.25" specified?
	case releaseOnly.MatchString(release):
		return LatestKubernetesVersionForRelease(release, cfg)

	default:
	}

	return semver.NewSemverOrDie(release)
}

// LatestKubernetesVersion returns the most recent supported patch release. Passing nil
// for the KubermaticConfiguration is fine and in this case the compiled-in defaults will
// be used.
func LatestKubernetesVersion(cfg *kubermaticv1.KubermaticConfiguration) *semver.Semver {
	versions := kubernetesVersions(cfg)

	var latest *semver.Semver
	for i, version := range versions {
		if latest == nil || version.GreaterThan(latest) {
			latest = &versions[i]
		}
	}

	return latest
}

// LatestStableKubernetesVersion returns the most recent patch release of the "stable" releases,
// which are latest-1 (i.e. if KKP is configured to support up to 1.29.7, then the stable
// releases would be all in the 1.28.x line). Passing nil for the KubermaticConfiguration
// is fine and in this case the compiled-in defaults will be used.
func LatestStableKubernetesVersion(cfg *kubermaticv1.KubermaticConfiguration) *semver.Semver {
	latest := LatestKubernetesVersion(cfg)
	if latest == nil {
		return nil
	}

	major := latest.Semver().Major()
	minor := latest.Semver().Minor() - 1

	return LatestKubernetesVersionForRelease(fmt.Sprintf("%d.%d", major, minor), cfg)
}

// LatestKubernetesVersionForRelease returns the most recent supported patch release
// for a given release branch (i.e. release="1.24" might return "1.24.7"). Passing nil for the
// KubermaticConfiguration is fine and in this case the compiled-in defaults will be used.
func LatestKubernetesVersionForRelease(release string, cfg *kubermaticv1.KubermaticConfiguration) *semver.Semver {
	parsed, err := semver.NewSemver(release)
	if err != nil {
		return nil
	}

	versions := kubernetesVersions(cfg)
	minor := parsed.Semver().Minor()

	var stable *semver.Semver
	for i, version := range versions {
		if version.Semver().Minor() != minor {
			continue
		}

		if stable == nil || version.GreaterThan(stable) {
			stable = &versions[i]
		}
	}

	return stable
}

// SafeBase64Decoding takes a value and decodes it with base64, but only
// if the given value can be decoded without errors. This primarily exists
// because in older KKP releases, we sometimes had pre-base64-encoded secrets
// in Vault, but during 2022 migrated to keeping plaintext in Vault.
func SafeBase64Decoding(value string) string {
	// If there was no error, the original value was encoded with base64.
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return string(decoded)
	}

	return value
}

// SafeBase64Encoding takes a value and encodes it with base64, but only
// if the given value was not already base64-encoded.
func SafeBase64Encoding(value string) string {
	// If there was no error, the original value was already encoded.
	if _, err := base64.StdEncoding.DecodeString(value); err == nil {
		return value
	}

	return base64.StdEncoding.EncodeToString([]byte(value))
}
