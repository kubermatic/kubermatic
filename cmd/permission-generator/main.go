package main

import (
	"encoding/json"
	"fmt"
	"go/types"
	"regexp"
	"sort"

	"golang.org/x/tools/go/packages"
)

type AWSPolicyActions map[string]struct{}

// FlatAWSPolicyCreator is being used to automatically create flat AWS policies. It uses the combination of
// resource and effect to ensure that policy statements targeting the same resource with the same effect
// are not being split up or used twice.
// It can be printed for use in the AWS console with json.Marshal.
type FlatAWSPolicyCreator struct {
	Version string
	// statements should only be added via AddPolicyStatement method, therefore keep field private
	statements map[AWSPolicyID]AWSPolicyActions
}

type AWSPolicyID struct {
	resource string
	effect   string
}

func NewFlatAWSPolicyCreator() *FlatAWSPolicyCreator {
	return &FlatAWSPolicyCreator{
		Version:    "2012-10-17",
		statements: make(map[AWSPolicyID]AWSPolicyActions),
	}
}

func (apd *FlatAWSPolicyCreator) MarshalJSON() ([]byte, error) {
	type printableAWSPolicyStatement struct {
		Effect   string   `json:"Effect"`
		Actions  []string `json:"Action"`
		Resource string   `json:"Resource"`
	}

	type printableAWSPolicyDocument struct {
		Version    string                         `json:"Version"`
		Statements []*printableAWSPolicyStatement `json:"Statement"`
	}

	out := &printableAWSPolicyDocument{}
	out.Version = apd.Version

	for policyID, aps := range apd.statements {
		printActions := make([]string, len(aps))
		i := 0
		for v := range aps {
			printActions[i] = v
			i++
		}
		printStatement := &printableAWSPolicyStatement{
			Effect:   policyID.effect,
			Actions:  printActions,
			Resource: policyID.resource,
		}
		out.Statements = append(out.Statements, printStatement)
	}

	// sort the actions alphabetically
	for _, statement := range out.Statements {
		sort.Strings(statement.Actions)
	}

	return json.Marshal(out)
}

func (apd *FlatAWSPolicyCreator) AddPolicyStatement(resource string, effect string, actions []string) {
	polID := AWSPolicyID{resource: resource, effect: effect}
	awsActions, found := apd.statements[polID]

	// if a statement with same effect and resource already exists, append to its actions
	if found {
		for _, action := range actions {
			awsActions[action] = struct{}{}
		}
		// otherwise create a new statement
	} else {
		newActions := make(map[string]struct{})
		for _, action := range actions {
			newActions[action] = struct{}{}
		}
		apd.statements[polID] = newActions
	}
}

type FuncCallID struct {
	ModulePath string
	Funcname   string
}

type FuncMetadata struct {
	ScopePermissions map[string][]string
	Definition       string
}

// SearchFuncInvocationsForPackages returns all unique functions that the passed packages use that match the filter.
// You can supply a list of full go module paths that should be searched and a regex string the imports should match
// example:  ([]string{"github.com/my-module/my-package"}, "github.com/aws/aws-sdk-go-v2/*") => returns all functions inside your package
// which are from any of the aws-sdk-go-v2 packages.
func SearchFuncInvocationsForPackages(mapper *AWSPermissionFuncMapping, pkgToSearch []string, filter string) (map[FuncCallID]FuncMetadata, error) {
	res := make(map[FuncCallID]FuncMetadata)

	r, err := regexp.Compile(filter)
	if err != nil {
		return nil, err
	}

	conf := &packages.Config{Mode: packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(conf, pkgToSearch...)
	if err != nil {
		return nil, err
	}

	// TODO I saw this somewhere, but not sure if we need to handle this as well
	/* if packages.PrintErrors(pkgs) > 0 { */
	/* 	t.Fatal("Encountered more Print errors") */
	/* } */

	for _, pkg := range pkgs {
		for _, obj := range pkg.TypesInfo.Uses {
			// filter out all the func types
			if _, ok := obj.(*types.Func); ok {
				// filter out only funcs where package matches
				if r.Match([]byte(obj.Pkg().Path())) {
					id := FuncCallID{ModulePath: obj.Pkg().Path(), Funcname: obj.Name()}
					scopPerm := make(map[string][]string)
					if mapper != nil {
						rd, err := mapper.LookUpPermissionsForFunc(id)
						if err != nil {
							// TODO change this to a log statement
							fmt.Printf("WARN: %q\n", err)
						}
						for scope, actions := range rd {
							scopPerm[scope] = actions.Actions
						}
					}
					res[id] = FuncMetadata{ScopePermissions: scopPerm, Definition: pkg.Fset.Position(obj.Pos()).String()}
					// TODO turn into a log debug statement
					// fmt.Printf("func %s\t%s\t%s\t%s\n", obj.Name(), obj.Pkg().Name(), obj.Pkg().Path(), pkg.Fset.Position(obj.Pos()))
				}
			}
		}
	}

	return res, nil
}

// LookUpPermissionsForFunc returns the permissions for a func
// It will return nil if the func has no permissions associated with it
// and an error if no permissions are found for the supplied func.
func (a *AWSPermissionFuncMapping) LookUpPermissionsForFunc(fid FuncCallID) (map[string]resourceDefinition, error) {
	if _, ok := a.Modules[fid.ModulePath].Funcs[fid.Funcname]; !ok {
		return nil, fmt.Errorf("mapper could not find permissions for func %q from module %q. Please check mapper yaml", fid.Funcname, fid.ModulePath)
	}

	return a.Modules[fid.ModulePath].Funcs[fid.Funcname].Permissions, nil
}

// An AWSPolicyFuncMapping describes a mapping of an AWS SDK func to its permissions
type AWSPermissionFuncMapping struct {
	Modules map[string]moduleDefinition `yaml:"modules"`
}

// func (a *AWSPermissionFuncMapping) AddAllowMapping(module string, fun string, resource string, actions []string) {
// 	// if module already exists, append to it
// 	if m, ok := a.Modules[module]; ok {
// 		// if func already exists, append to it
// 		if f, ok := m.Funcs[fun]; ok {
// 			// if resource exists, append the actions
// 			if r, ok := f.Permissions[resource]; ok {
// 				r.Actions = append(r.Actions, actions...)
// 			}
// 		} else {
// 			f := funcDefinition{}
// 		}
// 	} else {
// 	r := resourceDefinition{ Actions: actions}
// 	f := funcDefinition{}
// 	f[resource] = r
// 	m := moduleDefinition{}
// 	m[fun] = f
// 	a[module] = m
// }

type moduleDefinition struct {
	Funcs map[string]funcDefinition `yaml:"funcs"`
}

type funcDefinition struct {
	Permissions map[string]resourceDefinition `yaml:"permissions"`
}

type resourceDefinition struct {
	Actions []string `yaml:"actions"`
}
