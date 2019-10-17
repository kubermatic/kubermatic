/*
Copyright 2019 The Kubernetes Authors.

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

// Package genyaml can generate an example YAML snippet from
// an initialized struct and decorate it with godoc comments parsed
// from the AST of a given file.
//
// Example:
//	cm := NewCommentMap("example_config.go")

//	yamlSnippet, err := cm.GenYaml(&plugins.Configuration{
//		Approve: []plugins.Approve{
//			{
//				Repos: []string{
//					"ORGANIZATION",
//					"ORGANIZATION/REPOSITORY",
//				},
//				IssueRequired:       false,
//				RequireSelfApproval: new(bool),
//				LgtmActsAsApprove:   false,
//				IgnoreReviewState:   new(bool),
//			},
//		},
//	})
//
// 	yamlSnippet will be assigned a string containing the following YAML:
//
//	# Approve is the configuration for the Approve plugin.
//	approve:
//	  - # Repos is either of the form org/repos or just org.
//		repos:
//		  - ORGANIZATION
//		  - ORGANIZATION/REPOSITORY
//
//		# IssueRequired indicates if an associated issue is required for approval in the specified repos.
//		issue_required: true
//
//		# RequireSelfApproval requires PR authors to explicitly approve their PRs. Otherwise the plugin assumes the author of the PR approves the changes in the PR.
//		require_self_approval: false
//
//		# LgtmActsAsApprove indicates that the lgtm command should be used to indicate approval
//		lgtm_acts_as_approve: true
//
//		# IgnoreReviewState causes the approve plugin to ignore the GitHub review state. Otherwise: * an APPROVE github review is equivalent to leaving an \"/approve\" message. * A REQUEST_CHANGES github review is equivalent to leaving an /approve cancel\" message.
//		ignore_review_state: false
//
package genyaml

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/clarketm/json"
	yaml3 "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	jsonTag = "json"
)

// Comment is an abstract structure for storing mapped types to comments.
type CommentMap struct {
	// comments is a map of string(typeSpecName) -> string(tagName) -> Comment.
	comments map[string]map[string]Comment
	// RWMutex is a read/write mutex.
	sync.RWMutex
}

// NewCommentMap is the constructor for CommentMap accepting a variadic number of paths.
func NewCommentMap(paths ...string) *CommentMap {
	cm := &CommentMap{
		comments: make(map[string]map[string]Comment),
	}

	for _, path := range paths {
		cm.AddPath(path)
	}

	return cm
}

// Comment is an abstract structure for storing parsed AST comments decorated with contextual information.
type Comment struct {
	// Type is the underlying type of the identifier associated with the comment.
	Type string
	// IsObj determines if the underlying type is a object type (e.g. struct) or primitive type (e.g. string).
	IsObj bool
	// Doc is a comment string parsed from the AST of a node.
	Doc string
}

// marshal marshals the object into JSON then converts JSON to YAML and returns the YAML.
func marshal(o interface{}) ([]byte, error) {
	j, err := json.Marshal(o)
	if err != nil {
		return nil, fmt.Errorf("error marshaling into JSON: %v", err)
	}

	y, err := jsonToYaml(j)
	if err != nil {
		return nil, fmt.Errorf("error converting JSON to YAML: %v", err)
	}

	return y, nil
}

// jsonToYaml Converts JSON to YAML.
func jsonToYaml(j []byte) ([]byte, error) {
	// Convert the JSON to an object.
	var jsonObj interface{}
	// We are using yaml.Unmarshal here (instead of json.Unmarshal) because the
	// Go JSON library doesn't try to pick the right number type (int, float,
	// etc.) when unmarshalling to interface{}, it just picks float64
	// universally. go-yaml does go through the effort of picking the right
	// number type, so we can preserve number type throughout this process.
	err := yaml3.Unmarshal(j, &jsonObj)
	if err != nil {
		return nil, err
	}

	// marshal this object into YAML.
	return yaml3.Marshal(jsonObj)
}

// astFrom takes a path to a Go file and returns the abstract syntax tree (AST) for that file.
func astFrom(path string) (*doc.Package, error) {
	fset := token.NewFileSet()
	m := make(map[string]*ast.File)

	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("unable to parse file to AST from path: %s", path)
	}

	m[path] = f
	apkg, _ := ast.NewPackage(fset, m, nil, nil)

	astDoc := doc.New(apkg, "", 0)
	if astDoc == nil {
		return nil, fmt.Errorf("unable to parse AST documentation from path: %s", path)
	}

	return astDoc, nil
}

// fmtRawDoc formats/sanitizes a Go doc string removing TODOs, newlines, whitespace, and various other characters from the resultant string.
func fmtRawDoc(rawDoc string) string {
	var buffer bytes.Buffer

	// Ignore all lines after ---.
	rawDoc = strings.Split(rawDoc, "---")[0]

	for _, line := range strings.Split(rawDoc, "\n") {
		line = strings.TrimSpace(line) // Trim leading and trailing whitespace.
		switch {
		case strings.HasPrefix(line, "TODO"): // Ignore one line TODOs.
		case strings.HasPrefix(line, "+"): // Ignore instructions to the generators.
		default:
			line += "\n"
			buffer.WriteString(line)
		}
	}

	postDoc := strings.TrimRight(buffer.String(), "\n")               // Remove last newline.
	postDoc = strings.Replace(postDoc, "\t", " ", -1)                 // Replace tabs with spaces.
	postDoc = regexp.MustCompile(` +`).ReplaceAllString(postDoc, " ") // Compress multiple spaces to a single space.

	return postDoc
}

// fieldTag extracts the given tag or returns an empty string if the tag is not defined.
func fieldTag(field *ast.Field, tag string) string {
	if field.Tag == nil {
		return ""
	}

	return reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]).Get(tag)
}

// fieldName extracts the name of the field as it should appear in YAML format and returns the resultant string.
// "-" indicates that this field is not part of the YAML representation and is thus excluded.
func fieldName(field *ast.Field, tag string) string {
	tagVal := strings.Split(fieldTag(field, tag), ",")[0] // This can return "-".
	if tagVal == "" {
		// Set field name to the defined name in struct if defined.
		if field.Names != nil {
			return field.Names[0].Name
		}
		// Fallback field name to the immediate field type.
		name, _ := fieldType(field, false)
		return name
	}
	return tagVal
}

// fieldIsInlined returns true if the field is tagged with ",inline"
func fieldIsInlined(field *ast.Field, tag string) bool {
	values := sets.NewString(strings.Split(fieldTag(field, tag), ",")...)

	return values.Has("inline")
}

// fieldType extracts the type of the field and returns the resultant string type and a bool indicating if it is an object type.
func fieldType(field *ast.Field, recurse bool) (string, bool) {
	typeName := ""
	isObj, isSelect := false, false

	// Find leaf node.
	ast.Inspect(field, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Field:
			// First node is always a field; skip.
			return true
		case *ast.Ident:
			// Encountered a type, overwrite typeName and isObj.
			typeName = x.Name
			isObj = x.Obj != nil || isSelect
		case *ast.SelectorExpr:
			// SelectorExpr are not object types yet reference one, thus continue with DFS.
			isSelect = true
		}

		return recurse || isSelect
	})

	return typeName, isObj
}

// getType returns the type's name within its package for a defined type. For other (non-defined) types it returns the empty string.
func getType(typ interface{}) string {
	if t := reflect.TypeOf(typ); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

// genDocMap extracts the name of the field as it should appear in YAML format and returns the resultant string.
func (cm *CommentMap) genDocMap(path string) error {
	pkg, err := astFrom(path)
	if err != nil {
		return errors.New("unable to generate AST documentation map")
	}

	inlineFields := map[string][]string{}

	for _, t := range pkg.Types {
		if typeSpec, ok := t.Decl.Specs[0].(*ast.TypeSpec); ok {

			var lst []*ast.Field

			// Support struct type, interface type, and type alias.
			switch typ := typeSpec.Type.(type) {
			case *ast.InterfaceType:
				lst = typ.Methods.List
			case *ast.StructType:
				lst = typ.Fields.List
			case *ast.Ident:
				// ensure that aliases for non-struct/interface types continue to work
				if typ.Obj != nil {
					if alias, ok := typ.Obj.Decl.(*ast.TypeSpec).Type.(*ast.InterfaceType); ok {
						lst = alias.Methods.List
					} else if alias, ok := typ.Obj.Decl.(*ast.TypeSpec).Type.(*ast.StructType); ok {
						lst = alias.Fields.List
					}
				}
			}

			typeSpecName := typeSpec.Name.Name
			cm.comments[typeSpecName] = make(map[string]Comment)

			for _, field := range lst {

				if tagName := fieldName(field, jsonTag); tagName != "-" {
					typeName, isObj := fieldType(field, true)
					docString := fmtRawDoc(field.Doc.Text())
					cm.comments[typeSpecName][tagName] = Comment{typeName, isObj, docString}

					if fieldIsInlined(field, jsonTag) {
						existing, ok := inlineFields[typeSpecName]
						if !ok {
							existing = []string{}
						}
						inlineFields[typeSpecName] = append(existing, tagName)
					}
				}
			}
		}
	}

	// copy comments for inline fields from their original parent structures; this is needed
	// because when walking the generated YAML, the step to switch to the "correct" parent
	// struct is missing
	for typeSpecName, inlined := range inlineFields {
		for _, inlinedType := range inlined {
			for tagName, comment := range cm.comments[inlinedType] {
				cm.comments[typeSpecName][tagName] = comment
			}
		}
	}

	return nil
}

// injectComment reads a YAML node and injects a head comment based on its value and typeSpec.
func (cm *CommentMap) injectComment(parent *yaml3.Node, typeSpec []string, depth int) {
	if parent == nil || depth >= len(typeSpec) {
		return
	}

	typ := typeSpec[depth]

	// Decorate YAML node with comment.
	if v, ok := cm.comments[typ][parent.Value]; ok {
		parent.HeadComment = v.Doc
	}

	if parent.Content != nil {
		for i, child := range parent.Content {

			// Default type for node is current (i.e. most recent) type.
			nxtTyp := typeSpec[len(typeSpec)-1]

			if i > 0 {
				prevSibling := parent.Content[i-1]

				// Skip value nodes.
				if prevSibling.Kind == yaml3.ScalarNode && child.Kind == yaml3.ScalarNode && i%2 == 1 {
					continue
				}

				// New type detected; add type of key (i.e. prevSibling) to stack.
				if parent.Kind == yaml3.MappingNode && prevSibling.Kind == yaml3.ScalarNode {
					if subTypeSpec, ok := cm.comments[typ][prevSibling.Value]; ok && subTypeSpec.IsObj {
						nxtTyp = subTypeSpec.Type
					}
				}
			}

			// Recurse to inject comments on nested YAML nodes.
			cm.injectComment(child, append(typeSpec, nxtTyp), depth+1)
		}
	}

}

// PrintComments pretty prints comments.
func (cm *CommentMap) PrintComments() {
	cm.RLock()
	defer cm.RUnlock()

	data, err := json.MarshalIndent(cm.comments, "", "  ")
	if err == nil {
		fmt.Print(string(data))
	}
}

// AddPath allow for adding to the CommentMap via path specification to a `.go` file, returning a boolean indicating success.
func (cm *CommentMap) AddPath(path string) bool {
	cm.Lock()
	defer cm.Unlock()

	err := cm.genDocMap(path)
	if err != nil {
		return false
	}

	return true
}

// SetPath allow for setting of the CommentMap via path specification to a `.go` file, returning a boolean indicating success.
func (cm *CommentMap) SetPath(path string) bool {
	cm.Lock()
	defer cm.Unlock()

	// Empty map.
	cm.comments = make(map[string]map[string]Comment)

	err := cm.genDocMap(path)
	if err != nil {
		return false
	}

	return true
}

// GenYaml generates a fully commented YAML snippet for a given plugin configuration.
func (cm *CommentMap) GenYaml(config interface{}) (string, error) {
	cm.RLock()
	defer cm.RUnlock()

	var baseTypeSpec = getType(config)

	// Convert Config object to an abstract YAML node.
	y1, err := marshal(&config)
	if err != nil {
		return "", errors.New("failed to marshal config to yaml")
	}

	node := yaml3.Node{}
	err = yaml3.Unmarshal([]byte(y1), &node)
	if err != nil {
		return "", errors.New("failed to unmarshal yaml to yaml node")
	}

	// Inject comments
	cm.injectComment(&node, []string{baseTypeSpec}, 0)

	// Convert Yaml w/ comments to string.
	y2, err := yaml3.Marshal(&node)
	if err != nil {
		return "", errors.New("failed to marshal yaml node to yaml")
	}

	return string(y2), nil
}
