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
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
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
				recorder:                &events.FakeRecorder{},
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

func TestCleanupPolicyBindingsRemovesCleanupFinalizer(t *testing.T) {
	ctx := context.Background()

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: testNS,
		},
	}
	policyBinding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "policy-binding",
			Namespace:  testNS,
			Finalizers: []string{kubermaticv1.PolicyBindingCleanupFinalizer},
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{
				Name: "policy-template",
			},
		},
	}

	seedClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(policyBinding).
		Build()

	deletion := &Deletion{
		seedClient: seedClient,
		recorder:   &events.FakeRecorder{},
	}

	if err := deletion.cleanupPolicyBindings(ctx, zap.NewNop().Sugar(), cluster); err != nil {
		t.Fatalf("cleanupPolicyBindings failed: %v", err)
	}

	updatedPolicyBinding := &kubermaticv1.PolicyBinding{}
	err := seedClient.Get(ctx, types.NamespacedName{Name: policyBinding.Name, Namespace: policyBinding.Namespace}, updatedPolicyBinding)
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		t.Fatalf("failed to get PolicyBinding: %v", err)
	}
	for _, finalizer := range updatedPolicyBinding.Finalizers {
		if finalizer == kubermaticv1.PolicyBindingCleanupFinalizer {
			t.Fatalf("expected PolicyBinding cleanup finalizer to be removed, got %v", updatedPolicyBinding.Finalizers)
		}
	}
}

func TestCleanupPolicyBindingsUsesFallbackNamespace(t *testing.T) {
	ctx := context.Background()

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}
	clusterNamespace := kubernetesprovider.NamespaceName(cluster.Name)
	policyBinding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "policy-binding",
			Namespace:  clusterNamespace,
			Finalizers: []string{kubermaticv1.PolicyBindingCleanupFinalizer},
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{
				Name: "policy-template",
			},
		},
	}

	seedClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(policyBinding).
		Build()

	deletion := &Deletion{
		seedClient: seedClient,
		recorder:   &events.FakeRecorder{},
	}

	if err := deletion.cleanupPolicyBindings(ctx, zap.NewNop().Sugar(), cluster); err != nil {
		t.Fatalf("cleanupPolicyBindings failed: %v", err)
	}

	updatedPolicyBinding := &kubermaticv1.PolicyBinding{}
	err := seedClient.Get(ctx, types.NamespacedName{Name: policyBinding.Name, Namespace: policyBinding.Namespace}, updatedPolicyBinding)
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		t.Fatalf("failed to get PolicyBinding: %v", err)
	}
	for _, finalizer := range updatedPolicyBinding.Finalizers {
		if finalizer == kubermaticv1.PolicyBindingCleanupFinalizer {
			t.Fatalf("expected PolicyBinding cleanup finalizer to be removed from fallback namespace, got %v", updatedPolicyBinding.Finalizers)
		}
	}
}

func TestCleanupPolicyBindingsIgnoresMissingNamespace(t *testing.T) {
	ctx := context.Background()

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: testNS,
		},
	}

	testCases := []struct {
		name              string
		deleteAllOfFailed bool
		listFailed        bool
	}{
		{
			name:              "DeleteAllOf returns NotFound",
			deleteAllOfFailed: true,
		},
		{
			name:       "List returns NotFound",
			listFailed: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			seedClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithInterceptorFuncs(interceptor.Funcs{
					DeleteAllOf: func(ctx context.Context, client ctrlruntimeclient.WithWatch, obj ctrlruntimeclient.Object, opts ...ctrlruntimeclient.DeleteAllOfOption) error {
						deleteOptions := (&ctrlruntimeclient.DeleteAllOfOptions{}).ApplyOptions(opts)
						if tc.deleteAllOfFailed && deleteOptions.Namespace == testNS {
							return apierrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, testNS)
						}

						return client.DeleteAllOf(ctx, obj, opts...)
					},
					List: func(ctx context.Context, client ctrlruntimeclient.WithWatch, list ctrlruntimeclient.ObjectList, opts ...ctrlruntimeclient.ListOption) error {
						listOptions := (&ctrlruntimeclient.ListOptions{}).ApplyOptions(opts)
						if tc.listFailed && listOptions.Namespace == testNS {
							return apierrors.NewNotFound(schema.GroupResource{Resource: "namespaces"}, testNS)
						}

						return client.List(ctx, list, opts...)
					},
				}).
				Build()

			deletion := &Deletion{
				seedClient: seedClient,
				recorder:   &events.FakeRecorder{},
			}

			if err := deletion.cleanupPolicyBindings(ctx, zap.NewNop().Sugar(), cluster); err != nil {
				t.Fatalf("expected missing namespace to be treated as completed cleanup, got: %v", err)
			}
		})
	}
}

func TestCleanupClusterRemovesPolicyBindingFinalizersBeforeFinalizerGate(t *testing.T) {
	ctx := context.Background()

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
			Finalizers: []string{
				kubermaticv1.NodeDeletionFinalizer,
			},
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: testNS,
		},
	}
	policyBinding := &kubermaticv1.PolicyBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "policy-binding",
			Namespace:  testNS,
			Finalizers: []string{kubermaticv1.PolicyBindingCleanupFinalizer},
		},
		Spec: kubermaticv1.PolicyBindingSpec{
			PolicyTemplateRef: corev1.ObjectReference{
				Name: "policy-template",
			},
		},
	}

	seedClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(cluster, policyBinding).
		Build()

	deletion := &Deletion{
		seedClient: seedClient,
		recorder:   &events.FakeRecorder{},
		userClusterClientGetter: func() (ctrlruntimeclient.Client, error) {
			return nil, fmt.Errorf("node cleanup blocked")
		},
	}

	if err := deletion.CleanupCluster(ctx, zap.NewNop().Sugar(), cluster); err == nil {
		t.Fatal("expected CleanupCluster to return node cleanup error")
	}

	updatedPolicyBinding := &kubermaticv1.PolicyBinding{}
	err := seedClient.Get(ctx, types.NamespacedName{Name: policyBinding.Name, Namespace: policyBinding.Namespace}, updatedPolicyBinding)
	if apierrors.IsNotFound(err) {
		return
	}
	if err != nil {
		t.Fatalf("failed to get PolicyBinding: %v", err)
	}
	for _, finalizer := range updatedPolicyBinding.Finalizers {
		if finalizer == kubermaticv1.PolicyBindingCleanupFinalizer {
			t.Fatalf("expected PolicyBinding cleanup finalizer to be removed before finalizer gate, got %v", updatedPolicyBinding.Finalizers)
		}
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
