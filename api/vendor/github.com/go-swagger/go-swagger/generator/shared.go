// Copyright 2015 go-swagger maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/go-openapi/analysis"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
)

//go:generate go-bindata -mode 420 -modtime 1482416923 -pkg=generator -ignore=.*\.sw? -ignore=.*\.md ./templates/...

const (
	// default generation targets structure
	defaultModelsTarget     = "models"
	defaultServerTarget     = "restapi"
	defaultClientTarget     = "client"
	defaultOperationsTarget = "operations"
	defaultClientName       = "rest"
	defaultServerName       = "swagger"
	defaultScheme           = "http"
)

func init() {
	// all initializations for the generator package
	debugOptions()
	initLanguage()
	initTemplateRepo()
	initTypes()
}

// DefaultSectionOpts for a given opts, this is used when no config file is passed
// and uses the embedded templates when no local override can be found
func DefaultSectionOpts(gen *GenOpts) {
	sec := gen.Sections
	if len(sec.Models) == 0 {
		sec.Models = []TemplateOpts{
			{
				Name:     "definition",
				Source:   "asset:model",
				Target:   "{{ joinFilePath .Target (toPackagePath .ModelPackage) }}",
				FileName: "{{ (snakize (pascalize .Name)) }}.go",
			},
		}
	}

	if len(sec.Operations) == 0 {
		if gen.IsClient {
			sec.Operations = []TemplateOpts{
				{
					Name:     "parameters",
					Source:   "asset:clientParameter",
					Target:   "{{ joinFilePath .Target (toPackagePath .ClientPackage) (toPackagePath .Package) }}",
					FileName: "{{ (snakize (pascalize .Name)) }}_parameters.go",
				},
				{
					Name:     "responses",
					Source:   "asset:clientResponse",
					Target:   "{{ joinFilePath .Target (toPackagePath .ClientPackage) (toPackagePath .Package) }}",
					FileName: "{{ (snakize (pascalize .Name)) }}_responses.go",
				},
			}
		} else {
			ops := []TemplateOpts{}
			if gen.IncludeParameters {
				ops = append(ops, TemplateOpts{
					Name:     "parameters",
					Source:   "asset:serverParameter",
					Target:   "{{ if .UseTags }}{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .APIPackage) (toPackagePath .Package)  }}{{ else }}{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .Package) }}{{ end }}",
					FileName: "{{ (snakize (pascalize .Name)) }}_parameters.go",
				})
			}
			if gen.IncludeURLBuilder {
				ops = append(ops, TemplateOpts{
					Name:     "urlbuilder",
					Source:   "asset:serverUrlbuilder",
					Target:   "{{ if .UseTags }}{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .APIPackage) (toPackagePath .Package) }}{{ else }}{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .Package) }}{{ end }}",
					FileName: "{{ (snakize (pascalize .Name)) }}_urlbuilder.go",
				})
			}
			if gen.IncludeResponses {
				ops = append(ops, TemplateOpts{
					Name:     "responses",
					Source:   "asset:serverResponses",
					Target:   "{{ if .UseTags }}{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .APIPackage) (toPackagePath .Package) }}{{ else }}{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .Package) }}{{ end }}",
					FileName: "{{ (snakize (pascalize .Name)) }}_responses.go",
				})
			}
			if gen.IncludeHandler {
				ops = append(ops, TemplateOpts{
					Name:     "handler",
					Source:   "asset:serverOperation",
					Target:   "{{ if .UseTags }}{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .APIPackage) (toPackagePath .Package) }}{{ else }}{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .Package) }}{{ end }}",
					FileName: "{{ (snakize (pascalize .Name)) }}.go",
				})
			}
			sec.Operations = ops
		}
	}

	if len(sec.OperationGroups) == 0 {
		if gen.IsClient {
			sec.OperationGroups = []TemplateOpts{
				{
					Name:     "client",
					Source:   "asset:clientClient",
					Target:   "{{ joinFilePath .Target (toPackagePath .ClientPackage) (toPackagePath .Name)}}",
					FileName: "{{ (snakize (pascalize .Name)) }}_client.go",
				},
			}
		} else {
			sec.OperationGroups = []TemplateOpts{}
		}
	}

	if len(sec.Application) == 0 {
		if gen.IsClient {
			sec.Application = []TemplateOpts{
				{
					Name:     "facade",
					Source:   "asset:clientFacade",
					Target:   "{{ joinFilePath .Target (toPackagePath .ClientPackage) }}",
					FileName: "{{ snakize .Name }}Client.go",
				},
			}
		} else {
			sec.Application = []TemplateOpts{
				{
					Name:       "configure",
					Source:     "asset:serverConfigureapi",
					Target:     "{{ joinFilePath .Target (toPackagePath .ServerPackage) }}",
					FileName:   "configure_{{ (snakize (pascalize .Name)) }}.go",
					SkipExists: !gen.RegenerateConfigureAPI,
				},
				{
					Name:     "main",
					Source:   "asset:serverMain",
					Target:   "{{ joinFilePath .Target \"cmd\" .MainPackage }}",
					FileName: "main.go",
				},
				{
					Name:     "embedded_spec",
					Source:   "asset:swaggerJsonEmbed",
					Target:   "{{ joinFilePath .Target (toPackagePath .ServerPackage) }}",
					FileName: "embedded_spec.go",
				},
				{
					Name:     "server",
					Source:   "asset:serverServer",
					Target:   "{{ joinFilePath .Target (toPackagePath .ServerPackage) }}",
					FileName: "server.go",
				},
				{
					Name:     "builder",
					Source:   "asset:serverBuilder",
					Target:   "{{ joinFilePath .Target (toPackagePath .ServerPackage) (toPackagePath .APIPackage) }}",
					FileName: "{{ snakize (pascalize .Name) }}_api.go",
				},
				{
					Name:     "doc",
					Source:   "asset:serverDoc",
					Target:   "{{ joinFilePath .Target (toPackagePath .ServerPackage) }}",
					FileName: "doc.go",
				},
			}
		}
	}
	gen.Sections = sec

}

// TemplateOpts allows for codegen customization
type TemplateOpts struct {
	Name       string `mapstructure:"name"`
	Source     string `mapstructure:"source"`
	Target     string `mapstructure:"target"`
	FileName   string `mapstructure:"file_name"`
	SkipExists bool   `mapstructure:"skip_exists"`
	SkipFormat bool   `mapstructure:"skip_format"`
}

// SectionOpts allows for specifying options to customize the templates used for generation
type SectionOpts struct {
	Application     []TemplateOpts `mapstructure:"application"`
	Operations      []TemplateOpts `mapstructure:"operations"`
	OperationGroups []TemplateOpts `mapstructure:"operation_groups"`
	Models          []TemplateOpts `mapstructure:"models"`
}

// GenOpts the options for the generator
type GenOpts struct {
	IncludeModel               bool
	IncludeValidator           bool
	IncludeHandler             bool
	IncludeParameters          bool
	IncludeResponses           bool
	IncludeURLBuilder          bool
	IncludeMain                bool
	IncludeSupport             bool
	ExcludeSpec                bool
	DumpData                   bool
	ValidateSpec               bool
	FlattenOpts                *analysis.FlattenOpts
	IsClient                   bool
	defaultsEnsured            bool
	PropertiesSpecOrder        bool
	StrictAdditionalProperties bool
	AllowTemplateOverride      bool

	Spec                   string
	APIPackage             string
	ModelPackage           string
	ServerPackage          string
	ClientPackage          string
	Principal              string
	Target                 string
	Sections               SectionOpts
	LanguageOpts           *LanguageOpts
	TypeMapping            map[string]string
	Imports                map[string]string
	DefaultScheme          string
	DefaultProduces        string
	DefaultConsumes        string
	TemplateDir            string
	Template               string
	RegenerateConfigureAPI bool
	Operations             []string
	Models                 []string
	Tags                   []string
	Name                   string
	FlagStrategy           string
	CompatibilityMode      string
	ExistingModels         string
	Copyright              string
	SkipTagPackages        bool
	MainPackage            string
}

// CheckOpts carries out some global consistency checks on options.
func (g *GenOpts) CheckOpts() error {
	if g == nil {
		return errors.New("gen opts are required")
	}

	if !filepath.IsAbs(g.Target) {
		if _, err := filepath.Abs(g.Target); err != nil {
			return fmt.Errorf("could not locate target %s: %v", g.Target, err)
		}
	}

	if filepath.IsAbs(g.ServerPackage) {
		return fmt.Errorf("you shouldn't specify an absolute path in --server-package: %s", g.ServerPackage)
	}

	if strings.HasPrefix(g.Spec, "http://") || strings.HasPrefix(g.Spec, "https://") {
		return nil
	}

	pth, err := findSwaggerSpec(g.Spec)
	if err != nil {
		return err
	}

	// ensure spec path is absolute
	g.Spec, err = filepath.Abs(pth)
	if err != nil {
		return fmt.Errorf("could not locate spec: %s", g.Spec)
	}

	return nil
}

// TargetPath returns the target generation path relative to the server package.
// This method is used by templates, e.g. with {{ .TargetPath }}
//
// Errors cases are prevented by calling CheckOpts beforehand.
//
// Example:
// Target: ${PWD}/tmp
// ServerPackage: abc/efg
//
// Server is generated in ${PWD}/tmp/abc/efg
// relative TargetPath returned: ../../../tmp
//
func (g *GenOpts) TargetPath() string {
	var tgt string
	if g.Target == "" {
		tgt = "." // That's for windows
	} else {
		tgt = g.Target
	}
	tgtAbs, _ := filepath.Abs(tgt)
	srvPkg := filepath.FromSlash(g.LanguageOpts.ManglePackagePath(g.ServerPackage, "server"))
	srvrAbs := filepath.Join(tgtAbs, srvPkg)
	tgtRel, _ := filepath.Rel(srvrAbs, filepath.Dir(tgtAbs))
	tgtRel = filepath.Join(tgtRel, filepath.Base(tgtAbs))
	return tgtRel
}

// SpecPath returns the path to the spec relative to the server package.
// If the spec is remote keep this absolute location.
//
// If spec is not relative to server (e.g. lives on a different drive on windows),
// then the resolved path is absolute.
//
// This method is used by templates, e.g. with {{ .SpecPath }}
//
// Errors cases are prevented by calling CheckOpts beforehand.
func (g *GenOpts) SpecPath() string {
	if strings.HasPrefix(g.Spec, "http://") || strings.HasPrefix(g.Spec, "https://") {
		return g.Spec
	}
	// Local specifications
	specAbs, _ := filepath.Abs(g.Spec)
	var tgt string
	if g.Target == "" {
		tgt = "." // That's for windows
	} else {
		tgt = g.Target
	}
	tgtAbs, _ := filepath.Abs(tgt)
	srvPkg := filepath.FromSlash(g.LanguageOpts.ManglePackagePath(g.ServerPackage, "server"))
	srvAbs := filepath.Join(tgtAbs, srvPkg)
	specRel, err := filepath.Rel(srvAbs, specAbs)
	if err != nil {
		return specAbs
	}
	return specRel
}

// EnsureDefaults for these gen opts
func (g *GenOpts) EnsureDefaults() error {
	if g.defaultsEnsured {
		return nil
	}

	if g.LanguageOpts == nil {
		g.LanguageOpts = DefaultLanguageFunc()
	}

	DefaultSectionOpts(g)

	// set defaults for flattening options
	if g.FlattenOpts == nil {
		g.FlattenOpts = &analysis.FlattenOpts{
			Minimal:      true,
			Verbose:      true,
			RemoveUnused: false,
			Expand:       false,
		}
	}

	if g.DefaultScheme == "" {
		g.DefaultScheme = defaultScheme
	}

	if g.DefaultConsumes == "" {
		g.DefaultConsumes = runtime.JSONMime
	}

	if g.DefaultProduces == "" {
		g.DefaultProduces = runtime.JSONMime
	}

	// always include validator with models
	g.IncludeValidator = g.IncludeModel

	g.defaultsEnsured = true
	return nil
}

func (g *GenOpts) location(t *TemplateOpts, data interface{}) (string, string, error) {
	v := reflect.Indirect(reflect.ValueOf(data))
	fld := v.FieldByName("Name")
	var name string
	if fld.IsValid() {
		log.Println("name field", fld.String())
		name = fld.String()
	}

	fldpack := v.FieldByName("Package")
	pkg := g.APIPackage
	if fldpack.IsValid() {
		log.Println("package field", fldpack.String())
		pkg = fldpack.String()
	}

	var tags []string
	tagsF := v.FieldByName("Tags")
	if tagsF.IsValid() {
		tags = tagsF.Interface().([]string)
	}

	var useTags bool
	useTagsF := v.FieldByName("UseTags")
	if useTagsF.IsValid() {
		useTags = useTagsF.Interface().(bool)
	}

	funcMap := FuncMapFunc(g.LanguageOpts)

	pthTpl, err := template.New(t.Name + "-target").Funcs(funcMap).Parse(t.Target)
	if err != nil {
		return "", "", err
	}

	fNameTpl, err := template.New(t.Name + "-filename").Funcs(funcMap).Parse(t.FileName)
	if err != nil {
		return "", "", err
	}

	d := struct {
		Name, Package, APIPackage, ServerPackage, ClientPackage, ModelPackage, MainPackage, Target string
		Tags                                                                                       []string
		UseTags                                                                                    bool
		Context                                                                                    interface{}
	}{
		Name:          name,
		Package:       pkg,
		APIPackage:    g.APIPackage,
		ServerPackage: g.ServerPackage,
		ClientPackage: g.ClientPackage,
		ModelPackage:  g.ModelPackage,
		MainPackage:   g.MainPackage,
		Target:        g.Target,
		Tags:          tags,
		UseTags:       useTags,
		Context:       data,
	}

	var pthBuf bytes.Buffer
	if e := pthTpl.Execute(&pthBuf, d); e != nil {
		return "", "", e
	}

	var fNameBuf bytes.Buffer
	if e := fNameTpl.Execute(&fNameBuf, d); e != nil {
		return "", "", e
	}
	return pthBuf.String(), fileName(fNameBuf.String()), nil
}

func (g *GenOpts) render(t *TemplateOpts, data interface{}) ([]byte, error) {
	var templ *template.Template

	if strings.HasPrefix(strings.ToLower(t.Source), "asset:") {
		tt, err := templates.Get(strings.TrimPrefix(t.Source, "asset:"))
		if err != nil {
			return nil, err
		}
		templ = tt
	}

	if templ == nil {
		// try to load from repository (and enable dependencies)
		name := swag.ToJSONName(strings.TrimSuffix(t.Source, ".gotmpl"))
		tt, err := templates.Get(name)
		if err == nil {
			templ = tt
		}
	}

	if templ == nil {
		// try to load template from disk, in TemplateDir if specified
		// (dependencies resolution is limited to preloaded assets)
		var templateFile string
		if g.TemplateDir != "" {
			templateFile = filepath.Join(g.TemplateDir, t.Source)
		} else {
			templateFile = t.Source
		}
		content, err := ioutil.ReadFile(templateFile)
		if err != nil {
			return nil, fmt.Errorf("error while opening %s template file: %v", templateFile, err)
		}
		tt, err := template.New(t.Source).Funcs(FuncMapFunc(g.LanguageOpts)).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("template parsing failed on template %s: %v", t.Name, err)
		}
		templ = tt
	}

	if templ == nil {
		return nil, fmt.Errorf("template %q not found", t.Source)
	}

	var tBuf bytes.Buffer
	if err := templ.Execute(&tBuf, data); err != nil {
		return nil, fmt.Errorf("template execution failed for template %s: %v", t.Name, err)
	}
	log.Printf("executed template %s", t.Source)

	return tBuf.Bytes(), nil
}

// Render template and write generated source code
// generated code is reformatted ("linted"), which gives an
// additional level of checking. If this step fails, the generated
// code is still dumped, for template debugging purposes.
func (g *GenOpts) write(t *TemplateOpts, data interface{}) error {
	dir, fname, err := g.location(t, data)
	if err != nil {
		return fmt.Errorf("failed to resolve template location for template %s: %v", t.Name, err)
	}

	if t.SkipExists && fileExists(dir, fname) {
		debugLog("skipping generation of %s because it already exists and skip_exist directive is set for %s",
			filepath.Join(dir, fname), t.Name)
		return nil
	}

	log.Printf("creating generated file %q in %q as %s", fname, dir, t.Name)
	content, err := g.render(t, data)
	if err != nil {
		return fmt.Errorf("failed rendering template data for %s: %v", t.Name, err)
	}

	if dir != "" {
		_, exists := os.Stat(dir)
		if os.IsNotExist(exists) {
			debugLog("creating directory %q for \"%s\"", dir, t.Name)
			// Directory settings consistent with file privileges.
			// Environment's umask may alter this setup
			if e := os.MkdirAll(dir, 0755); e != nil {
				return e
			}
		}
	}

	// Conditionally format the code, unless the user wants to skip
	formatted := content
	var writeerr error

	if !t.SkipFormat {
		formatted, err = g.LanguageOpts.FormatContent(fname, content)
		if err != nil {
			log.Printf("source formatting failed on template-generated source (%q for %s). Check that your template produces valid code", filepath.Join(dir, fname), t.Name)
			writeerr = ioutil.WriteFile(filepath.Join(dir, fname), content, 0644)
			if writeerr != nil {
				return fmt.Errorf("failed to write (unformatted) file %q in %q: %v", fname, dir, writeerr)
			}
			log.Printf("unformatted generated source %q has been dumped for template debugging purposes. DO NOT build on this source!", fname)
			return fmt.Errorf("source formatting on generated source %q failed: %v", t.Name, err)
		}
	}

	writeerr = ioutil.WriteFile(filepath.Join(dir, fname), formatted, 0644)
	if writeerr != nil {
		return fmt.Errorf("failed to write file %q in %q: %v", fname, dir, writeerr)
	}
	return err
}

func fileName(in string) string {
	ext := filepath.Ext(in)
	return swag.ToFileName(strings.TrimSuffix(in, ext)) + ext
}

func (g *GenOpts) shouldRenderApp(t *TemplateOpts, app *GenApp) bool {
	switch swag.ToFileName(swag.ToGoName(t.Name)) {
	case "main":
		return g.IncludeMain
	case "embedded_spec":
		return !g.ExcludeSpec
	default:
		return true
	}
}

func (g *GenOpts) shouldRenderOperations() bool {
	return g.IncludeHandler || g.IncludeParameters || g.IncludeResponses
}

func (g *GenOpts) renderApplication(app *GenApp) error {
	log.Printf("rendering %d templates for application %s", len(g.Sections.Application), app.Name)
	for _, templ := range g.Sections.Application {
		if !g.shouldRenderApp(&templ, app) {
			continue
		}
		if err := g.write(&templ, app); err != nil {
			return err
		}
	}
	return nil
}

func (g *GenOpts) renderOperationGroup(gg *GenOperationGroup) error {
	log.Printf("rendering %d templates for operation group %s", len(g.Sections.OperationGroups), g.Name)
	for _, templ := range g.Sections.OperationGroups {
		if !g.shouldRenderOperations() {
			continue
		}

		if err := g.write(&templ, gg); err != nil {
			return err
		}
	}
	return nil
}

func (g *GenOpts) renderOperation(gg *GenOperation) error {
	log.Printf("rendering %d templates for operation %s", len(g.Sections.Operations), g.Name)
	for _, templ := range g.Sections.Operations {
		if !g.shouldRenderOperations() {
			continue
		}

		if err := g.write(&templ, gg); err != nil {
			return err
		}
	}
	return nil
}

func (g *GenOpts) renderDefinition(gg *GenDefinition) error {
	log.Printf("rendering %d templates for model %s", len(g.Sections.Models), gg.Name)
	for _, templ := range g.Sections.Models {
		if !g.IncludeModel {
			continue
		}

		if err := g.write(&templ, gg); err != nil {
			return err
		}
	}
	return nil
}

func (g *GenOpts) setTemplates() error {
	templates.LoadDefaults()

	if g.Template != "" {
		// set contrib templates
		if err := templates.LoadContrib(g.Template); err != nil {
			return err
		}
	}

	templates.SetAllowOverride(g.AllowTemplateOverride)

	if g.TemplateDir != "" {
		// set custom templates
		if err := templates.LoadDir(g.TemplateDir); err != nil {
			return err
		}
	}
	return nil
}

// defaultImports produces a default map for imports with models
func (g *GenOpts) defaultImports() map[string]string {
	baseImport := g.LanguageOpts.baseImport(g.Target)
	defaultImports := make(map[string]string, 50)

	if g.ExistingModels == "" {
		importPath := path.Join(
			baseImport,
			g.LanguageOpts.ManglePackagePath(g.ModelPackage, defaultModelsTarget))
		defaultImports[g.LanguageOpts.ManglePackageName(g.ModelPackage, defaultModelsTarget)] = importPath
	} else {
		// TODO(fredbi): mangle existing model pkg aliases
		importPath := g.LanguageOpts.ManglePackagePath(g.ExistingModels, "")
		defaultImports[importAlias(importPath)] = importPath
	}
	return defaultImports
}

// initImports produces a default map for import with the specified root for operations
func (g *GenOpts) initImports(operationsPackage string) map[string]string {
	baseImport := g.LanguageOpts.baseImport(g.Target)

	imports := make(map[string]string, 50)
	imports[g.LanguageOpts.ManglePackageName(operationsPackage, defaultOperationsTarget)] = path.Join(
		baseImport,
		g.LanguageOpts.ManglePackagePath(operationsPackage, defaultOperationsTarget))
	return imports
}

func fileExists(target, name string) bool {
	_, err := os.Stat(filepath.Join(target, name))
	return !os.IsNotExist(err)
}

func gatherModels(specDoc *loads.Document, modelNames []string) (map[string]spec.Schema, error) {
	models, mnc := make(map[string]spec.Schema), len(modelNames)
	defs := specDoc.Spec().Definitions

	if mnc > 0 {
		var unknownModels []string
		for _, m := range modelNames {
			_, ok := defs[m]
			if !ok {
				unknownModels = append(unknownModels, m)
			}
		}
		if len(unknownModels) != 0 {
			return nil, fmt.Errorf("unknown models: %s", strings.Join(unknownModels, ", "))
		}
	}
	for k, v := range defs {
		if mnc == 0 {
			models[k] = v
		}
		for _, nm := range modelNames {
			if k == nm {
				models[k] = v
			}
		}
	}
	return models, nil
}

// titleOrDefault infers a name for the app from the title of the spec
func titleOrDefault(specDoc *loads.Document, name, defaultName string) string {
	if strings.TrimSpace(name) == "" {
		if specDoc.Spec().Info != nil && strings.TrimSpace(specDoc.Spec().Info.Title) != "" {
			name = specDoc.Spec().Info.Title
		} else {
			name = defaultName
		}
	}
	return swag.ToGoName(name)
}

func mainNameOrDefault(specDoc *loads.Document, name, defaultName string) string {
	// _test won't do as main server name
	return strings.TrimSuffix(titleOrDefault(specDoc, name, defaultName), "Test")
}

func appNameOrDefault(specDoc *loads.Document, name, defaultName string) string {
	// _test_api, _api_test, _test, _api won't do as app names
	return strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(titleOrDefault(specDoc, name, defaultName), "Test"), "API"), "Test")
}

type opRef struct {
	Method string
	Path   string
	Key    string
	ID     string
	Op     *spec.Operation
}

type opRefs []opRef

func (o opRefs) Len() int           { return len(o) }
func (o opRefs) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o opRefs) Less(i, j int) bool { return o[i].Key < o[j].Key }

func gatherOperations(specDoc *analysis.Spec, operationIDs []string) map[string]opRef {
	var oprefs opRefs

	for method, pathItem := range specDoc.Operations() {
		for path, operation := range pathItem {
			// nm := ensureUniqueName(operation.ID, method, path, operations)
			vv := *operation
			oprefs = append(oprefs, opRef{
				Key:    swag.ToGoName(strings.ToLower(method) + " " + path),
				Method: method,
				Path:   path,
				ID:     vv.ID,
				Op:     &vv,
			})
		}
	}

	sort.Sort(oprefs)

	operations := make(map[string]opRef)
	for _, opr := range oprefs {
		nm := opr.ID
		if nm == "" {
			nm = opr.Key
		}

		oo, found := operations[nm]
		if found && oo.Method != opr.Method && oo.Path != opr.Path {
			nm = opr.Key
		}
		if len(operationIDs) == 0 || swag.ContainsStrings(operationIDs, opr.ID) || swag.ContainsStrings(operationIDs, nm) {
			opr.ID = nm
			opr.Op.ID = nm
			operations[nm] = opr
		}
	}

	return operations
}

func pruneEmpty(in []string) (out []string) {
	for _, v := range in {
		if v != "" {
			out = append(out, v)
		}
	}
	return
}

func trimBOM(in string) string {
	return strings.Trim(in, "\xef\xbb\xbf")
}

// gatherSecuritySchemes produces a sorted representation from a map of spec security schemes
func gatherSecuritySchemes(securitySchemes map[string]spec.SecurityScheme, appName, principal, receiver string) (security GenSecuritySchemes) {
	for scheme, req := range securitySchemes {
		isOAuth2 := strings.ToLower(req.Type) == "oauth2"
		var scopes []string
		if isOAuth2 {
			for k := range req.Scopes {
				scopes = append(scopes, k)
			}
		}
		sort.Strings(scopes)

		security = append(security, GenSecurityScheme{
			AppName:      appName,
			ID:           scheme,
			ReceiverName: receiver,
			Name:         req.Name,
			IsBasicAuth:  strings.ToLower(req.Type) == "basic",
			IsAPIKeyAuth: strings.ToLower(req.Type) == "apikey",
			IsOAuth2:     isOAuth2,
			Scopes:       scopes,
			Principal:    principal,
			Source:       req.In,
			// from original spec
			Description:      req.Description,
			Type:             strings.ToLower(req.Type),
			In:               req.In,
			Flow:             req.Flow,
			AuthorizationURL: req.AuthorizationURL,
			TokenURL:         req.TokenURL,
			Extensions:       req.Extensions,
		})
	}
	sort.Sort(security)
	return
}

// gatherExtraSchemas produces a sorted list of extra schemas.
//
// ExtraSchemas are inlined types rendered in the same model file.
func gatherExtraSchemas(extraMap map[string]GenSchema) (extras GenSchemaList) {
	var extraKeys []string
	for k := range extraMap {
		extraKeys = append(extraKeys, k)
	}
	sort.Strings(extraKeys)
	for _, k := range extraKeys {
		// figure out if top level validations are needed
		p := extraMap[k]
		p.HasValidations = shallowValidationLookup(p)
		extras = append(extras, p)
	}
	return
}

func sharedValidationsFromSimple(v spec.CommonValidations, isRequired bool) (sh sharedValidations) {
	sh = sharedValidations{
		Required:         isRequired,
		Maximum:          v.Maximum,
		ExclusiveMaximum: v.ExclusiveMaximum,
		Minimum:          v.Minimum,
		ExclusiveMinimum: v.ExclusiveMinimum,
		MaxLength:        v.MaxLength,
		MinLength:        v.MinLength,
		Pattern:          v.Pattern,
		MaxItems:         v.MaxItems,
		MinItems:         v.MinItems,
		UniqueItems:      v.UniqueItems,
		MultipleOf:       v.MultipleOf,
		Enum:             v.Enum,
	}
	return
}

func sharedValidationsFromSchema(v spec.Schema, isRequired bool) (sh sharedValidations) {
	sh = sharedValidations{
		Required:         isRequired,
		Maximum:          v.Maximum,
		ExclusiveMaximum: v.ExclusiveMaximum,
		Minimum:          v.Minimum,
		ExclusiveMinimum: v.ExclusiveMinimum,
		MaxLength:        v.MaxLength,
		MinLength:        v.MinLength,
		Pattern:          v.Pattern,
		MaxItems:         v.MaxItems,
		MinItems:         v.MinItems,
		UniqueItems:      v.UniqueItems,
		MultipleOf:       v.MultipleOf,
		Enum:             v.Enum,
	}
	return
}

func dumpData(data interface{}) error {
	bb, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(bb))
	return nil
}

func importAlias(pkg string) string {
	_, k := path.Split(pkg)
	return k
}
