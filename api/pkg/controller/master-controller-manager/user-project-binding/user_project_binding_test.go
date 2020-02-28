package userprojectbinding

import (
	"context"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac/test"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"

	controllerclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				prj.ResourceVersion = "1"
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
				prj.ResourceVersion = "1"
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
			objs := []runtime.Object{}

			for _, user := range test.existingUsers {
				objs = append(objs, user)
			}
			for _, binding := range test.existingBindings {

				objs = append(objs, binding)
			}

			if test.existingProject != nil {
				objs = append(objs, test.existingProject)
			}

			kubermaticFakeClient := fake.NewFakeClient(objs...)

			// act
			target := reconcileSyncProjectBinding{ctx: context.TODO(), Client: kubermaticFakeClient}

			err := target.ensureNotProjectOwnerForBinding(test.bindingToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}
			updatedProject := &kubermaticv1.Project{}

			err = kubermaticFakeClient.Get(target.ctx, controllerclient.ObjectKey{Name: "thunderball"}, updatedProject)
			if err != nil {
				t.Fatal(err)
			}

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
			existingUsers: []*kubermaticv1.User{test.CreateUser("James Bond")},
			bindingToSync: test.CreateExpectedOwnerBinding("James Bond", test.CreateProject("thunderball", test.CreateUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.ResourceVersion = "1"
				return prj
			}(),
		},
		{
			name:            "scenario 3: expected owner reference was added to a project - with previous owners)",
			existingProject: test.CreateProject("thunderball", test.CreateUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{test.CreateUser("James Bond"), test.CreateUser("Bob")},
			bindingToSync:   test.CreateExpectedOwnerBinding("Bob", test.CreateProject("thunderball", test.CreateUser("Bob"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := test.CreateProject("thunderball", test.CreateUser("James Bond"))
				prj.ResourceVersion = "1"
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
			objs := []runtime.Object{}
			for _, user := range test.existingUsers {
				objs = append(objs, user)
			}

			for _, binding := range test.existingBindings {
				objs = append(objs, binding)
			}
			if test.existingProject != nil {
				objs = append(objs, test.existingProject)
			}

			kubermaticFakeClient := fake.NewFakeClient(objs...)

			// act
			target := reconcileSyncProjectBinding{ctx: context.TODO(), Client: kubermaticFakeClient}
			err := target.ensureProjectOwnerForBinding(test.bindingToSync)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			updatedProject := &kubermaticv1.Project{}

			err = kubermaticFakeClient.Get(target.ctx, controllerclient.ObjectKey{Name: "thunderball"}, updatedProject)
			if err != nil {
				t.Fatal(err)
			}
			if !equality.Semantic.DeepEqual(updatedProject, test.expectedProject) {
				t.Fatalf("%v", diff.ObjectDiff(updatedProject, test.expectedProject))
			}
		})
	}
}
