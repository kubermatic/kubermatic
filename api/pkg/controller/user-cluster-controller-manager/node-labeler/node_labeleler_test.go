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

package nodelabeler

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	"github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/node-labeler/api"
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
		expectedErr      string
	}{
		{
			name: "node not found, no error",
		},
		{
			name: "labels get added",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: requestName,
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: "ubuntu",
					},
				},
			},
			reconcilerLabels: map[string]string{"foo": "bar"},
			expectedLabels:   map[string]string{"foo": "bar", "x-kubernetes.io/distribution": "ubuntu"},
		},
		{
			name: "adding new labels keeps existing labels",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   requestName,
					Labels: map[string]string{"baz": "boo"},
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: "ubuntu",
					},
				},
			},
			reconcilerLabels: map[string]string{"foo": "bar"},
			expectedLabels:   map[string]string{"foo": "bar", "baz": "boo", "x-kubernetes.io/distribution": "ubuntu"},
		},
		{
			name: "ubuntu label gets added",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: requestName,
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: "ubuntu",
					},
				},
			},
			expectedLabels: map[string]string{"x-kubernetes.io/distribution": "ubuntu"},
		},
		{
			name: "centos label gets added",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: requestName,
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: "centos",
					},
				},
			},
			expectedLabels: map[string]string{"x-kubernetes.io/distribution": "centos"},
		},
		{
			name: "container linux label gets added",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: requestName,
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: "container linux",
					},
				},
			},
			expectedLabels: map[string]string{"x-kubernetes.io/distribution": "container-linux"},
		},
		{
			name: "sles label gets added",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: requestName,
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: "sles",
					},
				},
			},
			expectedLabels: map[string]string{"x-kubernetes.io/distribution": "sles"},
		},
		{
			name: "rhel label gets added",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: requestName,
				},
				Status: corev1.NodeStatus{
					NodeInfo: corev1.NodeSystemInfo{
						OSImage: "rhel",
					},
				},
			},
			expectedLabels: map[string]string{"x-kubernetes.io/distribution": "rhel"},
		},
		{
			name: "unknown os, error",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: requestName,
				},
			},
			expectedErr: `failed to apply distribution label: could not detect distribution from image name ""`,
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
			_, err := r.Reconcile(request)
			var actualErr string
			if err != nil {
				actualErr = err.Error()
			}
			if actualErr != tc.expectedErr {
				t.Fatalf("Got err %q, expected err %q", actualErr, tc.expectedErr)
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

func TestMatchOSLabels(t *testing.T) {
	tests := []struct {
		osImage  string
		expected string
	}{
		{
			osImage:  "container linux",
			expected: api.ContainerLinuxLabelValue,
		},
		{
			osImage:  "flatcar container linux",
			expected: api.FlatcarLabelValue,
		},
	}

	for n, test := range tests {
		// go map iterations are randomized, so we just hammer out the entropy
		// by running the same test a gazillion times
		for i := 0; i < 1000; i++ {
			result := findDistributionLabel(test.osImage)

			if result != test.expected {
				t.Fatalf("Test case %d: expected label %q, but got %q.", n+1, test.expected, result)
			}
		}
	}
}
