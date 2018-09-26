package kubernetes

import (
	"testing"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// Tests creating a selector with an invalid user id
func TestSelector(t *testing.T) {
	tests := []struct {
		name             string
		id               string
		expectedSelector string
		err              string
	}{
		{
			name:             "valid selector",
			id:               "123456",
			expectedSelector: "user=123456",
			err:              "",
		},
		{
			name:             "id exceeds max length",
			id:               "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
			expectedSelector: "",
			err:              "invalid label value: \"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\": must be no more than 63 characters",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selector := labels.NewSelector()
			req, err := labels.NewRequirement(userLabelKey, selection.Equals, []string{test.id})
			if err != nil {
				if test.err == "" {
					t.Errorf("expected no error when creating the requirement, but got one: %v", err)
				}

				if test.err != err.Error() {
					t.Errorf("expected error %s, got %s", test.err, err.Error())
				}

				return
			}
			selector = selector.Add(*req)

			if selector.String() != test.expectedSelector {
				t.Errorf("expected %s, got %s", test.expectedSelector, selector.String())
			}
		})
	}
}
