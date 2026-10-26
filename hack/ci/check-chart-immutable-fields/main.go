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

// check-chart-immutable-fields compares the golden Helm renders (*.yaml.out)
// of a base git ref with the working tree and fails when a change touches a
// field that `helm upgrade` cannot apply in place, or removes an object that
// holds data.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

const (
	defaultBase      = "origin/main"
	defaultAllowlist = "hack/ci/testdata/chart-immutable-allowlist.yaml"
	helmHookKey      = "helm.sh/hook"
	removedField     = "removed"
)

type allowEntry struct {
	Object    string `json:"object"`
	Field     string `json:"field"`
	Reason    string `json:"reason"`
	Migration string `json:"migration"`
}

type violation struct {
	File   string
	Object string
	Field  string
	Old    any
	New    any
}

// renderPair holds one golden file at the base ref and in the working tree.
// A nil Base means the file does not exist at the base ref, a nil Current
// means it was deleted.
type renderPair struct {
	Path    string
	Base    []byte
	Current []byte
}

type object struct {
	ID   string
	Kind string
	Raw  map[string]any
}

func main() {
	base := flag.String("base", baseFromEnv(), "git ref to compare against")
	allowlistPath := flag.String("allowlist", defaultAllowlist, "path to the allowlist YAML file")
	flag.Parse()

	if err := run(*base, *allowlistPath, flag.Args(), os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func baseFromEnv() string {
	if sha := os.Getenv("PULL_BASE_SHA"); sha != "" {
		return sha
	}

	return defaultBase
}

func run(base, allowlistPath string, files []string, out io.Writer) error {
	allowlistData, err := os.ReadFile(allowlistPath)
	if err != nil {
		return fmt.Errorf("failed to read allowlist: %w", err)
	}

	allowlist, err := parseAllowlist(allowlistData)
	if err != nil {
		return fmt.Errorf("invalid allowlist %s: %w", allowlistPath, err)
	}

	if _, err := git("rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		return fmt.Errorf("base ref %q is not a commit: %w", base, err)
	}

	root, err := git("rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	repoRoot := strings.TrimSpace(string(root))

	if len(files) == 0 {
		changed, err := git("diff", "--name-only", base, "--", "charts/**/*.yaml.out")
		if err != nil {
			return err
		}
		for line := range strings.SplitSeq(strings.TrimSpace(string(changed)), "\n") {
			if line != "" {
				files = append(files, filepath.Join(repoRoot, line))
			}
		}
	}

	var pairs []renderPair
	for _, file := range files {
		pair, err := loadPair(repoRoot, base, file)
		if err != nil {
			return err
		}
		pairs = append(pairs, pair)
	}

	ok, err := check(pairs, allowlist, out)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("found Helm chart changes that cannot be applied in place")
	}

	return nil
}

func loadPair(repoRoot, base, file string) (renderPair, error) {
	abs, err := filepath.Abs(file)
	if err != nil {
		return renderPair{}, err
	}

	rel, err := filepath.Rel(repoRoot, abs)
	if err != nil {
		return renderPair{}, err
	}
	rel = filepath.ToSlash(rel)

	pair := renderPair{Path: rel}

	// cat-file -e exits non-zero when the path does not exist at base.
	if _, err := git("cat-file", "-e", base+":"+rel); err == nil {
		content, err := git("show", base+":"+rel)
		if err != nil {
			return renderPair{}, err
		}
		// keep the slice non-nil so an empty file at base is not taken as new
		pair.Base = append([]byte{}, content...)
	}

	content, err := os.ReadFile(abs)
	switch {
	case err == nil:
		pair.Current = content
	case errors.Is(err, fs.ErrNotExist):
		pair.Current = nil
	default:
		return renderPair{}, err
	}

	return pair, nil
}

func git(args ...string) ([]byte, error) {
	var stderr bytes.Buffer

	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return output, nil
}

func parseAllowlist(data []byte) ([]allowEntry, error) {
	var entries []allowEntry
	if err := yaml.UnmarshalStrict(data, &entries); err != nil {
		return nil, err
	}

	var errs []error
	for i, e := range entries {
		if e.Object == "" || e.Field == "" {
			errs = append(errs, fmt.Errorf("entry %d: object and field are required", i))
		}
		if e.Reason == "" {
			errs = append(errs, fmt.Errorf("entry %d (%s %s): reason is required", i, e.Object, e.Field))
		}
		if e.Migration == "" {
			errs = append(errs, fmt.Errorf("entry %d (%s %s): migration is required, point it to the code that migrates existing installations", i, e.Object, e.Field))
		}
	}

	return entries, errors.Join(errs...)
}

// check compares all render pairs and prints the result. It returns false
// when at least one violation is not covered by the allowlist.
func check(pairs []renderPair, allowlist []allowEntry, out io.Writer) (bool, error) {
	ok := true

	for _, pair := range pairs {
		if pair.Base == nil {
			fmt.Fprintf(out, "SKIP %s: new file\n", pair.Path)
			continue
		}

		violations, err := compareRenders(pair.Path, pair.Base, pair.Current)
		if err != nil {
			return false, err
		}

		for _, v := range violations {
			if entry := findAllowEntry(allowlist, v); entry != nil {
				fmt.Fprintf(out, "ALLOWED %s: %s %s (reason: %s, migration: %s)\n", v.File, v.Object, v.Field, entry.Reason, entry.Migration)
				continue
			}

			ok = false
			fmt.Fprintf(out, "FAIL %s: %s %s\n", v.File, v.Object, v.Field)
			fmt.Fprintf(out, "  old: %s\n", formatValue(v.Old))
			fmt.Fprintf(out, "  new: %s\n", formatValue(v.New))
			fmt.Fprintf(out, "  to allow it, add {object: %q, field: %q, reason: ..., migration: <file:line of the upgrade migration>} to %s\n", v.Object, v.Field, defaultAllowlist)
		}
	}

	return ok, nil
}

func findAllowEntry(allowlist []allowEntry, v violation) *allowEntry {
	for i := range allowlist {
		if allowlist[i].Object == v.Object && allowlist[i].Field == v.Field {
			return &allowlist[i]
		}
	}

	return nil
}

func formatValue(v any) string {
	if v == nil {
		return "<unset>"
	}

	encoded, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	return string(encoded)
}

func compareRenders(file string, base, current []byte) ([]violation, error) {
	oldObjects, err := parseObjects(base)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s at base: %w", file, err)
	}

	newObjects, err := parseObjects(current)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", file, err)
	}

	ids := make([]string, 0, len(oldObjects))
	for id := range oldObjects {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var violations []violation
	for _, id := range ids {
		for _, v := range compareObject(oldObjects[id], newObjects[id]) {
			v.File = file
			violations = append(violations, v)
		}
	}

	return violations, nil
}

func parseObjects(data []byte) (map[string]object, error) {
	objects := map[string]object{}

	reader := utilyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))
	for {
		doc, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		var raw map[string]any
		if err := yaml.Unmarshal(doc, &raw); err != nil {
			return nil, err
		}

		kind, _ := raw["kind"].(string)
		if kind == "" {
			continue
		}

		if _, isHook := nestedMap(raw, "metadata", "annotations")[helmHookKey]; isHook {
			continue
		}

		name, _ := nested(raw, "metadata", "name").(string)
		namespace, _ := nested(raw, "metadata", "namespace").(string)

		id := kind + "/" + name
		if namespace != "" {
			id = kind + "/" + namespace + "/" + name
		}

		objects[id] = object{ID: id, Kind: kind, Raw: raw}
	}

	return objects, nil
}

func compareObject(old object, current object) []violation {
	removed := current.Raw == nil

	var violations []violation
	field := func(path ...string) {
		if removed {
			return
		}
		oldValue := nested(old.Raw, path...)
		newValue := nested(current.Raw, path...)
		if !reflect.DeepEqual(oldValue, newValue) {
			violations = append(violations, violation{Object: old.ID, Field: strings.Join(path, "."), Old: oldValue, New: newValue})
		}
	}
	removal := func() {
		if removed {
			violations = append(violations, violation{Object: old.ID, Field: removedField, Old: "present", New: nil})
		}
	}

	switch old.Kind {
	case "StatefulSet":
		removal()
		field("spec", "selector")
		field("spec", "serviceName")
		field("spec", "podManagementPolicy")
		if !removed {
			violations = append(violations, compareVolumeClaimTemplates(old, current)...)
		}

	case "Deployment", "DaemonSet":
		field("spec", "selector")

	case "Service":
		if removed {
			break
		}
		oldIP, _ := nested(old.Raw, "spec", "clusterIP").(string)
		newIP, _ := nested(current.Raw, "spec", "clusterIP").(string)
		if oldIP != "" && newIP != "" && oldIP != newIP {
			violations = append(violations, violation{Object: old.ID, Field: "spec.clusterIP", Old: oldIP, New: newIP})
		}

	case "PersistentVolumeClaim":
		removal()
		field("spec", "storageClassName")
		field("spec", "accessModes")

	case "Job":
		field("spec", "template")
	}

	return violations
}

type claimTemplate struct {
	StorageClassName any    `json:"storageClassName"`
	AccessModes      any    `json:"accessModes"`
	Storage          string `json:"storage"`

	amount string
}

func compareVolumeClaimTemplates(old object, current object) []violation {
	oldTemplates := claimTemplates(old.Raw)
	newTemplates := claimTemplates(current.Raw)

	names := map[string]struct{}{}
	for name := range oldTemplates {
		names[name] = struct{}{}
	}
	for name := range newTemplates {
		names[name] = struct{}{}
	}

	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)

	var violations []violation
	for _, name := range sorted {
		oldTemplate, inOld := oldTemplates[name]
		newTemplate, inNew := newTemplates[name]
		if inOld && inNew && oldTemplate.amount == newTemplate.amount &&
			reflect.DeepEqual(oldTemplate.StorageClassName, newTemplate.StorageClassName) &&
			reflect.DeepEqual(oldTemplate.AccessModes, newTemplate.AccessModes) {
			continue
		}

		v := violation{Object: old.ID, Field: fmt.Sprintf("spec.volumeClaimTemplates[%s]", name)}
		if inOld {
			v.Old = oldTemplate
		}
		if inNew {
			v.New = newTemplate
		}
		violations = append(violations, v)
	}

	return violations
}

func claimTemplates(raw map[string]any) map[string]claimTemplate {
	result := map[string]claimTemplate{}

	list, _ := nested(raw, "spec", "volumeClaimTemplates").([]any)
	for _, item := range list {
		template, ok := item.(map[string]any)
		if !ok {
			continue
		}

		name, _ := nested(template, "metadata", "name").(string)
		var storage string
		switch value := nested(template, "spec", "resources", "requests", "storage").(type) {
		case string:
			storage = value
		case float64:
			// a plain YAML number like 10737418240 comes back as float64
			storage = strconv.FormatFloat(value, 'f', -1, 64)
		}
		// keep the original text for printing, compare by the parsed amount so
		// that 10Gi and 10737418240 count as equal
		amount := storage
		if q, err := resource.ParseQuantity(storage); err == nil {
			amount = q.AsDec().String()
		}

		result[name] = claimTemplate{
			StorageClassName: nested(template, "spec", "storageClassName"),
			AccessModes:      nested(template, "spec", "accessModes"),
			Storage:          storage,
			amount:           amount,
		}
	}

	return result
}

func nested(obj map[string]any, path ...string) any {
	var current any = obj
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}

	return current
}

func nestedMap(obj map[string]any, path ...string) map[string]any {
	m, _ := nested(obj, path...).(map[string]any)
	return m
}
