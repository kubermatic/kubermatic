package yaml

import (
	"io"

	yaml3 "gopkg.in/yaml.v3"

	"k8s.io/test-infra/pkg/genyaml"
)

// Encode takes a runtime object and creates a YAML encoded version in the
// given output. The special aspect of this function is that it does not
// output the creationTimestamp when marshalling a syntetic resource. This
// just makes the YAML look nicer when presented to the enduser.
func Encode(resource interface{}, output io.Writer) error {
	encoder := yaml3.NewEncoder(output)
	encoder.SetIndent(2)

	// genyaml is smart enough to not output a creationTimestamp when marshalling as YAML
	return genyaml.NewCommentMap().EncodeYaml(resource, encoder)
}
