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
	"reflect"
	"testing"

	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"

	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const ctName = "requiredlabels"

func TestReconcile(t *testing.T) {
	sch, err := v1beta1.SchemeBuilder.Build()
	if err != nil {
		t.Fatalf("building gatekeeper scheme failed: %v", err)
	}

	workerSelector, err := workerlabel.LabelSelector("")
	if err != nil {
		t.Fatalf("failed to build worker-name selector: %v", err)
	}

	testCases := []struct {
		name                 string
		requestName          string
		expectedCT           *kubermaticv1.ConstraintTemplate
		expectedGetErrStatus metav1.StatusReason
		masterClient         ctrlruntimeclient.Client
		seedClient           ctrlruntimeclient.Client
		userClient           ctrlruntimeclient.Client
	}{
		{
			name:         "scenario 1: sync ct to user cluster",
			requestName:  ctName,
			expectedCT:   genConstraintTemplate(ctName),
			masterClient: fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, genConstraintTemplate(ctName)),
			seedClient:   fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, genCluster("cluster", true)),
			userClient:   fakectrlruntimeclient.NewFakeClientWithScheme(sch),
		},
		{
			name:                 "scenario 2: dont sync ct to user cluster which has opa-integration off",
			requestName:          ctName,
			expectedGetErrStatus: metav1.StatusReasonNotFound,
			masterClient:         fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, genConstraintTemplate(ctName)),
			seedClient:           fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, genCluster("cluster", false)),
			userClient:           fakectrlruntimeclient.NewFakeClientWithScheme(sch),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			r := &reconciler{
				ctx:                     context.Background(),
				log:                     kubermaticlog.Logger,
				workerNameLabelSelector: workerSelector,
				masterClient:            tc.masterClient,
				seedClientProviders: map[string]*SeedClientProvider{
					"testSeed": {
						seedClient:                tc.seedClient,
						userClusterClientProvider: newFakeClientProvider(tc.userClient),
					},
				},
				userClusterClients: map[string]ctrlruntimeclient.Client{},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			ct := &v1beta1.ConstraintTemplate{}
			err := tc.userClient.Get(context.Background(), request.NamespacedName, ct)
			if tc.expectedGetErrStatus != "" {
				if tc.expectedGetErrStatus != errors.ReasonForError(err) {
					t.Fatalf("Expected error status %s differs from the expected one %s", tc.expectedGetErrStatus, errors.ReasonForError(err))
				}
				return
			}

			if err != nil {
				t.Fatalf("failed to get constraint template: %v", err)
			}

			if !reflect.DeepEqual(ct.Spec, tc.expectedCT.Spec) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(ct, tc.expectedCT))
			}

			if !reflect.DeepEqual(ct.Name, tc.expectedCT.Name) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(ct, tc.expectedCT))
			}
		})
	}
}

func genConstraintTemplate(name string) *kubermaticv1.ConstraintTemplate {
	ct := &kubermaticv1.ConstraintTemplate{}
	ct.Name = name
	ct.Spec = v1beta1.ConstraintTemplateSpec{
		CRD: v1beta1.CRD{
			Spec: v1beta1.CRDSpec{
				Names: v1beta1.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
				},
				Validation: &v1beta1.Validation{
					OpenAPIV3Schema: &apiextensionv1beta1.JSONSchemaProps{
						Properties: map[string]apiextensionv1beta1.JSONSchemaProps{
							"labels": {
								Type: "array",
								Items: &apiextensionv1beta1.JSONSchemaPropsOrArray{
									Schema: &apiextensionv1beta1.JSONSchemaProps{
										Type: "string",
									},
								},
							},
						},
					},
				},
			},
		},
		Targets: []v1beta1.Target{
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

	return ct
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

func (f *fakeClientProvider) GetClient(c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.client, nil
}
