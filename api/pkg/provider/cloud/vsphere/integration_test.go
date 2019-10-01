// +build integration

package vsphere

import (
	"os"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

var (
	vSphereDatacenter = os.Getenv("VSPHERE_E2E_TEST_DATACENTER")
	vSphereEndpoint   = os.Getenv("VSPHERE_E2E_ADDRESS")
	vSphereUsername   = os.Getenv("VSPHERE_E2E_USERNAME")
	vSpherePassword   = os.Getenv("VSPHERE_E2E_PASSWORD")
)

func getTestDC() *kubermaticv1.DatacenterSpecVSphere {
	return &kubermaticv1.DatacenterSpecVSphere{
		Datacenter: vSphereDatacenter,
		Endpoint:   vSphereEndpoint,
	}
}
