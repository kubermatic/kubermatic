/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package workerlabel

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LabelSelector returns a label selector to only process clusters with a matching worker-name label.
func LabelSelector(workerName string) (labels.Selector, error) {
	var req *labels.Requirement
	var err error
	if workerName == "" {
		req, err = labels.NewRequirement(kubermaticv1.WorkerNameLabelKey, selection.DoesNotExist, nil)
	} else {
		req, err = labels.NewRequirement(kubermaticv1.WorkerNameLabelKey, selection.Equals, []string{workerName})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build label selector: %w", err)
	}

	return labels.Parse(req.String())
}

// Predicate returns a predicate func to only process objects with a matching worker-name label
// This works regardless of the underlying type
// Once https://github.com/kubernetes-sigs/controller-runtime/issues/244 is fixed we won't
// need this anymore.
func Predicate(workerName string) predicate.Funcs {
	return TypedPredicate[ctrlruntimeclient.Object](workerName)
}

func TypedPredicate[T ctrlruntimeclient.Object](workerName string) predicate.TypedFuncs[T] {
	return kubermaticpred.TypedFactory(func(object T) bool {
		return object.GetLabels()[kubermaticv1.WorkerNameLabelKey] == workerName
	})
}
