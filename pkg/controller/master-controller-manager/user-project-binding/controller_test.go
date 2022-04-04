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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac/test"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/diff"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
}

var (
	jamesBond = test.CreateUser("James Bond")
	bob       = test.CreateUser("Bob")
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
			existingProject: test.CreateProject("thunderball", jamesBond),
			existingUsers:   []*kubermaticv1.User{jamesBond},
			bindingToSync:   test.CreateExpectedEditorBinding("James Bond", test.CreateProject("thunderball", jamesBond)),
			expectedProject: test.CreateProject("thunderball"),
		},
		{
			name:            "scenario 2: no - op the owner reference already removed from a project (no previous owners) for James Bond - an editor",
			existingProject: test.CreateProject("thunderball"),
			existingUsers:   []*kubermaticv1.User{jamesBond},
			bindingToSync:   test.CreateExpectedEditorBinding("James Bond", test.CreateProject("thunderball", jamesBond)),
			expectedProject: test.CreateProject("thunderball"),
		},
		{
			name:            "scenario 3: the owner reference was removed from a project (with previous owners) for James Bond - an editor",
			existingProject: test.CreateProject("thunderball", jamesBond, bob),
			existingUsers:   []*kubermaticv1.User{jamesBond, bob},
			bindingToSync:   test.CreateExpectedEditorBinding("James Bond", test.CreateProject("thunderball", jamesBond)),
			expectedProject: test.CreateProject("thunderball", bob),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{test.bindingToSync}

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
			existingProject: test.CreateProject("thunderball", jamesBond),
			existingUsers:   []*kubermaticv1.User{jamesBond},
			bindingToSync:   test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", jamesBond)),
			expectedProject: test.CreateProject("thunderball", jamesBond),
		},
		{
			name:            "scenario 2: expected owner reference was added to a project - no previous owners)",
			existingProject: test.CreateProject("thunderball"),
			existingUsers:   []*kubermaticv1.User{jamesBond},
			bindingToSync:   test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", jamesBond)),
			expectedProject: test.CreateProject("thunderball", jamesBond),
		},
		{
			name:            "scenario 3: expected owner reference was added to a project - with previous owners)",
			existingProject: test.CreateProject("thunderball", jamesBond),
			existingUsers:   []*kubermaticv1.User{jamesBond, bob},
			bindingToSync:   test.CreateExpectedOwnerBinding("Bob", test.CreateProject("thunderball", bob)),
			expectedProject: test.CreateProject("thunderball", jamesBond, bob),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// setup the test scenario
			ctx := context.Background()
			objs := []ctrlruntimeclient.Object{test.bindingToSync}
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

			updatedProject.ObjectMeta.ResourceVersion = ""
			test.expectedProject.ObjectMeta.ResourceVersion = ""

			if !equality.Semantic.DeepEqual(updatedProject, test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(updatedProject, test.expectedProject))
			}
		})
	}
}
