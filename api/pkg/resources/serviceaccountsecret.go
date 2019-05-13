package resources

import (
	"encoding/base64"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountSecretCreator returns a creator function to create a Google Service Account.
func ServiceAccountSecretCreator(serviceAccount string) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return GoogleServiceAccountSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			b, err := base64.StdEncoding.DecodeString(serviceAccount)
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
