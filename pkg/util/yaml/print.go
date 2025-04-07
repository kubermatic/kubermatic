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
	"io"

	"k8s.io/test-infra/pkg/genyaml"
	yaml3 "sigs.k8s.io/yaml/goyaml.v3"
)

// Encode takes a runtime object and creates a YAML encoded version in the
// given output. The special aspect of this function is that it does not
// output the creationTimestamp when marshalling a synthetic resource. This
// just makes the YAML look nicer when presented to the enduser.
func Encode(resource interface{}, output io.Writer) error {
	encoder := yaml3.NewEncoder(output)
	encoder.SetIndent(2)

	cm, err := genyaml.NewCommentMap(nil)
	if err != nil {
		return err
	}

	// genyaml is smart enough to not output a creationTimestamp when marshalling as YAML
	return cm.EncodeYaml(resource, encoder)
}
