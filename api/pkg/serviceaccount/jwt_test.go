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

package serviceaccount_test

import (
	"fmt"
	"testing"
	"time"

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

			token, err := tokenGenerator.Generate(serviceaccount.Claims(tc.expectedEmail, tc.expectedProject, tc.expectedToken))
			if err != nil {
				t.Fatal(err)
			}

			tokenAuthenticator := serviceaccount.JWTTokenAuthenticator([]byte(test.TestServiceAccountHashKey))
			public, custom, err := tokenAuthenticator.Authenticate(token)
			if err != nil {
				t.Fatal(err)
			}

			if custom.Email != tc.expectedEmail {
				t.Fatalf("expected email %s got %s", tc.expectedEmail, custom.Email)
			}

			if custom.ProjectID != tc.expectedProject {
				t.Fatalf("expected project %s got %s", tc.expectedProject, custom.ProjectID)
			}

			if custom.TokenID != tc.expectedToken {
				t.Fatalf("expected token %s got %s", tc.expectedToken, custom.TokenID)
			}

			threeYearsString := formatTime(serviceaccount.Now().AddDate(3, 0, 0))
			expiryString := formatTime(public.Expiry.Time())

			if threeYearsString != expiryString {
				t.Fatalf("expected expire after 3 years from Now. Expected %s got %s", threeYearsString, expiryString)
			}

		})
	}
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("%d-%02d-%02d",
		t.Year(), t.Month(), t.Day())
}
