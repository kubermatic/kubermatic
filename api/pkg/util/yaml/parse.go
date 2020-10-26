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

package yaml

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/runtime"
	kyaml "k8s.io/apimachinery/pkg/util/yaml"
)

func ParseMultipleDocuments(input io.Reader) ([]runtime.RawExtension, error) {
	reader := kyaml.NewYAMLReader(bufio.NewReader(input))
	objects := []runtime.RawExtension{}

	for {
		b, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed reading from YAML reader: %v", err)
		}

		b = bytes.TrimSpace(b)
		if len(b) == 0 {
			continue
		}

		decoder := kyaml.NewYAMLToJSONDecoder(bytes.NewBuffer(b))
		raw := runtime.RawExtension{}
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decoding failed: %v", err)
		}

		// skip empty documents (e.g. documents that are only comments)
		if len(raw.Raw) == 0 {
			continue
		}

		objects = append(objects, raw)
	}

	return objects, nil
}
