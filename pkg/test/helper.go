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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	"k8c.io/kubermatic/v2/pkg/provider"

	"sigs.k8s.io/yaml"
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

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(expected)),
		B:        difflib.SplitLines(output),
		FromFile: "Fixture",
		ToFile:   "Current",
		Context:  3,
	}
	diffStr, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.Fatal(err)
	}

	if diffStr != "" {
		t.Errorf("got diff between expected and actual result: \n%s\n", diffStr)
	}
}

func NewSeedGetter(seed *kubermaticv1.Seed) provider.SeedGetter {
	return func() (*kubermaticv1.Seed, error) {
		return seed, nil
	}
}

func NewConfigGetter(config *kubermaticv1.KubermaticConfiguration) provider.KubermaticConfigurationGetter {
	defaulted, err := defaults.DefaultConfiguration(config, zap.NewNop().Sugar())
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

	expectedEncoded, err := yaml.Marshal(expectedObj)
	if err != nil {
		return fmt.Errorf("failed to encode old object as YAML: %w", err)
	}

	actualEncoded, err := yaml.Marshal(actualObj)
	if err != nil {
		return fmt.Errorf("failed to encode new object as YAML: %w", err)
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(expectedEncoded)),
		B:        difflib.SplitLines(string(actualEncoded)),
		FromFile: "Expected",
		ToFile:   "Actual",
		Context:  3,
	}

	diffStr, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return fmt.Errorf("failed to create diff: %w", err)
	}

	if diffStr != "" {
		return errors.New(diffStr)
	}

	return nil
}
