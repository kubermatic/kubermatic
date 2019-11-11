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
