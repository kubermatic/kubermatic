package resources

import (
	corev1 "k8s.io/api/core/v1"
)

type ConfigMapCreator = func(data *TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error)

type SecretCreator = func(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error)
