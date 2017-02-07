package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	texttemplate "text/template"

	"k8s.io/client-go/pkg/util/yaml"
)

// Template holds the actual cluster template
type Template struct {
	tpl *texttemplate.Template
}

// ParseFiles creates a new template for the given filenames
// and parses the template definitions from the named files.
func ParseFiles(filenames string) (*Template, error) {
	tpl, err := texttemplate.ParseFiles(filenames)
	if err != nil {
		return nil, err
	}

	return &Template{tpl}, nil
}

// Execute applies a parsed template to the specified data object,
// and stores the result in the value pointed to by v.
func (t *Template) Execute(data, v interface{}) error {
	var buf bytes.Buffer
	if err := t.tpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed executing template: %v", err)
	}

	jsonBytes, err := yaml.ToJSON(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed converting yaml to json: %v", err)
	}

	if err := json.Unmarshal(jsonBytes, &v); err != nil {
		return fmt.Errorf("failed unmarshaling: %v", err)
	}

	return nil
}
