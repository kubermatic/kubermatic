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

// Command promql-metrics lists every Prometheus metric name that KKP's
// Prometheus rule files and Grafana dashboards reference, and can check
// that list against a metric contract file.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"

	"sigs.k8s.io/yaml"
)

const (
	defaultRules      = "charts/monitoring/prometheus/rules/*.yaml"
	defaultDashboards = "charts/monitoring/grafana/dashboards/**/*.json,charts/mla/grafana/files/*.json,charts/mla/grafana/dashboards/*.json"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("promql-metrics", flag.ContinueOnError)
	fs.SetOutput(stderr)

	root := fs.String("root", "", "repository root; defaults to the first parent directory that contains charts/monitoring")
	rules := fs.String("rules", defaultRules, "comma-separated globs of Prometheus rule files, relative to -root")
	dashboards := fs.String("dashboards", defaultDashboards, "comma-separated globs of Grafana dashboard files, relative to -root")
	format := fs.String("format", "text", "output format: text or json")
	contract := fs.String("contract", "", "path to the metric contract YAML file")
	check := fs.Bool("check", false, "compare the referenced metrics against -contract and exit 1 on drift")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *check && *contract == "" {
		fmt.Fprintln(stderr, "-check requires -contract")
		return 2
	}

	if *root == "" {
		r, err := findRoot()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		*root = r
	}

	ex := newExtractor()

	ruleFiles, err := globAll(*root, *rules)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	dashboardFiles, err := globAll(*root, *dashboards)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	for _, f := range ruleFiles {
		if err := ex.addRuleFile(*root, f); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	for _, f := range dashboardFiles {
		if err := ex.addDashboardFile(*root, f); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}

	if *check {
		c, err := loadContract(*contract)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}

		problems := c.check(ex.names)
		if len(problems) > 0 {
			fmt.Fprintf(stderr, "metric contract %s is out of date:\n", *contract)
			for _, p := range problems {
				fmt.Fprintf(stderr, "  %s\n", p)
			}
			return 1
		}

		fmt.Fprintf(stdout, "metric contract OK: %d referenced metrics, %d skipped expressions\n", len(ex.names), len(ex.skipped))
		return 0
	}

	switch *format {
	case "text":
		for _, name := range ex.sortedNames() {
			fmt.Fprintln(stdout, name)
		}
		fmt.Fprintf(stderr, "%d metrics from %d rule files and %d dashboards, %d expressions skipped\n",
			len(ex.names), len(ruleFiles), len(dashboardFiles), len(ex.skipped))
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(ex.report()); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	default:
		fmt.Fprintf(stderr, "unknown -format %q\n", *format)
		return 2
	}

	return 0
}

func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if info, err := os.Stat(filepath.Join(dir, "charts", "monitoring")); err == nil && info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("cannot find repository root, pass -root")
		}
		dir = parent
	}
}

// globAll expands comma-separated glob patterns relative to root. Unlike
// filepath.Glob it supports "**" to match any number of directories.
func globAll(root, patterns string) ([]string, error) {
	seen := map[string]struct{}{}

	for pattern := range strings.SplitSeq(patterns, ",") {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		matches, err := glob(root, filepath.ToSlash(pattern))
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", pattern, err)
		}

		for _, m := range matches {
			seen[m] = struct{}{}
		}
	}

	return slices.Sorted(func(yield func(string) bool) {
		for k := range seen {
			if !yield(k) {
				return
			}
		}
	}), nil
}

func glob(root, pattern string) ([]string, error) {
	if !strings.Contains(pattern, "**") {
		return filepath.Glob(filepath.Join(root, pattern))
	}

	// walk from the static part of the pattern and match the rest as a regex
	base, _, _ := strings.Cut(pattern, "**")
	base = base[:strings.LastIndex(base, "/")+1]

	re, err := globToRegexp(pattern)
	if err != nil {
		return nil, err
	}

	var matches []string

	walkRoot := filepath.Join(root, base)
	if _, err := os.Stat(walkRoot); errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	err = filepath.WalkDir(walkRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if re.MatchString(filepath.ToSlash(rel)) {
			matches = append(matches, path)
		}

		return nil
	})

	return matches, err
}

func globToRegexp(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")

	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case strings.HasPrefix(pattern[i:], "**/"):
			b.WriteString("(.*/)?")
			i += 2
		case strings.HasPrefix(pattern[i:], "**"):
			b.WriteString(".*")
			i++
		case c == '*':
			b.WriteString("[^/]*")
		case c == '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(c)))
		}
	}

	b.WriteString("$")

	return regexp.Compile(b.String())
}

type skippedExpr struct {
	File   string `json:"file"`
	Expr   string `json:"expr"`
	Reason string `json:"reason"`
}

type metricRef struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

type report struct {
	Metrics []metricRef   `json:"metrics"`
	Skipped []skippedExpr `json:"skipped"`
}

type extractor struct {
	parser  parser.Parser
	names   map[string]map[string]struct{}
	skipped []skippedExpr
}

func newExtractor() *extractor {
	return &extractor{
		parser: parser.NewParser(parser.Options{EnableExperimentalFunctions: true}),
		names:  map[string]map[string]struct{}{},
	}
}

func (e *extractor) sortedNames() []string {
	names := make([]string, 0, len(e.names))
	for n := range e.names {
		names = append(names, n)
	}
	sort.Strings(names)

	return names
}

func (e *extractor) report() report {
	r := report{
		Metrics: []metricRef{},
		Skipped: e.skipped,
	}
	if r.Skipped == nil {
		r.Skipped = []skippedExpr{}
	}

	for _, n := range e.sortedNames() {
		files := make([]string, 0, len(e.names[n]))
		for f := range e.names[n] {
			files = append(files, f)
		}
		sort.Strings(files)
		r.Metrics = append(r.Metrics, metricRef{Name: n, Files: files})
	}

	return r
}

func (e *extractor) skip(file, expr, reason string) {
	e.skipped = append(e.skipped, skippedExpr{File: file, Expr: expr, Reason: reason})
}

// addExpr substitutes Grafana variables, parses expr and records every
// metric name found in its vector selectors.
func (e *extractor) addExpr(file, expr string) {
	if strings.TrimSpace(expr) == "" {
		return
	}

	parsed, err := e.parser.ParseExpr(substituteVariables(expr))
	if err != nil {
		e.skip(file, expr, "parse error: "+firstLine(err.Error()))
		return
	}

	parser.Inspect(parsed, func(node parser.Node, _ []parser.Node) error {
		vs, ok := node.(*parser.VectorSelector)
		if !ok {
			return nil
		}

		if vs.Name != "" {
			e.record(vs.Name, file)
			return nil
		}

		for _, m := range vs.LabelMatchers {
			if m.Name != model.MetricNameLabel {
				continue
			}

			if m.Type == labels.MatchEqual {
				e.record(m.Value, file)
			} else {
				e.skip(file, expr, fmt.Sprintf("non-equality __name__ matcher: %s", m.String()))
			}
		}

		return nil
	})
}

func (e *extractor) record(name, file string) {
	if e.names[name] == nil {
		e.names[name] = map[string]struct{}{}
	}
	e.names[name][file] = struct{}{}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

type ruleFile struct {
	Groups []struct {
		Rules []struct {
			Expr string `json:"expr"`
		} `json:"rules"`
	} `json:"groups"`
}

func (e *extractor) addRuleFile(root, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var rf ruleFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return fmt.Errorf("parse rule file %s: %w", path, err)
	}

	rel := relPath(root, path)
	for _, g := range rf.Groups {
		for _, r := range g.Rules {
			e.addExpr(rel, r.Expr)
		}
	}

	return nil
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
