package serviceaccount_test

import (
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/handler/test"
	"github.com/kubermatic/kubermatic/api/pkg/serviceaccount"
)

func TestServiceAccountIssuer(t *testing.T) {

	testcases := []struct {
		name            string
		expectedEmail   string
		expectedProject string
		expectedToken   string
	}{
		{
			name:            "scenario 1, check signed token",
			expectedEmail:   "test@example.com",
			expectedProject: "testProject",
			expectedToken:   "testToken",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			tokenGenerator, err := serviceaccount.JWTTokenGenerator([]byte(test.TestServiceAccountHashKey))
			if err != nil {
				t.Fatal(err)
			}

			token, err := tokenGenerator.GenerateToken(serviceaccount.Claims(tc.expectedEmail, tc.expectedProject, tc.expectedToken))
			if err != nil {
				t.Fatal(err)
			}

			tokenAuthenticator := serviceaccount.JWTTokenAuthenticator([]byte(test.TestServiceAccountHashKey))
			result, err := tokenAuthenticator.AuthenticateToken(token)
			if err != nil {
				t.Fatal(err)
			}

			if result.Email != tc.expectedEmail {
				t.Fatalf("expected email %s got %s", tc.expectedEmail, result.Email)
			}

			if result.ProjectID != tc.expectedProject {
				t.Fatalf("expected project %s got %s", tc.expectedProject, result.ProjectID)
			}

			if result.TokenID != tc.expectedToken {
				t.Fatalf("expected token %s got %s", tc.expectedToken, result.TokenID)
			}

		})
	}
}
