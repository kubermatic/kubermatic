package workerlabel

import (
	"fmt"

	kubermaticpred "github.com/kubermatic/kubermatic/pkg/controller/util/predicate"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LabelSelector returns a label selector to only process clusters with a matching worker-name label
func LabelSelector(workerName string) (labels.Selector, error) {
	var req *labels.Requirement
	var err error
	if workerName == "" {
		req, err = labels.NewRequirement(kubermaticv1.WorkerNameLabelKey, selection.DoesNotExist, nil)
	} else {
		req, err = labels.NewRequirement(kubermaticv1.WorkerNameLabelKey, selection.Equals, []string{workerName})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build label selector: %v", err)
	}

	return labels.Parse(req.String())
}

// Predicates returns a predicate func to only process objects with a matching worker-name label
// This works regardless of the underlying type
// Once https://github.com/kubernetes-sigs/controller-runtime/issues/244 is fixed we wont
// need this anymore
func Predicates(workerName string) predicate.Funcs {
	return kubermaticpred.Factory(func(m metav1.Object, r runtime.Object) bool {
		return m.GetLabels()[kubermaticv1.WorkerNameLabelKey] == workerName
	})
}
