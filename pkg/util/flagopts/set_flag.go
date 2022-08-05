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
	"flag"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// SetFlag wraps a given set so it can be used as a CLI flag.
func SetFlag(set sets.String) flag.Value {
	return &setFlag{set: set}
}

type setFlag struct {
	set sets.String
}

func (f *setFlag) String() string {
	return strings.Join(f.set.List(), ",")
}

func (f *setFlag) Set(value string) error {
	// clear set content
	f.set.Delete(f.set.UnsortedList()...)

	if value != "" {
		for _, val := range strings.Split(value, ",") {
			val = strings.TrimSpace(val)
			if val != "" {
				f.set.Insert(val)
			}
		}
	}

	return nil
}
