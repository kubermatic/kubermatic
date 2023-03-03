package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"testing"
	"text/tabwriter"

	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v3"
)

func TestAddPolicyStatement(t *testing.T) {
	fapc := NewFlatAWSPolicyCreator()

	fapc.AddPolicyStatement("*", "Allow", []string{"elasticloadbalancing:CreateListener", "elasticloadbalancing:CreateRule", "elasticloadbalancing:CreateTargetGroup", "elasticloadbalancing:CreateLoadBalancer", "elasticloadbalancing:ConfigureHealthCheck"})
	fapc.AddPolicyStatement("*", "Deny", []string{"elasticloadbalancing:CreateListener", "elasticloadbalancing:CreateRule", "elasticloadbalancing:CreateTargetGroup", "elasticloadbalancing:CreateLoadBalancer", "elasticloadbalancing:ConfigureHealthCheck"})
	fapc.AddPolicyStatement("some-resource", "Allow", []string{"elasticloadbalancing:CreateListener", "elasticloadbalancing:CreateRule", "elasticloadbalancing:CreateTargetGroup", "elasticloadbalancing:CreateLoadBalancer", "elasticloadbalancing:ConfigureHealthCheck"})
	fapc.AddPolicyStatement("*", "Allow", []string{"something-new"})

	b, err := json.MarshalIndent(fapc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(b))
}

func TestAstParsing(t *testing.T) {
	fpath := "/Users/simonbein/github/simontheleg/kubermatic/pkg/provider/cloud/aws/api.go"
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, fpath, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	type result = struct {
		funcname string
	}
	res := []result{}

	// for _, decl := range node.Decls {
	// 	ast.Inspect(decl, func(n ast.Node) bool {
	// 		f, ok := n.(*ast.FuncDecl)
	// 		if ok {
	// 			res = append(res, result{funcname: f.Name.Name})
	// 			return true
	// 		}
	// 		return false
	// 	})
	// }

	getSubNetsF := node.Decls[1]

	ast.Inspect(getSubNetsF, func(node ast.Node) bool {
		if n, ok := node.(*ast.SelectorExpr); ok {
			fmt.Println(n.Sel.Name)
			// ident, ok := n.Fun.(*ast.Ident)
			// if ok {
			// 	res = append(res, result{funcname: ident.Name})
			// }
		}
		return true
	})

	fmt.Println(len(res))
	// for _, p := range res {
	// 	fmt.Println(p.funcname)
	// }
}

// func TestNewApproach(t *testing.T) {
// 	fpath := "/Users/simonbein/github/simontheleg/kubermatic/pkg/provider/cloud/aws/api.go"
// 	fset := token.NewFileSet()
// 	f, err := parser.ParseFile(fset, fpath, nil, parser.ParseComments)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	conf := packages.Config{Importer: importer.Default()}
// 	pkg, err := conf.Check("k8c.io/kubermatic/v2/pkg/provider/cloud/aws", fset, []*ast.File{f}, nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	fmt.Printf("Package  %q\n", pkg.Path())
// 	fmt.Printf("Name:    %s\n", pkg.Name())
// 	fmt.Printf("Imports: %s\n", pkg.Imports())
// 	fmt.Printf("Scope:   %s\n", pkg.Scope())
// }

func TestNewApproach(t *testing.T) {
	// fpath := "/Users/simonbein/github/simontheleg/kubermatic/pkg/provider/cloud/aws/api.go"
	// fset := token.NewFileSet()
	// f, err := parser.ParseFile(fset, fpath, nil, parser.ParseComments)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	conf := &packages.Config{Mode: packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesInfo}
	pkgs, err := packages.Load(conf, "k8c.io/kubermatic/v2/pkg/provider/cloud/aws")
	if err != nil {
		t.Fatal(err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		// TODO proper error message
		t.Fatal("Encountered more Print errors")
	}

	for _, pkg := range pkgs {
		for _, obj := range pkg.TypesInfo.Uses {
			// filter out all the func types
			if _, ok := obj.(*types.Func); ok {
				// filter out only funcs where package matches
				if strings.Contains(obj.Pkg().Path(), "github.com/aws/aws-sdk-go-v2") {
					fmt.Printf("func %s\t%s\t%s\t%s\n", obj.Name(), obj.Pkg().Name(), obj.Pkg().Path(), pkg.Fset.Position(obj.Pos()))
				}
			}
		}
		// fmt.Println(pkg.TypesInfo)
		// scope := pkg.Types.Scope()

		// for _, name := range scope.Names() {
		// 	obj := scope.Lookup(name)
		// 	fmt.Println(obj)
		// 	// filter out all the func types
		// 	// if _, ok := obj.(*types.Func); ok {
		// 	// 	fmt.Printf(obj.Name()+" %s\n", (pkg.Fset.Position(obj.Pos())))
		// 	// }

		// }
		// if obj.Type() == types.{
		// 	fmt.Println(obj)
		// }
	}

}

func TestSearchFuncInvocationsForPackages(t *testing.T) {
	pkgs := []string{"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"}
	filter := "github.com/aws/aws-sdk-go-v2/*"

	res, err := SearchFuncInvocationsForPackages(nil, pkgs, filter)
	if err != nil {
		t.Fatal(err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)
	defer w.Flush()

	for k, v := range res {
		fmt.Fprintf(w, "%s\t%s\t%s\n", k.Funcname, k.ModulePath, v.Definition)
	}
}

//go:embed mapper.yml
var mapperyml []byte

func TestAWSPermissionFuncMapping(t *testing.T) {
	mapper := &AWSPermissionFuncMapping{}
	err := yaml.Unmarshal(mapperyml, mapper)
	if err != nil {
		t.Fatal(err)
	}

	pkgs := []string{"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"}
	filter := "github.com/aws/aws-sdk-go-v2/*"

	res, err := SearchFuncInvocationsForPackages(mapper, pkgs, filter)
	if err != nil {
		t.Fatal(err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)

	for k, v := range res {
		fmt.Fprintf(w, "%s\t%s\t%s\n", k.Funcname, k.ModulePath, v.ScopePermissions)
	}
	w.Flush()
}

func TestGeneratingAWSPolicy(t *testing.T) {
	mapper := &AWSPermissionFuncMapping{}
	err := yaml.Unmarshal(mapperyml, mapper)
	if err != nil {
		t.Fatal(err)
	}

	pkgs := []string{"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"}
	filter := "github.com/aws/aws-sdk-go-v2/*"

	res, err := SearchFuncInvocationsForPackages(mapper, pkgs, filter)
	if err != nil {
		t.Fatal(err)
	}

	// create a policy from our funcs
	fapc := NewFlatAWSPolicyCreator()
	for _, metadata := range res {
		for scope, actions := range metadata.ScopePermissions {
			fapc.AddPolicyStatement(scope, "Allow", actions)
		}
	}

	b, err := json.MarshalIndent(fapc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(b))
}
