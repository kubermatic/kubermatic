package reconciling

import (
	"encoding/json"
	"github.com/go-test/deep"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
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

	// For some reason unstructured objects returned from the api have types for their fields
	// that are not map[string]interface{} and don't even exist in our codebase like
	// `openshift.infrastructureStatus`, so we have to compare the wire format here.
	// We only do this for unstrucutred as this comparison is pretty expensive.
	if _, isUnstructured := a.(*unstructured.Unstructured); isUnstructured && jsonEqual(a, b) {
		return true
	}

	// For informational purpose we use deep.equal as it tells us what the difference is.
	// We need to calculate the difference in both ways as deep.equal only does a one-way comparison
	diff := deep.Equal(a, b)
	if diff == nil {
		diff = deep.Equal(b, a)
	}

	klog.V(4).Infof("Object %T %s/%s differs from the generated one: %v", a, a.GetNamespace(), a.GetName(), diff)
	return false
}

func jsonEqual(a, b interface{}) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		klog.Errorf("failed to marshal aJSON: %v", err)
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		klog.Errorf("failed to marshal bJSON: %v", err)
		return false
	}
	return string(aJSON) == string(bJSON)
}
