package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

var _ PolicyCreator = (*AWSPolicyCreator)(nil)

type AWSPolicyCreator struct {
	mapper *AWSPermissionMapper
}

func NewAWSPolicyCreator(mapper *AWSPermissionMapper) *AWSPolicyCreator {
	return &AWSPolicyCreator{mapper: mapper}
}

func (apc *AWSPolicyCreator) GeneratePolicy(fi FuncInvocations) ([]byte, error) {

	fapc := NewFlatAWSPolicyCreator()

	for fid := range fi {
		perm, err := apc.mapper.LookUpPermissionsForFunc(fid)
		if err != nil {
			return nil, err
		}
		for scope, resourceDef := range perm {
			fapc.AddPolicyStatement(scope, "Allow", resourceDef.Actions)
		}
	}

	b, err := json.MarshalIndent(fapc, "", "  ") // use double space indent to make policy more readable
	if err != nil {
		return nil, err
	}

	return b, nil
}

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

type printableAWSPolicyStatement struct {
	Effect   string   `json:"Effect"`
	Actions  []string `json:"Action"`
	Resource string   `json:"Resource"`
}

type printableAWSPolicyDocument struct {
	Version    string                         `json:"Version"`
	Statements []*printableAWSPolicyStatement `json:"Statement"`
}

// Sorting a printableAWSPolicyDocument sorts its statements by the effect name
func (p *printableAWSPolicyDocument) Less(i, j int) bool {
	return p.Statements[i].Effect < p.Statements[j].Effect
}

func (p *printableAWSPolicyDocument) Len() int {
	return len(p.Statements)
}

func (p *printableAWSPolicyDocument) Swap(i, j int) {
	p.Statements[i], p.Statements[j] = p.Statements[j], p.Statements[i]
}

func (apd *FlatAWSPolicyCreator) MarshalJSON() ([]byte, error) {
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

	sort.Sort(out)

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

// LookUpPermissionsForFunc returns the permissions for a func
// It will return nil if the func has no permissions associated with it
// and an error if no permissions are found for the supplied func.
func (a *AWSPermissionMapper) LookUpPermissionsForFunc(fid FuncCallID) (map[string]resourceDefinition, error) {
	if _, ok := a.Modules[fid.ModulePath].Funcs[fid.Funcname]; !ok {
		return nil, fmt.Errorf("mapper could not find permissions for func %q from module %q. Please check mapper yaml", fid.Funcname, fid.ModulePath)
	}

	return a.Modules[fid.ModulePath].Funcs[fid.Funcname].Permissions, nil
}

// An AWSPolicyFuncMapping describes a mapping of an AWS SDK func to its permissions
type AWSPermissionMapper struct {
	Modules map[string]moduleDefinition `yaml:"modules"`
}

type moduleDefinition struct {
	Funcs map[string]funcDefinition `yaml:"funcs"`
}

type funcDefinition struct {
	Permissions map[string]resourceDefinition `yaml:"permissions"`
}

type resourceDefinition struct {
	Actions []string `yaml:"actions"`
}

//go:embed defaultmapper.yml
var defaultmapperyml []byte

func NewAWSDefaultMapper() (*AWSPermissionMapper, error) {
	mapper := &AWSPermissionMapper{}
	err := yaml.Unmarshal(defaultmapperyml, mapper)
	if err != nil {
		return nil, err
	}
	return mapper, nil
}
