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

package test

import (
	"context"
	"testing"
	"time"

	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateNamespaceWithCleanup creates a namespace with a generated name test-<something> and registers a hook on T.Cleanup
// that removes it at the end of the test.
func CreateNamespaceWithCleanup(t *testing.T, ctx context.Context, client ctrlruntimeclient.Client) *corev1.Namespace {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}
	if err := client.Create(ctx, ns); err != nil {
		t.Fatalf("failed to create test namespace: %s", err)
	}

	t.Cleanup(func() {
		if err := client.Delete(ctx, ns); err != nil {
			t.Fatalf("failed to cleanup test namespace: %s", err)
		}
	})

	if !utils.WaitFor(time.Second*1, time.Second*10, func() bool {
		namespace := &corev1.Namespace{}
		return client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(ns), namespace) == nil
	}) {
		t.Fatalf("timeout waiting for namespace creation")
	}
	return ns
}
