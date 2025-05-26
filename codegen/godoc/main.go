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

package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log"
	"os"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"

	"k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/applications/providers/template"
	"k8c.io/kubermatic/v2/pkg/resources/prometheus"
)

var packageCache = map[string]*packages.Package{}

func main() {
	// document addon template data
	if err := generate("zz_generated.addondata.go.txt", addon.TemplateData{}); err != nil {
		log.Fatalf("Failed to create documentation: %v", err)
	}

	// document application template data
	if err := generate("zz_generated.applicationdata.go.txt", template.TemplateData{}); err != nil {
		log.Fatalf("Failed to create documentation: %v", err)
	}

	// document Prometheus custom rule template data
	if err := generate("zz_generated.prometheusdata.go.txt", prometheus.CustomizationData{}); err != nil {
		log.Fatalf("Failed to create documentation: %v", err)
	}
}

func generate(filename string, i interface{}) error {
	snippets := generateDocumentation([]string{}, reflect.ValueOf(i))
	code := strings.Join(snippets, "\n\n") + "\n"

	return os.WriteFile(filename, []byte(code), 0644)
}

func generateDocumentation(docs []string, v reflect.Value) []string {
	t := v.Type()

	if v.Kind() == reflect.Struct {
		pkgName := t.PkgPath()
		symbol := t.Name()

		log.Printf("Documenting %s.%s ...", pkgName, symbol)

		pkg, err := loadPackage(pkgName)
		if err != nil {
			log.Fatalf("Failed to load package: %v", err)
		}

		doc, err := getDocumentation(pkg, symbol)
		if err != nil {
			log.Fatalf("Failed to generate docs: %v", err)
		}

		docs = append(docs, doc)

		for n := range v.NumField() {
			docs = generateDocumentation(docs, v.Field(n))
		}
	}

	return docs
}

func loadPackage(name string) (*packages.Package, error) {
	if p, ok := packageCache[name]; ok {
		return p, nil
	}

	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax,
	}

	pkgs, err := packages.Load(&cfg, name)
	if err != nil {
		return nil, err
	}

	if len(pkgs) != 1 {
		return nil, fmt.Errorf("expected to load 1 package, but got %d", len(pkgs))
	}

	pkg := pkgs[0]
	packageCache[name] = pkg

	return pkg, nil
}

func getDocumentation(pkg *packages.Package, symbolName string) (string, error) {
	for _, s := range pkg.Syntax {
		for _, d := range s.Decls {
			if genDecl, ok := d.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE && len(genDecl.Specs) == 1 {
				if typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec); ok && typeSpec.Name.String() == symbolName {
					var buf bytes.Buffer
					err := format.Node(&buf, pkg.Fset, genDecl)

					return buf.String(), err
				}
			}
		}
	}

	return "", errors.New("symbol not found")
}
