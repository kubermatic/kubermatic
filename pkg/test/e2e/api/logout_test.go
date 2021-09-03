// +build logout

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

package api

import (
	"context"
	"net/http"
	"testing"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/credentials"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/datacenter"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestLogout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		isAdmin bool
	}{
		{
			name:    "test endpoints after logout",
			isAdmin: false,
		},
		{
			name:    "test admin endpoints after logout",
			isAdmin: true,
		},
	}

	ctx := context.Background()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var masterToken string
			var err error

			if tc.isAdmin {
				masterToken, err = utils.RetrieveAdminMasterToken(ctx)
			} else {
				masterToken, err = utils.RetrieveMasterToken(ctx)
			}
			if err != nil {
				t.Fatalf("failed to get master token: %v", err)
			}

			testClient := utils.NewTestClient(masterToken, t)

			// logout
			if err := testClient.Logout(); err != nil {
				t.Fatal(err)
			}

			// test projection creation
			_, err = testClient.CreateProjectWithoutChecks(rand.String(10))
			if err == nil {
				t.Fatal("create project: expected error")
			}
			if _, ok := err.(*project.CreateProjectUnauthorized); !ok {
				t.Fatalf("create project: expected unauthorized error code, but got %#v", err)
			}

			// test listing datacenters
			_, err = testClient.ListDC()
			if err == nil {
				t.Fatal("list datacenter: expected error")
			}
			dcErr, ok := err.(*datacenter.ListDatacentersDefault)
			if !ok {
				t.Fatalf("list datacenter: expected error")
			}
			if dcErr.Code() != http.StatusUnauthorized {
				t.Fatalf("list datacenter: expected unauthorized error code, but got %v", dcErr.Code())
			}

			// test listing credentials
			_, err = testClient.ListCredentials("gcp", "gcp-westeurope")
			if err == nil {
				t.Fatal("list credentials: expected error")
			}
			credentialErr, ok := err.(*credentials.ListCredentialsDefault)
			if !ok {
				t.Fatalf("list credentials: expected error")
			}
			if credentialErr.Code() != http.StatusUnauthorized {
				t.Fatalf("list credentials: expected unauthorized error code, but got %v", credentialErr.Code())
			}
		})
	}
}
