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
	"k8s.io/apimachinery/pkg/util/diff"

	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name               string
		constraintTemplate *v1.ConstraintTemplate
		requestName        string
		expectedCT         *v1.ConstraintTemplate
	}{
		{
			name:               "sync ct to user cluster",
			constraintTemplate: genConstraintTemplate("requiredlabels"),
			requestName:        "requiredlabels",
			expectedCT:         genConstraintTemplate("requiredlabels"),
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sch, err := v1beta1.SchemeBuilder.Build()
			if err != nil {
				t.Fatalf("building gatekeeper scheme failed: %v", err)
			}
			seedClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.constraintTemplate)
			userClient := fakectrlruntimeclient.NewFakeClientWithScheme(sch)

			r := &reconciler{
				ctx:        context.Background(),
				log:        kubermaticlog.Logger,
				userClient: userClient,
				seedClient: seedClient,
				recorder:   record.NewFakeRecorder(10),
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			ct := &v1beta1.ConstraintTemplate{}
			if err := userClient.Get(context.Background(), request.NamespacedName, ct); err != nil {
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

func genConstraintTemplate(name string) *v1.ConstraintTemplate {
	ct := &v1.ConstraintTemplate{}
	ct.Name = name
	ct.Spec = v1beta1.ConstraintTemplateSpec{
		CRD: v1beta1.CRD{
			Spec: v1beta1.CRDSpec{
				Names: v1beta1.Names{
					Kind:       "labelconstraint",
					ShortNames: []string{"lc"},
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
