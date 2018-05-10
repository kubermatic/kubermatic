package resources

import (
	corev1 "k8s.io/api/core/v1"
)

// ConfigMapCreator defines an interface to create/update ConfigMap's
type ConfigMapCreator = func(data *TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error)

// SecretCreator defines an interface to create/update Secret's
type SecretCreator = func(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error)
