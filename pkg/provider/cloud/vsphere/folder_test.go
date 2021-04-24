// +build integration

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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"

	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestCreateVMFolder(t *testing.T) {
	dc := &kubermaticv1.DatacenterSpecVSphere{
		Datacenter: vSphereDatacenter,
		Endpoint:   vSphereEndpoint,
		RootPath:   path.Join("/", vSphereDatacenter, "vm"),
	}

	ctx := context.Background()
	session, err := newSession(ctx, dc, vSphereUsername, vSpherePassword, certificates.NewFakeCABundle().CertPool())
	if err != nil {
		t.Fatal(err)
	}

	// Use a unique ID to prevent side effects when running this test concurrently
	id := "kubermatic-e2e-" + rand.String(8)
	folder := path.Join(dc.RootPath, id)

	// Cheap way to test idempotency
	for i := 0; i < 2; i++ {
		if err := createVMFolder(ctx, session, folder); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := deleteVMFolder(ctx, session, folder); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProvider_GetVMFolders(t *testing.T) {
	tests := []struct {
		name            string
		dc              *kubermaticv1.DatacenterSpecVSphere
		expectedFolders sets.String
		// If we check for folders on the root we might have other tests creating folders, so we have to skip that check
		skipAdditionalFoldersCheck bool
	}{
		{
			name:                       "successfully-list-default-folders",
			skipAdditionalFoldersCheck: true,
			dc:                         getTestDC(),
			expectedFolders: sets.NewString(
				path.Join("/", vSphereDatacenter, "vm"),
				path.Join("/", vSphereDatacenter, "vm", "kubermatic-e2e-tests"),
				path.Join("/", vSphereDatacenter, "vm", "kubermatic-e2e-tests", "test-1"),
				path.Join("/", vSphereDatacenter, "vm", "kubermatic-e2e-tests-2"),
			),
		},
		{
			name: "successfully-list-folders-below-custom-root",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				Datacenter: vSphereDatacenter,
				Endpoint:   vSphereEndpoint,
				RootPath:   path.Join("/", vSphereDatacenter, "vm", "kubermatic-e2e-tests"),
			},
			expectedFolders: sets.NewString(
				path.Join("/", vSphereDatacenter, "vm", "kubermatic-e2e-tests"),
				path.Join("/", vSphereDatacenter, "vm", "kubermatic-e2e-tests", "test-1"),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			folders, err := GetVMFolders(test.dc, vSphereUsername, vSpherePassword, certificates.NewFakeCABundle().CertPool())
			if err != nil {
				t.Fatal(err)
			}

			gotFolders := sets.NewString()
			for _, folder := range folders {
				gotFolders.Insert(folder.Path)
			}
			t.Logf("Got folders: %v", gotFolders.List())

			if diff := test.expectedFolders.Difference(gotFolders); diff.Len() > 0 {
				t.Errorf("Response is missing expected folders: %v", diff.List())
			}
			if !test.skipAdditionalFoldersCheck {
				if diff := gotFolders.Difference(test.expectedFolders); diff.Len() > 0 {
					t.Errorf("Response contains unexpected folders: %v", diff.List())
				}
			}
		})
	}
}
