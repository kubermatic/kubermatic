package kubernetes

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HasFinalizer tells if a object has the given finalizer
func HasFinalizer(o metav1.Object, name string) bool {
	for _, f := range o.GetFinalizers() {
		if f == name {
			return true
		}
	}
	return false
}

// RemoveFinalizer removes the given finalizer and returns the cleaned list
func RemoveFinalizer(finalizers []string, toRemove string) []string {
	for i, f := range finalizers {
		if f == toRemove {
			return append(finalizers[:i], finalizers[i+1:]...)
		}
	}
	return finalizers
}

// ToLabelValue returns the base64 encoded sha1 sum of s
func ToLabelValue(s string) string {
	sh := sha1.New()
	fmt.Fprint(sh, s)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sh.Sum(nil))
}
