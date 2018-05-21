package rbac

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

func TestGenerateRBACRole(t *testing.T) {
	tests := []struct {
		name          string
		projectToSync *kubermaticv1.Project
		existingUser  *kubermaticv1.User
		expectedUser  *kubermaticv1.User
	}{

		{
			name: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
		})
	}
}
