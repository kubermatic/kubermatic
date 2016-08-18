package fixchain

import "testing"

func TestTypeString(t *testing.T) {
	fixErrorTests := []struct {
		ferr     FixError
		expected string
	}{
		{
			FixError{Type: None},
			"None",
		},
		{
			FixError{Type: ParseFailure},
			"ParseFailure",
		},
		{
			FixError{Type: CannotFetchURL},
			"CannotFetchURL",
		},
		{
			FixError{Type: FixFailed},
			"FixFailed",
		},
		{
			FixError{Type: LogPostFailed},
			"LogPostFailed",
		},
		{
			FixError{Type: VerifyFailed},
			"VerifyFailed",
		},
		{
			FixError{},
			"None",
		},
	}

	for _, test := range fixErrorTests {
		if got, want := test.ferr.TypeString(), test.expected; got != want {
			t.Errorf("TypeString() returned %s, expected %s.", got, want)
		}
	}
}
