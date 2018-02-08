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
	"log"
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"sort"
	"strings"

	"github.com/go-openapi/analysis"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/swag"
)

// GenerateServer generates a server application
func GenerateServer(name string, modelNames, operationIDs []string, opts *GenOpts) error {
	generator, err := newAppGenerator(name, modelNames, operationIDs, opts)
	if err != nil {
		return err
	}
	return generator.Generate()
}

// GenerateSupport generates the supporting files for an API
func GenerateSupport(name string, modelNames, operationIDs []string, opts *GenOpts) error {

	generator, err := newAppGenerator(name, modelNames, operationIDs, opts)
	if err != nil {
		return err
	}
	return generator.GenerateSupport(nil)
}

func newAppGenerator(name string, modelNames, operationIDs []string, opts *GenOpts) (*appGenerator, error) {
	if opts == nil {
		return nil, errors.New("gen opts are required")
	}
	if err := opts.EnsureDefaults(false); err != nil {
		return nil, err
	}

	if opts.TemplateDir != "" {
		if err := templates.LoadDir(opts.TemplateDir); err != nil {
			return nil, err
		}
	}

	// Load the spec
	var err error
	var specDoc *loads.Document
	opts.Spec, specDoc, err = loadSpec(opts.Spec)
	if err != nil {
		return nil, err
	}

	// Validate and Expand. specDoc is in/out param.
	specDoc, err = validateAndFlattenSpec(opts, specDoc)
	if err != nil {
		return nil, err
	}

	analyzed := analysis.New(specDoc.Spec())

	models, err := gatherModels(specDoc, modelNames)
	if err != nil {
		return nil, err
	}

	operations := gatherOperations(analyzed, operationIDs)
	if len(operations) == 0 {
		return nil, errors.New("no operations were selected")
	}

	defaultScheme := opts.DefaultScheme
	if defaultScheme == "" {
		defaultScheme = "http"
	}

	defaultProduces := opts.DefaultProduces
	if defaultProduces == "" {
		defaultProduces = runtime.JSONMime
	}

	defaultConsumes := opts.DefaultConsumes
	if defaultConsumes == "" {
		defaultConsumes = runtime.JSONMime
	}

	apiPackage := opts.LanguageOpts.MangleName(swag.ToFileName(opts.APIPackage), "api")
	return &appGenerator{
		Name:       appNameOrDefault(specDoc, name, "swagger"),
		Receiver:   "o",
		SpecDoc:    specDoc,
		Analyzed:   analyzed,
		Models:     models,
		Operations: operations,
		Target:     opts.Target,
		// Package:       filepath.Base(opts.Target),
		DumpData:        opts.DumpData,
		Package:         apiPackage,
		APIPackage:      apiPackage,
		ModelsPackage:   opts.LanguageOpts.MangleName(swag.ToFileName(opts.ModelPackage), "definitions"),
		ServerPackage:   opts.LanguageOpts.MangleName(swag.ToFileName(opts.ServerPackage), "server"),
		ClientPackage:   opts.LanguageOpts.MangleName(swag.ToFileName(opts.ClientPackage), "client"),
		Principal:       opts.Principal,
		DefaultScheme:   defaultScheme,
		DefaultProduces: defaultProduces,
		DefaultConsumes: defaultConsumes,
		GenOpts:         opts,
	}, nil
}

type appGenerator struct {
	Name            string
	Receiver        string
	SpecDoc         *loads.Document
	Analyzed        *analysis.Spec
	Package         string
	APIPackage      string
	ModelsPackage   string
	ServerPackage   string
	ClientPackage   string
	Principal       string
	Models          map[string]spec.Schema
	Operations      map[string]opRef
	Target          string
	DumpData        bool
	DefaultScheme   string
	DefaultProduces string
	DefaultConsumes string
	GenOpts         *GenOpts
}

// 1. Checks if the child path and parent path coincide.
// 2. If they do return child path  relative to parent path.
// 3. Everything else return false
func checkPrefixAndFetchRelativePath(childpath string, parentpath string) (bool, string) {

	if strings.HasPrefix(childpath, parentpath) {
		pth, err := filepath.Rel(parentpath, childpath)
		if err != nil {
			log.Fatalln(err)
		}
		return true, pth
	}

	return false, ""

}

func baseImport(tgt string) string {
	tgtAbsPath, err := filepath.Abs(tgt)
	if err != nil {
		log.Fatalln(err)
	}
	var tgtAbsPathExtended string
	tgtAbsPathExtended, err = filepath.EvalSymlinks(tgtAbsPath)
	if err != nil {
		log.Fatalln(err)
	}

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(os.Getenv("HOME"), "go")
	}

	var pth string
	for _, gp := range filepath.SplitList(gopath) {
		// EvalSymLinks also calls the Clean
		gopathExtended, err := filepath.EvalSymlinks(gp)
		if err != nil {
			log.Fatalln(err)
		}
		gopathExtended = filepath.Join(gopathExtended, "src")
		gp = filepath.Join(gp, "src")

		// Windows (local) file systems - NTFS, as well as FAT and variants
		// are case insensitive.
		if goruntime.GOOS == "windows" {
			tgtAbsPath = strings.ToLower(tgtAbsPath)
			tgtAbsPathExtended = strings.ToLower(tgtAbsPathExtended)
			gopathExtended = strings.ToLower(gopathExtended)
			gp = strings.ToLower(gp)
		}

		// At this stage we have expanded and unexpanded target path. GOPATH is fully expanded.
		// Expanded means symlink free.
		// We compare both types of targetpath<s> with gopath.
		// If any one of them coincides with gopath , it is imperative that
		// target path lies inside gopath. How?
		// 		- Case 1: Irrespective of symlinks paths coincide. Both non-expanded paths.
		// 		- Case 2: Symlink in target path points to location inside GOPATH. (Expanded Target Path)
		//    - Case 3: Symlink in target path points to directory outside GOPATH (Unexpanded target path)

		// Case 1: - Do nothing case. If non-expanded paths match just genrate base import path as if
		//				   there are no symlinks.

		// Case 2: - Symlink in target path points to location inside GOPATH. (Expanded Target Path)
		//					 First if will fail. Second if will succeed.

		// Case 3: - Symlink in target path points to directory outside GOPATH (Unexpanded target path)
		// 					 First if will succeed and break.

		//compares non expanded path for both
		if ok, relativepath := checkPrefixAndFetchRelativePath(tgtAbsPath, gp); ok {
			pth = relativepath
			break
		}

		// Compares non-expanded target path
		if ok, relativepath := checkPrefixAndFetchRelativePath(tgtAbsPath, gopathExtended); ok {
			pth = relativepath
			break
		}

		// Compares expanded target path.
		if ok, relativepath := checkPrefixAndFetchRelativePath(tgtAbsPathExtended, gopathExtended); ok {
			pth = relativepath
			break
		}

	}

	if pth == "" {
		log.Fatalln("target must reside inside a location in the $GOPATH/src")
	}
	return pth
}

func (a *appGenerator) Generate() error {

	app, err := a.makeCodegenApp()
	if err != nil {
		return err
	}

	if a.DumpData {
		bb, err := json.MarshalIndent(app, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, string(bb))
		return nil
	}

	// IPC removed concurrent execution because of the FuncMap that is being shared
	// templates are now lazy loaded so there is concurrent map access I can't guard

	// errChan := make(chan error, 100)
	// wg := nsync.NewControlWaitGroup(20)

	if a.GenOpts.IncludeModel {
		log.Printf("rendering %d models", len(app.Models))
		for _, mod := range app.Models {
			// if len(errChan) > 0 {
			// 	wg.Wait()
			// 	return <-errChan
			// }
			modCopy := mod
			// wg.Do(func() {
			modCopy.IncludeValidator = true // a.GenOpts.IncludeValidator
			modCopy.IncludeModel = true
			if err := a.GenOpts.renderDefinition(&modCopy); err != nil {
				return err
			}
			// })
		}
	}
	// wg.Wait()

	if a.GenOpts.IncludeHandler {
		log.Printf("rendering %d operation groups (tags)", app.OperationGroups.Len())
		for _, opg := range app.OperationGroups {
			opgCopy := opg
			log.Printf("rendering %d operations for %s", opg.Operations.Len(), opg.Name)
			for _, op := range opgCopy.Operations {
				// if len(errChan) > 0 {
				// 	wg.Wait()
				// 	return <-errChan
				// }
				opCopy := op
				// wg.Do(func() {

				if err := a.GenOpts.renderOperation(&opCopy); err != nil {
					return err
				}
				// })
			}
		}
	}

	if a.GenOpts.IncludeSupport {
		log.Printf("rendering support")
		// wg.Do(func() {
		if err := a.GenerateSupport(&app); err != nil {
			// errChan <- err
			return err
		}
		// })
	}
	// wg.Wait()
	// if len(errChan) > 0 {
	// 	return <-errChan
	// }
	return nil
}

func (a *appGenerator) GenerateSupport(ap *GenApp) error {
	app := ap
	if ap == nil {
		ca, err := a.makeCodegenApp()
		if err != nil {
			return err
		}
		app = &ca
	}

	importPath := filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage, a.APIPackage))
	app.DefaultImports = append(
		app.DefaultImports,
		filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage)),
		importPath,
	)

	return a.GenOpts.renderApplication(app)
}

var mediaTypeNames = map[*regexp.Regexp]string{
	regexp.MustCompile("application/.*json"):                "json",
	regexp.MustCompile("application/.*yaml"):                "yaml",
	regexp.MustCompile("application/.*protobuf"):            "protobuf",
	regexp.MustCompile("application/.*capnproto"):           "capnproto",
	regexp.MustCompile("application/.*thrift"):              "thrift",
	regexp.MustCompile("(?:application|text)/.*xml"):        "xml",
	regexp.MustCompile("text/.*markdown"):                   "markdown",
	regexp.MustCompile("text/.*html"):                       "html",
	regexp.MustCompile("text/.*csv"):                        "csv",
	regexp.MustCompile("text/.*tsv"):                        "tsv",
	regexp.MustCompile("text/.*javascript"):                 "js",
	regexp.MustCompile("text/.*css"):                        "css",
	regexp.MustCompile("text/.*plain"):                      "txt",
	regexp.MustCompile("application/.*octet-stream"):        "bin",
	regexp.MustCompile("application/.*tar"):                 "tar",
	regexp.MustCompile("application/.*gzip"):                "gzip",
	regexp.MustCompile("application/.*gz"):                  "gzip",
	regexp.MustCompile("application/.*raw-stream"):          "bin",
	regexp.MustCompile("application/x-www-form-urlencoded"): "urlform",
	regexp.MustCompile("multipart/form-data"):               "multipartform",
}

var knownProducers = map[string]string{
	"json":          "runtime.JSONProducer()",
	"yaml":          "yamlpc.YAMLProducer()",
	"xml":           "runtime.XMLProducer()",
	"txt":           "runtime.TextProducer()",
	"bin":           "runtime.ByteStreamProducer()",
	"urlform":       "runtime.DiscardProducer",
	"multipartform": "runtime.DiscardProducer",
}

var knownConsumers = map[string]string{
	"json":          "runtime.JSONConsumer()",
	"yaml":          "yamlpc.YAMLConsumer()",
	"xml":           "runtime.XMLConsumer()",
	"txt":           "runtime.TextConsumer()",
	"bin":           "runtime.ByteStreamConsumer()",
	"urlform":       "runtime.DiscardConsumer",
	"multipartform": "runtime.DiscardConsumer",
}

func getSerializer(sers []GenSerGroup, ext string) (*GenSerGroup, bool) {
	for i := range sers {
		s := &sers[i]
		if s.Name == ext {
			return s, true
		}
	}
	return nil, false
}

func mediaTypeName(tn string) (string, bool) {
	for k, v := range mediaTypeNames {
		if k.MatchString(tn) {
			return v, true
		}
	}
	return "", false
}

func (a *appGenerator) makeConsumes() (consumes GenSerGroups, consumesJSON bool) {
	for _, cons := range a.Analyzed.RequiredConsumes() {
		cn, ok := mediaTypeName(cons)
		if !ok {
			continue
		}
		nm := swag.ToJSONName(cn)
		if nm == "json" {
			consumesJSON = true
		}

		if ser, ok := getSerializer(consumes, cn); ok {
			ser.AllSerializers = append(ser.AllSerializers, GenSerializer{
				AppName:        ser.AppName,
				ReceiverName:   ser.ReceiverName,
				Name:           ser.Name,
				MediaType:      cons,
				Implementation: knownConsumers[nm],
			})
			sort.Sort(ser.AllSerializers)
			continue
		}

		ser := GenSerializer{
			AppName:        a.Name,
			ReceiverName:   a.Receiver,
			Name:           nm,
			MediaType:      cons,
			Implementation: knownConsumers[nm],
		}

		consumes = append(consumes, GenSerGroup{
			AppName:        ser.AppName,
			ReceiverName:   ser.ReceiverName,
			Name:           ser.Name,
			MediaType:      cons,
			AllSerializers: []GenSerializer{ser},
			Implementation: ser.Implementation,
		})
	}
	if len(consumes) == 0 {
		consumes = append(consumes, GenSerGroup{
			AppName:      a.Name,
			ReceiverName: a.Receiver,
			Name:         "json",
			MediaType:    runtime.JSONMime,
			AllSerializers: []GenSerializer{GenSerializer{
				AppName:        a.Name,
				ReceiverName:   a.Receiver,
				Name:           "json",
				MediaType:      runtime.JSONMime,
				Implementation: knownConsumers["json"],
			}},
			Implementation: knownConsumers["json"],
		})
		consumesJSON = true
	}
	sort.Sort(consumes)
	return
}

func (a *appGenerator) makeProduces() (produces GenSerGroups, producesJSON bool) {
	for _, prod := range a.Analyzed.RequiredProduces() {
		pn, ok := mediaTypeName(prod)
		if !ok {
			continue
		}
		nm := swag.ToJSONName(pn)
		if nm == "json" {
			producesJSON = true
		}

		if ser, ok := getSerializer(produces, pn); ok {
			ser.AllSerializers = append(ser.AllSerializers, GenSerializer{
				AppName:        ser.AppName,
				ReceiverName:   ser.ReceiverName,
				Name:           ser.Name,
				MediaType:      prod,
				Implementation: knownProducers[nm],
			})
			sort.Sort(ser.AllSerializers)
			continue
		}

		ser := GenSerializer{
			AppName:        a.Name,
			ReceiverName:   a.Receiver,
			Name:           nm,
			MediaType:      prod,
			Implementation: knownProducers[nm],
		}
		produces = append(produces, GenSerGroup{
			AppName:        ser.AppName,
			ReceiverName:   ser.ReceiverName,
			Name:           ser.Name,
			MediaType:      prod,
			Implementation: ser.Implementation,
			AllSerializers: []GenSerializer{ser},
		})
	}
	if len(produces) == 0 {
		produces = append(produces, GenSerGroup{
			AppName:      a.Name,
			ReceiverName: a.Receiver,
			Name:         "json",
			MediaType:    runtime.JSONMime,
			AllSerializers: []GenSerializer{GenSerializer{
				AppName:        a.Name,
				ReceiverName:   a.Receiver,
				Name:           "json",
				MediaType:      runtime.JSONMime,
				Implementation: knownProducers["json"],
			}},
			Implementation: knownProducers["json"],
		})
		producesJSON = true
	}
	sort.Sort(produces)
	return
}

func (a *appGenerator) makeSecuritySchemes() (security GenSecuritySchemes) {

	prin := a.Principal
	if prin == "" {
		prin = "interface{}"
	}
	for _, scheme := range a.Analyzed.RequiredSecuritySchemes() {
		if req, ok := a.SpecDoc.Spec().SecurityDefinitions[scheme]; ok {
			isOAuth2 := strings.ToLower(req.Type) == "oauth2"
			var scopes []string
			if isOAuth2 {
				for k := range req.Scopes {
					scopes = append(scopes, k)
				}
			}

			security = append(security, GenSecurityScheme{
				AppName:      a.Name,
				ID:           scheme,
				ReceiverName: a.Receiver,
				Name:         req.Name,
				IsBasicAuth:  strings.ToLower(req.Type) == "basic",
				IsAPIKeyAuth: strings.ToLower(req.Type) == "apikey",
				IsOAuth2:     isOAuth2,
				Scopes:       scopes,
				Principal:    prin,
				Source:       req.In,
			})
		}
	}
	sort.Sort(security)
	return
}

func (a *appGenerator) makeCodegenApp() (GenApp, error) {
	log.Println("building a plan for generation")
	sw := a.SpecDoc.Spec()
	receiver := a.Receiver

	var defaultImports []string

	jsonb, _ := json.MarshalIndent(a.SpecDoc.OrigSpec(), "", "  ")

	consumes, _ := a.makeConsumes()
	produces, _ := a.makeProduces()
	sort.Sort(consumes)
	sort.Sort(produces)
	prin := a.Principal
	if prin == "" {
		prin = "interface{}"
	}
	security := a.makeSecuritySchemes()

	var genMods []GenDefinition
	importPath := a.GenOpts.ExistingModels
	if a.GenOpts.ExistingModels == "" {
		importPath = filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ModelsPackage))
	}

	defaultImports = append(defaultImports, importPath)

	log.Println("planning definitions")
	for mn, m := range a.Models {
		mod, err := makeGenDefinition(
			mn,
			a.ModelsPackage,
			m,
			a.SpecDoc,
			a.GenOpts,
		)
		if err != nil {
			return GenApp{}, err
		}
		if mod != nil {
			//mod.ReceiverName = receiver
			genMods = append(genMods, *mod)
		}
	}

	log.Println("planning operations")
	tns := make(map[string]struct{})
	var genOps GenOperations
	for on, opp := range a.Operations {
		o := opp.Op
		o.Tags = pruneEmpty(o.Tags)
		o.ID = on
		var bldr codeGenOpBuilder
		bldr.ModelsPackage = a.ModelsPackage
		bldr.Principal = prin
		bldr.Target = a.Target
		bldr.DefaultImports = defaultImports
		bldr.DefaultScheme = a.DefaultScheme
		bldr.Doc = a.SpecDoc
		bldr.Analyzed = a.Analyzed
		bldr.BasePath = a.SpecDoc.BasePath()
		bldr.GenOpts = a.GenOpts

		// TODO: change operation name to something safe
		bldr.Name = on
		bldr.Operation = *o
		bldr.Method = opp.Method
		bldr.Path = opp.Path
		bldr.Authed = len(a.Analyzed.SecurityRequirementsFor(o)) > 0
		bldr.Security = a.Analyzed.SecurityRequirementsFor(o)
		bldr.SecurityDefinitions = a.Analyzed.SecurityDefinitionsFor(o)
		bldr.RootAPIPackage = swag.ToFileName(a.APIPackage)
		bldr.WithContext = a.GenOpts != nil && a.GenOpts.WithContext

		bldr.APIPackage = bldr.RootAPIPackage
		st := o.Tags
		if a.GenOpts != nil {
			st = a.GenOpts.Tags
		}
		intersected := intersectTags(o.Tags, st)
		if len(st) > 0 && len(intersected) == 0 {
			continue
		}

		if len(intersected) == 1 {
			tag := intersected[0]
			bldr.APIPackage = a.GenOpts.LanguageOpts.MangleName(swag.ToFileName(tag), a.APIPackage)
			for _, t := range intersected {
				tns[t] = struct{}{}
			}
		}
		op, err := bldr.MakeOperation()
		if err != nil {
			return GenApp{}, err
		}
		op.ReceiverName = receiver
		op.Tags = intersected
		genOps = append(genOps, op)

	}
	for k := range tns {
		importPath := filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage, a.APIPackage, swag.ToFileName(k)))
		defaultImports = append(defaultImports, importPath)
	}
	sort.Sort(genOps)

	log.Println("grouping operations into packages")
	opsGroupedByPackage := make(map[string]GenOperations)
	for _, operation := range genOps {
		if operation.Package == "" {
			operation.Package = a.Package
		}
		opsGroupedByPackage[operation.Package] = append(opsGroupedByPackage[operation.Package], operation)
	}

	var opGroups GenOperationGroups
	for k, v := range opsGroupedByPackage {
		sort.Sort(v)
		opGroup := GenOperationGroup{
			GenCommon: GenCommon{
				Copyright: a.GenOpts.Copyright,
			},
			Name:           k,
			Operations:     v,
			DefaultImports: []string{filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ModelsPackage))},
			RootPackage:    a.APIPackage,
			WithContext:    a.GenOpts != nil && a.GenOpts.WithContext,
		}
		opGroups = append(opGroups, opGroup)
		var importPath string
		if k == a.APIPackage {
			importPath = filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage, a.APIPackage))
		} else {
			importPath = filepath.ToSlash(filepath.Join(baseImport(a.Target), a.ServerPackage, a.APIPackage, k))
		}
		defaultImports = append(defaultImports, importPath)
	}
	sort.Sort(opGroups)

	log.Println("planning meta data and facades")

	var collectedSchemes []string
	var extraSchemes []string
	for _, op := range genOps {
		collectedSchemes = concatUnique(collectedSchemes, op.Schemes)
		extraSchemes = concatUnique(extraSchemes, op.ExtraSchemes)
	}
	sort.Strings(collectedSchemes)
	sort.Strings(extraSchemes)

	host := "localhost"
	if sw.Host != "" {
		host = sw.Host
	}

	basePath := "/"
	if sw.BasePath != "" {
		basePath = sw.BasePath
	}

	return GenApp{
		GenCommon: GenCommon{
			Copyright: a.GenOpts.Copyright,
		},
		APIPackage:          a.ServerPackage,
		Package:             a.Package,
		ReceiverName:        receiver,
		Name:                a.Name,
		Host:                host,
		BasePath:            basePath,
		Schemes:             schemeOrDefault(collectedSchemes, a.DefaultScheme),
		ExtraSchemes:        extraSchemes,
		ExternalDocs:        sw.ExternalDocs,
		Info:                sw.Info,
		Consumes:            consumes,
		Produces:            produces,
		DefaultConsumes:     a.DefaultConsumes,
		DefaultProduces:     a.DefaultProduces,
		DefaultImports:      defaultImports,
		SecurityDefinitions: security,
		Models:              genMods,
		Operations:          genOps,
		OperationGroups:     opGroups,
		Principal:           prin,
		SwaggerJSON:         generateReadableSpec(jsonb),
		ExcludeSpec:         a.GenOpts != nil && a.GenOpts.ExcludeSpec,
		WithContext:         a.GenOpts != nil && a.GenOpts.WithContext,
		GenOpts:             a.GenOpts,
	}, nil
}

// generateReadableSpec makes swagger json spec as a string instead of bytes
// the only character that needs to be escaped is '`' symbol, since it cannot be escaped in the GO string
// that is quoted as `string data`. The function doesn't care about the beginning or the ending of the
// string it escapes since all data that needs to be escaped is always in the middle of the swagger spec.
func generateReadableSpec(spec []byte) string {
	buf := &bytes.Buffer{}
	for _, b := range string(spec) {
		if b == '`' {
			buf.WriteString("`+\"`\"+`")
		} else {
			buf.WriteRune(b)
		}
	}
	return buf.String()
}
