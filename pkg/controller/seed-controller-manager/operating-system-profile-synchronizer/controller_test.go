/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package operatingsystemprofilesynchronizer

import (
	"context"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(osmv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
}

func TestReconcile(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	testCases := []struct {
		name                            string
		namespacedName                  types.NamespacedName
		existingClusters                []*kubermaticv1.Cluster
		existingOperatingSystemProfiles []*osmv1alpha1.OperatingSystemProfile
		expectedOperatingSystemProfiles []*osmv1alpha1.OperatingSystemProfile
	}{
		{
			name: "scenario 1: sync OSP to user cluster namespace",
			existingClusters: []*kubermaticv1.Cluster{
				genCluster("cluster-1", true),
			},
			namespacedName: types.NamespacedName{Name: "osp", Namespace: "kubermatic"},
			existingOperatingSystemProfiles: []*osmv1alpha1.OperatingSystemProfile{
				getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", false, false),
			},
			expectedOperatingSystemProfiles: []*osmv1alpha1.OperatingSystemProfile{
				getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", false, false),
				getOperatingSystemProfile("osp", "cluster-cluster-1", "v1.0.0", false, false),
			},
		},
		{
			name: "scenario 2: sync OSP to multiple user cluster namespaces",
			existingClusters: []*kubermaticv1.Cluster{
				genCluster("cluster-1", true),
				genCluster("cluster-2", true),
				genCluster("cluster-3", true),
				genCluster("cluster-4", false),
			},
			namespacedName: types.NamespacedName{Name: "osp", Namespace: "kubermatic"},
			existingOperatingSystemProfiles: []*osmv1alpha1.OperatingSystemProfile{
				getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", false, false),
			},
			expectedOperatingSystemProfiles: []*osmv1alpha1.OperatingSystemProfile{
				getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", false, false),
				getOperatingSystemProfile("osp", "cluster-cluster-1", "v1.0.0", false, false),
				getOperatingSystemProfile("osp", "cluster-cluster-2", "v1.0.0", false, false),
				getOperatingSystemProfile("osp", "cluster-cluster-3", "v1.0.0", false, false),
			},
		},
		{
			name: "scenario 3: deleting OSP from kubermatic namespace should delete it from the cluster namespace",
			existingClusters: []*kubermaticv1.Cluster{
				genCluster("cluster-1", true),
			},
			namespacedName: types.NamespacedName{Name: "osp", Namespace: "kubermatic"},
			existingOperatingSystemProfiles: []*osmv1alpha1.OperatingSystemProfile{
				getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", true, false),
				getOperatingSystemProfile("osp", "cluster-cluster-1", "v1.0.0", false, false),
			},
			expectedOperatingSystemProfiles: nil,
		},
		{
			name: "scenario 4: deleting OSP from kubermatic namespace should delete it from multiple user cluster namespaces",
			existingClusters: []*kubermaticv1.Cluster{
				genCluster("cluster-1", true),
				genCluster("cluster-2", true),
				genCluster("cluster-3", true),
				genCluster("cluster-4", false),
			},
			namespacedName: types.NamespacedName{Name: "osp", Namespace: "kubermatic"},
			existingOperatingSystemProfiles: []*osmv1alpha1.OperatingSystemProfile{
				getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", true, false),
				getOperatingSystemProfile("osp", "cluster-cluster-1", "v1.0.0", false, false),
				getOperatingSystemProfile("osp", "cluster-cluster-2", "v1.0.0", false, false),
				getOperatingSystemProfile("osp", "cluster-cluster-3", "v1.0.0", false, false),
			},
			expectedOperatingSystemProfiles: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var (
				obj []ctrlruntimeclient.Object
				err error
			)

			for _, c := range tc.existingClusters {
				obj = append(obj, c)
			}

			for _, c := range tc.existingOperatingSystemProfiles {
				obj = append(obj, c)
			}

			seedClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(obj...).
				Build()

			r := &Reconciler{
				log:                     kubermaticlog.Logger,
				workerNameLabelSelector: workerSelector,
				recorder:                &record.FakeRecorder{},
				seedClient:              seedClient,
				namespace:               "kubermatic",
			}

			request := reconcile.Request{NamespacedName: tc.namespacedName}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			ospList := &osmv1alpha1.OperatingSystemProfileList{}
			err = seedClient.List(context.Background(), ospList)
			if err != nil {
				t.Fatalf("failed to list operatingSystemProfiles: %v", err)
			}

			if len(ospList.Items) != len(tc.expectedOperatingSystemProfiles) {
				t.Fatalf("expected count %d differs from the observed count %d for operatingSystemProfiles", len(tc.expectedOperatingSystemProfiles), len(ospList.Items))
			}

			for _, observedOSP := range ospList.Items {
				for _, expectedOSP := range tc.expectedOperatingSystemProfiles {
					if expectedOSP.Name == observedOSP.Name && expectedOSP.Namespace == observedOSP.Namespace {
						if !diff.SemanticallyEqual(expectedOSP.Spec, observedOSP.Spec) {
							t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(expectedOSP.Spec, observedOSP.Spec))
						}
					}
				}
			}
		})
	}
}

func TestReconcileForUpdate(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	testCases := []struct {
		name                            string
		namespacedName                  types.NamespacedName
		updatedVersion                  *string
		existingClusters                []*kubermaticv1.Cluster
		existingOperatingSystemProfile  *osmv1alpha1.OperatingSystemProfile
		expectedOperatingSystemProfiles []*osmv1alpha1.OperatingSystemProfile
	}{
		{
			name: "scenario 1: updating custom OSP should update OSPs in user clusters",
			existingClusters: []*kubermaticv1.Cluster{
				genCluster("cluster-1", true),
				genCluster("cluster-2", true),
				genCluster("cluster-3", true),
			},
			namespacedName:                 types.NamespacedName{Name: "osp", Namespace: "kubermatic"},
			updatedVersion:                 pointer.String("v1.2.3"),
			existingOperatingSystemProfile: getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", false, false),
			expectedOperatingSystemProfiles: []*osmv1alpha1.OperatingSystemProfile{
				getOperatingSystemProfile("osp", "kubermatic", "v1.2.3", false, true),
				getOperatingSystemProfile("osp", "cluster-cluster-1", "v1.2.3", false, true),
				getOperatingSystemProfile("osp", "cluster-cluster-2", "v1.2.3", false, true),
				getOperatingSystemProfile("osp", "cluster-cluster-3", "v1.2.3", false, true),
			},
		},
		{
			name: "scenario 2: updating custom OSP without version update should result in no update",
			existingClusters: []*kubermaticv1.Cluster{
				genCluster("cluster-1", true),
				genCluster("cluster-2", true),
				genCluster("cluster-3", true),
			},
			namespacedName:                 types.NamespacedName{Name: "osp", Namespace: "kubermatic"},
			existingOperatingSystemProfile: getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", false, false),
			expectedOperatingSystemProfiles: []*osmv1alpha1.OperatingSystemProfile{
				getOperatingSystemProfile("osp", "kubermatic", "v1.0.0", false, true),
				getOperatingSystemProfile("osp", "cluster-cluster-1", "v1.0.0", false, false),
				getOperatingSystemProfile("osp", "cluster-cluster-2", "v1.0.0", false, false),
				getOperatingSystemProfile("osp", "cluster-cluster-3", "v1.0.0", false, false),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var (
				obj []ctrlruntimeclient.Object
				err error
			)

			for _, c := range tc.existingClusters {
				obj = append(obj, c)
			}

			obj = append(obj, tc.existingOperatingSystemProfile)

			seedClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(obj...).
				Build()

			r := &Reconciler{
				log:                     kubermaticlog.Logger,
				workerNameLabelSelector: workerSelector,
				recorder:                &record.FakeRecorder{},
				seedClient:              seedClient,
				namespace:               "kubermatic",
			}

			request := reconcile.Request{NamespacedName: tc.namespacedName}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			osp := &osmv1alpha1.OperatingSystemProfile{}
			err = seedClient.Get(context.Background(), tc.namespacedName, osp)
			if err != nil {
				t.Fatalf("failed to get operatingSystemProfile: %v", err)
			}

			osp.Spec.OSVersion = "2.0"
			if tc.updatedVersion != nil {
				osp.Spec.Version = *tc.updatedVersion
			}

			err = seedClient.Update(context.Background(), osp)
			if err != nil {
				t.Fatalf("failed to update operatingSystemProfile: %v", err)
			}

			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			ospList := &osmv1alpha1.OperatingSystemProfileList{}
			err = seedClient.List(context.Background(), ospList)
			if err != nil {
				t.Fatalf("failed to list operatingSystemProfiles: %v", err)
			}

			if len(ospList.Items) != len(tc.expectedOperatingSystemProfiles) {
				t.Fatalf("expected count %d differs from the observed count %d for operatingSystemProfiles", len(tc.expectedOperatingSystemProfiles), len(ospList.Items))
			}

			for _, observedOSP := range ospList.Items {
				for _, expectedOSP := range tc.expectedOperatingSystemProfiles {
					if expectedOSP.Name == observedOSP.Name && expectedOSP.Namespace == observedOSP.Namespace {
						if !diff.SemanticallyEqual(expectedOSP.Spec, observedOSP.Spec) {
							t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(expectedOSP.Spec, observedOSP.Spec))
						}
					}
				}
			}
		})
	}
}

func genCluster(name string, osmEnabled bool) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			EnableOperatingSystemManager: pointer.Bool(osmEnabled),
			HumanReadableName:            name,
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: kubernetes.NamespaceName(name),
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Etcd:       kubermaticv1.HealthStatusUp,
				Apiserver:  kubermaticv1.HealthStatusUp,
				Controller: kubermaticv1.HealthStatusUp,
				Scheduler:  kubermaticv1.HealthStatusUp,
			},
		},
	}
}

func getOperatingSystemProfile(name string, namespace string, version string, markedForDeletion bool, update bool) *osmv1alpha1.OperatingSystemProfile {
	var (
		deletionTimestamp *metav1.Time
		finalizers        []string
	)
	if markedForDeletion {
		deletionTimestamp = &metav1.Time{Time: time.Now()}
		finalizers = []string{cleanupFinalizer}
	}

	osVersion := "1.0"
	if update {
		osVersion = "2.0"
	}

	return &osmv1alpha1.OperatingSystemProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			DeletionTimestamp: deletionTimestamp,
			Finalizers:        finalizers,
		},
		Spec: osmv1alpha1.OperatingSystemProfileSpec{
			OSName:    "ubuntu",
			OSVersion: osVersion,
			Version:   version,
			SupportedCloudProviders: []osmv1alpha1.CloudProviderSpec{
				{
					Name: "aws",
				},
			},
			ProvisioningConfig: osmv1alpha1.OSPConfig{
				SupportedContainerRuntimes: []osmv1alpha1.ContainerRuntimeSpec{
					{
						Name: "containerd",
					},
				},
				Files: []osmv1alpha1.File{
					{
						Path: "/etc/systemd/journald.conf.d/max_disk_use.conf",
						Content: osmv1alpha1.FileContent{
							Inline: &osmv1alpha1.FileContentInline{
								Encoding: "b64",
								Data:     "test",
							},
						},
					},
				},
			},
		},
	}
}
