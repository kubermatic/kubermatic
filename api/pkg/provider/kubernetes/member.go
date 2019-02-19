package kubernetes

import (
	"fmt"
	"strings"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
)

// NewProjectMemberProvider returns a project members provider
func NewProjectMemberProvider(createMasterImpersonatedClient kubermaticImpersonationClient, membersLister kubermaticv1lister.UserProjectBindingLister) *ProjectMemberProvider {
	return &ProjectMemberProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		membersLister:                  membersLister,
	}
}

var _ provider.ProjectMemberProvider = &ProjectMemberProvider{}

// ProjectMemberProvider binds users with projects
type ProjectMemberProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient kubermaticImpersonationClient

	// membersLister local cache that stores bindings for members and projects
	membersLister kubermaticv1lister.UserProjectBindingLister
}

// Create creates a binding for the given member and the given project
func (p *ProjectMemberProvider) Create(userInfo *provider.UserInfo, project *kubermaticapiv1.Project, memberEmail, group string) (*kubermaticapiv1.UserProjectBinding, error) {
	binding := &kubermaticapiv1.UserProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
					Kind:       kubermaticapiv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				},
			},
			Name: rand.String(10),
		},
		Spec: kubermaticapiv1.UserProjectBindingSpec{
			ProjectID: project.Name,
			UserEmail: memberEmail,
			Group:     group,
		},
	}

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserProjectBindings().Create(binding)
}

// List gets all members of the given project
func (p *ProjectMemberProvider) List(userInfo *provider.UserInfo, project *kubermaticapiv1.Project, options *provider.ProjectMemberListOptions) ([]*kubermaticapiv1.UserProjectBinding, error) {
	allMembers, err := p.membersLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	projectMembers := []*kubermaticapiv1.UserProjectBinding{}
	for _, member := range allMembers {
		if member.Spec.ProjectID == project.Name {
			projectMembers = append(projectMembers, member)
		}
	}

	if options == nil {
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

// Delete simply deletes the given binding
// Note:
// Use List to get binding for the specific member of the given project
func (p *ProjectMemberProvider) Delete(userInfo *provider.UserInfo, bindingName string) error {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}
	return masterImpersonatedClient.UserProjectBindings().Delete(bindingName, &metav1.DeleteOptions{})
}

// Update simply updates the given binding
func (p *ProjectMemberProvider) Update(userInfo *provider.UserInfo, binding *kubermaticapiv1.UserProjectBinding) (*kubermaticapiv1.UserProjectBinding, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserProjectBindings().Update(binding)
}

// MapUserToGroup maps the given user to a specific group of the given project
// This function is unsafe in a sense that it uses privileged account to list all members in the system
func (p *ProjectMemberProvider) MapUserToGroup(userEmail string, projectID string) (string, error) {
	allMembers, err := p.membersLister.List(labels.Everything())
	if err != nil {
		return "", err
	}

	for _, member := range allMembers {
		if strings.EqualFold(member.Spec.UserEmail, userEmail) && member.Spec.ProjectID == projectID {
			return member.Spec.Group, nil
		}
	}

	return "", kerrors.NewForbidden(schema.GroupResource{}, projectID, fmt.Errorf("The user %q doesn't belong to the given project = %s", userEmail, projectID))
}

// MappingsFor returns the list of projects (bindings) for the given user
// This function is unsafe in a sense that it uses privileged account to list all members in the system
func (p *ProjectMemberProvider) MappingsFor(userEmail string) ([]*kubermaticapiv1.UserProjectBinding, error) {
	allMemberMappings, err := p.membersLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	memberMappings := []*kubermaticapiv1.UserProjectBinding{}
	for _, memberMapping := range allMemberMappings {
		if strings.EqualFold(memberMapping.Spec.UserEmail, userEmail) {
			memberMappings = append(memberMappings, memberMapping)
		}
	}

	return memberMappings, nil
}
