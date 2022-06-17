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
	"fmt"
	"strings"
)

type Step interface{}

type Path []Step

func (p Path) Append(s Step) Path {
	return append(p, s)
}

// Parent returns the path except for the last element.
func (p Path) Parent() Path {
	if len(p) < 1 {
		return nil
	}

	return p[0 : len(p)-1]
}

func (p Path) End() Step {
	if len(p) == 0 {
		return nil
	}

	return p[len(p)-1]
}

func (p Path) String() string {
	parts := []string{}

	for _, p := range p {
		if s, ok := p.(string); ok {
			parts = append(parts, s)
			continue
		}

		if i, ok := p.(int); ok {
			parts = append(parts, fmt.Sprintf("[%d]", i))
			continue
		}

		parts = append(parts, fmt.Sprintf("%v", p))
	}

	return strings.Join(parts, ".")
}
