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

package serviceaccountprojectbindingcontroller

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcileBindingForProjectServiceAccount(t *testing.T) {
	tests := []struct {
		name                      string
		saName                    string
		existingKubermaticObjects []ctrlruntimeclient.Object
		expectedBinding           *kubermaticv1.UserProjectBinding
	}{
		{
			name:   "scenario 1: this test creates binding for service account",
			saName: "serviceaccount-abcd",
			existingKubermaticObjects: []ctrlruntimeclient.Object{
				genProject("my-first-project-ID"),
				genServiceAccount("abcd", "editors", "my-first-project-ID"),
			},
			expectedBinding: genSABindingWithOwnerRefs("my-first-project-ID", "serviceaccount-abcd", "serviceaccount-abcd@sa.kubermatic.io", "editors"),
		},
		{
			name:   "scenario 2: this test update binding group from viewers to editors",
			saName: "serviceaccount-abcd",
			existingKubermaticObjects: []ctrlruntimeclient.Object{
				genProject("my-first-project-ID"),
				genServiceAccount("abcd", "editors", "my-first-project-ID"),
				genSABinding("my-first-project-ID", "serviceaccount-abcd@sa.kubermatic.io", "viewers"),
			},
			expectedBinding: genSABindingWithOwnerRefs("my-first-project-ID", "serviceaccount-abcd", "serviceaccount-abcd@sa.kubermatic.io", "editors"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			kubermaticFakeClient := fake.NewClientBuilder().
				WithObjects(test.existingKubermaticObjects...).
				Build()

			// act
			ctx := context.Background()
			target := reconcileServiceAccountProjectBinding{Client: kubermaticFakeClient, log: zap.NewNop().Sugar()}

			_, err := target.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: test.saName}})

			// validate
			if err != nil {
				t.Fatal(err)
			}
			bindings := &kubermaticv1.UserProjectBindingList{}
			err = kubermaticFakeClient.List(ctx, bindings)
			if err != nil {
				t.Fatal(err)
			}

			if len(bindings.Items) != 1 {
				t.Fatalf("wrong number of bindigs, expected 1 got %d", len(bindings.Items))
			}

			binding := bindings.Items[0]
			binding.Name = test.expectedBinding.Name
			binding.ResourceVersion = ""

			if !diff.SemanticallyEqual(*test.expectedBinding, binding) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(test.expectedBinding, binding))
			}
		})
	}
}

func genSABinding(projectID, email, group string) *kubermaticv1.UserProjectBinding {
	binding := generator.GenBinding(projectID, email, group)
	binding.Labels = map[string]string{kubermaticv1.ProjectIDLabelKey: projectID}
	binding.Spec.Group = fmt.Sprintf("%s-%s", group, projectID)
	return binding
}

func genSABindingWithOwnerRefs(projectID, saName, email, group string) *kubermaticv1.UserProjectBinding {
	binding := genSABinding(projectID, email, group)
	binding.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.UserKindName,
			Name:       saName,
		},
	}
	return binding
}

func genProject(name string) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		Spec: kubermaticv1.ProjectSpec{Name: name},
		Status: kubermaticv1.ProjectStatus{
			Phase: kubermaticv1.ProjectActive,
		},
	}
}

func genServiceAccount(id, group, projectName string) *kubermaticv1.User {
	user := &kubermaticv1.User{}
	user.Labels = map[string]string{kubermaticv1.ServiceAccountInitialGroupLabel: fmt.Sprintf("%s-%s", group, projectName)}
	user.Name = kubermaticv1helper.EnsureProjectServiceAccountPrefix(id)
	user.Spec.Email = fmt.Sprintf("%s@sa.kubermatic.io", user.Name)
	user.Spec.Project = projectName

	return user
}
