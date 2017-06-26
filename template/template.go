package template

import (
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"io/ioutil"
	"text/template"
)

// FuncMap defines the available functions to kubermatic templates.
var funcs = template.FuncMap{
	"apiBytesToB64":    apiBytesToB64,
	"apiBytesToString": apiBytesToString,
}

func apiBytesToB64(b api.Bytes) string {
	return b.Base64()
}

func apiBytesToString(b api.Bytes) string {
	return string(b)
}

// TxtFuncMap returns an aggregated template function map. Currently (custom functions + sprig)
func TxtFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	for name, f := range funcs {
		funcMap[name] = f
	}

	return funcMap
}

// ParseFile creates a new template for the given filenames
// and parses the template definitions from the named files.
func ParseFile(filename string) (*template.Template, error) {
	glog.V(6).Infof("Loading template %q", filename)

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	tpl, err := template.New("base").Funcs(TxtFuncMap()).Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %q: %v", filename, err)
	}

	return tpl, nil
}
