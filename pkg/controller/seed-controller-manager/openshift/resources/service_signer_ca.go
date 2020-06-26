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

package resources

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

const ServiceSignerCASecretName = "service-signer-ca"

// ServiceSignerCA is Openshift-specific CA used to create serving certs for workloads on-demand
// See https://github.com/openshift/openshift-docs/pull/2324/files
func ServiceSignerCA() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return ServiceSignerCASecretName, certificates.GetCACreator("service-signer-ca")
	}
}
