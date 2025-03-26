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

package usercluster

import (
	"encoding/json"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetLabelArgsValue(t *testing.T) {
	testCases := []struct {
		name           string
		initialLabels  map[string]string
		expectedLabels map[string]string
	}{
		{
			name:           "Labels get applied",
			initialLabels:  map[string]string{"foo": "bar"},
			expectedLabels: map[string]string{"foo": "bar"},
		},
		{
			name:           "Protected labels do not get applied",
			initialLabels:  map[string]string{"foo": "bar", "project-id": "my-project", "worker-name": "w"},
			expectedLabels: map[string]string{"foo": "bar"},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cluster := &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Labels: tc.initialLabels,
				},
			}
			result, err := getLabelsArgValue(cluster)
			if err != nil {
				t.Fatalf("error when calling getLabelsArgValue: %v", err)
			}

			actualLabels := map[string]string{}
			if err := json.Unmarshal([]byte(result), &actualLabels); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedLabels, actualLabels) {
				t.Fatalf("actual labels do not match expected labels:\n%v", diff.ObjectDiff(tc.expectedLabels, actualLabels))
			}
		})
	}
}
