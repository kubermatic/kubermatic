/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package clustertemplatecontroller

import (
	"context"
	"reflect"
	"sort"
	"testing"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}
	seedNamespace := "namespace"
	projectName := test.GenDefaultProject().Name

	testCases := []struct {
		name                 string
		namespacedName       types.NamespacedName
		expectedClusters     []*kubermaticv1.Cluster
		expectedGetErrStatus metav1.StatusReason
		seedClient           ctrlruntimeclient.Client
	}{
		{
			name: "scenario 1: generates new clusters according to the template instance object",
			namespacedName: types.NamespacedName{
				Name: "my-first-project-ID-ctID2",
			},
			expectedClusters: []*kubermaticv1.Cluster{
				genCluster("ct2-0", "john@acme.com", *test.GenClusterTemplateInstance(projectName, "ctID2", 3)),
				genCluster("ct2-1", "john@acme.com", *test.GenClusterTemplateInstance(projectName, "ctID2", 3)),
				genCluster("ct2-2", "john@acme.com", *test.GenClusterTemplateInstance(projectName, "ctID2", 3)),
			},
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(
					test.GenClusterTemplate("ct1", "ctID1", projectName, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
					test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
					test.GenClusterTemplate("ct3", "ctID3", projectName, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
					test.GenClusterTemplate("ct4", "ctID4", projectName, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
					test.GenClusterTemplateInstance(projectName, "ctID1", 2),
					test.GenClusterTemplateInstance(projectName, "ctID2", 3),
					test.GenClusterTemplateInstance(projectName, "ctID3", 10),
				).
				Build(),
		},

		{
			name: "scenario 2: shrank number of clusters from 3 to 1",
			namespacedName: types.NamespacedName{
				Name: "my-first-project-ID-ctID2",
			},
			expectedClusters: []*kubermaticv1.Cluster{
				genCluster("ct2-0", "john@acme.com", *test.GenClusterTemplateInstance(projectName, "ctID2", 3)),
			},
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(
					test.GenClusterTemplate("ct1", "ctID1", projectName, kubermaticv1.UserClusterTemplateScope, test.GenDefaultAPIUser().Email),
					test.GenClusterTemplate("ct2", "ctID2", "", kubermaticv1.GlobalClusterTemplateScope, "john@acme.com"),
					test.GenClusterTemplate("ct3", "ctID3", projectName, kubermaticv1.UserClusterTemplateScope, "john@acme.com"),
					test.GenClusterTemplate("ct4", "ctID4", projectName, kubermaticv1.ProjectClusterTemplateScope, "john@acme.com"),
					test.GenClusterTemplateInstance(projectName, "ctID1", 2),
					test.GenClusterTemplateInstance(projectName, "ctID2", 1),
					test.GenClusterTemplateInstance(projectName, "ctID3", 10),
					genCluster("ct2-0", "john@acme.com", *test.GenClusterTemplateInstance(projectName, "ctID2", 3)),
					genCluster("ct2-1", "john@acme.com", *test.GenClusterTemplateInstance(projectName, "ctID2", 3)),
					genCluster("ct2-2", "john@acme.com", *test.GenClusterTemplateInstance(projectName, "ctID2", 3)),
				).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				namespace:               seedNamespace,
				log:                     kubermaticlog.Logger,
				workerNameLabelSelector: workerSelector,
				seedClient:              tc.seedClient,
			}

			request := reconcile.Request{NamespacedName: tc.namespacedName}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			clusterTemplateLabelSelector := ctrlruntimeclient.MatchingLabels{kubernetes.ClusterTemplateInstanceLabelKey: tc.namespacedName.Name}
			clusters := &kubermaticv1.ClusterList{}
			err = tc.seedClient.List(ctx, clusters, clusterTemplateLabelSelector)

			if tc.expectedGetErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %v", tc.expectedGetErrStatus)
				}
				if tc.expectedGetErrStatus != errors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, errors.ReasonForError(err))
				}
				return
			}
			if err != nil {
				t.Fatalf("failed get constraint: %v", err)
			}

			// remove autogenerated name;
			clusterList := []*kubermaticv1.Cluster{}
			for _, cluster := range clusters.Items {
				// ignore clusters that only have a deletion timestampa and
				// the CredentialsSecretsCleanupFinalizer finalizer, as those
				// would be cleaned up by another controller
				if kuberneteshelper.HasOnlyFinalizer(&cluster, kubermaticapiv1.CredentialsSecretsCleanupFinalizer) && !cluster.DeletionTimestamp.IsZero() {
					continue
				}

				modifiedCluster := cluster.DeepCopy()
				modifiedCluster.Name = ""

				clusterList = append(clusterList, modifiedCluster)
			}
			expectedClusterList := []*kubermaticv1.Cluster{}
			for _, cluster := range tc.expectedClusters {
				modifiedCluster := cluster.DeepCopy()
				modifiedCluster.Name = ""
				expectedClusterList = append(expectedClusterList, modifiedCluster)
			}

			sortClusters(clusterList)
			sortClusters(expectedClusterList)
			if !reflect.DeepEqual(clusterList, expectedClusterList) {
				t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(clusterList, expectedClusterList))
			}
		})
	}
}

func sortClusters(clusters []*kubermaticv1.Cluster) {
	sort.SliceStable(clusters, func(i, j int) bool {
		mi, mj := clusters[i], clusters[j]
		return mi.Spec.HumanReadableName < mj.Spec.HumanReadableName
	})
}

func genCluster(name, userEmail string, instance kubermaticv1.ClusterTemplateInstance) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          map[string]string{kubermaticv1.ProjectIDLabelKey: instance.Spec.ProjectID, kubernetes.ClusterTemplateInstanceLabelKey: instance.Name},
			ResourceVersion: "1",
			Finalizers:      []string{kubermaticapiv1.CredentialsSecretsCleanupFinalizer},
			Annotations:     map[string]string{kubermaticv1.ClusterTemplateUserAnnotationKey: userEmail},
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: name,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "fake-dc",
				Fake:           &kubermaticv1.FakeCloudSpec{},
			},
		},
		Status: kubermaticv1.ClusterStatus{
			UserEmail: userEmail,
		},
	}
}
