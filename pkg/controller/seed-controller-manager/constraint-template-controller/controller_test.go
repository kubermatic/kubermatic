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

package constrainttemplatecontroller

import (
	"context"
	"testing"
	"time"

	constrainttemplatev1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"
	regoschema "github.com/open-policy-agent/frameworks/constraint/pkg/client/drivers/rego/schema"
	"github.com/open-policy-agent/frameworks/constraint/pkg/core/templates"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const ctName = "requiredlabels"

func TestReconcile(t *testing.T) {
	sch := fake.NewScheme()
	utilruntime.Must(constrainttemplatev1.AddToScheme(sch))

	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	testCases := []struct {
		name                 string
		requestName          string
		expectedCT           *kubermaticv1.ConstraintTemplate
		expectedGetErrStatus metav1.StatusReason
		seedClient           ctrlruntimeclient.Client
		userClient           ctrlruntimeclient.Client
	}{
		{
			name:        "scenario 1: sync ct to user cluster",
			requestName: ctName,
			expectedCT:  genConstraintTemplate(ctName, false),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genConstraintTemplate(ctName, false), genCluster("cluster", true)).
				Build(),
			userClient: fake.NewClientBuilder().WithScheme(sch).Build(),
		},
		{
			name:                 "scenario 2: dont sync ct to user cluster which has opa-integration off",
			requestName:          ctName,
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genConstraintTemplate(ctName, false), genCluster("cluster", false)).
				Build(),
			userClient: fake.NewClientBuilder().WithScheme(sch).Build(),
		},
		{
			name:                 "scenario 3: cleanup ct on user cluster when master ct is being terminated",
			requestName:          ctName,
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genConstraintTemplate(ctName, true), genCluster("cluster", true)).
				Build(),
			userClient: fake.
				NewClientBuilder().
				WithScheme(sch).
				WithObjects(genGKConstraintTemplate(ctName)).
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:                       kubermaticlog.Logger,
				workerNameLabelSelector:   workerSelector,
				recorder:                  &record.FakeRecorder{},
				seedClient:                tc.seedClient,
				userClusterClientProvider: newFakeClientProvider(tc.userClient),
				userClusterClients:        map[string]ctrlruntimeclient.Client{},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			ct := &constrainttemplatev1.ConstraintTemplate{}
			err := tc.userClient.Get(ctx, request.NamespacedName, ct)
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

			ct.ResourceVersion = ""
			ct.APIVersion = ""
			ct.Kind = ""

			if tc.expectedCT.Name != ct.Name {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedCT, ct))
			}

			if !diff.SemanticallyEqual(tc.expectedCT.Spec.CRD, ct.Spec.CRD) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedCT, ct))
			}

			if !diff.SemanticallyEqual(tc.expectedCT.Spec.Targets, ct.Spec.Targets) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedCT, ct))
			}
		})
	}
}

func TestDeleteWhenCTOnUserClusterIsMissing(t *testing.T) {
	sch := fake.NewScheme()
	utilruntime.Must(constrainttemplatev1.AddToScheme(sch))

	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	seedClient := fake.
		NewClientBuilder().
		WithObjects(genConstraintTemplate(ctName, true), genCluster("cluster", true)).
		Build()
	userClient := fake.
		NewClientBuilder().
		WithScheme(sch).
		WithObjects().
		Build()

	ctx := context.Background()
	r := &reconciler{
		log:                       kubermaticlog.Logger,
		workerNameLabelSelector:   workerSelector,
		recorder:                  &record.FakeRecorder{},
		seedClient:                seedClient,
		userClusterClientProvider: newFakeClientProvider(userClient),
		userClusterClients:        map[string]ctrlruntimeclient.Client{},
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: ctName}}
	if _, err := r.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconciling failed: %v", err)
	}

	err = userClient.Get(ctx, request.NamespacedName, &constrainttemplatev1.ConstraintTemplate{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if apierrors.ReasonForError(err) != metav1.StatusReasonNotFound {
		t.Fatalf("expected err: %v, got %v", metav1.StatusReasonNotFound, apierrors.ReasonForError(err))
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

func genGKConstraintTemplate(name string) *constrainttemplatev1.ConstraintTemplate {
	ct := &constrainttemplatev1.ConstraintTemplate{}
	ct.Name = name
	ct.Spec = constrainttemplatev1.ConstraintTemplateSpec{
		CRD:     genCTSpec().CRD,
		Targets: genCTSpec().Targets,
	}

	return ct
}

func genCTSpec() kubermaticv1.ConstraintTemplateSpec {
	return kubermaticv1.ConstraintTemplateSpec{
		CRD: constrainttemplatev1.CRD{
			Spec: constrainttemplatev1.CRDSpec{
				Names: constrainttemplatev1.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
				},
				Validation: &constrainttemplatev1.Validation{
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
		Targets: []constrainttemplatev1.Target{
			{
				Target: "admission.k8s.gatekeeper.sh",
				Code: []constrainttemplatev1.Code{
					{
						Engine: regoschema.Name,
						Source: &templates.Anything{
							Value: (&regoschema.Source{
								Rego: `package k8srequiredlabels

deny[{"msg": msg, "details": {"missing_labels": missing}}] {
  provided := {label | input.review.object.metadata.labels[label]}
  required := {label | label := input.parameters.labels[_]}
  missing := required - provided
  count(missing) > 0
  msg := sprintf("you must provide labels: %v", [missing])
}`,
							}).ToUnstructured(),
						},
					},
				},
			},
		},
	}
}

func genCluster(name string, opaEnabled bool) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			OPAIntegration: &kubermaticv1.OPAIntegrationSettings{
				Enabled: opaEnabled,
			},
			HumanReadableName: name,
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

type fakeClientProvider struct {
	client ctrlruntimeclient.Client
}

func newFakeClientProvider(client ctrlruntimeclient.Client) *fakeClientProvider {
	return &fakeClientProvider{
		client: client,
	}
}

func (f *fakeClientProvider) GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.client, nil
}
