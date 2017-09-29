package kubernetes

import (
	"bytes"
	"encoding/json"
	"fmt"
	texttemplate "text/template"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/template"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Template holds the actual cluster template
type Template struct {
	tpl *texttemplate.Template
}

// ParseFile creates a new template for the given filenames
// and parses the template definitions from the named files.
func ParseFile(filename string) (*Template, error) {
	t, err := template.ParseFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	return &Template{t}, nil
}

// Execute applies a parsed template to the specified data object,
// and stores the result in the value pointed to by v.
func (t *Template) Execute(data, v interface{}) error {
	var buf bytes.Buffer
	if err := t.tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed executing template: %v", err)
	}

	glog.Info(buf.String())

	jsonBytes, err := yaml.ToJSON(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed converting yaml to json: %v", err)
	}

	if err := json.Unmarshal(jsonBytes, &v); err != nil {
		return fmt.Errorf("failed unmarshaling: %v", err)
	}

	return nil
}
