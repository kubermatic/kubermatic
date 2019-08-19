// +build integration

package vsphere

import (
	"context"
	"path"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/rand"
)

func TestCreateVMFolder(t *testing.T) {
	dc := &kubermaticv1.DatacenterSpecVSphere{
		Datacenter: vSphereDatacenter,
		Endpoint:   vSphereEndpoint,
		RootPath:   path.Join("/", vSphereDatacenter, "vm"),
	}

	cloudSpec := kubermaticv1.CloudSpec{
		VSphere: &kubermaticv1.VSphereCloudSpec{
			InfraManagementUser: kubermaticv1.VSphereCredentials{
				Username: vSphereUsername,
				Password: vSpherePassword,
			},
		},
	}

	p := &Provider{dc: dc}
	client, err := p.getClient(cloudSpec)
	if err != nil {
		t.Fatal(err)
	}

	// Use a unique ID to prevent side effects when running this test concurrently
	id := "kubermatic-e2e-" + rand.String(8)
	folder := path.Join(dc.RootPath, id)

	// Cheap way to test idempotency
	for i := 0; i < 2; i++ {
		if err := createVMFolder(context.Background(), client, folder); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := deleteVMFolder(context.Background(), client, folder); err != nil {
			t.Fatal(err)
		}
	}
}
