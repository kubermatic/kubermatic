package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"
	"text/tabwriter"
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

var pkgs = []string{"k8c.io/kubermatic/v2/pkg/provider/cloud/aws", "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/aws"}

var filter = regexp.MustCompile("github.com/aws/aws-sdk-go-v2/*")

func TestSearchFuncInvocationsForPackages(t *testing.T) {
	res, err := SearchFuncInvocationsForPackages("", pkgs, filter)
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

func TestAWSPermissionFuncMapping(t *testing.T) {
	res, err := SearchFuncInvocationsForPackages("", pkgs, filter)
	if err != nil {
		t.Fatal(err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, '\t', 0)

	for k := range res {
		fmt.Fprintf(w, "%s\t%s\n", k.Funcname, k.ModulePath)
	}
	w.Flush()
}

func TestGeneratingAWSPolicy(t *testing.T) {
	mapper, err := NewAWSDefaultMapper()
	if err != nil {
		t.Fatal(err)
	}

	invoc, err := SearchFuncInvocationsForPackages("", pkgs, filter)
	if err != nil {
		t.Fatal(err)
	}

	apc := NewAWSPolicyCreator(mapper)
	b, err := apc.GeneratePolicy(invoc)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(string(b))
}
