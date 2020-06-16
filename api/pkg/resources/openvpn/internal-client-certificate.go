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

package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

type internalClientCertificateCreatorData interface {
	GetOpenVPNCA() (*resources.ECDSAKeyPair, error)
}

// InternalClientCertificateCreator returns a function to create/update the secret with a client certificate for the openvpn clients in the seed cluster.
func InternalClientCertificateCreator(data internalClientCertificateCreatorData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.OpenVPNClientCertificatesSecretName, certificates.GetECDSAClientCertificateCreator(
			resources.OpenVPNClientCertificatesSecretName,
			"internal-client",
			[]string{},
			resources.OpenVPNInternalClientCertSecretKey,
			resources.OpenVPNInternalClientKeySecretKey,
			data.GetOpenVPNCA,
		)
	}
}
