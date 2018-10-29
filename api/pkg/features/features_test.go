package features

import (
	"testing"
)

func TestFeatureGates(t *testing.T) {
	scenarios := []struct {
		name   string
		input  string
		output map[string]bool
	}{
		{
			name:  "scenario 1: happy path provides valid input and makes sure it was parsed correctly",
			input: "feature1=false,feature2=true",
			output: map[string]bool{
				"feature1": false,
				"feature2": true,
			},
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			target, err := NewFeatures(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			for feature, shouldBeEnabled := range tc.output {
				isEnabled := target.Enabled(feature)
				if isEnabled != shouldBeEnabled {
					t.Fatalf("expected feature = %s to be set to %v but was set to %v", feature, shouldBeEnabled, isEnabled)
				}
			}
		})
	}
}
