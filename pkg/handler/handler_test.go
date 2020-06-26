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

package handler

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEncodeJSON(t *testing.T) {
	var empty error

	testcases := []struct {
		input    interface{}
		expected string
	}{
		{nil, `{}`},
		{empty, `{}`},
		{[]int{}, `[]`},
		{[]int{1, 2, 3}, `[1,2,3]`},
		{12, `12`},
	}

	ctx := context.TODO()

	for _, testcase := range testcases {
		writer := httptest.NewRecorder()

		err := encodeJSON(ctx, writer, testcase.input)
		if err != nil {
			t.Errorf("failed to encode %#v as JSON: %v", testcase.input, err)
		}

		encoded, err := ioutil.ReadAll(writer.Body)
		if err != nil {
			t.Fatal("unable to read response body")
		}

		trimmed := strings.TrimSpace(string(encoded))

		if trimmed != testcase.expected {
			t.Errorf("expected to encode %#v as '%s', but got '%s'", testcase.input, testcase.expected, trimmed)
		}
	}
}
