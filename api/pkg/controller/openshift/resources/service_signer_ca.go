package resources

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
)

const ServiceSignerCASecretName = "service-signer-ca"

// ServiceSignerCA is Openshift-specific CA used to create serving certs for workloads on-demand
// See https://github.com/openshift/openshift-docs/pull/2324/files
func ServiceSignerCA() resources.NamedSecretCreatorGetter {
	return func() (string, resources.SecretCreator) {
		return ServiceSignerCASecretName, certificates.GetCACreator("service-signer-ca")
	}
}
