package kubermatic

import (
	"fmt"
	"strings"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

func featureGates(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	features := make([]string, 0)
	for _, feature := range cfg.Spec.FeatureGates.List() {
		features = append(features, fmt.Sprintf("%s=true", feature))
	}

	return strings.Join(features, ",")
}

func createSecretData(s *corev1.Secret, data map[string]string) *corev1.Secret {
	if s.Data == nil {
		s.Data = make(map[string][]byte)
	}

	for k, v := range data {
		s.Data[k] = []byte(v)
	}

	return s
}
