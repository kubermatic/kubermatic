package template

import (
	"fmt"
	"net"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

func ipSliceToCommaSeparatedString(ips []net.IP) string {
	var s string
	for _, ip := range ips {
		s = s + fmt.Sprintf("%s,", ip.String())
	}

	return strings.TrimSuffix(s, ",")
}

// TxtFuncMap returns an aggregated template function map. Currently (custom functions + sprig)
func TxtFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	funcMap["ipSliceToCommaSeparatedString"] = ipSliceToCommaSeparatedString

	return funcMap
}
