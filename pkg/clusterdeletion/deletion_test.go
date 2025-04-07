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

package clusterdeletion

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var testScheme = fake.NewScheme()

func init() {
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(testScheme); err != nil {
		panic(fmt.Sprintf("failed to add clusterv1alpha1 to scheme: %v", err))
	}
}

const testNS = "test-ns"

func getPod(ownerRefKind, ownerRefName string, hasPV bool) *corev1.Pod {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNS,
			Name:      "my-pod",
		},
	}

	if ownerRefKind != "" {
		p.OwnerReferences = []metav1.OwnerReference{{Kind: ownerRefKind, Name: ownerRefName}}
	}

	if hasPV {
		p.Spec.Volumes = []corev1.Volume{
			{
				Name: "my-vol",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{},
				},
			},
		}
	}

	return p
}

func TestCleanupPVUsingWorkloads(t *testing.T) {
	testCases := []struct {
		name                string
		objects             []ctrlruntimeclient.Object
		errExpected         bool
		objDeletionExpected bool
	}{
		{
			name:                "Delete Pod",
			objects:             []ctrlruntimeclient.Object{getPod("", "", true)},
			objDeletionExpected: true,
		},
		{
			name:    "Dont delete pod without PV",
			objects: []ctrlruntimeclient.Object{getPod("", "", false)},
		},
	}

	log := zap.NewNop().Sugar()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(tc.objects...).
				Build()

			d := &Deletion{}
			ctx := context.Background()

			if err := d.cleanupPVCUsingPods(ctx, log, client); (err != nil) != tc.errExpected {
				t.Fatalf("Expected err=%v, got err=%v", tc.errExpected, err)
			}
			if tc.errExpected {
				return
			}

			for _, object := range tc.objects {
				metav1Object := object.(metav1.Object)
				nn := types.NamespacedName{
					Namespace: metav1Object.GetNamespace(),
					Name:      metav1Object.GetName(),
				}

				objCopy := object.DeepCopyObject().(ctrlruntimeclient.Object)

				err := client.Get(ctx, nn, objCopy)
				if apierrors.IsNotFound(err) != tc.objDeletionExpected {
					t.Errorf("Expected object %q to be deleted=%t", nn.String(), tc.objDeletionExpected)
				}
			}
		})
	}
}

func TestNodesRemainUntilInClusterResourcesAreGone(t *testing.T) {
	const clusterName = "cluster"
	testCases := []struct {
		name    string
		cluster *kubermaticv1.Cluster
		objects []ctrlruntimeclient.Object
	}{
		{
			name:    "Nodes remain because LB finalizer exists",
			cluster: getClusterWithFinalizer(clusterName, kubermaticv1.InClusterLBCleanupFinalizer),
			objects: []ctrlruntimeclient.Object{&corev1.Service{
				Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
			}},
		},
		{
			name:    "Nodes remain because PV finalizer exists",
			cluster: getClusterWithFinalizer(clusterName, kubermaticv1.InClusterPVCleanupFinalizer),
			objects: []ctrlruntimeclient.Object{&corev1.PersistentVolume{}},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := &clusterv1alpha1.Machine{}
			tc.objects = append(tc.objects, m)

			userClusterClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(tc.objects...).
				Build()

			userClusterClientGetter := func() (ctrlruntimeclient.Client, error) {
				return userClusterClient, nil
			}
			seedClient := fake.NewClientBuilder().WithObjects(tc.cluster).Build()

			ctx := context.Background()
			deletion := &Deletion{
				seedClient:              seedClient,
				recorder:                &record.FakeRecorder{},
				userClusterClientGetter: userClusterClientGetter,
			}

			if err := deletion.CleanupCluster(ctx, kubermaticlog.Logger, tc.cluster); err != nil {
				t.Fatalf("Deletion failed: %v", err)
			}

			resultingMachines := &clusterv1alpha1.MachineList{}
			if err := userClusterClient.List(ctx, resultingMachines); err != nil {
				t.Fatalf("failed to list machines: %v", err)
			}
			if len(resultingMachines.Items) < 1 {
				t.Errorf("machines got deleted before in-cluster cleanup was done")
			}
		})
	}
}

func getClusterWithFinalizer(name string, finalizers ...string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Finalizers: finalizers,
		},
	}
}

// Short circuit linter, we want to use this once https://github.com/kubernetes-sigs/controller-runtime/issues/702
// is resolved and we can enable all tests.
var _ = unstructuredWithAPIVersionAndKind

func unstructuredWithAPIVersionAndKind(apiVersion, kind string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)
	return u
}
