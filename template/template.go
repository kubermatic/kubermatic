package template

import (
	"github.com/Masterminds/sprig"
	"github.com/kubermatic/api"
	"text/template"
)

// FuncMap defines the available functions to kubermatic templates.
var funcs = template.FuncMap{
	"apiBytesToB64": apiBytesToB64,
}

func apiBytesToB64(b api.Bytes) string {
	return b.Base64()
}

// TxtFuncMap returns an aggregated template function map. Currently (custom functions + sprig)
func TxtFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	for name, f := range funcs {
		funcMap[name] = f
	}

	return funcMap
}
