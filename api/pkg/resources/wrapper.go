package resources

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ClusterRefWrapper is responsible for wrapping a ObjectCreator function, solely to set the OwnerReference to the cluster object
func ClusterRefWrapper(c *kubermaticv1.Cluster) ObjectModifier {
	return func(create ObjectCreator) ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			obj.(metav1.Object).SetOwnerReferences([]metav1.OwnerReference{GetClusterRef(c)})
			return obj, nil
		}
	}
}
