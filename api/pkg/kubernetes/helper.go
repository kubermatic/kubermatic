package kubernetes

import (
	"fmt"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
)

var tokenValidator = regexp.MustCompile(`[bcdfghjklmnpqrstvwxz2456789]{6}\.[bcdfghjklmnpqrstvwxz2456789]{16}`)

// HasFinalizer tells if a object has all the given finalizers
func HasFinalizer(o metav1.Object, names ...string) bool {
	return sets.NewString(o.GetFinalizers()...).HasAll(names...)
}

// HasOnlyFinalizer tells if an object has only the given finalizer
func HasOnlyFinalizer(o metav1.Object, name string) bool {
	set := sets.NewString(o.GetFinalizers()...)
	return set.Has(name) && set.Len() == 1
}

// RemoveFinalizer removes the given finalizer from the object
func RemoveFinalizer(obj metav1.Object, toRemove string) {
	set := sets.NewString(obj.GetFinalizers()...)
	set.Delete(toRemove)
	obj.SetFinalizers(set.List())
}

// AddFinalizer will add the given finalizer to the object. It uses a StringSet to avoid duplicates
func AddFinalizer(obj metav1.Object, finalizers ...string) {
	set := sets.NewString(obj.GetFinalizers()...)
	set.Insert(finalizers...)
	obj.SetFinalizers(set.List())
}

// GenerateToken generates a new, random token that can be used
// as an admin and kubelet token.
func GenerateToken() string {
	return fmt.Sprintf("%s.%s", rand.String(6), rand.String(16))
}

// ValidateKubernetesToken checks if a given token is syntactically correct.
func ValidateKubernetesToken(token string) error {
	if !tokenValidator.MatchString(token) {
		return fmt.Errorf("token is malformed, must match %s", tokenValidator.String())
	}

	return nil
}

func ValidateSecretKeySelector(selector *providerconfig.GlobalSecretKeySelector, key string) error {
	if selector.Name == "" && selector.Namespace == "" && key == "" {
		return fmt.Errorf("%q cannot be empty", key)
	}
	return nil
}
