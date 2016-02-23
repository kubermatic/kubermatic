package template

import (
	"bytes"
	"encoding/json"
	texttemplate "text/template"

	"k8s.io/kubernetes/pkg/util/yaml"
)

// Template holds the data to be passed to a template
type Template struct {
	data interface{}
}

// New creates a new template with the given template data
func New(data interface{}) *Template {
	return &Template{data}
}

// Unmarshal parses the given filename and stores the result in
// the value pointed to by v.
func (t *Template) Unmarshal(filename string, v interface{}) error {
	tpl, err := texttemplate.ParseFiles(filename)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err = tpl.Execute(&buf, t.data); err != nil {
		return err
	}

	jsonBytes, err := yaml.ToJSON(buf.Bytes())
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonBytes, &v)
	if err != nil {
		return err
	}

	return nil
}
