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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac/test"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	jamesBond   = test.CreateUser("James Bond")
	bob         = test.CreateUser("Bob")
	thunderball = test.CreateProject("thunderball")

	jamesAsOwner = metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.UserKindName,
		UID:        jamesBond.GetUID(),
		Name:       jamesBond.Name,
	}

	bobAsOwner = metav1.OwnerReference{
		APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		Kind:       kubermaticv1.UserKindName,
		UID:        bob.GetUID(),
		Name:       bob.Name,
	}
)

func projectWithOwner(p *kubermaticv1.Project, ownerRefs ...metav1.OwnerReference) *kubermaticv1.Project {
	project := p.DeepCopy()

	for _, ref := range ownerRefs {
		kuberneteshelper.EnsureOwnerReference(project, ref)
	}

	return project
}

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
			existingUsers:   []*kubermaticv1.User{jamesBond},
			existingProject: projectWithOwner(thunderball, jamesAsOwner),
			bindingToSync:   test.CreateExpectedEditorBinding("James Bond", thunderball),
			expectedProject: thunderball,
		},
		{
			name:            "scenario 2: no - op the owner reference already removed from a project (no previous owners) for James Bond - an editor",
			existingUsers:   []*kubermaticv1.User{jamesBond},
			existingProject: thunderball,
			bindingToSync:   test.CreateExpectedEditorBinding("James Bond", thunderball),
			expectedProject: thunderball,
		},
		{
			name:            "scenario 3: the owner reference was removed from a project (with previous owners) for James Bond - an editor",
			existingUsers:   []*kubermaticv1.User{jamesBond, bob},
			existingProject: projectWithOwner(thunderball, jamesAsOwner, bobAsOwner),
			existingBindings: []*kubermaticv1.UserProjectBinding{
				test.CreateExpectedOwnerBinding(bob.Name, thunderball),
			},
			bindingToSync:   test.CreateExpectedEditorBinding("James Bond", thunderball),
			expectedProject: projectWithOwner(thunderball, bobAsOwner),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{test.bindingToSync, test.existingProject}

			for _, user := range test.existingUsers {
				objs = append(objs, user)
			}
			for _, binding := range test.existingBindings {
				objs = append(objs, binding)
			}

			kubermaticFakeClient := fake.
				NewClientBuilder().
				WithObjects(objs...).
				Build()

			// act
			target := reconcileSyncProjectBinding{Client: kubermaticFakeClient}

			err := target.reconcile(ctx, zap.NewNop().Sugar(), test.bindingToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			updatedProject := &kubermaticv1.Project{}

			err = kubermaticFakeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: "thunderball"}, updatedProject)
			if err != nil {
				t.Fatal(err)
			}

			updatedProject.ResourceVersion = ""
			test.expectedProject.ResourceVersion = ""

			if !diff.SemanticallyEqual(test.expectedProject, updatedProject) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(test.expectedProject, updatedProject))
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
			existingUsers:   []*kubermaticv1.User{jamesBond},
			existingProject: projectWithOwner(thunderball, jamesAsOwner),
			bindingToSync:   test.CreateExpectedOwnerBinding(jamesBond.Name, thunderball),
			expectedProject: projectWithOwner(thunderball, jamesAsOwner),
		},
		{
			name:            "scenario 2: expected owner reference was added to a project - no previous owners)",
			existingUsers:   []*kubermaticv1.User{jamesBond},
			existingProject: thunderball,
			bindingToSync:   test.CreateExpectedOwnerBinding(jamesBond.Name, thunderball),
			expectedProject: projectWithOwner(thunderball, jamesAsOwner),
		},
		{
			name:            "scenario 3: expected owner reference was added to a project - with previous owners)",
			existingUsers:   []*kubermaticv1.User{jamesBond, bob},
			existingProject: projectWithOwner(thunderball, jamesAsOwner),
			bindingToSync:   test.CreateExpectedOwnerBinding(bob.Name, thunderball),
			expectedProject: projectWithOwner(thunderball, jamesAsOwner, bobAsOwner),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{test.bindingToSync, test.existingProject}

			for _, user := range test.existingUsers {
				objs = append(objs, user)
			}
			for _, binding := range test.existingBindings {
				objs = append(objs, binding)
			}

			kubermaticFakeClient := fake.
				NewClientBuilder().
				WithObjects(objs...).
				Build()

			// act
			target := reconcileSyncProjectBinding{Client: kubermaticFakeClient}
			err := target.reconcile(ctx, zap.NewNop().Sugar(), test.bindingToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			updatedProject := &kubermaticv1.Project{}

			err = kubermaticFakeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: "thunderball"}, updatedProject)
			if err != nil {
				t.Fatal(err)
			}

			updatedProject.ResourceVersion = ""
			test.expectedProject.ResourceVersion = ""

			if !diff.SemanticallyEqual(test.expectedProject, updatedProject) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(test.expectedProject, updatedProject))
			}
		})
	}
}
