package workerlabel

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LabelSelector returns a label selector to only process clusters with a matching worker-name label
func LabelSelector(workerName string) (func(*metav1.ListOptions), error) {
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

	return func(options *metav1.ListOptions) {
		options.LabelSelector = req.String()
	}, nil
}

// Predicates returns a prediate func to only process objects with a matching worker-name label
// This works regardless of the underlying type
// Once https://github.com/kubernetes-sigs/controller-runtime/issues/244 is fixed we wont
// need this anymore
func Predicates(workerName string) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return e.Meta.GetLabels()[kubermaticv1.WorkerNameLabelKey] == workerName
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.MetaNew.GetLabels()[kubermaticv1.WorkerNameLabelKey] == workerName
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return e.Meta.GetLabels()[kubermaticv1.WorkerNameLabelKey] == workerName
		},
	}
}
