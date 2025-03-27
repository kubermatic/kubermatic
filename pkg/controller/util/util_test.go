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

package util

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"
)

func TestConcurrencyLimitReached(t *testing.T) {
	concurrencyLimitReachedTestCases := []struct {
		name                 string
		maxConcurrentLimit   int
		expectedLimitReached bool
	}{
		{
			name:                 "concurrency limit has not reached",
			maxConcurrentLimit:   2,
			expectedLimitReached: false,
		},
		{
			name:                 "concurrency limit has reached",
			maxConcurrentLimit:   1,
			expectedLimitReached: true,
		},
	}

	for _, testCase := range concurrencyLimitReachedTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			reached, err := ConcurrencyLimitReached(context.Background(), fake.NewClientBuilder().WithObjects(&kubermaticv1.Cluster{}).Build(), testCase.maxConcurrentLimit)

			if err != nil {
				t.Fatalf("failed to run test: %v with error: %v", testCase.name, err)
			}

			if reached != testCase.expectedLimitReached {
				t.Fatalf("failed to run test: %v, expects: %v, got: %v", testCase.name, testCase.expectedLimitReached, reached)
			}
		})
	}
}
