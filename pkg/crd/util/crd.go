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

package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

func LoadFromDirectory(directory string) ([]apiextensionsv1beta1.CustomResourceDefinition, error) {
	files, err := filepath.Glob(filepath.Join(directory, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list YAML files in %q: %v", directory, err)
	}

	crds := []apiextensionsv1beta1.CustomResourceDefinition{}

	for _, filename := range files {
		loaded, err := LoadFromFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to load %q: %v", filename, err)
		}

		crds = append(crds, loaded...)
	}

	return crds, nil
}

func LoadFromFile(filename string) ([]apiextensionsv1beta1.CustomResourceDefinition, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	crds := []apiextensionsv1beta1.CustomResourceDefinition{}
	decoder := yamlutil.NewYAMLOrJSONDecoder(f, 1024)

	for {
		crd := apiextensionsv1beta1.CustomResourceDefinition{}

		err = decoder.Decode(&crd)
		if err != nil {
			break
		}

		crds = append(crds, crd)
	}

	if err != io.EOF {
		return crds, fmt.Errorf("failed to decode YAML: %v", err)
	}

	return crds, nil
}
