package workerlabel

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// LabelSelector returns a label selector to only process clusters with a matching machine.k8s.io/controller label
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
