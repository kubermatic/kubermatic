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

package flagopts

import "testing"

func TestStringArray_Set(t *testing.T) {
	tests := []struct {
		name    string
		s       *StringArray
		args    string
		wantErr bool
	}{
		{
			name: "simple",
			args: "hello",
			s:    &StringArray{"hello"},
		},
		{
			name: "empty",
			args: "",
			s:    &StringArray{},
		},
		{
			name: "few of them",
			args: "hello,world",
			s:    &StringArray{"hello", "world"},
		},
		{
			name: "with gaps",
			args: "hello,,,world",
			s:    &StringArray{"hello", "world"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Set(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("StringArray.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
