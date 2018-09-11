package kubernetes

import (
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"github.com/kubermatic/kubermatic/api/pkg/provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// NewProjectMemberProvider returns a project members provider
func NewProjectMemberProvider(createMasterImpersonatedClient kubermaticImpersonationClient, membersLister kubermaticv1lister.UserProjectBindingLister) *ProjectMemberProvider {
	return &ProjectMemberProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		membersLister:                  membersLister,
	}
}

var _ provider.ProjectMemberProvider = &ProjectMemberProvider{}

// ProjectMembersProvider binds users with projects
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

	// Note:
	// After we get the list of members we try to get at least one item using unprivileged account to see if the user have read access
	if len(projectMembers) > 0 {
		masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
		if err != nil {
			return nil, err
		}

		memberToGet := projectMembers[0]
		_, err = masterImpersonatedClient.UserProjectBindings().Get(memberToGet.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	if options == nil {
		return projectMembers, nil
	}

	filteredMembers := []*kubermaticapiv1.UserProjectBinding{}
	if options != nil {
		for _, member := range projectMembers {
			if member.Spec.UserEmail == options.MemberEmail {
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
func (p *ProjectMemberProvider) Delete(userInfo *provider.UserInfo, bindinName string) error {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}
	return masterImpersonatedClient.UserProjectBindings().Delete(bindinName, &metav1.DeleteOptions{})
}

// Update simply updates the given binding
func (p *ProjectMemberProvider) Update(userInfo *provider.UserInfo, binding *kubermaticapiv1.UserProjectBinding) (*kubermaticapiv1.UserProjectBinding, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserProjectBindings().Update(binding)
}
