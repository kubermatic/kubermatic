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

package openstack

import (
	"errors"
	"testing"

	"github.com/gophercloud/gophercloud"
)

func NewPointerEndpointNotFoundErr() error {
	return &gophercloud.ErrEndpointNotFound{}
}
func NewValueEndpointNotFoundErr() error {
	return gophercloud.ErrEndpointNotFound{}
}
func NewPointerNotFoundErr() error {
	return &gophercloud.ErrDefault404{}
}
func NewValueNotFoundErr() error {
	return &gophercloud.ErrDefault404{}
}

func Test_isEndpointNotFoundErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Correct endpoint not found error (as pointer)",
			err:  NewPointerEndpointNotFoundErr(),
			want: true,
		},
		{
			name: "Correct endpoint not found error (as value)",
			err:  NewValueEndpointNotFoundErr(),
			want: true,
		},
		{
			name: "Incorrect not found error (as pointer)",
			err:  NewPointerNotFoundErr(),
			want: false,
		},
		{
			name: "Incorrect not found error (as value)",
			err:  NewValueNotFoundErr(),
			want: false,
		},
		{
			name: "Incorrect different error",
			err:  errors.New("different one"),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEndpointNotFoundErr(tt.err); got != tt.want {
				t.Errorf("isEndpointNotFoundErr() = %v, want %v", got, tt.want)
			}
		})
	}
}
