package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"text/tabwriter"

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

var pkgs = []string{"k8c.io/kubermatic/v2/pkg/provider/cloud/aws"}

const filter = "github.com/aws/aws-sdk-go-v2/*"

func TestSearchFuncInvocationsForPackages(t *testing.T) {
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
