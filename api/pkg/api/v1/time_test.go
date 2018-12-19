package v1_test

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

type TimeHolder struct {
	T Time `json:"t"`
}

func TestTimeMarshalJSON(t *testing.T) {
	cases := []struct {
		input  Time
		result string
	}{
		{Time{}, "{\"t\":\"0001-01-01T00:00:00Z\"}"},
		{Date(1998, time.May, 5, 5, 5, 5, 50, time.UTC), "{\"t\":\"1998-05-05T05:05:05Z\"}"},
		{Date(1998, time.May, 5, 5, 5, 5, 0, time.UTC), "{\"t\":\"1998-05-05T05:05:05Z\"}"},
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
		result Time
	}{
		{"{\"t\":\"0001-01-01T00:00:00Z\"}", Time{}},
		{"{\"t\":\"1998-05-05T05:05:05Z\"}", NewTime(Date(1998, time.May, 5, 5, 5, 5, 0, time.UTC).Local())},
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
