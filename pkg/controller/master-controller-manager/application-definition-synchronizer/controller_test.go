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
	"testing"
	"time"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(appskubermaticv1.AddToScheme(scheme.Scheme))
}

const applicationDefinitionName = "app-def-1"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                          string
		requestName                   string
		expectedApplicationDefinition *appskubermaticv1.ApplicationDefinition
		masterClient                  ctrlruntimeclient.Client
		seedClient                    ctrlruntimeclient.Client
	}{
		{
			name:                          "scenario 1: sync application definition from master cluster to seed cluster",
			requestName:                   applicationDefinitionName,
			expectedApplicationDefinition: generateApplicationDef(applicationDefinitionName, false),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateApplicationDef(applicationDefinitionName, false), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				Build(),
		},
		{
			name:                          "scenario 2: cleanup application definition on the seed cluster when master application definition is being terminated\"",
			requestName:                   applicationDefinitionName,
			expectedApplicationDefinition: nil,
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generateApplicationDef(applicationDefinitionName, true), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generateApplicationDef(applicationDefinitionName, false), generator.GenTestSeed()).
				Build(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &events.FakeRecorder{},
				masterClient: tc.masterClient,
				seedClients:  map[string]ctrlruntimeclient.Client{"first": tc.seedClient},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			seedApplicationDef := &appskubermaticv1.ApplicationDefinition{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedApplicationDef)

			if tc.expectedApplicationDefinition == nil {
				if err == nil {
					t.Fatal("failed clean up application definition on the seed cluster")
				} else if !apierrors.IsNotFound(err) {
					t.Fatalf("failed to get application definition: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to get application definition: %v", err)
				}

				seedApplicationDef.ResourceVersion = ""
				seedApplicationDef.APIVersion = ""
				seedApplicationDef.Kind = ""

				if !diff.SemanticallyEqual(tc.expectedApplicationDefinition, seedApplicationDef) {
					t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedApplicationDefinition, seedApplicationDef))
				}

				// todo label and annotation
			}
		})
	}
}

func generateApplicationDef(name string, deleted bool) *appskubermaticv1.ApplicationDefinition {
	applicationDef := &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"someLabelKey": "someLabelValue",
			},
			Annotations: map[string]string{
				"someAnnotationKey": "someAnnotationValue",
			},
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			Description: "sample app",
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "version 1",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
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
		applicationDef.Finalizers = append(applicationDef.Finalizers, appskubermaticv1.ApplicationDefinitionSeedCleanupFinalizer)
	}

	return applicationDef
}
