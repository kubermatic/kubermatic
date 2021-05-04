/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package prometheus

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

func ClientCertificateCreator(ca *resources.ECDSAKeyPair) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.UserClusterPrometheusCertificatesSecretName,
			certificates.GetECDSAClientCertificateCreator(
				resources.UserClusterPrometheusCertificatesSecretName,
				resources.UserClusterPrometheusCertificateCommonName,
				[]string{},
				resources.UserClusterPrometheusClientCertSecretKey,
				resources.UserClusterPrometheusClientKeySecretKey,
				func() (*resources.ECDSAKeyPair, error) { return ca, nil })
	}
}
