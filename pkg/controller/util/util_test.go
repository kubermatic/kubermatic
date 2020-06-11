package util

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
			reached, err := ConcurrencyLimitReached(context.Background(), fake.NewFakeClient(&kubermaticv1.Cluster{}), testCase.maxConcurrentLimit)

			if err != nil {
				t.Fatalf("failed to run test: %v with error: %v", testCase.name, err)
			}

			if reached != testCase.expectedLimitReached {
				t.Fatalf("failed to run test: %v, expects: %v, got: %v", testCase.name, testCase.expectedLimitReached, reached)
			}
		})
	}
}
