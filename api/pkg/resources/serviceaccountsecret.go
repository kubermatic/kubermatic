package resources

import (
	"encoding/base64"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountSecretCreator returns a creator function to create a Google Service Account.
func ServiceAccountSecretCreator(data CredentialsData) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return GoogleServiceAccountSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			credentials, err := GetCredentials(data)
			if err != nil {
				return nil, err
			}

			b, err := base64.StdEncoding.DecodeString(credentials.GCP.ServiceAccount)
			if err != nil {
				return nil, fmt.Errorf("error decoding service account: %v", err)
			}

			se.Type = corev1.SecretTypeOpaque

			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			se.Data["serviceAccount"] = b

			return se, nil
		}
	}
}
