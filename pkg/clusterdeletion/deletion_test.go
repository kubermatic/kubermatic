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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	controllerruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
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

func TestCleanUpPVUsingWorkloads(t *testing.T) {
	testCases := []struct {
		name                string
		objects             []runtime.Object
		errExpected         bool
		objDeletionExpected bool
	}{
		{
			name:                "Delete Pod",
			objects:             []runtime.Object{getPod("", "", true)},
			objDeletionExpected: true,
		},
		{
			name:    "Dont delete pod without PV",
			objects: []runtime.Object{getPod("", "", false)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClientWithScheme(scheme.Scheme, tc.objects...)
			d := &Deletion{}
			ctx := context.Background()

			if err := d.cleanupPVCUsingPods(ctx, client); (err != nil) != tc.errExpected {
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

				err := client.Get(ctx, nn, object.DeepCopyObject())
				if kerrors.IsNotFound(err) != tc.objDeletionExpected {
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
		objects []runtime.Object
	}{
		{
			name:    "Nodes remain because LB finalizer exists",
			cluster: getClusterWithFinalizer(clusterName, kubermaticapiv1.InClusterLBCleanupFinalizer),
			objects: []runtime.Object{&corev1.Service{
				Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
			}},
		},
		{
			name:    "Nodes remain because PV finalizer exists",
			cluster: getClusterWithFinalizer(clusterName, kubermaticapiv1.InClusterPVCleanupFinalizer),
			objects: []runtime.Object{&corev1.PersistentVolume{}},
		},
		// https://github.com/kubernetes-sigs/controller-runtime/issues/702
		//	{
		//		name:    "Nodes remain because credentialRequests finalizer exists",
		//		cluster: getClusterWithFinalizer(clusterName, kubermaticapiv1.InClusterCredentialsRequestsCleanupFinalizer),
		//		objects: []runtime.Object{unstructuredWithAPIVersionAndKind("cloudcredential.openshift.io/v1", "CredentialsRequest")},
		//	},
		//	{
		//		name:    "Nodes remain because imageRegistryConfigs finalizer exists",
		//		cluster: getClusterWithFinalizer(clusterName, kubermaticapiv1.InClusterImageRegistryConfigCleanupFinalizer),
		//		objects: []runtime.Object{unstructuredWithAPIVersionAndKind("imageregistry.operator.openshift.io/v1", "Config")},
		//	},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := &clusterv1alpha1.Machine{}
			tc.objects = append(tc.objects, m)
			userClusterClient := fake.NewFakeClient(tc.objects...)
			userClusterClientGetter := func() (controllerruntimeclient.Client, error) {
				return userClusterClient, nil
			}
			seedClient := fake.NewFakeClient(tc.cluster)

			ctx := context.Background()
			deletion := &Deletion{
				seedClient:              seedClient,
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
