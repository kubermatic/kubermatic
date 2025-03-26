/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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
	"path"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

// getVMRootPath is a helper func to get the root path for VM's
// We extracted it because we use it in several places.
func getVMRootPath(dc *kubermaticv1.DatacenterSpecVSphere) string {
	// Each datacenter root directory for VMs is: ${DATACENTER_NAME}/vm
	rootPath := path.Join("/", dc.Datacenter, "vm")
	// We offer a different root path though in case people would like to store all Kubermatic VMs below a certain directory
	if dc.RootPath != "" {
		rootPath = path.Clean(dc.RootPath)
	}
	return rootPath
}
