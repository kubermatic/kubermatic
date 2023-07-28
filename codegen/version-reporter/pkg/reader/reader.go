/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package reader

import (
	"errors"
	"log"
	"os"

	"k8c.io/kubermatic/v2/codegen/version-reporter/pkg/config"
)

var (
	ErrVersionNotFound = errors.New("no such constant")
)

func ResolveConfig(cfg *config.Config) bool {
	success := true

	for pdx, product := range cfg.Products {
		log.Printf("Fetching versions for %s…", product.Name)

		for odx, occ := range product.Occurrences {
			versions, err := ResolveReference(occ)
			if err != nil {
				log.Printf("Error: failed to read version #%d for %s: %v", odx, product.Name, err)
				success = false
				continue
			}

			cfg.Products[pdx].Occurrences[odx].Versions = versions
		}
	}

	return success
}

func ResolveReference(ref config.VersionReference) (map[string]string, error) {
	switch {
	case ref.GoConstant != nil:
		return readGoConstantVersion(ref.GoConstant)
	case ref.GoFunction != nil:
		return readGoFunctionVersion(ref.GoFunction)
	case ref.HelmChart != nil:
		return readHelmChartVersion(ref.HelmChart)
	case ref.YAMLFile != nil:
		return readYAMLVersion(ref.YAMLFile)
	default:
		return nil, errors.New("unknown reference type")
	}
}

func readGoConstantVersion(ref *config.GoConstantReference) (map[string]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	version, err := ReadGoConstantFromPackage(cwd, ref.Package, ref.Constant)
	if err != nil {
		return nil, err
	}

	return map[string]string{config.Unversioned: version}, nil
}

func readGoFunctionVersion(ref *config.GoFunctionReference) (map[string]string, error) {
	return CallGoFunction(ref.Function)
}

func readHelmChartVersion(ref *config.HelmChartReference) (map[string]string, error) {
	version, err := ReadHelmChartVersion(ref.Directory, ref.ValuePath)
	if err != nil {
		return nil, err
	}

	return map[string]string{config.Unversioned: version}, nil
}

func readYAMLVersion(ref *config.YAMLFileReference) (map[string]string, error) {
	version, err := ReadYAMLVersion(ref.File, ref.ValuePath)
	if err != nil {
		return nil, err
	}

	return map[string]string{config.Unversioned: version}, nil
}
