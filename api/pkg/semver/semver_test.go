package semver

import (
	"bytes"
	"testing"
)

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		inputSemver    *Semver
		expectedResult []byte
	}{
		{
			name:           "simple semver struct",
			inputSemver:    NewSemverOrDie("v1.0.0"),
			expectedResult: []byte("\"1.0.0\""),
		},
		{
			name:           "simple semver struct 2",
			inputSemver:    NewSemverOrDie("v2.1.0"),
			expectedResult: []byte("\"2.1.0\""),
		},
		{
			name:           "simple semver struct 3",
			inputSemver:    NewSemverOrDie("v3.2.1"),
			expectedResult: []byte("\"3.2.1\""),
		},
		{
			name:           "no-v semver",
			inputSemver:    NewSemverOrDie("4.3.2"),
			expectedResult: []byte("\"4.3.2\""),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := tc.inputSemver.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Compare(tc.expectedResult, b) != 0 {
				t.Errorf("expected to get %s, but got %s", string(tc.expectedResult), string(b))
			}

			var s Semver
			err = s.UnmarshalJSON(b)
			if err != nil {
				t.Fatal(err)
			}
			if s.Compare(tc.inputSemver.Semver()) != 0 {
				t.Errorf("expected to get %s, but got %s", tc.inputSemver.String(), s.String())
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		inputByte      []byte
		expectedSemver *Semver
	}{
		{
			name:           "simple semver struct",
			inputByte:      []byte("\"1.0.0\""),
			expectedSemver: NewSemverOrDie("v1.0.0"),
		},
		{
			name:           "simple semver struct 2",
			inputByte:      []byte("\"2.1.0\""),
			expectedSemver: NewSemverOrDie("v2.1.0"),
		},
		{
			name:           "simple semver struct 3",
			inputByte:      []byte("\"3.2.1\""),
			expectedSemver: NewSemverOrDie("v3.2.1"),
		},
		{
			name:           "no-v semver",
			inputByte:      []byte("\"4.3.2\""),
			expectedSemver: NewSemverOrDie("4.3.2"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var s Semver
			err := s.UnmarshalJSON(tc.inputByte)
			if err != nil {
				t.Fatal(err)
			}
			if s.Compare(tc.expectedSemver.Semver()) != 0 {
				t.Errorf("expected to get %s, but got %s", tc.expectedSemver.String(), s.String())
			}

			b, err := s.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Compare(b, tc.inputByte) != 0 {
				t.Errorf("epxected to get %s, but got %s", string(tc.inputByte), string(b))
			}
		})
	}
}
