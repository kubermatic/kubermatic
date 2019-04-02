package apiserver

import (
	"bytes"
	"encoding/csv"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// TokenUsers returns a secret containing the tokens csv
func TokenUsersCreator(data resources.SecretDataProvider) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.TokensSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			buffer := &bytes.Buffer{}
			writer := csv.NewWriter(buffer)

			if err := writer.Write([]string{data.Cluster().Address.AdminToken, "admin", "10000", "system:masters"}); err != nil {
				return nil, err
			}
			writer.Flush()
			if err := writer.Error(); err != nil {
				return nil, err
			}

			se.Data[resources.TokensSecretKey] = buffer.Bytes()

			return se, nil
		}
	}
}
