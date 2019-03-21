package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OwnerRefWrapper is responsible for wrapping a ObjectCreator function, solely to set the OwnerReference to the cluster object
func OwnerRefWrapper(ref metav1.OwnerReference) ObjectModifier {
	return func(create ObjectCreator) ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			obj.(metav1.Object).SetOwnerReferences([]metav1.OwnerReference{ref})
			return obj, nil
		}
	}
}
