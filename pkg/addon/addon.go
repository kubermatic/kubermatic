/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package addon

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"github.com/Masterminds/sprig/v3"

	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/util/yaml"

	"k8s.io/apimachinery/pkg/runtime"
)

// Addon is a single addon loaded from a single directory. Within this
// directory usually all .yaml files will have been loaded and concatenated
// into a single large template to render.
type Addon struct {
	combinedTemplate *template.Template
	lock             *sync.Mutex
}

func addonFunctions(overwriteRegistry string) template.FuncMap {
	funcs := sprig.TxtFuncMap()
	// Registry is deprecated and should not be used anymore.
	funcs["Registry"] = registry.GetOverwriteFunc(overwriteRegistry)
	funcs["Image"] = registry.GetImageRewriterFunc(overwriteRegistry)
	funcs["join"] = strings.Join
	funcs["semverCompare"] = semverCompare

	return funcs
}

// semverCompare checks if a given version matches the given constraint.
// Both version and constraint can be either strings or already parsed semver
// objects. If parsing fails or an incompatible type is given, the function
// will silently return false to aid in using it in Go templates.
func semverCompare(version any, constraint any) bool {
	if sver, ok := version.(string); ok {
		parsed, err := semver.NewVersion(sver)
		if err != nil {
			return false
		}

		return semverCompare(parsed, constraint)
	}

	if scon, ok := constraint.(string); ok {
		parsed, err := semver.NewConstraint(scon)
		if err != nil {
			return false
		}

		return semverCompare(version, parsed)
	}

	ver, ok := version.(*semver.Version)
	if !ok {
		return false
	}

	con, ok := constraint.(*semver.Constraints)
	if !ok {
		return false
	}

	return con.Check(ver)
}

func listAddonFilesRecursively(root string) ([]string, error) {
	files := []string{}

	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func LoadAddonsFromDirectory(directory string) (map[string]*Addon, error) {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}

	fileInfos, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	result := map[string]*Addon{}
	for _, info := range fileInfos {
		if !info.IsDir() {
			continue
		}

		addonName := info.Name()

		addon, err := LoadAddonFromDirectory(filepath.Join(directory, addonName))
		if err != nil {
			return nil, fmt.Errorf("failed to load %q addon: %w", addonName, err)
		}

		result[addonName] = addon
	}

	return result, nil
}

func LoadAddonFromDirectory(directory string) (*Addon, error) {
	allAddonFiles, err := listAddonFilesRecursively(directory)
	if err != nil {
		return nil, err
	}

	// to provide more helpful error messages, we parse each addon file individually,
	// even though later during runtime we deal with one big combined manifest
	combined := strings.Builder{}
	parser := template.New("temp").Funcs(addonFunctions("WILL_BE_INJECTED_LATER"))

	for _, path := range allAddonFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", path, err)
		}

		if _, err := parser.Parse(string(content)); err != nil {
			return nil, fmt.Errorf("failed to parse file %s: %w", path, err)
		}

		combined.Write(content)
		combined.WriteString("\n\n---\n\n")
	}

	// parse the big combined manifest
	parsed, err := parser.Parse(combined.String())
	if err != nil {
		return nil, err
	}

	return &Addon{
		combinedTemplate: parsed,
		lock:             &sync.Mutex{},
	}, nil
}

func (a *Addon) Render(overwriteRegistry string, data *TemplateData) ([]runtime.RawExtension, error) {
	// Now that we know the overwrite registry (which can be cluster-dependent), we have
	// to update the Registry/Image funcs, which rely on the registry name to be injected
	// via a closure. Since both functions are part of the "addon interface" for addon
	// authors, their signature cannot simply be changed to work based on the TemplateData.
	// Because of this switcheroo we must ensure that this addon is not rendered in parallel
	// by multiple goroutines.
	a.lock.Lock()
	defer a.lock.Unlock()

	a.combinedTemplate.Funcs(addonFunctions(overwriteRegistry))

	var buffer bytes.Buffer
	if err := a.combinedTemplate.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	sd := strings.TrimSpace(buffer.String())
	if len(sd) == 0 {
		return nil, nil
	}

	manifests, err := yaml.ParseMultipleDocuments(&buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to decode templated YAML: %w", err)
	}

	return manifests, nil
}
