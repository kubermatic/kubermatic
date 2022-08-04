/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package scenarios

import (
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

func getOSNameFromSpec(spec apimodels.OperatingSystemSpec) providerconfig.OperatingSystem {
	if spec.Centos != nil {
		return providerconfig.OperatingSystemCentOS
	}
	if spec.Ubuntu != nil {
		return providerconfig.OperatingSystemUbuntu
	}
	if spec.Sles != nil {
		return providerconfig.OperatingSystemSLES
	}
	if spec.Rhel != nil {
		return providerconfig.OperatingSystemRHEL
	}
	if spec.Flatcar != nil {
		return providerconfig.OperatingSystemFlatcar
	}
	if spec.RockyLinux != nil {
		return providerconfig.OperatingSystemRockyLinux
	}

	return ""
}
