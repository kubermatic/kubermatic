package test

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func CreateProject(name string, owner *kubermaticv1.User) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       kubermaticv1.ProjectKindName,
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:  types.UID(name) + "ID",
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
					UID:        owner.GetUID(),
					Name:       owner.Name,
				},
			},
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: name,
		},
		Status: kubermaticv1.ProjectStatus{
			Phase: kubermaticv1.ProjectInactive,
		},
	}
}

func CreateUser(name string) *kubermaticv1.User {
	return &kubermaticv1.User{
		TypeMeta: metav1.TypeMeta{
			Kind:       kubermaticv1.UserKindName,
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:  "",
			Name: name,
		},
		Spec: kubermaticv1.UserSpec{
			Email: fmt.Sprintf("%s@acme.com", name),
		},
	}
}

func CreateExpectedBindingFor(userName string, userGroup string, project *kubermaticv1.Project) *kubermaticv1.UserProjectBinding {
	user := CreateUser(userName)
	return &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("binding-for-%s", userName),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				},
			},
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			UserEmail: user.Spec.Email,
			ProjectID: project.Name,
			Group:     fmt.Sprintf("%s-%s", userGroup, project.Name),
		},
	}
}

func CreateExpectedOwnerBinding(userName string, project *kubermaticv1.Project) *kubermaticv1.UserProjectBinding {
	return CreateExpectedBindingFor(userName, "owners", project)
}

func CreateExpectedEditorBinding(userName string, project *kubermaticv1.Project) *kubermaticv1.UserProjectBinding {
	return CreateExpectedBindingFor(userName, "editors", project)
}
