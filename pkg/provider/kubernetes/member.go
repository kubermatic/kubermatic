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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/controller/master-controller-manager/rbac"
	"k8c.io/kubermatic/v2/pkg/provider"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kubectl/pkg/util/slice"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewProjectMemberProvider returns a project members provider.
func NewProjectMemberProvider(createMasterImpersonatedClient ImpersonationClient, clientPrivileged ctrlruntimeclient.Client) *ProjectMemberProvider {
	return &ProjectMemberProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               clientPrivileged,
	}
}

var _ provider.ProjectMemberProvider = &ProjectMemberProvider{}
var _ provider.PrivilegedProjectMemberProvider = &ProjectMemberProvider{}

// ProjectMemberProvider binds users with projects.
type ProjectMemberProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient ImpersonationClient

	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged ctrlruntimeclient.Client
}

var _ provider.ProjectMemberProvider = &ProjectMemberProvider{}

// Create creates a binding for the given member and the given project.
func (p *ProjectMemberProvider) Create(ctx context.Context, userInfo *provider.UserInfo, project *kubermaticv1.Project, memberEmail, group string) (*kubermaticv1.UserProjectBinding, error) {
	if kubermaticv1helper.IsProjectServiceAccount(memberEmail) {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("cannot add the given member %s to the project %s because the email indicates a service account", memberEmail, project.Spec.Name))
	}

	binding := genBinding(project, memberEmail, group)

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := masterImpersonatedClient.Create(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// List gets all members of the given project.
func (p *ProjectMemberProvider) List(ctx context.Context, userInfo *provider.UserInfo, project *kubermaticv1.Project, options *provider.ProjectMemberListOptions) ([]*kubermaticv1.UserProjectBinding, error) {
	allMembers := &kubermaticv1.UserProjectBindingList{}
	if err := p.clientPrivileged.List(ctx, allMembers); err != nil {
		return nil, err
	}

	projectMembers := []*kubermaticv1.UserProjectBinding{}
	for _, member := range allMembers.Items {
		if member.Spec.ProjectID == project.Name {
			// The provider should serve only regular users as a members.
			// The ServiceAccount is another type of the user and should not be append to project members.
			if kubermaticv1helper.IsProjectServiceAccount(member.Spec.UserEmail) {
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
			err = masterImpersonatedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: memberToGet.Name}, &kubermaticv1.UserProjectBinding{})
			if err != nil {
				return nil, err
			}
		}
	}

	if len(options.MemberEmail) == 0 {
		return projectMembers, nil
	}

	filteredMembers := []*kubermaticv1.UserProjectBinding{}
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
// Use List to get binding for the specific member of the given project.
func (p *ProjectMemberProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, bindingName string) error {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}
	return masterImpersonatedClient.Delete(ctx, &kubermaticv1.UserProjectBinding{ObjectMeta: metav1.ObjectMeta{Name: bindingName}})
}

// Update updates the given binding.
func (p *ProjectMemberProvider) Update(ctx context.Context, userInfo *provider.UserInfo, binding *kubermaticv1.UserProjectBinding) (*kubermaticv1.UserProjectBinding, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := masterImpersonatedClient.Update(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// MapUserToGroup maps the given user to a specific group of the given project
// This function is unsafe in a sense that it uses privileged account to list all members in the system.
func (p *ProjectMemberProvider) MapUserToGroup(ctx context.Context, userEmail string, projectID string) (string, error) {
	group, err := getUserBindingRole(ctx, userEmail, projectID, p.clientPrivileged)
	if err != nil {
		return "", err
	}
	if group == "" {
		return "", apierrors.NewForbidden(schema.GroupResource{}, projectID, fmt.Errorf("there is no user binding between %q user and %s project", userEmail, projectID))
	}
	return group, nil
}

// MapUserToGroups returns the groups of the user in the project. It combines identity provider groups with
// group from UserProjectBinding (if exists). Groups returned by this function are suffixed with project's ID to
// avoid leaking permissions among projects having binding with the same group but different roles.
// This function is unsafe in a sense that it uses privileged account to list all userProjectBindings in the system.
func (p *ProjectMemberProvider) MapUserToGroups(ctx context.Context, user *kubermaticv1.User, projectID string) (sets.String, error) {
	groups := sets.NewString()

	userBindingGroup, err := getUserBindingRole(ctx, user.Spec.Email, projectID, p.clientPrivileged)
	if err != nil {
		return nil, err
	}

	if userBindingGroup != "" {
		groups.Insert(userBindingGroup)
	}

	idpGroups := user.Spec.Groups

	groupBindings, err := p.GroupMappingsFor(ctx, idpGroups)
	if err != nil {
		return nil, err
	}
	for _, binding := range groupBindings {
		if binding.Spec.ProjectID == projectID {
			suffixedGroupName := fmt.Sprintf("%s-%s", binding.Spec.Group, projectID)
			groups.Insert(suffixedGroupName)
		}
	}

	if groups.Len() > 0 {
		return groups, nil
	} else {
		return nil, apierrors.NewForbidden(
			schema.GroupResource{},
			projectID,
			fmt.Errorf("%q doesn't belong to project %s", user.Spec.Email, projectID),
		)
	}
}

func getUserBindingRole(ctx context.Context, userEmail, projectID string, client ctrlruntimeclient.Client) (string, error) {
	allMembers := &kubermaticv1.UserProjectBindingList{}
	if err := client.List(ctx, allMembers); err != nil {
		return "", err
	}

	for _, member := range allMembers.Items {
		if strings.EqualFold(member.Spec.UserEmail, userEmail) && member.Spec.ProjectID == projectID {
			return member.Spec.Group, nil
		}
	}
	return "", nil
}

// MappingsFor returns the list of projects (bindings) for the given user
// This function is unsafe in a sense that it uses privileged account to list all members in the system.
func (p *ProjectMemberProvider) MappingsFor(ctx context.Context, userEmail string) ([]*kubermaticv1.UserProjectBinding, error) {
	allMemberMappings := &kubermaticv1.UserProjectBindingList{}
	if err := p.clientPrivileged.List(ctx, allMemberMappings); err != nil {
		return nil, err
	}

	memberMappings := []*kubermaticv1.UserProjectBinding{}
	for _, memberMapping := range allMemberMappings.Items {
		if strings.EqualFold(memberMapping.Spec.UserEmail, userEmail) {
			memberMappings = append(memberMappings, memberMapping.DeepCopy())
		}
	}

	return memberMappings, nil
}

// GroupMappingsFor returns the list of projects (bindings) for the given set of groups.
// This function is unsafe in a sense that it uses privileged account to list all members in the system.
func (p *ProjectMemberProvider) GroupMappingsFor(ctx context.Context, groups []string) ([]*kubermaticv1.GroupProjectBinding, error) {
	groupSet := sets.NewString(groups...)

	allBindings := &kubermaticv1.GroupProjectBindingList{}
	if err := p.clientPrivileged.List(ctx, allBindings); err != nil {
		return nil, err
	}

	var bindingsForGivenGroups []*kubermaticv1.GroupProjectBinding
	for _, binding := range allBindings.Items {
		if groupSet.Has(binding.Spec.Group) {
			bindingsForGivenGroups = append(bindingsForGivenGroups, binding.DeepCopy())
		}
	}

	return bindingsForGivenGroups, nil
}

// MapUserToRoles returns the roles of the user in the project. It searches across the user project bindings and the group
// project bindings for the user and returns the role set.
// This function is unsafe in a sense that it uses privileged account to list all userProjectBindings and groupProjectBindings in the system.
func (p *ProjectMemberProvider) MapUserToRoles(ctx context.Context, user *kubermaticv1.User, projectID string) (sets.String, error) {
	projectReq, err := labels.NewRequirement(kubermaticv1.ProjectIDLabelKey, selection.Equals, []string{projectID})
	if err != nil {
		return sets.String{}, fmt.Errorf("failed to construct project label selector: %w", err)
	}

	// Get group project bindings
	groupProjectBindings := &kubermaticv1.GroupProjectBindingList{}
	if err := p.clientPrivileged.List(ctx, groupProjectBindings, &ctrlruntimeclient.ListOptions{LabelSelector: labels.NewSelector().Add(*projectReq)}); err != nil {
		return sets.String{}, err
	}

	roles := sets.NewString()
	for _, gpb := range groupProjectBindings.Items {
		if slice.ContainsString(user.Spec.Groups, gpb.Spec.Group, nil) && gpb.Spec.ProjectID == projectID {
			roles.Insert(gpb.Spec.Role)
		}
	}

	// get the userprojectBinding group
	userBindingRole, err := getUserBindingRole(ctx, user.Spec.Email, projectID, p.clientPrivileged)
	if err != nil {
		return sets.String{}, err
	}
	if userBindingRole == "" {
		return roles, nil
	}
	// extract just the role
	userBindingRole = apiv1.ExtractGroupPrefix(userBindingRole)
	roles.Insert(userBindingRole)

	return roles, nil
}

// CreateUnsecured creates a binding for the given member and the given project
// This function is unsafe in a sense that it uses privileged account to create the resource.
func (p *ProjectMemberProvider) CreateUnsecured(ctx context.Context, project *kubermaticv1.Project, memberEmail, group string) (*kubermaticv1.UserProjectBinding, error) {
	if kubermaticv1helper.IsProjectServiceAccount(memberEmail) {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("cannot add the given member %s to the project %s because the email indicates a service account", memberEmail, project.Spec.Name))
	}

	binding := genBinding(project, memberEmail, group)

	if err := p.clientPrivileged.Create(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// CreateUnsecuredForServiceAccount creates a binding for the given service account and the given project
// This function is unsafe in a sense that it uses privileged account to create the resource.
func (p *ProjectMemberProvider) CreateUnsecuredForServiceAccount(ctx context.Context, project *kubermaticv1.Project, memberEmail, group string) (*kubermaticv1.UserProjectBinding, error) {
	if kubermaticv1helper.IsProjectServiceAccount(memberEmail) && !strings.HasPrefix(group, rbac.ProjectManagerGroupNamePrefix) {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("cannot add the given member %s to the project %s because the email indicates a service account", memberEmail, project.Spec.Name))
	}

	binding := genBinding(project, memberEmail, group)

	if err := p.clientPrivileged.Create(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// DeleteUnsecured deletes the given binding
// Note:
// Use List to get binding for the specific member of the given project
// This function is unsafe in a sense that it uses privileged account to delete the resource.
func (p *ProjectMemberProvider) DeleteUnsecured(ctx context.Context, bindingName string) error {
	return p.clientPrivileged.Delete(ctx, &kubermaticv1.UserProjectBinding{ObjectMeta: metav1.ObjectMeta{Name: bindingName}})
}

// UpdateUnsecured updates the given binding
// This function is unsafe in a sense that it uses privileged account to update the resource.
func (p *ProjectMemberProvider) UpdateUnsecured(ctx context.Context, binding *kubermaticv1.UserProjectBinding) (*kubermaticv1.UserProjectBinding, error) {
	err := p.clientPrivileged.Update(ctx, binding)

	return binding, err
}

func genBinding(project *kubermaticv1.Project, memberEmail, group string) *kubermaticv1.UserProjectBinding {
	return &kubermaticv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: rand.String(10),
		},
		Spec: kubermaticv1.UserProjectBindingSpec{
			ProjectID: project.Name,
			UserEmail: memberEmail,
			Group:     group,
		},
	}
}
