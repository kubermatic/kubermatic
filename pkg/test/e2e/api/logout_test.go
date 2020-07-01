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
	"net/http"
	"testing"

	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/credentials"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/datacenter"
	"github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/client/project"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestLogout(t *testing.T) {
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
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var masterToken string
			var err error

			if tc.isAdmin {
				masterToken, err = retrieveAdminMasterToken()
			} else {
				masterToken, err = retrieveMasterToken()
			}
			if err != nil {
				t.Fatalf("can not get master token due error: %v", err)
			}

			apiRunner := createRunner(masterToken, t)
			if err := apiRunner.Logout(); err != nil {
				t.Fatal(err)
			}

			_, err = apiRunner.CreateProject(rand.String(10))
			if err == nil {
				t.Fatalf("create project: expected error")
			}
			if _, ok := err.(*project.CreateProjectUnauthorized); !ok {
				t.Fatalf("create project: expected unauthorized error code")
			}

			_, err = apiRunner.ListDC()
			if err == nil {
				t.Fatalf("list datacenter: expected error")
			}
			rawListDatacentersDefaultErr, ok := err.(*datacenter.ListDatacentersDefault)
			if !ok {
				t.Fatalf("list datacenter: expected error")
			}
			if rawListDatacentersDefaultErr.Code() != http.StatusUnauthorized {
				t.Fatalf("list datacenter: expected unauthorized error code")
			}
			_, err = apiRunner.ListCredentials("gcp", "gcp-westeurope")
			if err == nil {
				t.Fatalf("list credentials: expected error")
			}
			rawListCredentialsDefaultErr, ok := err.(*credentials.ListCredentialsDefault)
			if !ok {
				t.Fatalf("list credentials: expected error")
			}
			if rawListCredentialsDefaultErr.Code() != http.StatusUnauthorized {
				t.Fatalf("list credentials: expected unauthorized error code")
			}

		})
	}
}
