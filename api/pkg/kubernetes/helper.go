package kubernetes

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
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

// AddFinalizer will add the given finalizer to the object. It uses a StringSet to avoid duplicates
func AddFinalizer(finalizers []string, toAdd string) []string {
	set := sets.NewString(finalizers...)
	set.Insert(toAdd)
	return set.List()
}

// GenerateToken generates a new, random token that can be used
// as an admin and kubelet token.
func GenerateToken() string {
	return fmt.Sprintf("%s.%s", rand.String(6), rand.String(16))
}
