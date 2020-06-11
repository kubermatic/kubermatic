package resources

import (
	"fmt"

	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const ExternalX509KubeconfigName = "kubermatic-cluster-admin-secret"

func ExternalX509KubeconfigCreator(data openshiftData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return ExternalX509KubeconfigName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			b := secret.Data[resources.KubeconfigSecretKey]
			ca, err := data.GetRootCA()
			if err != nil {
				return nil, fmt.Errorf("failed to get root ca: %v", err)
			}
			url := data.Cluster().Address.URL
			cn := ExternalX509KubeconfigName
			organizations := []string{"system:masters"}
			valid, err := resources.IsValidKubeconfig(b, ca.Cert, url, cn, organizations, data.Cluster().Name)
			if err != nil {
				return nil, fmt.Errorf("failed to validate kubeconfig: %v", err)
			}
			if valid {
				return secret, nil
			}
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}
			secret.Data[resources.KubeconfigSecretKey], err = resources.BuildNewKubeconfigAsByte(ca, url, cn, organizations, data.Cluster().Name)
			if err != nil {
				return nil, fmt.Errorf("failed to build kubeconfig: %v", err)
			}
			return secret, nil
		}
	}
}
