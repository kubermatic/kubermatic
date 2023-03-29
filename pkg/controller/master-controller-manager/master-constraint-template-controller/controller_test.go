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

package masterconstrainttemplatecontroller

import (
	"context"
	"testing"
	"time"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	opa "k8c.io/api/v2/pkg/apis/open-policy-agent"
	kubermaticlog "k8c.io/kubermatic/v3/pkg/log"
	"k8c.io/kubermatic/v3/pkg/test/diff"
	"k8c.io/kubermatic/v3/pkg/test/generator"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const ctName = "requiredlabels"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                 string
		requestName          string
		expectedCT           *kubermaticv1.ConstraintTemplate
		expectedGetErrStatus metav1.StatusReason
		masterClient         ctrlruntimeclient.Client
		seedClient           ctrlruntimeclient.Client
	}{
		{
			name:        "scenario 1: sync ct to seed cluster",
			requestName: ctName,
			expectedCT:  genConstraintTemplate(ctName, false),
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(genConstraintTemplate(ctName, false), generator.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				Build(),
		},
		{
			name:                 "scenario 2: cleanup ct on seed cluster when master ct is being terminated",
			requestName:          ctName,
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			masterClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(genConstraintTemplate(ctName, true), generator.GenTestSeed()).
				Build(),
			seedClient: fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(genConstraintTemplate(ctName, false)).
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

			ct := &kubermaticv1.ConstraintTemplate{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, ct)
			if tc.expectedGetErrStatus != "" {
				if err == nil {
					t.Fatalf("expected error status %s, instead got ct: %v", tc.expectedGetErrStatus, ct)
				}

				if tc.expectedGetErrStatus != apierrors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, apierrors.ReasonForError(err))
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to get constraint template: %v", err)
			}

			ct.ResourceVersion = tc.expectedCT.ResourceVersion
			ct.Namespace = tc.expectedCT.Namespace
			ct.APIVersion = ""
			ct.Kind = ""

			if !diff.SemanticallyEqual(tc.expectedCT, ct) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedCT, ct))
			}
		})
	}
}

func genConstraintTemplate(name string, deleted bool) *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}
	ct.Name = name

	ct.Spec = genCTSpec()
	if deleted {
		deleteTime := metav1.NewTime(time.Now())
		ct.DeletionTimestamp = &deleteTime
		ct.Finalizers = append(ct.Finalizers, cleanupFinalizer)
	}

	return ct
}

func genCTSpec() kubermaticv1.ConstraintTemplateSpec {
	return kubermaticv1.ConstraintTemplateSpec{
		CRD: opa.CRD{
			Spec: opa.CRDSpec{
				Names: opa.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
				},
				Validation: &opa.Validation{
					OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
						Properties: map[string]apiextensionsv1.JSONSchemaProps{
							"labels": {
								Type: "array",
								Items: &apiextensionsv1.JSONSchemaPropsOrArray{
									Schema: &apiextensionsv1.JSONSchemaProps{
										Type: "string",
									},
								},
							},
						},
					},
				},
			},
		},
		Targets: []opa.Target{
			{
				Target: "admission.k8s.gatekeeper.sh",
				Rego: `
		package k8srequiredlabels

        deny[{"msg": msg, "details": {"missing_labels": missing}}] {
          provided := {label | input.review.object.metadata.labels[label]}
          required := {label | label := input.parameters.labels[_]}
          missing := required - provided
          count(missing) > 0
          msg := sprintf("you must provide labels: %v", [missing])
        }`,
			},
		},
	}
}
