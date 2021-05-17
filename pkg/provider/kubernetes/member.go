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

package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewProjectMemberProvider returns a project members provider
func NewProjectMemberProvider(createMasterImpersonatedClient impersonationClient, clientPrivileged ctrlruntimeclient.Client, isServiceAccountFunc func(string) bool) *ProjectMemberProvider {
	return &ProjectMemberProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               clientPrivileged,
		isServiceAccountFunc:           isServiceAccountFunc,
	}
}

var _ provider.ProjectMemberProvider = &ProjectMemberProvider{}

// ProjectMemberProvider binds users with projects
type ProjectMemberProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient impersonationClient

	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged ctrlruntimeclient.Client

	// since service account are special type of user this functions
	// helps to determine if the given email address belongs to a service account
	isServiceAccountFunc func(email string) bool
}

// Create creates a binding for the given member and the given project
func (p *ProjectMemberProvider) Create(userInfo *provider.UserInfo, project *kubermaticapiv1.Project, memberEmail, group string) (*kubermaticapiv1.UserProjectBinding, error) {
	if p.isServiceAccountFunc(memberEmail) {
		return nil, kerrors.NewBadRequest(fmt.Sprintf("cannot add the given member %s to the project %s because the email indicates a service account", memberEmail, project.Spec.Name))
	}

	binding := genBinding(project, memberEmail, group)

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := masterImpersonatedClient.Create(context.Background(), binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// List gets all members of the given project
func (p *ProjectMemberProvider) List(userInfo *provider.UserInfo, project *kubermaticapiv1.Project, options *provider.ProjectMemberListOptions) ([]*kubermaticapiv1.UserProjectBinding, error) {
	allMembers := &kubermaticapiv1.UserProjectBindingList{}
	if err := p.clientPrivileged.List(context.Background(), allMembers); err != nil {
		return nil, err
	}

	projectMembers := []*kubermaticapiv1.UserProjectBinding{}
	for _, member := range allMembers.Items {
		if member.Spec.ProjectID == project.Name {

			// The provider should serve only regular users as a members.
			// The ServiceAccount is another type of the user and should not be append to project members.
			if p.isServiceAccountFunc(member.Spec.UserEmail) {
				continue
			}
			projectMembers = append(projectMembers, member.DeepCopy())
		}
	}

	if options == nil {
		options = &provider.ProjectMemberListOptions{}
	}

	// Note:
	// After we get the list of members we try to get at least one item using unprivileged account to see if the user have read access
	if len(projectMembers) > 0 {
		if !options.SkipPrivilegeVerification {
			masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
			if err != nil {
				return nil, err
			}

			memberToGet := projectMembers[0]
			err = masterImpersonatedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: memberToGet.Name}, &kubermaticapiv1.UserProjectBinding{})
			if err != nil {
				return nil, err
			}
		}
	}

	if len(options.MemberEmail) == 0 {
		return projectMembers, nil
	}

	filteredMembers := []*kubermaticapiv1.UserProjectBinding{}
	if options != nil {
		for _, member := range projectMembers {
			if strings.EqualFold(member.Spec.UserEmail, options.MemberEmail) {
				filteredMembers = append(filteredMembers, member)
				break
			}
		}
	}

	return filteredMembers, nil
}

// Delete deletes the given binding
// Note:
// Use List to get binding for the specific member of the given project
func (p *ProjectMemberProvider) Delete(userInfo *provider.UserInfo, bindingName string) error {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}
	return masterImpersonatedClient.Delete(context.Background(), &kubermaticapiv1.UserProjectBinding{ObjectMeta: metav1.ObjectMeta{Name: bindingName}})
}

// Update updates the given binding
func (p *ProjectMemberProvider) Update(userInfo *provider.UserInfo, binding *kubermaticapiv1.UserProjectBinding) (*kubermaticapiv1.UserProjectBinding, error) {
	if rbac.ExtractGroupPrefix(binding.Spec.Group) == rbac.OwnerGroupNamePrefix && !kuberneteshelper.HasFinalizer(binding, rbac.CleanupFinalizerName) {
		kuberneteshelper.AddFinalizer(binding, rbac.CleanupFinalizerName)
	}
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := masterImpersonatedClient.Update(context.Background(), binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// MapUserToGroup maps the given user to a specific group of the given project
// This function is unsafe in a sense that it uses privileged account to list all members in the system
func (p *ProjectMemberProvider) MapUserToGroup(userEmail string, projectID string) (string, error) {
	allMembers := &kubermaticapiv1.UserProjectBindingList{}
	if err := p.clientPrivileged.List(context.Background(), allMembers); err != nil {
		return "", err
	}

	for _, member := range allMembers.Items {
		if strings.EqualFold(member.Spec.UserEmail, userEmail) && member.Spec.ProjectID == projectID {
			return member.Spec.Group, nil
		}
	}

	return "", kerrors.NewForbidden(schema.GroupResource{}, projectID, fmt.Errorf("%q doesn't belong to the given project = %s", userEmail, projectID))
}

// MappingsFor returns the list of projects (bindings) for the given user
// This function is unsafe in a sense that it uses privileged account to list all members in the system
func (p *ProjectMemberProvider) MappingsFor(userEmail string) ([]*kubermaticapiv1.UserProjectBinding, error) {
	allMemberMappings := &kubermaticapiv1.UserProjectBindingList{}
	if err := p.clientPrivileged.List(context.Background(), allMemberMappings); err != nil {
		return nil, err
	}

	memberMappings := []*kubermaticapiv1.UserProjectBinding{}
	for _, memberMapping := range allMemberMappings.Items {
		if strings.EqualFold(memberMapping.Spec.UserEmail, userEmail) {
			memberMappings = append(memberMappings, memberMapping.DeepCopy())
		}
	}

	return memberMappings, nil
}

// CreateUnsecured creates a binding for the given member and the given project
// This function is unsafe in a sense that it uses privileged account to create the resource
func (p *ProjectMemberProvider) CreateUnsecured(project *kubermaticapiv1.Project, memberEmail, group string) (*kubermaticapiv1.UserProjectBinding, error) {
	if p.isServiceAccountFunc(memberEmail) {
		return nil, kerrors.NewBadRequest(fmt.Sprintf("cannot add the given member %s to the project %s because the email indicates a service account", memberEmail, project.Spec.Name))
	}

	binding := genBinding(project, memberEmail, group)

	if err := p.clientPrivileged.Create(context.Background(), binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// CreateUnsecuredForServiceAccount creates a binding for the given service account and the given project
// This function is unsafe in a sense that it uses privileged account to create the resource
func (p *ProjectMemberProvider) CreateUnsecuredForServiceAccount(project *kubermaticapiv1.Project, memberEmail, group string) (*kubermaticapiv1.UserProjectBinding, error) {
	if p.isServiceAccountFunc(memberEmail) && !strings.HasPrefix(group, rbac.ProjectManagerGroupNamePrefix) {
		return nil, kerrors.NewBadRequest(fmt.Sprintf("cannot add the given member %s to the project %s because the email indicates a service account", memberEmail, project.Spec.Name))
	}

	binding := genBinding(project, memberEmail, group)

	if err := p.clientPrivileged.Create(context.Background(), binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// DeleteUnsecured deletes the given binding
// Note:
// Use List to get binding for the specific member of the given project
// This function is unsafe in a sense that it uses privileged account to delete the resource
func (p *ProjectMemberProvider) DeleteUnsecured(bindingName string) error {
	return p.clientPrivileged.Delete(context.Background(), &kubermaticapiv1.UserProjectBinding{ObjectMeta: metav1.ObjectMeta{Name: bindingName}})
}

// UpdateUnsecured updates the given binding
// This function is unsafe in a sense that it uses privileged account to update the resource
func (p *ProjectMemberProvider) UpdateUnsecured(binding *kubermaticapiv1.UserProjectBinding) (*kubermaticapiv1.UserProjectBinding, error) {
	if rbac.ExtractGroupPrefix(binding.Spec.Group) == rbac.OwnerGroupNamePrefix && !kuberneteshelper.HasFinalizer(binding, rbac.CleanupFinalizerName) {
		kuberneteshelper.AddFinalizer(binding, rbac.CleanupFinalizerName)
	}

	if err := p.clientPrivileged.Update(context.Background(), binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func genBinding(project *kubermaticapiv1.Project, memberEmail, group string) *kubermaticapiv1.UserProjectBinding {
	finalizers := []string{}
	if rbac.ExtractGroupPrefix(group) == rbac.OwnerGroupNamePrefix {
		finalizers = append(finalizers, rbac.CleanupFinalizerName)
	}
	return &kubermaticapiv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
					Kind:       kubermaticapiv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				},
			},
			Name:       rand.String(10),
			Finalizers: finalizers,
		},
		Spec: kubermaticapiv1.UserProjectBindingSpec{
			ProjectID: project.Name,
			UserEmail: memberEmail,
			Group:     group,
		},
	}
}
