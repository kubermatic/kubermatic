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

package clustertemplatesynchronizer

import (
	"context"
	"reflect"
	"testing"
	"time"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const clusterTemplateName = "cluster-template-test"

func TestReconcile(t *testing.T) {

	testCases := []struct {
		name                    string
		requestName             string
		expectedClusterTemplate *kubermaticv1.ClusterTemplate
		masterClient            ctrlruntimeclient.Client
		seedClient              ctrlruntimeclient.Client
	}{
		{
			name:                    "scenario 1: sync cluster template from master cluster to seed cluster",
			requestName:             clusterTemplateName,
			expectedClusterTemplate: generateClusterTemplate(clusterTemplateName, false),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateClusterTemplate(clusterTemplateName, false), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				Build(),
		},
		{
			name:                    "scenario 2: cleanup cluster template on the seed cluster when master cluster template is being terminated",
			requestName:             clusterTemplateName,
			expectedClusterTemplate: nil,
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateClusterTemplate(clusterTemplateName, true), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(generateClusterTemplate(clusterTemplateName, false), test.GenTestSeed()).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &record.FakeRecorder{},
				masterClient: tc.masterClient,
				seedClients:  map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			seedClusterTemplate := &kubermaticv1.ClusterTemplate{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedClusterTemplate)
			if tc.expectedClusterTemplate == nil {
				if err == nil {
					t.Fatal("failed clean up template on the seed cluster")
				} else if !errors.IsNotFound(err) {
					t.Fatalf("failed to get template: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to get template: %v", err)
				}
				if !reflect.DeepEqual(seedClusterTemplate.Spec, tc.expectedClusterTemplate.Spec) {
					t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(seedClusterTemplate, tc.expectedClusterTemplate))
				}
				if !reflect.DeepEqual(seedClusterTemplate.Name, tc.expectedClusterTemplate.Name) {
					t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(seedClusterTemplate, tc.expectedClusterTemplate))
				}
			}
		})
	}
}

func generateClusterTemplate(name string, deleted bool) *kubermaticv1.ClusterTemplate {
	ct := &kubermaticv1.ClusterTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: test.GenDefaultCluster().Spec.Cloud,
		},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		ct.DeletionTimestamp = &deleteTime
		ct.Finalizers = append(ct.Finalizers, v1.ClusterTemplateSeedCleanupFinalizer)
	}
	return ct
}
