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

package crd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/sets"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func LoadFromDirectory(directory string) ([]ctrlruntimeclient.Object, error) {
	files, err := filepath.Glob(filepath.Join(directory, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to list YAML files in %q: %w", directory, err)
	}

	crds := []ctrlruntimeclient.Object{}

	for _, filename := range files {
		loaded, err := LoadFromFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to load %q: %w", filename, err)
		}

		crds = append(crds, loaded...)
	}

	return crds, nil
}

func LoadFromFile(filename string) ([]ctrlruntimeclient.Object, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	crds := []ctrlruntimeclient.Object{}
	decoder := yamlutil.NewYAMLOrJSONDecoder(f, 1024)

	for {
		crd := unstructured.Unstructured{}

		err = decoder.Decode(&crd)
		if err != nil {
			break
		}

		if crd.GetAPIVersion() != apiextensionsv1.SchemeGroupVersion.String() {
			continue
		}

		if crd.GetKind() != "CustomResourceDefinition" {
			continue
		}

		crds = append(crds, &crd)
	}

	if !errors.Is(err, io.EOF) {
		return crds, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return crds, nil
}

type ClusterKind string

const (
	MasterCluster ClusterKind = "master"
	SeedCluster   ClusterKind = "seed"

	// LocationAnnotation is the annotation on CRD object that contains a comma separated list
	// of cluster kinds where this CRD should be installed into.
	LocationAnnotation = "kubermatic.k8c.io/location"
)

func SkipCRDOnCluster(crd ctrlruntimeclient.Object, kind ClusterKind) bool {
	location := crd.GetAnnotations()[LocationAnnotation]

	// only filter out if a label exists
	if location == "" {
		return false
	}

	return !sets.New[string](strings.Split(location, ",")...).Has(string(kind))
}
