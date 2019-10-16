package usercluster

import (
	"encoding/json"
	"testing"

	"github.com/go-test/deep"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
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

			if diff := deep.Equal(tc.expectedLabels, actualLabels); diff != nil {
				t.Errorf("actual labels do not match expected labels, diff: %v", err)
			}
		})
	}
}
