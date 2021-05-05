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

package userprojectbinding

import (
	"context"
	"testing"

	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac/test"
	"k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/diff"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEnsureNotProjectOwnerForBinding(t *testing.T) {
	tests := []struct {
		name             string
		bindingToSync    *kubermaticv1.UserProjectBinding
		existingProject  *kubermaticv1.Project
		existingUsers    []*kubermaticv1.User
		expectedProject  *kubermaticv1.Project
		existingBindings []*kubermaticv1.UserProjectBinding
	}{
		{
			name:            "scenario 1: the owner reference is removed from a project (no previous owners) for James Bond - an editor",
			existingProject: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{test.CreateUser("James Bond")},
			bindingToSync:   test.CreateExpectedEditorBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
		},
		{
			name: "scenario 2: no - op the owner reference already removed from a project (no previous owners) for James Bond - an editor",
			existingProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
			existingUsers: []*kubermaticv1.User{test.CreateUser("James Bond")},
			bindingToSync: test.CreateExpectedEditorBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
		},
		{
			name: "scenario 3: the owner reference was removed from a project (with previous owners) for James Bond - an editor",
			existingProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.OwnerReferences = append(prj.OwnerReferences, metav1.OwnerReference{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
					UID:        "",
					Name:       "Bob",
				})
				return prj
			}(),
			existingUsers: []*kubermaticv1.User{test.CreateUser("James Bond"), test.CreateUser("Bob")},
			bindingToSync: test.CreateExpectedEditorBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{
					{
						APIVersion: kubermaticv1.SchemeGroupVersion.String(),
						Kind:       kubermaticv1.UserKindName,
						UID:        "",
						Name:       "Bob",
					},
				}
				return prj
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}

			for _, user := range test.existingUsers {
				objs = append(objs, user)
			}
			for _, binding := range test.existingBindings {

				objs = append(objs, binding)
			}

			if test.existingProject != nil {
				objs = append(objs, test.existingProject)
			}

			kubermaticFakeClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(objs...).
				Build()

			// act
			target := reconcileSyncProjectBinding{Client: kubermaticFakeClient}

			err := target.ensureNotProjectOwnerForBinding(ctx, test.bindingToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			updatedProject := &kubermaticv1.Project{}

			err = kubermaticFakeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: "thunderball"}, updatedProject)
			if err != nil {
				t.Fatal(err)
			}

			updatedProject.ObjectMeta.ResourceVersion = ""
			test.expectedProject.ObjectMeta.ResourceVersion = ""

			if !equality.Semantic.DeepEqual(updatedProject, test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(updatedProject, test.expectedProject))
			}
		})
	}
}

func TestEnsureProjectOwnerForBinding(t *testing.T) {
	tests := []struct {
		name             string
		bindingToSync    *kubermaticv1.UserProjectBinding
		existingProject  *kubermaticv1.Project
		existingUsers    []*kubermaticv1.User
		expectedProject  *kubermaticv1.Project
		existingBindings []*kubermaticv1.UserProjectBinding
	}{
		{
			name:            "scenario 1: no-op the owner reference already attached to the project",
			existingProject: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{test.CreateUser("James Bond")},
			bindingToSync:   test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond"))),
			expectedProject: test.CreateProject("thunderball", test.CreateUser("James Bond")),
		},
		{
			name: "scenario 2: expected owner reference was added to a project - no previous owners)",
			existingProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
			existingUsers:   []*kubermaticv1.User{test.CreateUser("James Bond")},
			bindingToSync:   test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond"))),
			expectedProject: test.CreateProject("thunderball", test.CreateUser("James Bond")),
		},
		{
			name:            "scenario 3: expected owner reference was added to a project - with previous owners)",
			existingProject: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{test.CreateUser("James Bond"), test.CreateUser("Bob")},
			bindingToSync:   test.CreateExpectedOwnerBinding("Bob", test.CreateProject("thunderball", test.CreateUser("Bob"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.OwnerReferences = append(prj.OwnerReferences, metav1.OwnerReference{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
					UID:        "",
					Name:       "Bob",
				})
				return prj
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{}
			for _, user := range test.existingUsers {
				objs = append(objs, user)
			}

			for _, binding := range test.existingBindings {
				objs = append(objs, binding)
			}
			if test.existingProject != nil {
				objs = append(objs, test.existingProject)
			}

			kubermaticFakeClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(objs...).
				Build()

			// act
			target := reconcileSyncProjectBinding{Client: kubermaticFakeClient}
			err := target.ensureProjectOwnerForBinding(ctx, test.bindingToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			updatedProject := &kubermaticv1.Project{}

			err = kubermaticFakeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: "thunderball"}, updatedProject)
			if err != nil {
				t.Fatal(err)
			}

			updatedProject.ObjectMeta.ResourceVersion = ""
			test.expectedProject.ObjectMeta.ResourceVersion = ""

			if !equality.Semantic.DeepEqual(updatedProject, test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(updatedProject, test.expectedProject))
			}
		})
	}
}
