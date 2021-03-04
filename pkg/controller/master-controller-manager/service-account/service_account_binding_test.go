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

package serviceaccount

import (
	"context"
	"fmt"
	"testing"

	"k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				genServiceAccount("abcd", "test", "editors", "my-first-project-ID"),
			},
			expectedBinding: genSABinding("my-first-project-ID", "serviceaccount-abcd", "serviceaccount-abcd@sa.kubermatic.io", "editors"),
		},
		{
			name:   "scenario 2: this test update binding group from viewers to editors",
			saName: "serviceaccount-abcd",
			existingKubermaticObjects: []ctrlruntimeclient.Object{
				genProject("my-first-project-ID"),
				genServiceAccount("abcd", "test", "editors", "my-first-project-ID"),
				genSABinding("my-first-project-ID", "serviceaccount-abcd", "serviceaccount-abcd@sa.kubermatic.io", "viewers"),
			},
			expectedBinding: genSABinding("my-first-project-ID", "serviceaccount-abcd", "serviceaccount-abcd@sa.kubermatic.io", "editors"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			kubermaticFakeClient := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(test.existingKubermaticObjects...).
				Build()

			// act
			ctx := context.Background()
			target := reconcileServiceAccountProjectBinding{Client: kubermaticFakeClient}

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

			if !equality.Semantic.DeepEqual(bindings.Items[0].Labels, test.expectedBinding.Labels) {
				t.Fatalf("%v", diff.ObjectDiff(bindings.Items[0].Labels, test.expectedBinding.Labels))
			}

			if !equality.Semantic.DeepEqual(bindings.Items[0].OwnerReferences, test.expectedBinding.OwnerReferences) {
				t.Fatalf("%v", diff.ObjectDiff(bindings.Items[0].OwnerReferences, test.expectedBinding.OwnerReferences))
			}

			if !equality.Semantic.DeepEqual(bindings.Items[0].Spec, test.expectedBinding.Spec) {
				t.Fatalf("%v", diff.ObjectDiff(bindings.Items[0].Spec, test.expectedBinding.Spec))
			}
		})
	}
}

func TestReconcileBindingForMainServiceAccount(t *testing.T) {

	tests := []struct {
		name                      string
		saName                    string
		existingKubermaticObjects []ctrlruntimeclient.Object
		expectedBindingNames      sets.String
	}{
		{
			name:                      "scenario 1: the human user has one owned project, new binding will be created",
			saName:                    "main-serviceaccount-abcd",
			existingKubermaticObjects: test.GenDefaultKubermaticObjects(genMainServiceAccount("abcd", "editors", test.GenDefaultUser().Spec.Email)),
			expectedBindingNames:      sets.NewString("my-first-project-ID-main-service", "my-first-project-ID-bob@acme.com"),
		},
		{
			name:   "scenario 2: the human doesn't have owned projects",
			saName: "main-serviceaccount-abcd",
			existingKubermaticObjects: []ctrlruntimeclient.Object{
				// add a project
				test.GenDefaultProject(),
				// add a user
				test.GenDefaultUser(),
				// add binding
				test.GenBinding(test.GenDefaultProject().Name, test.GenDefaultUser().Spec.Email, "editors"),
				genMainServiceAccount("abcd", "editors", test.GenDefaultUser().Spec.Email),
			},
			expectedBindingNames: sets.NewString("my-first-project-ID-bob@acme.com"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			kubermaticFakeClient := fakectrlruntimeclient.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(test.existingKubermaticObjects...).
				Build()

			// act
			ctx := context.Background()
			target := reconcileServiceAccountProjectBinding{Client: kubermaticFakeClient}

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

			if len(bindings.Items) != len(test.expectedBindingNames) {
				t.Fatalf("expected %d bindings got %d", len(test.expectedBindingNames), len(bindings.Items))
			}

			for _, binding := range bindings.Items {
				if test.expectedBindingNames.Has(binding.Name) {
					t.Fatalf("expected binding name")
				}
			}
		})
	}
}

func genSABinding(projectID, saName, email, group string) *kubermaticv1.UserProjectBinding {
	binding := test.GenBinding(projectID, email, group)
	binding.OwnerReferences[0].Kind = kubermaticv1.UserKindName
	binding.OwnerReferences[0].Name = saName
	binding.Labels = map[string]string{kubermaticv1.ProjectIDLabelKey: projectID}
	binding.Spec.Group = fmt.Sprintf("%s-%s", group, projectID)
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

func genServiceAccount(id, name, group, projectName string) *kubermaticv1.User {
	user := &kubermaticv1.User{}
	user.Labels = map[string]string{kubernetes.ServiceAccountLabelGroup: fmt.Sprintf("%s-%s", group, projectName)}
	user.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ProjectKindName,
			Name:       projectName,
			UID:        types.UID(id),
		},
	}
	user.Name = fmt.Sprintf("serviceaccount-%s", id)
	user.Spec.Email = "serviceaccount-abcd@sa.kubermatic.io"

	return user
}

func genMainServiceAccount(id, group, owner string) *kubermaticv1.User {
	user := &kubermaticv1.User{}
	user.Labels = map[string]string{kubernetes.ServiceAccountLabelGroup: group}
	user.Annotations = map[string]string{OwnerAnnotationKey: owner}

	user.Name = fmt.Sprintf("main-serviceaccount-%s", id)
	user.Spec.Email = fmt.Sprintf("main-serviceaccount-%s@sa.kubermatic.io", id)
	user.Spec.ID = id

	return user
}
