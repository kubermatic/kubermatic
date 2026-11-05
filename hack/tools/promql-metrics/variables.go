/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package main

import "strings"

// durationVariables are Grafana built-ins that expand to a PromQL duration.
var durationVariables = map[string]bool{
	"__rate_interval": true,
	"__interval":      true,
	"__range":         true,
}

// numberVariables are Grafana built-ins that expand to a plain number.
var numberVariables = map[string]bool{
	"__interval_ms": true,
	"__range_s":     true,
	"__range_ms":    true,
}

// substituteVariables replaces Grafana template variables ($var, ${var},
// ${var:format} and [[var]]) with values that keep the expression valid
// PromQL. The replacement depends on where the variable sits:
//
//   - inside a quoted string (matcher value or regex): .+
//   - as an unquoted matcher value, e.g. {pod=$pod}: ".+"
//   - inside a range or subquery bracket, or a duration built-in: 5m
//   - anywhere else, e.g. a threshold like "> $min": 1
func substituteVariables(expr string) string {
	var (
		out          strings.Builder
		quote        byte
		braceDepth   int
		bracketDepth int
	)

	for i := 0; i < len(expr); {
		c := expr[i]

		if name, n := readVariable(expr[i:]); n > 0 {
			out.WriteString(variableValue(name, quote != 0, bracketDepth > 0, braceDepth > 0 && afterMatcherOp(out.String())))
			i += n
			continue
		}

		if quote != 0 {
			out.WriteByte(c)
			switch {
			case c == '\\' && quote != '`' && i+1 < len(expr):
				out.WriteByte(expr[i+1])
				i += 2
				continue
			case c == quote:
				quote = 0
			}
			i++
			continue
		}

		switch c {
		case '"', '\'', '`':
			quote = c
		case '{':
			braceDepth++
		case '}':
			braceDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		}

		out.WriteByte(c)
		i++
	}

	return out.String()
}

func variableValue(name string, inString, inBracket, matcherValue bool) string {
	switch {
	case inString:
		return ".+"
	case durationVariables[name]:
		return "5m"
	case numberVariables[name]:
		return "300"
	case inBracket:
		return "5m"
	case matcherValue:
		return `".+"`
	default:
		return "1"
	}
}

// readVariable returns the name of the Grafana variable at the start of s
// and the number of bytes it spans, or 0 when s does not start with one.
func readVariable(s string) (string, int) {
	switch {
	case strings.HasPrefix(s, "${"):
		end := strings.IndexByte(s, '}')
		if end < 0 {
			return "", 0
		}
		name := s[2:end]
		if i := strings.IndexAny(name, ":."); i >= 0 {
			name = name[:i]
		}
		if name == "" || !isIdent(name) {
			return "", 0
		}
		return name, end + 1
	case strings.HasPrefix(s, "[["):
		end := strings.Index(s, "]]")
		if end < 0 {
			return "", 0
		}
		name := s[2:end]
		if i := strings.IndexByte(name, ':'); i >= 0 {
			name = name[:i]
		}
		if name == "" || !isIdent(name) {
			return "", 0
		}
		return name, end + 2
	case strings.HasPrefix(s, "$"):
		n := 1
		for n < len(s) && isIdentByte(s[n]) {
			n++
		}
		if n == 1 {
			return "", 0
		}
		return s[1:n], n
	}

	return "", 0
}

func afterMatcherOp(s string) bool {
	s = strings.TrimRight(s, " \t\n")
	return strings.HasSuffix(s, "=") || strings.HasSuffix(s, "=~") || strings.HasSuffix(s, "!~")
}

func isIdent(s string) bool {
	for i := range len(s) {
		if !isIdentByte(s[i]) {
			return false
		}
	}
	return true
}

func isIdentByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}
