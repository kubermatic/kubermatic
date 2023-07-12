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
	res, err := SearchFuncInvocationsForPackages(nil, "", pkgs, filter)
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
	res, err := SearchFuncInvocationsForPackages(nil, "", pkgs, filter)
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

	invoc, err := SearchFuncInvocationsForPackages(nil, "", pkgs, filter)
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
