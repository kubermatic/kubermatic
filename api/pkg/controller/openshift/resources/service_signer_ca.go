package resources

import (
	"context"

	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates"
)

// ServiceSignerCA is Openshift-specific CA used to create serving certs for workloads on-demand
// See https://github.com/openshift/openshift-docs/pull/2324/files
func ServiceSignerCA(_ context.Context, od openshiftData) (string, SecretCreator) {
	return ServiceSignerCASecretName, certificates.GetCACreator(ServiceSignerCASecretName, "service-signer-ca")
}
