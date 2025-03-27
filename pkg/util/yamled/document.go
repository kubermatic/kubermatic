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

package yamled

import (
	"errors"
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v3"

	"k8c.io/kubermatic/sdk/v2/apis/equality"
)

type Document struct {
	// root is the first real node, i.e. the documentNode's
	// first and only child
	root *yaml.Node
}

func Load(r io.Reader) (*Document, error) {
	var data yaml.Node
	if err := yaml.NewDecoder(r).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode input YAML: %w", err)
	}

	return NewDocumentFromNode(&data)
}

func NewDocumentFromNode(m *yaml.Node) (*Document, error) {
	if m.Kind != yaml.DocumentNode {
		return nil, errors.New("node must be a DocumentNode")
	}

	if len(m.Content) == 0 {
		return nil, errors.New("documentNode must not be empty")
	}

	return &Document{
		root: m.Content[0],
	}, nil
}

func (d *Document) MarshalYAML() (interface{}, error) {
	var data interface{}
	if err := d.root.Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}

func (d *Document) Has(path Path) bool {
	_, exists := traversePath(d.root, path)

	return exists
}

func (d *Document) DecodeAtPath(path Path, dst interface{}) error {
	node, ok := d.GetNode(path)
	if !ok {
		return nil
	}

	return node.Decode(dst)
}

func (d *Document) GetNode(path Path) (*yaml.Node, bool) {
	if d.IsEmpty() {
		return nil, false
	}
	return traversePath(d.root, path)
}

func (d *Document) GetValue(path Path) (interface{}, bool) {
	node, exists := d.GetNode(path)
	if !exists {
		return nil, exists
	}

	return scalarNodeToGo(node)
}

func (d *Document) GetString(path Path) (string, bool) {
	node, exists := d.GetNode(path)
	if !exists {
		return "", exists
	}

	return nodeToString(node)
}

func (d *Document) GetInt(path Path) (int, bool) {
	node, exists := d.GetNode(path)
	if !exists {
		return 0, exists
	}

	return nodeToInt(node)
}

func (d *Document) GetBool(path Path) (bool, bool) {
	node, exists := d.GetNode(path)
	if !exists {
		return false, exists
	}

	return nodeToBool(node)
}

func (d *Document) GetArray(path Path) ([]interface{}, bool) {
	node, exists := d.GetNode(path)
	if !exists {
		return nil, exists
	}

	return sequenceNodeToGo(node)
}

func (d *Document) Set(path Path, newValue interface{}) bool {
	// we always need a key or array position to work with
	if len(path) == 0 {
		return false
	}

	newRoot, ok := setValue(d.root, path, newValue)
	d.root = newRoot

	return ok
}

func (d *Document) Append(path Path, newValue interface{}) bool {
	// we require maps at the root level, so the path cannot be empty
	if len(path) == 0 {
		return false
	}

	node, ok := d.GetNode(path)
	if !ok {
		return d.Set(path, []interface{}{newValue})
	}

	if node.Kind != yaml.SequenceNode {
		return false
	}

	newNode, ok := createNode(newValue)
	if !ok {
		return false
	}

	node.Content = append(node.Content, newNode)

	return true
}

func (d *Document) Remove(path Path) bool {
	// nuke everything
	if len(path) == 0 {
		newNode, ok := createNode(map[string]interface{}{})
		if !ok {
			return false
		}

		d.root = newNode
		return true
	}

	endKey := path.End()
	parentPath := path.Parent()

	parent, exists := d.GetNode(parentPath)
	if !exists {
		return true
	}

	if pos, ok := endKey.(int); ok {
		if parent.Kind == yaml.SequenceNode {
			if pos >= 0 && pos < len(parent.Content) {
				parent.Content = append(parent.Content[:pos], parent.Content[pos+1:]...)
			}

			return true
		}
	} else if key, ok := endKey.(string); ok {
		// check if we are really in a map
		if parent.Kind == yaml.MappingNode {
			for i, node := range parent.Content {
				if i%2 == 0 && node.Kind == yaml.ScalarNode && node.Value == key {
					parent.Content = append(parent.Content[:i], parent.Content[i+2:]...)
					break
				}
			}

			return true
		}
	}

	return false
}

// Fill will set the value at the path to the newValue, but keeps any existing
// sub values intact.
func (d *Document) Fill(path Path, newValue interface{}) bool {
	return mergeIntoPath(d.root, path, newValue)
}

func (d *Document) Equal(other *Document) bool {
	var (
		dData     interface{}
		otherData interface{}
	)

	if d.DecodeAtPath(nil, &dData) != nil {
		return false
	}

	if other.DecodeAtPath(nil, &otherData) != nil {
		return false
	}

	return equality.Semantic.DeepEqual(dData, otherData)
}

func (d *Document) IsEmpty() bool {
	contents, _ := d.MarshalYAML()
	return contents == nil
}
