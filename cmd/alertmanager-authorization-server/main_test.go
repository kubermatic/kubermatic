/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(testScheme))
}

func TestAuthorize(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                      string
		userEmail                 string
		clusterID                 string
		existingKubermaticObjects []ctrlruntimeclient.Object
		expectedError             bool
		expectedAuthorized        bool
	}{
		{
			name:      "user is authorized to access alertmanager of cluster which belongs to the user",
			userEmail: "bob@acme.com",
			clusterID: test.DefaultClusterID,
			existingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
			),
			expectedError:      false,
			expectedAuthorized: true,
		},
		{
			name:                      "authorization should fail if cluster doesn't exist",
			userEmail:                 "john@acme.com",
			clusterID:                 test.DefaultClusterID,
			existingKubermaticObjects: test.GenDefaultKubermaticObjects(),
			expectedError:             true,
			expectedAuthorized:        false,
		},
		{
			name:      "user is NOT authorized to access alertmanager of cluster which does NOT belong to the user",
			userEmail: "john@acme.com",
			clusterID: test.DefaultClusterID,
			existingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", false),
			),
			expectedError:      false,
			expectedAuthorized: false,
		},
		{
			name:      "admin user is authorized to access alertmanager of any clusters",
			userEmail: "john@acme.com",
			clusterID: test.DefaultClusterID,
			existingKubermaticObjects: test.GenDefaultKubermaticObjects(
				test.GenDefaultCluster(),
				test.GenAdminUser("John", "john@acme.com", true),
			),
			expectedError:      false,
			expectedAuthorized: true,
		},
	}

	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(tc.existingKubermaticObjects...).
				Build()

			server := authorizationServer{
				client: client,
				log:    log,
			}
			authorized, err := server.authorize(context.Background(), tc.userEmail, tc.clusterID)
			assert.Equal(t, tc.expectedAuthorized, authorized)
			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}
