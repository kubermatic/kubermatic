package kubernetes

import (
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
