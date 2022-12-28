/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/gobuffalo/flect"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// embeddedFS is an embedded fs that contains kubermatic CRD manifest
//
//go:embed k8c.io/*
var embeddedFS embed.FS

const rootDir = "k8c.io"

// Groups returns a list of all known API groups for which CRDs are available.
func Groups() ([]string, error) {
	entries, err := embeddedFS.ReadDir(rootDir)
	if err != nil {
		return nil, err
	}

	groups := sets.New[string]()

	for _, entry := range entries {
		name := strings.Split(entry.Name(), "_")
		groups.Insert(name[0])
	}

	return sets.List(groups), nil
}

// CRDForType returns the CRD for a given object or returns an error if the
// object is not using one of the known types (*.k8c.io).
func CRDForObject(obj runtime.Object) (*apiextensionsv1.CustomResourceDefinition, error) {
	return CRDForGVK(obj.GetObjectKind().GroupVersionKind())
}

// CRDForGVK returns the CRD for the given GKV or an error if there is no
// CRD available (i.e. any GVK outside of *.k8c.io was provided).
func CRDForGVK(gvk schema.GroupVersionKind) (*apiextensionsv1.CustomResourceDefinition, error) {
	// as filenames are being generated by controller-gen, use the same pluralization mechanism
	// https://github.com/kubernetes-sigs/controller-tools/blob/8cb5ce83c3cca425a4de0af3d2578e31a3cd6a48/pkg/crd/spec.go#L58
	kindPlural := strings.ToLower(flect.Pluralize(gvk.Kind))
	filename := gvk.Group + "_" + kindPlural + ".yaml"

	crd, err := loadCRD(filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("no CRD available for \"%s_%s\"", gvk.Group, kindPlural)
		}

		return nil, err
	}

	return crd, nil
}

// CRDsForGroup returns all CRDs for the given API group (e.g. "kubermatic.k8c.io").
func CRDsForGroup(apiGroup string) ([]apiextensionsv1.CustomResourceDefinition, error) {
	files, err := embeddedFS.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("x: %w", err)
	}

	prefix := fmt.Sprintf("%s_", apiGroup)
	result := []apiextensionsv1.CustomResourceDefinition{}

	for _, info := range files {
		filename := info.Name()
		if !strings.HasPrefix(filename, prefix) {
			continue
		}

		crd, err := loadCRD(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to open CRD: %w", err)
		}

		result = append(result, *crd)
	}

	return result, nil
}

func loadCRD(filename string) (*apiextensionsv1.CustomResourceDefinition, error) {
	f, err := embeddedFS.Open(rootDir + "/" + filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	crd := &apiextensionsv1.CustomResourceDefinition{}
	dec := yaml.NewYAMLOrJSONDecoder(f, 1024)
	if err := dec.Decode(crd); err != nil {
		return nil, err
	}

	return crd, nil
}
