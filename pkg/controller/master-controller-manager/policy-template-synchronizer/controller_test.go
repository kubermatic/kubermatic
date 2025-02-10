/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package policytemplatesynchronizer

import (
	"context"
	"testing"
	"time"

	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	podsecurityapi "k8s.io/pod-security-admission/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
}

const policyTemplateName = "policy-template-test"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                   string
		requestName            string
		expectedPolicyTemplate *kubermaticv1.PolicyTemplate
		masterClient           ctrlruntimeclient.Client
		seedClient             ctrlruntimeclient.Client
	}{
		{
			name:                   "scenario 1: sync policy template from master cluster to seed cluster",
			requestName:            policyTemplateName,
			expectedPolicyTemplate: generatePolicyTemplate(policyTemplateName, false),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generatePolicyTemplate(policyTemplateName, false), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				Build(),
		},
		{
			name:                   "scenario 2: cleanup policy template on the seed cluster when master policy template is being terminated",
			requestName:            policyTemplateName,
			expectedPolicyTemplate: nil,
			masterClient: fake.
				NewClientBuilder().
				WithObjects(generatePolicyTemplate(policyTemplateName, true), generator.GenTestSeed()).
				Build(),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(generatePolicyTemplate(policyTemplateName, false), generator.GenTestSeed()).
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

			seedPolicyTemplate := &kubermaticv1.PolicyTemplate{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, seedPolicyTemplate)
			if tc.expectedPolicyTemplate == nil {
				if err == nil {
					t.Fatal("failed clean up policy template on the seed cluster")
				} else if !apierrors.IsNotFound(err) {
					t.Fatalf("failed to get policy template: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("failed to get policy template: %v", err)
				}

				seedPolicyTemplate.ResourceVersion = ""
				seedPolicyTemplate.APIVersion = ""
				seedPolicyTemplate.Kind = ""

				if !diff.SemanticallyEqual(tc.expectedPolicyTemplate, seedPolicyTemplate) {
					t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedPolicyTemplate, seedPolicyTemplate))
				}
			}
		})
	}
}

func generatePolicyTemplate(name string, deleted bool) *kubermaticv1.PolicyTemplate {
	pt := &kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Title:       "test policy template",
			Description: "test policy template description",
			Visibility:  "project",
			KyvernoPolicySpec: kyvernov1.Spec{
				Rules: []kyvernov1.Rule{
					{
						Name: "test-rule",
						MatchResources: kyvernov1.MatchResources{
							Any: []kyvernov1.ResourceFilter{
								{
									ResourceDescription: kyvernov1.ResourceDescription{
										Kinds: []string{"v1/Pod"},
									},
								},
							},
						},
						Validation: &kyvernov1.Validation{
							Message: "test message",
							PodSecurity: &kyvernov1.PodSecurity{
								Level:   podsecurityapi.LevelBaseline,
								Version: "latest",
							},
						},
					},
				},
			},
		},
	}
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		pt.DeletionTimestamp = &deleteTime
		pt.Finalizers = append(pt.Finalizers, kubermaticv1.PolicyTemplateSeedCleanupFinalizer)
	}
	return pt
}
