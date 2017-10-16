package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HasFinalizer(o metav1.Object, name string) bool {
	for _, f := range o.GetFinalizers() {
		if f == name {
			return true
		}
	}
	return false
}

func RemoveFinalizer(finalizers []string, toRemove string) []string {
	for i, f := range finalizers {
		if f == toRemove {
			return append(finalizers[:i], finalizers[i+1:]...)
		}
	}
	return finalizers
}
