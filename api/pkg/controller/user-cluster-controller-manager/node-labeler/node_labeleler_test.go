package nodelabeler

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	const requestName = "my-node"
	testCases := []struct {
		name             string
		node             *corev1.Node
		reconcilerLabels map[string]string
		expectedLabels   map[string]string
	}{
		{
			name: "node not found, no error",
		},
		{
			name: "labels get added",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{
				Name: requestName,
			}},
			reconcilerLabels: map[string]string{"foo": "bar"},
			expectedLabels:   map[string]string{"foo": "bar"},
		},
		{
			name: "adding new labels keeps existing labels",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{
				Name:   requestName,
				Labels: map[string]string{"baz": "boo"},
			}},
			reconcilerLabels: map[string]string{"foo": "bar"},
			expectedLabels:   map[string]string{"foo": "bar", "baz": "boo"},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var client ctrlruntimeclient.Client
			if tc.node != nil {
				client = fakectrlruntimeclient.NewFakeClient(tc.node)
			} else {
				client = fakectrlruntimeclient.NewFakeClient()
			}
			r := &reconciler{
				log:      kubermaticlog.Logger,
				client:   client,
				recorder: record.NewFakeRecorder(10),
				labels:   tc.reconcilerLabels,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: requestName}}
			if _, err := r.Reconcile(request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			if tc.node == nil {
				return
			}

			node := &corev1.Node{}
			if err := client.Get(context.Background(), request.NamespacedName, node); err != nil {
				t.Fatalf("failed to get node: %v", err)
			}

			if diff := deep.Equal(node.Labels, tc.expectedLabels); diff != nil {
				t.Errorf("node doesn't have expected labels, diff: %v", diff)
			}
		})
	}
}
