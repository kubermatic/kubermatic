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
	"bytes"

	yaml "gopkg.in/yaml.v3"
)

// nullNode returns a new scalar node with "null"
// as its value.
func nullNode() *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "null",
		Tag:   "!!null",
	}
}

func traversePath(root *yaml.Node, path Path) (*yaml.Node, bool) {
	currentNode := root

	for _, step := range path {
		stepFound := false

		if sstep, ok := step.(string); ok {
			// step is string => try descending down a map
			if currentNode.Kind != yaml.MappingNode {
				return nil, false // not a map, cannot descend
			}

			// maps like {a: 1, b: 2} are represented as content=[key, value, key, value, ...],
			for idx, node := range currentNode.Content {
				if idx%2 != 0 {
					continue // skip values
				}

				if node.Value == sstep {
					stepFound = true
					currentNode = currentNode.Content[idx+1]
					break
				}
			}
		} else if istep, ok := step.(int); ok {
			// step is int => try getting Nth element of list
			if currentNode.Kind != yaml.SequenceNode {
				return nil, false // not a sequence, cannot descend
			}

			if istep < 0 || istep >= len(currentNode.Content) {
				return nil, false
			}

			stepFound = true
			currentNode = currentNode.Content[istep]
		}

		if !stepFound {
			return nil, false
		}
	}

	return currentNode, true
}

func createNode(newValue interface{}) (*yaml.Node, bool) {
	var buf bytes.Buffer

	if err := yaml.NewEncoder(&buf).Encode(newValue); err != nil {
		return nil, false
	}

	doc, err := Load(&buf)
	if err != nil {
		return nil, false
	}

	return doc.root, true
}

func setValue(root *yaml.Node, path Path, newValue interface{}) (*yaml.Node, bool) {
	// when we have reached the root level,
	// replace our root element with the new data structure
	if len(path) == 0 {
		return root, true
	}

	endKey := path.End()
	parentPath := path.Parent()
	target := root

	// create parent if missing
	if len(parentPath) > 0 {
		var exists bool

		target, exists = traversePath(root, parentPath)
		if !exists {
			// depending on the type of the path step, we must either
			// create a new empty sequence or a new empty map
			var newValue interface{}

			if _, ok := endKey.(int); ok {
				newValue = []interface{}{}
			} else if _, ok := endKey.(string); ok {
				newValue = map[string]interface{}{}
			} else {
				return nil, false
			}

			if _, success := setValue(root, parentPath, newValue); !success {
				return nil, false
			}

			target, _ = traversePath(root, parentPath)
		}
	}

	// Now we know that the parent element exists.

	if pos, ok := endKey.(int); ok {
		// check if we are really in an array
		if target.Kind != yaml.SequenceNode {
			return nil, false
		}

		// invalid input
		if pos < 0 {
			return nil, false
		}

		// add new nil nodes until we have enough
		for len(target.Content) <= pos {
			target.Content = append(target.Content, nullNode())
		}

		// overwrite the target null node with the actual content
		newNode, ok := createNode(newValue)
		if !ok {
			return nil, false
		}

		target.Content[pos] = newNode

		return root, true
	}

	if headKey, ok := endKey.(string); ok {
		// check if we are really in a map
		if target.Kind != yaml.MappingNode {
			// if we're at the root and the document was empty (=> null), we
			// default the target node to an empty object
			if target == root {
				*target = yaml.Node{
					Kind: yaml.MappingNode,
				}
			} else {
				return nil, false
			}
		}

		node, exists := traversePath(target, Path{headKey})
		if !exists {
			node = nullNode()

			// add a new node for the key (the current step)
			// and the new node (will be overwritten later)
			target.Content = append(target.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: headKey,
				Tag:   "!!str",
			}, node)
		}

		newNode, ok := createNode(newValue)
		if !ok {
			return nil, false
		}

		// update the node
		*node = *newNode

		return root, true
	}

	return nil, false
}

func mergeIntoPath(root *yaml.Node, path Path, newValue interface{}) bool {
	node, exists := traversePath(root, path)
	if !exists {
		// exit early if there is nothing fancy to do
		_, ok := setValue(root, path, newValue)
		return ok
	}

	if slice, ok := newValue.([]interface{}); ok {
		for idx, value := range slice {
			mergeIntoPath(node, Path{idx}, value)
		}
	} else if mappy, ok := newValue.(map[string]interface{}); ok {
		for k, v := range mappy {
			mergeIntoPath(node, Path{k}, v)
		}
	} else {
		setValue(node, nil, newValue)
	}

	return true
}

func scalarNodeToGo(node *yaml.Node) (interface{}, bool) {
	if node.Kind != yaml.ScalarNode {
		return nil, false
	}

	var data interface{}
	if err := node.Decode(&data); err != nil {
		return nil, false
	}

	return data, true
}

func sequenceNodeToGo(node *yaml.Node) ([]interface{}, bool) {
	if node.Kind != yaml.SequenceNode {
		return nil, false
	}

	var data []interface{}
	if err := node.Decode(&data); err != nil {
		return nil, false
	}

	return data, true
}

func nodeToString(node *yaml.Node) (string, bool) {
	val, ok := scalarNodeToGo(node)
	if s, isString := val.(string); isString && ok {
		return s, true
	}

	return "", false
}

func nodeToInt(node *yaml.Node) (int, bool) {
	val, ok := scalarNodeToGo(node)
	if i, isInt := val.(int); isInt && ok {
		return i, true
	}

	return 0, false
}

func nodeToBool(node *yaml.Node) (bool, bool) {
	val, ok := scalarNodeToGo(node)
	if b, isBool := val.(bool); isBool && ok {
		return b, true
	}

	return false, false
}
