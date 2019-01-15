package resources

import (
	"github.com/go-test/deep"
	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	// Kubernetes Objects can be deeper than the default 10 levels.
	deep.MaxDepth = 20
	deep.LogErrors = true
}

// DeepEqual compares both objects for equality
func DeepEqual(a, b metav1.Object) bool {
	if equality.Semantic.DeepEqual(a, b) {
		return true
	}

	// For informational purpose we use deep.equal as it tells us what the difference is.
	// We need to calculate the difference in both ways as deep.equal only does a one-way comparison
	diff := deep.Equal(a, b)
	if diff == nil {
		diff = deep.Equal(b, a)
	}

	glog.V(4).Infof("Object %T %s/%s differs from the one, generated: %v", a, a.GetNamespace(), a.GetName(), diff)
	return false
}
