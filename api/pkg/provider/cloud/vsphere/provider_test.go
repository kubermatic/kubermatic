// +build e2e

package vsphere

import (
	"os"
	"path"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/sets"
)

var (
	vSphereDatacenter = os.Getenv("TEST_VSPHERE_DATACENTER")
	vSphereEndpoint   = os.Getenv("TEST_VSPHERE_ENDPOINT")
	vSphereUsername   = os.Getenv("TEST_VSPHERE_USERNAME")
	vSpherePassword   = os.Getenv("TEST_VSPHERE_PASSWORD")
)

func TestProvider_GetVMFolders(t *testing.T) {
	tests := []struct {
		name            string
		dc              *kubermaticv1.DatacenterSpecVSphere
		expectedFolders sets.String
	}{
		{
			name: "successfully-list-default-folders",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				Datacenter: vSphereDatacenter,
				Endpoint:   vSphereEndpoint,
			},
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
			p := &Provider{dc: test.dc}

			cloudSpec := kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					InfraManagementUser: kubermaticv1.VSphereCredentials{
						Username: vSphereUsername,
						Password: vSpherePassword,
					},
				},
			}

			folders, err := p.GetVMFolders(cloudSpec)
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
			if diff := gotFolders.Difference(test.expectedFolders); diff.Len() > 0 {
				t.Errorf("Response contains unexpected folders: %v", diff.List())
			}
		})
	}
}
