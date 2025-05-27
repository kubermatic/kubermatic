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

package test

import (
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func CreateProject(name string) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       kubermaticv1.ProjectKindName,
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			UID:             types.UID(name) + "ID",
			Name:            name,
			ResourceVersion: "1",
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

func CreateBindingFor(userName string, userGroup string, project *kubermaticv1.Project) *kubermaticv1.UserProjectBinding {
	user := CreateUser(userName)
	return &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("binding-for-%s", strings.ReplaceAll(userName, " ", "-")),
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			UserEmail: user.Spec.Email,
			ProjectID: project.Name,
			Group:     fmt.Sprintf("%s-%s", userGroup, project.Name),
		},
	}
}

func CreateExpectedOwnerBinding(userName string, project *kubermaticv1.Project) *kubermaticv1.UserProjectBinding {
	return CreateBindingFor(userName, "owners", project)
}

func CreateExpectedEditorBinding(userName string, project *kubermaticv1.Project) *kubermaticv1.UserProjectBinding {
	return CreateBindingFor(userName, "editors", project)
}
