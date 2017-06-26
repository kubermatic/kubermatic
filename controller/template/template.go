package template

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	texttemplate "text/template"

	"github.com/golang/glog"
	"github.com/kubermatic/api/template"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// Template holds the actual cluster template
type Template struct {
	tpl *texttemplate.Template
}

// ParseFiles creates a new template for the given filenames
// and parses the template definitions from the named files.
func ParseFiles(filename string) (*Template, error) {
	glog.V(6).Infof("Loading template %q", filename)

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	tpl, err := texttemplate.New("base").Funcs(template.TxtFuncMap()).Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %v", filename, err)
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
