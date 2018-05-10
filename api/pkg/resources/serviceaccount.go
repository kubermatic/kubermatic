package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccount returns a service account with the given name and sets the owner reference.
// If existing is set, it will get updated
func ServiceAccount(name string, owner *metav1.OwnerReference, existing *corev1.ServiceAccount) *corev1.ServiceAccount {
	var sa *corev1.ServiceAccount
	if existing != nil {
		sa = existing
	} else {
		sa = &corev1.ServiceAccount{}
	}

	sa.Name = name
	sa.OwnerReferences = []metav1.OwnerReference{*owner}
	return sa
}
