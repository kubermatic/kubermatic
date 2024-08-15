/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package ini

import (
	"bytes"
	"testing"

	"gopkg.in/gcfg.v1"
)

type globalSection struct {
	Password string
}

type testData struct {
	Global globalSection
}

var testStrings = []string{
	"test",
	`foo"bar"`,
	"foo`bar",
	"\\hello\\world\\",
	"another 'test' `string`",
}

func TestQuote(t *testing.T) {
	for _, testString := range testStrings {
		t.Run(testString, func(t *testing.T) {
			// generate an ini string
			f := New()

			s := f.Section("Global", "")
			s.AddStringKey("Password", testString)

			var buf bytes.Buffer
			if err := f.Render(&buf); err != nil {
				t.Fatal(err)
			}

			// parse it back (using the most commonly used ini library among CCM/CSIs)
			parsed := &testData{}
			if err := gcfg.ReadStringInto(parsed, buf.String()); err != nil {
				t.Fatalf("failed to load string into config object: %v", err)
			}

			if testString != parsed.Global.Password {
				t.Fatalf("after unmarshalling the config into a string an reading it back in, the value changed. Password before:\n%s Password after:\n%s", testString, parsed.Global.Password)
			}
		})
	}
}
