package main

import (
	"fmt"
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"
)

type FuncCallID struct {
	ModulePath string
	Funcname   string
}

type FuncMetadata struct {
	Definition string
}

// FuncInvocations is a list of unique func calls and associated metadata
type FuncInvocations map[FuncCallID]FuncMetadata

// SearchFuncInvocationsForPackages returns all unique functions that the passed packages use that match the filter.
// You can supply a list of full go module paths that should be searched and a regex the imports should match
// example:  ([]string{"github.com/my-module/my-package"}, "github.com/aws/aws-sdk-go-v2/*") => returns all functions inside your package
// which are from any of the aws-sdk-go-v2 packages.
func SearchFuncInvocationsForPackages(pkgToSearch []string, filter *regexp.Regexp) (FuncInvocations, error) {
	res := make(map[FuncCallID]FuncMetadata)

	conf := &packages.Config{Mode: packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(conf, pkgToSearch...)
	if err != nil {
		return nil, err
	}

	// handle errors like no 'Go files found in pkg' upfront
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			// TODO turn into log statement
			fmt.Printf("WARN: error loading pkg %q: %q\n", pkg.ID, err.Msg) // we have to use pkg.ID here, as fields like name are not set
		}
	}

	for _, pkg := range pkgs {
		for _, obj := range pkg.TypesInfo.Uses {
			// filter out all the func types
			if _, ok := obj.(*types.Func); ok {
				// some (error).Error() objects do not have a Pkg. Filter these out so .Pkg().Path() does not panic
				if obj.Pkg() == nil {
					// fmt.Printf("%s xxxxxx %s xxxxx %s\n", obj, obj.Pkg(), pkg.Fset.Position(obj.Pos()).String())
					continue
				}

				// filter out only funcs where package matches
				if filter.Match([]byte(obj.Pkg().Path())) {
					id := FuncCallID{ModulePath: obj.Pkg().Path(), Funcname: obj.Name()}
					res[id] = FuncMetadata{Definition: pkg.Fset.Position(obj.Pos()).String()}
					// TODO turn into a log debug statement
					// fmt.Printf("func %s\t%s\t%s\t%s\n", obj.Name(), obj.Pkg().Name(), obj.Pkg().Path(), pkg.Fset.Position(obj.Pos()))
				}
			}
		}
	}

	return res, nil
}
