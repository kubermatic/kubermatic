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

package v1_test

import (
	"encoding/json"
	"testing"
	"time"

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
)

type TimeHolder struct {
	T apiv1.Time `json:"t"`
}

func TestTimeMarshalJSON(t *testing.T) {
	cases := []struct {
		input  apiv1.Time
		result string
	}{
		{apiv1.Time{}, "{\"t\":\"0001-01-01T00:00:00Z\"}"},
		{apiv1.Date(1998, time.May, 5, 5, 5, 5, 50, time.UTC), "{\"t\":\"1998-05-05T05:05:05Z\"}"},
		{apiv1.Date(1998, time.May, 5, 5, 5, 5, 0, time.UTC), "{\"t\":\"1998-05-05T05:05:05Z\"}"},
	}

	for _, c := range cases {
		input := TimeHolder{c.input}
		result, err := json.Marshal(&input)
		if err != nil {
			t.Errorf("Failed to marshal input: '%v': %v", input, err)
		}
		if string(result) != c.result {
			t.Errorf("Failed to marshal input: '%v': expected %+v, got %q", input, c.result, string(result))
		}
	}
}

func TestTimeUnmarshalJSON(t *testing.T) {
	cases := []struct {
		input  string
		result apiv1.Time
	}{
		{"{\"t\":\"0001-01-01T00:00:00Z\"}", apiv1.Time{}},
		{"{\"t\":\"1998-05-05T05:05:05Z\"}", apiv1.NewTime(apiv1.Date(1998, time.May, 5, 5, 5, 5, 0, time.UTC).Local())},
	}

	for _, c := range cases {
		var result TimeHolder
		if err := json.Unmarshal([]byte(c.input), &result); err != nil {
			t.Errorf("Failed to unmarshal input '%v': %v", c.input, err)
		}
		if result.T != c.result {
			t.Errorf("Failed to unmarshal input '%v': expected %+v, got %+v", c.input, c.result, result)
		}
	}
}
