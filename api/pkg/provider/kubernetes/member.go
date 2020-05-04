package kubernetes

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/controller/master-controller-manager/rbac"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/rest"
)

// NewProjectMemberProvider returns a project members provider
func NewProjectMemberProvider(createMasterImpersonatedClient kubermaticImpersonationClient, membersLister kubermaticv1lister.UserProjectBindingLister, userLister kubermaticv1lister.UserLister, isServiceAccountFunc func(string) bool) *ProjectMemberProvider {
	return &ProjectMemberProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		membersLister:                  membersLister,
		userLister:                     userLister,
		isServiceAccountFunc:           isServiceAccountFunc,
	}
}

var _ provider.ProjectMemberProvider = &ProjectMemberProvider{}

// ProjectMemberProvider binds users with projects
type ProjectMemberProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient kubermaticImpersonationClient

	// membersLister local cache that stores bindings for members and projects
	membersLister kubermaticv1lister.UserProjectBindingLister

	// userLister local cache that stores users
	userLister kubermaticv1lister.UserLister

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

	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
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
			masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
			if err != nil {
				return nil, err
			}

			memberToGet := projectMembers[0]
			_, err = masterImpersonatedClient.UserProjectBindings().Get(memberToGet.Name, metav1.GetOptions{})
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
	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}
	return masterImpersonatedClient.UserProjectBindings().Delete(bindingName, &metav1.DeleteOptions{})
}

// Update updates the given binding
func (p *ProjectMemberProvider) Update(userInfo *provider.UserInfo, binding *kubermaticapiv1.UserProjectBinding) (*kubermaticapiv1.UserProjectBinding, error) {
	if rbac.ExtractGroupPrefix(binding.Spec.Group) == rbac.OwnerGroupNamePrefix && !kuberneteshelper.HasFinalizer(binding, rbac.CleanupFinalizerName) {
		kuberneteshelper.AddFinalizer(binding, rbac.CleanupFinalizerName)
	}
	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
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

	return "", kerrors.NewForbidden(schema.GroupResource{}, projectID, fmt.Errorf("%q doesn't belong to the given project = %s", userEmail, projectID))
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

// CreateUnsecured creates a binding for the given member and the given project
// This function is unsafe in a sense that it uses privileged account to create the resource
func (p *ProjectMemberProvider) CreateUnsecured(project *kubermaticapiv1.Project, memberEmail, group string) (*kubermaticapiv1.UserProjectBinding, error) {
	if p.isServiceAccountFunc(memberEmail) {
		return nil, kerrors.NewBadRequest(fmt.Sprintf("cannot add the given member %s to the project %s because the email indicates a service account", memberEmail, project.Spec.Name))
	}

	binding := genBinding(project, memberEmail, group)

	masterImpersonatedClient, err := p.createMasterImpersonatedClient(rest.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserProjectBindings().Create(binding)
}

// DeleteUnsecured deletes the given binding
// Note:
// Use List to get binding for the specific member of the given project
// This function is unsafe in a sense that it uses privileged account to delete the resource
func (p *ProjectMemberProvider) DeleteUnsecured(bindingName string) error {
	masterImpersonatedClient, err := p.createMasterImpersonatedClient(rest.ImpersonationConfig{})
	if err != nil {
		return err
	}
	return masterImpersonatedClient.UserProjectBindings().Delete(bindingName, &metav1.DeleteOptions{})
}

// UpdateUnsecured updates the given binding
// This function is unsafe in a sense that it uses privileged account to update the resource
func (p *ProjectMemberProvider) UpdateUnsecured(binding *kubermaticapiv1.UserProjectBinding) (*kubermaticapiv1.UserProjectBinding, error) {
	if rbac.ExtractGroupPrefix(binding.Spec.Group) == rbac.OwnerGroupNamePrefix && !kuberneteshelper.HasFinalizer(binding, rbac.CleanupFinalizerName) {
		kuberneteshelper.AddFinalizer(binding, rbac.CleanupFinalizerName)
	}
	masterImpersonatedClient, err := p.createMasterImpersonatedClient(rest.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserProjectBindings().Update(binding)
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
