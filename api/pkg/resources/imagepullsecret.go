package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImagePullSecretCreator returns a creator function to create a ImagePullSecret
func ImagePullSecretCreator(name string, dockerPullConfigJSON []byte) func(data SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
	return func(data SecretDataProvider, existing *corev1.Secret) (*corev1.Secret, error) {
		var secret *corev1.Secret
		if existing != nil {
			secret = existing
		} else {
			secret = &corev1.Secret{}
		}

		secret.Name = name
		secret.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

		secret.Type = corev1.SecretTypeDockerConfigJson

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		secret.Data[corev1.DockerConfigJsonKey] = dockerPullConfigJSON

		return secret, nil
	}
}
