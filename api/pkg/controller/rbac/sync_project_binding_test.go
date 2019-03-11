package rbac

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"

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
			existingProject: createProject("thunderball", createUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{createUser("James Bond")},
			bindingToSync:   createExpectedEditorBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
		},
		{
			name: "scenario 2: no - op the owner reference already removed from a project (no previous owners) for James Bond - an editor",
			existingProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
			existingUsers: []*kubermaticv1.User{createUser("James Bond")},
			bindingToSync: createExpectedEditorBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
		},
		{
			name: "scenario 3: the owner reference was removed from a project (with previous owners) for James Bond - an editor",
			existingProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = append(prj.OwnerReferences, metav1.OwnerReference{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
					UID:        "",
					Name:       "Bob",
				})
				return prj
			}(),
			existingUsers: []*kubermaticv1.User{createUser("James Bond"), createUser("Bob")},
			bindingToSync: createExpectedEditorBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
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

			scheme := scheme.Scheme
			if err := kubermaticv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			kubermaticFakeClient := fake.NewFakeClientWithScheme(scheme, objs...)

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
			existingProject: createProject("thunderball", createUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{createUser("James Bond")},
			bindingToSync:   createExpectedOwnerBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
			expectedProject: createProject("thunderball", createUser("James Bond")),
		},
		{
			name: "scenario 2: expected owner reference was added to a project - no previous owners)",
			existingProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
				prj.OwnerReferences = []metav1.OwnerReference{}
				return prj
			}(),
			existingUsers:   []*kubermaticv1.User{createUser("James Bond")},
			bindingToSync:   createExpectedOwnerBinding("James Bond", createProject("thunderball", createUser("James Bond"))),
			expectedProject: createProject("thunderball", createUser("James Bond")),
		},
		{
			name:            "scenario 3: expected owner reference was added to a project - with previous owners)",
			existingProject: createProject("thunderball", createUser("James Bond")),
			existingUsers:   []*kubermaticv1.User{createUser("James Bond"), createUser("Bob")},
			bindingToSync:   createExpectedOwnerBinding("Bob", createProject("thunderball", createUser("Bob"))),
			expectedProject: func() *kubermaticv1.Project {
				prj := createProject("thunderball", createUser("James Bond"))
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

			scheme := scheme.Scheme
			if err := kubermaticv1.AddToScheme(scheme); err != nil {
				t.Fatal(err)
			}
			kubermaticFakeClient := fake.NewFakeClientWithScheme(scheme, objs...)

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
