/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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
	"fmt"
	"go/types"
	"io"
	"regexp"
	"text/tabwriter"

	"go.uber.org/zap"
	"golang.org/x/tools/go/packages"
)

type FuncCallID struct {
	ModulePath string
	Funcname   string
}

type FuncMetadata struct {
	Definition string
}

// FuncInvocations is a list of unique func calls and associated metadata.
type FuncInvocations map[FuncCallID]FuncMetadata

// SearchFuncInvocationsForPackages returns all unique functions that the passed packages use that match the filter.
// You can supply a list of full go module paths that should be searched and a regex the imports should match
// example:  ([]string{"github.com/my-module/my-package"}, "github.com/aws/aws-sdk-go-v2/*") => returns all functions inside your package
// which are from any of the aws-sdk-go-v2 packages.
func SearchFuncInvocationsForPackages(log *zap.SugaredLogger, dir string, pkgToSearch []string, filter *regexp.Regexp) (FuncInvocations, error) {
	res := make(map[FuncCallID]FuncMetadata)

	conf := &packages.Config{Dir: dir, Mode: packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(conf, pkgToSearch...)
	if err != nil {
		return nil, err
	}

	// handle errors like no 'Go files found in pkg' upfront
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			log.Warnf("error loading pkg %q: %q\n", pkg.ID, err.Msg) // we have to use pkg.ID here, as fields like name are not set
		}
	}

	for _, pkg := range pkgs {
		for _, obj := range pkg.TypesInfo.Uses {
			// filter out all the func types
			if _, ok := obj.(*types.Func); ok {
				// some (error).Error() objects do not have a Pkg. Filter these out so .Pkg().Path() does not panic
				if obj.Pkg() == nil {
					log.Debugf("%s xxxxxx %s xxxxx %s\n", obj, obj.Pkg(), pkg.Fset.Position(obj.Pos()).String())
					continue
				}

				// filter out only funcs where package matches
				if filter.Match([]byte(obj.Pkg().Path())) {
					id := FuncCallID{ModulePath: obj.Pkg().Path(), Funcname: obj.Name()}
					res[id] = FuncMetadata{Definition: pkg.Fset.Position(obj.Pos()).String()}
					log.Debugf("func %s\t%s\t%s\t%s\n", obj.Name(), obj.Pkg().Name(), obj.Pkg().Path(), pkg.Fset.Position(obj.Pos()))
				}
			}
		}
	}

	return res, nil
}

func PrintFuncInvocations(writer io.Writer, fi FuncInvocations) {
	w := new(tabwriter.Writer)
	w.Init(writer, 8, 8, 0, '\t', 0)
	defer w.Flush()

	for k, v := range fi {
		fmt.Fprintf(w, "%s\t%s\t%s\n", k.Funcname, k.ModulePath, v.Definition)
	}
}
