package apiserver

import (
	"bytes"
	"encoding/csv"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TokenUsers returns a secret containing the tokens csv
func TokenUsers(data *resources.TemplateData, existing *corev1.Secret) (*corev1.Secret, error) {
	var se *corev1.Secret
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Secret{}
	}

	se.Name = resources.TokensSecretName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	if se.Data == nil {
		se.Data = map[string][]byte{}
	}

	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	if err := writer.Write([]string{data.Cluster.Address.AdminToken, "admin", "10000", "system:masters"}); err != nil {
		return nil, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	se.Data[resources.TokensSecretKey] = buffer.Bytes()

	return se, nil
}
