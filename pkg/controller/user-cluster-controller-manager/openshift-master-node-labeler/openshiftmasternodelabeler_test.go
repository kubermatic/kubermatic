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

package openshiftmasternodelabeler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconciliation(t *testing.T) {
	testCases := []struct {
		name   string
		nodes  []ctrlruntimeclient.Object
		verify func(*reconcile.Result, error, ctrlruntimeclient.Client) error
	}{
		{
			name: "Labeled nodes already exist, nothing to do",
			nodes: []ctrlruntimeclient.Object{
				&corev1.Node{ObjectMeta: metav1.ObjectMeta{
					Name:   "one",
					Labels: map[string]string{"node-role.kubernetes.io/master": ""},
				}},
			},
			verify: func(r *reconcile.Result, err error, client ctrlruntimeclient.Client) error {
				if r != nil {
					return fmt.Errorf("expected reconcile.Result to be nil, was %v", r)
				}
				if err != nil {
					return fmt.Errorf("expected err to be nil, was %v", err)
				}
				nodes := &corev1.NodeList{}
				if err := client.List(context.Background(), nodes); err != nil {
					return fmt.Errorf("error listing nodes: %v", err)
				}
				if n := len(nodes.Items); n != 1 {
					return fmt.Errorf("expected three nodes, got %d", n)
				}
				for _, node := range nodes.Items {
					if _, exists := node.Labels["node-role.kubernetes.io/master"]; !exists {
						return fmt.Errorf("node %q didn't have the master label anymore", node.Name)
					}
				}
				return nil
			},
		},
		{
			name: "Labeling one node",
			nodes: []ctrlruntimeclient.Object{
				&corev1.Node{ObjectMeta: metav1.ObjectMeta{
					Name: "one",
				}},
				&corev1.Node{ObjectMeta: metav1.ObjectMeta{
					Name: "two",
				}},
				&corev1.Node{ObjectMeta: metav1.ObjectMeta{
					Name: "three",
				}},
			},
			verify: func(r *reconcile.Result, err error, client ctrlruntimeclient.Client) error {
				if r != nil {
					return fmt.Errorf("expected reconcile.Result to be nil, was %v", r)
				}
				if err != nil {
					return fmt.Errorf("expected err to be nil, was %v", err)
				}
				nodes := &corev1.NodeList{}
				if err := client.List(context.Background(), nodes); err != nil {
					return fmt.Errorf("error listing nodes: %v", err)
				}
				if n := len(nodes.Items); n != 3 {
					return fmt.Errorf("expected three nodes, got %d", n)
				}
				var nodesWithLabels int
				for _, node := range nodes.Items {
					if _, exists := node.Labels["node-role.kubernetes.io/master"]; exists {
						nodesWithLabels++
					}
				}
				if nodesWithLabels != 1 {
					return fmt.Errorf("expected one labeled node, got %d", nodesWithLabels)
				}
				return nil
			},
		},
		{
			name: "Labeling one node",
			nodes: []ctrlruntimeclient.Object{
				&corev1.Node{ObjectMeta: metav1.ObjectMeta{
					Name: "three",
				}},
			},
			verify: func(r *reconcile.Result, err error, client ctrlruntimeclient.Client) error {
				if r != nil {
					return fmt.Errorf("expected reconcile.Result to be nil, was %v", r)
				}
				if err != nil {
					return fmt.Errorf("expected err to be nil, was %v", err)
				}
				nodes := &corev1.NodeList{}
				if err := client.List(context.Background(), nodes); err != nil {
					return fmt.Errorf("error listing nodes: %v", err)
				}
				if n := len(nodes.Items); n != 1 {
					return fmt.Errorf("expected one node, got %d", n)
				}
				for _, node := range nodes.Items {
					if _, exists := node.Labels["node-role.kubernetes.io/master"]; !exists {
						return fmt.Errorf("node %q didn't have the master label", node.Name)
					}
				}
				return nil
			},
		},
		{
			name: "Not enough nodes exist, retrying later",
			verify: func(r *reconcile.Result, err error, client ctrlruntimeclient.Client) error {
				if err != nil {
					return fmt.Errorf("expected err to be nil, was %v", err)
				}
				if r == nil {
					return errors.New("expected to get reconcile.Result, but was nil")
				}
				if r.RequeueAfter != time.Minute {
					return fmt.Errorf("expected RequeueAfter to be 1 Minute, was %v", r.RequeueAfter)
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := fake.NewClientBuilder().WithObjects(tc.nodes...).Build()

			r := &reconciler{client: client}
			result, err := r.reconcile(ctx)
			if err := tc.verify(result, err, client); err != nil {
				t.Fatalf("verification failed: %v", err)
			}
		})
	}
}
