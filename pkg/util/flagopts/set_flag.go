/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package flagopts

import (
	"strings"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/util/sets"
)

// SetFlag wraps a given set so it can be used as a CLI flag.
func SetFlag[T ~string](set sets.Set[T]) pflag.Value {
	return &setFlag[T]{set: set}
}

type setFlag[T ~string] struct {
	set sets.Set[T]
}

func (f *setFlag[T]) String() string {
	values := []string{}
	for _, val := range sets.List(f.set) {
		values = append(values, string(val))
	}

	return strings.Join(values, ",")
}

func (f *setFlag[T]) Set(value string) error {
	// clear set content
	f.set.Delete(f.set.UnsortedList()...)

	if value != "" {
		for _, val := range strings.Split(value, ",") {
			val = strings.TrimSpace(val)
			if val != "" {
				f.set.Insert(T(val))
			}
		}
	}

	return nil
}

func (f *setFlag[T]) Type() string {
	return "strings" // see pflag's flag.go
}
