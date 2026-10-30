/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestTypedPredicate(t *testing.T) {
	testCases := []struct {
		name       string
		workerName string
		labels     map[string]string
		expected   bool
	}{
		{
			name:       "an empty worker name matches an object without the worker-name label",
			workerName: "",
			labels:     nil,
			expected:   true,
		},
		{
			name:       "a named worker does not match an object without the worker-name label",
			workerName: "dev2",
			labels:     nil,
			expected:   false,
		},
		{
			name:       "a named worker matches an object carrying its own worker-name label",
			workerName: "dev2",
			labels:     map[string]string{kubermaticv1.WorkerNameLabelKey: "dev2"},
			expected:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := &kubermaticv1.ClusterTemplateInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "instance",
					Labels: tc.labels,
				},
			}

			pred := TypedPredicate[*kubermaticv1.ClusterTemplateInstance](tc.workerName)

			createEvent := event.TypedCreateEvent[*kubermaticv1.ClusterTemplateInstance]{Object: instance}
			if pred.CreateFunc(createEvent) != tc.expected {
				t.Errorf("create event: expected match %t, got %t", tc.expected, !tc.expected)
			}

			deleteEvent := event.TypedDeleteEvent[*kubermaticv1.ClusterTemplateInstance]{Object: instance}
			if pred.DeleteFunc(deleteEvent) != tc.expected {
				t.Errorf("delete event: expected match %t, got %t", tc.expected, !tc.expected)
			}
		})
	}
}

func TestTypedPredicateUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		workerName string
		oldLabels  map[string]string
		newLabels  map[string]string
		expected   bool
	}{
		{
			name:       "an update from a matching old object to an unmatched new object passes",
			workerName: "dev2",
			oldLabels:  map[string]string{kubermaticv1.WorkerNameLabelKey: "dev2"},
			newLabels:  nil,
			expected:   true,
		},
		{
			name:       "an update from an unmatched old object to a matching new object passes",
			workerName: "dev2",
			oldLabels:  nil,
			newLabels:  map[string]string{kubermaticv1.WorkerNameLabelKey: "dev2"},
			expected:   true,
		},
		{
			name:       "an update between two unmatched objects does not pass",
			workerName: "dev2",
			oldLabels:  nil,
			newLabels:  nil,
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldInstance := &kubermaticv1.ClusterTemplateInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "instance",
					Labels: tc.oldLabels,
				},
			}
			newInstance := &kubermaticv1.ClusterTemplateInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "instance",
					Labels: tc.newLabels,
				},
			}

			pred := TypedPredicate[*kubermaticv1.ClusterTemplateInstance](tc.workerName)

			updateEvent := event.TypedUpdateEvent[*kubermaticv1.ClusterTemplateInstance]{ObjectOld: oldInstance, ObjectNew: newInstance}
			if pred.UpdateFunc(updateEvent) != tc.expected {
				t.Errorf("update event: expected match %t, got %t", tc.expected, !tc.expected)
			}
		})
	}
}
