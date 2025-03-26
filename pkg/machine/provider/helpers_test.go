/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package provider

import (
	"encoding/json"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func genCluster(cloudSpec kubermaticv1.CloudSpec) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: "testproject",
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: cloudSpec,
		},
	}
}

func cloneBuilder[T any](builder T) T {
	encoded, err := json.Marshal(builder)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal machine builder %T: %v", builder, err))
	}

	newBuilder := new(T)
	if err := json.Unmarshal(encoded, newBuilder); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal machine builder %T: %v", builder, err))
	}

	return *newBuilder
}

type testcase[T any] interface {
	Name() string
	Expected() *T
	ExpectedError() bool
	Run(cluster *kubermaticv1.Cluster) (*T, error)
}

type specBuilder[T any] interface {
	Build() T
}

type baseTestcase[T any, U any] struct {
	name        string
	inputSpec   specBuilder[T]
	datacenter  *U
	expected    specBuilder[T]
	expectedErr bool
}

func (tt *baseTestcase[T, U]) Name() string {
	return tt.name
}

func (tt *baseTestcase[T, U]) Input() *T {
	if tt.inputSpec == nil {
		return nil
	}

	spec := tt.inputSpec.Build()
	return &spec
}

func (tt *baseTestcase[T, U]) Expected() *T {
	if tt.expected == nil {
		return nil
	}

	spec := tt.expected.Build()
	return &spec
}

func (tt *baseTestcase[T, U]) ExpectedError() bool {
	return tt.expectedErr
}

func runProviderTestcases[T any](t *testing.T, cluster *kubermaticv1.Cluster, testcases []testcase[T]) {
	for _, tt := range testcases {
		t.Run(tt.Name(), func(t *testing.T) {
			completed, err := tt.Run(cluster.DeepCopy())
			if tt.ExpectedError() != (err != nil) {
				t.Fatalf("Unexpected error response, expectedErr=%v, but got %v", tt.ExpectedError(), err)
			}

			// when no special expectation is given, just assert that _something_ was returned
			if tt.Expected() == nil {
				if completed == nil {
					t.Fatal("Should not have returned nil.")
				}
			} else {
				if changes := diff.ObjectDiff(tt.Expected(), completed); changes != "" {
					t.Fatalf("Did not receive the expected provider spec:\n%s", changes)
				}
			}
		})
	}
}
