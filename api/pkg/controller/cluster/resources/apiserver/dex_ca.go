package apiserver

import (
	"encoding/pem"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DexCACertificate returns a secret with the certificate for TLS verification against dex
func DexCACertificate(data resources.SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
	var se *corev1.Secret
	if existing != nil {
		se = existing
	} else {
		se = &corev1.Secret{}
	}
	se.Name = resources.DexCASecretName
	se.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	ca, err := data.GetDexCA()
	if err != nil {
		return nil, fmt.Errorf("failed to get dex public keys: %v", err)
	}

	if se.Data == nil {
		se.Data = map[string][]byte{}
	}

	if _, exists := se.Data[resources.DexCAFileName]; exists {
		return se, nil
	}

	var cert []byte
	for _, certRaw := range ca {
		block := pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certRaw.Raw,
		}
		cert = append(cert, pem.EncodeToMemory(&block)...)
	}

	se.Data[resources.DexCAFileName] = cert
	return se, nil
}
