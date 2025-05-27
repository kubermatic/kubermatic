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
	"os"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

var (
	vSphereDatacenter = os.Getenv("VSPHERE_E2E_TEST_DATACENTER")
	vSphereEndpoint   = os.Getenv("VSPHERE_E2E_ADDRESS")
	vSphereUsername   = os.Getenv("VSPHERE_E2E_USERNAME")
	vSpherePassword   = os.Getenv("VSPHERE_E2E_PASSWORD")

	vSphereVMRootFolder = "vm/Kubermatic-dev"
)

func getTestDC() *kubermaticv1.DatacenterSpecVSphere {
	return &kubermaticv1.DatacenterSpecVSphere{
		Endpoint:      vSphereEndpoint,
		AllowInsecure: true,
		Datacenter:    vSphereDatacenter,
	}
}
