package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ConfigMapCreator defines an interface to create/update ConfigMap's
type ConfigMapCreator = func(data *TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error)

// SecretCreator defines an interface to create/update Secret's
type SecretCreator = func(data *TemplateData, existing *corev1.Secret) (*corev1.Secret, error)

// StatefulSetCreator defines an interface to create/update StatefulSet
type StatefulSetCreator = func(data *TemplateData, existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error)

// ServiceCreator defines an interface to create/update Services
type ServiceCreator = func(data *TemplateData, existing *corev1.Service) (*corev1.Service, error)
