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
