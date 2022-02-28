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

package applicationdefinitionsynchronizer

import (
	"context"
	"reflect"
	"testing"
	"time"

	appkubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(appkubermaticv1.AddToScheme(scheme.Scheme))
}

const applicationDefinitionName = "app-def-1"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                          string
		requestName                   string
		expectedApplicationDefinition *appkubermaticv1.ApplicationDefinition
		masterClient                  ctrlruntimeclient.Client
		seedClient                    ctrlruntimeclient.Client
	}{
		{
			name:                          "scenario 1: sync application definition from master cluster to seed cluster",
			requestName:                   applicationDefinitionName,
			expectedApplicationDefinition: generateApplicationDef(applicationDefinitionName, false),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(generateApplicationDef(applicationDefinitionName, false), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				Build(),
		},
		{
			name:                          "scenario 2: cleanup application definition on the seed cluster when master application definition is being terminated\"",
			requestName:                   applicationDefinitionName,
			expectedApplicationDefinition: nil,
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(generateApplicationDef(applicationDefinitionName, true), test.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithObjects(generateApplicationDef(applicationDefinitionName, false), test.GenTestSeed()).
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

			seedApplicationDef := &appkubermaticv1.ApplicationDefinition{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedApplicationDef)

			if tc.expectedApplicationDefinition == nil {
				if err == nil {
					t.Fatal("failed clean up application definition on the seed cluster")
				} else if !errors.IsNotFound(err) {
					t.Fatalf("failed to get application definition: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to get application definition: %v", err)
				}
				if !reflect.DeepEqual(seedApplicationDef.Name, tc.expectedApplicationDefinition.Name) {
					t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(seedApplicationDef, tc.expectedApplicationDefinition))
				}
				if !reflect.DeepEqual(seedApplicationDef.Labels, tc.expectedApplicationDefinition.Labels) {
					t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(seedApplicationDef, tc.expectedApplicationDefinition))
				}
				if !reflect.DeepEqual(seedApplicationDef.Annotations, tc.expectedApplicationDefinition.Annotations) {
					t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(seedApplicationDef, tc.expectedApplicationDefinition))
				}
				if !reflect.DeepEqual(seedApplicationDef.Spec, tc.expectedApplicationDefinition.Spec) {
					t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(seedApplicationDef, tc.expectedApplicationDefinition))
				}
				// todo label and annotation
			}
		})
	}
}

func generateApplicationDef(name string, deleted bool) *appkubermaticv1.ApplicationDefinition {
	applicationDef := &appkubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"someLabelKey": "someLabelValue",
			},
			Annotations: map[string]string{
				"someAnnotationKey": "someAnnotationValue",
			},
		},
		Spec: appkubermaticv1.ApplicationDefinitionSpec{
			Description: "sample app",
			Versions: []appkubermaticv1.ApplicationVersion{
				{
					Version: "version 1",
					Constraints: appkubermaticv1.ApplicationConstraints{
						K8sVersion: "> 1.0.0",
						KKPVersion: "> 1.1.1",
					},
					Template: appkubermaticv1.ApplicationTemplate{
						Source: appkubermaticv1.ApplicationSource{
							Helm: &appkubermaticv1.HelmSource{
								URL:          "https://my-chart-repo.local",
								ChartName:    "sample-app",
								ChartVersion: "1.0",
							},
						},
					},
				},
			},
		},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		applicationDef.DeletionTimestamp = &deleteTime
		applicationDef.Finalizers = append(applicationDef.Finalizers, appkubermaticv1.ApplicationDefinitionSeedCleanupFinalizer)
	}

	return applicationDef
}
