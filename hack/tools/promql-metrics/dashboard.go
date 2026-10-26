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

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// dashboardWalker holds the state needed while walking a single dashboard.
type dashboardWalker struct {
	ex   *extractor
	file string

	// dsVars maps the name of a templating variable of type "datasource"
	// to the plugin type it selects, for example "prometheus" or "loki".
	dsVars map[string]string
}

func (e *extractor) addDashboardFile(root, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var dashboard map[string]any
	if err := json.Unmarshal(data, &dashboard); err != nil {
		return fmt.Errorf("parse dashboard %s: %w", path, err)
	}

	w := &dashboardWalker{
		ex:     e,
		file:   relPath(root, path),
		dsVars: map[string]string{},
	}

	variables := templatingList(dashboard)
	for _, v := range variables {
		if str(v["type"]) == "datasource" {
			w.dsVars[str(v["name"])] = strings.ToLower(str(v["query"]))
		}
	}

	w.walkPanels(asList(dashboard["panels"]))

	// dashboards from before Grafana 5 keep panels inside top-level rows
	for _, row := range asList(dashboard["rows"]) {
		if r, ok := row.(map[string]any); ok {
			w.walkPanels(asList(r["panels"]))
		}
	}

	for _, v := range variables {
		if str(v["type"]) == "query" {
			w.addVariableQuery(v)
		}
	}

	return nil
}

func templatingList(dashboard map[string]any) []map[string]any {
	templating, ok := dashboard["templating"].(map[string]any)
	if !ok {
		return nil
	}

	var out []map[string]any
	for _, item := range asList(templating["list"]) {
		if v, ok := item.(map[string]any); ok {
			out = append(out, v)
		}
	}

	return out
}

// walkPanels does not pass a row's datasource down to its children: Grafana
// falls back to the dashboard default for a panel without a datasource.
func (w *dashboardWalker) walkPanels(panels []any) {
	for _, item := range panels {
		panel, ok := item.(map[string]any)
		if !ok {
			continue
		}

		panelDS := panel["datasource"]

		for _, t := range asList(panel["targets"]) {
			target, ok := t.(map[string]any)
			if !ok {
				continue
			}

			expr := str(target["expr"])
			if expr == "" {
				continue
			}

			ds := target["datasource"]
			if ds == nil {
				ds = panelDS
			}

			if w.isLoki(ds) {
				w.ex.skip(w.file, expr, "loki datasource")
				continue
			}

			w.ex.addExpr(w.file, expr)
		}

		// rows (collapsed or not) and library panels nest their children here
		w.walkPanels(asList(panel["panels"]))
	}
}

var (
	labelValuesRe = regexp.MustCompile(`(?s)^\s*label_values\((.*)\)\s*$`)
	queryResultRe = regexp.MustCompile(`(?s)^\s*query_result\((.*)\)\s*$`)
	metricsRe     = regexp.MustCompile(`(?s)^\s*metrics\((.*)\)\s*$`)
	labelNamesRe  = regexp.MustCompile(`(?s)^\s*label_names\((.*)\)\s*$`)
)

func (w *dashboardWalker) addVariableQuery(v map[string]any) {
	var query string
	switch q := v["query"].(type) {
	case string:
		query = q
	case map[string]any:
		query = str(q["query"])
	}

	if strings.TrimSpace(query) == "" {
		return
	}

	if w.isLoki(v["datasource"]) {
		w.ex.skip(w.file, query, "loki datasource")
		return
	}

	switch {
	case labelValuesRe.MatchString(query):
		args := labelValuesRe.FindStringSubmatch(query)[1]
		i := lastTopLevelComma(args)
		if i < 0 {
			w.ex.skip(w.file, query, "label_values without a metric selector")
			return
		}
		w.ex.addExpr(w.file, args[:i])
	case queryResultRe.MatchString(query):
		w.ex.addExpr(w.file, queryResultRe.FindStringSubmatch(query)[1])
	case metricsRe.MatchString(query):
		w.ex.skip(w.file, query, "metrics() variable query matches names by regex")
	case labelNamesRe.MatchString(query):
		args := strings.TrimSpace(labelNamesRe.FindStringSubmatch(query)[1])
		if args == "" {
			return
		}
		w.ex.addExpr(w.file, args)
	default:
		w.ex.addExpr(w.file, query)
	}
}

// lastTopLevelComma returns the index of the last comma that is not inside
// quotes, braces, brackets or parentheses, or -1.
func lastTopLevelComma(s string) int {
	depth := 0
	var quote byte
	last := -1

	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			switch c {
			case '\\':
				i++
			case quote:
				quote = 0
			}
			continue
		}

		switch c {
		case '"', '\'', '`':
			quote = c
		case '(', '{', '[':
			depth++
		case ')', '}', ']':
			depth--
		case ',':
			if depth == 0 {
				last = i
			}
		}
	}

	return last
}

// isLoki reports whether a Grafana datasource reference points to Loki. The
// reference is either a plain string (name, uid or variable) or an object
// with "type" and "uid".
func (w *dashboardWalker) isLoki(ds any) bool {
	var typ, uid string
	switch d := ds.(type) {
	case nil:
		return false
	case string:
		uid = d
	case map[string]any:
		typ = str(d["type"])
		uid = str(d["uid"])
	}

	// a datasource variable is resolved to the plugin it selects; this wins
	// over the declared type because Grafana resolves the reference by uid
	if name := variableName(uid); name != "" {
		if plugin, ok := w.dsVars[name]; ok {
			return plugin == "loki"
		}
	}

	return strings.EqualFold(typ, "loki") || strings.Contains(strings.ToLower(uid), "loki")
}

var variableNameRe = regexp.MustCompile(`^\s*(?:\$\{([A-Za-z0-9_]+)(?:[:.][^}]*)?\}|\$([A-Za-z0-9_]+)|\[\[([A-Za-z0-9_]+)(?::[^\]]*)?\]\])\s*$`)

// variableName returns the variable name if s is a single Grafana variable
// reference such as $ds, ${ds} or [[ds]].
func variableName(s string) string {
	m := variableNameRe.FindStringSubmatch(s)
	if m == nil {
		return ""
	}

	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}

	return ""
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
