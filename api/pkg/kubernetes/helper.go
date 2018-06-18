package kubernetes

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// HasFinalizer tells if a object has the given finalizer
func HasFinalizer(o metav1.Object, name string) bool {
	return sets.NewString(o.GetFinalizers()...).Has(name)
}

// RemoveFinalizer removes the given finalizer and returns the cleaned list
func RemoveFinalizer(finalizers []string, toRemove string) []string {
	set := sets.NewString(finalizers...)
	set.Delete(toRemove)
	return set.List()
}

// ToLabelValue returns the base64 encoded sha1 sum of s
func ToLabelValue(s string) string {
	sh := sha1.New()
	fmt.Fprint(sh, s)
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sh.Sum(nil))
}
