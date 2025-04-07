//go:build integration

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

package vsphere

import (
	"context"
	"path"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestProvider_GetVMFolders(t *testing.T) {
	tests := []struct {
		name           string
		dc             *kubermaticv1.DatacenterSpecVSphere
		expectedFolder string
	}{
		{
			name: "successfully-create-and-list-folders-below-custom-root",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				Datacenter:    vSphereDatacenter,
				Endpoint:      vSphereEndpoint,
				AllowInsecure: true,
				RootPath:      path.Join("/", vSphereDatacenter, vSphereVMRootFolder),
			},
			expectedFolder: path.Join("/", vSphereDatacenter, vSphereVMRootFolder, generateTestFolder()),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			session, err := newSession(ctx, test.dc, vSphereUsername, vSpherePassword, nil)
			if err != nil {
				t.Fatal(err)
			}

			restSession, err := newRESTSession(ctx, test.dc, vSphereUsername, vSpherePassword, nil)
			if err != nil {
				t.Fatal("failed to create REST client session: %w", err)
			}
			defer restSession.Logout(ctx)

			if err := ensureVMFolder(ctx, session, restSession, test.expectedFolder, nil); err != nil {
				t.Fatal(err)
			}
			folders, err := GetVMFolders(ctx, test.dc, vSphereUsername, vSpherePassword, nil)
			if err != nil {
				t.Fatal(err)
			}

			folderFound := false
			gotFolders := sets.New[string]()
			for _, folder := range folders {
				if folder.Path == test.expectedFolder {
					folderFound = true
					if err := deleteVMFolder(ctx, session, test.expectedFolder); err != nil {
						t.Fatal(err)
					}
				}
			}

			if !folderFound {
				t.Fatalf("Response is missing expected folders:\n%v", diff.SetDiff(sets.New(test.expectedFolder), gotFolders))
			}
		})
	}
}

func generateTestFolder() string {
	return "kubermatic-e2e-" + rand.String(8)
}
