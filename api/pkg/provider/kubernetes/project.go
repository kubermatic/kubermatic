package kubernetes

import (
	"errors"

	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/label"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	restclient "k8s.io/client-go/rest"
)

// NewProjectProvider returns a project provider
func NewProjectProvider(createMasterImpersonatedClient kubermaticImpersonationClient, projectLister kubermaticv1lister.ProjectLister) (*ProjectProvider, error) {
	kubermaticClient, err := createMasterImpersonatedClient(restclient.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	return &ProjectProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               kubermaticClient.Projects(),
		projectLister:                  projectLister,
	}, nil
}

// NewPrivilegedProjectProvider returns a privileged project provider
func NewPrivilegedProjectProvider(createMasterImpersonatedClient kubermaticImpersonationClient) (*PrivilegedProjectProvider, error) {
	kubermaticClient, err := createMasterImpersonatedClient(restclient.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	return &PrivilegedProjectProvider{
		clientPrivileged: kubermaticClient.Projects(),
	}, nil
}

// ProjectProvider represents a data structure that knows how to manage projects
type ProjectProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient kubermaticImpersonationClient

	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged kubermaticclientv1.ProjectInterface

	// projectLister local cache that stores projects objects
	projectLister kubermaticv1lister.ProjectLister
}

// PrivilegedProjectProvider represents a data structure that knows how to manage projects in a privileged way
type PrivilegedProjectProvider struct {
	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged kubermaticclientv1.ProjectInterface
}

// New creates a brand new project in the system with the given name
//
// Note:
// a user cannot own more than one project with the given name
// since we get the list of the current projects from a cache (lister) there is a small time window
// during which a user can create more that one project with the given name.
func (p *ProjectProvider) New(user *kubermaticapiv1.User, projectName string, labels map[string]string) (*kubermaticapiv1.Project, error) {
	if user == nil {
		return nil, errors.New("a user is missing but required")
	}

	project := &kubermaticapiv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
					Kind:       kubermaticapiv1.UserKindName,
					UID:        user.GetUID(),
					Name:       user.Name,
				},
			},
			Name:   rand.String(10),
			Labels: labels,
		},
		Spec: kubermaticapiv1.ProjectSpec{
			Name: projectName,
		},
		Status: kubermaticapiv1.ProjectStatus{
			Phase: kubermaticapiv1.ProjectInactive,
		},
	}

	return p.clientPrivileged.Create(project)
}

// Update update a specific project for a specific user and returns the updated project
func (p *ProjectProvider) Update(userInfo *provider.UserInfo, newProject *kubermaticapiv1.Project) (*kubermaticapiv1.Project, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}
	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	return masterImpersonatedClient.Projects().Update(newProject)
}

// Delete deletes the given project as the given user
//
// Note:
// Before deletion project's status.phase is set to ProjectTerminating
func (p *ProjectProvider) Delete(userInfo *provider.UserInfo, projectInternalName string) error {
	if userInfo == nil {
		return errors.New("a user is missing but required")
	}
	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}

	existingProject, err := masterImpersonatedClient.Projects().Get(projectInternalName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	existingProject.Status.Phase = kubermaticapiv1.ProjectTerminating
	if _, err := masterImpersonatedClient.Projects().Update(existingProject); err != nil {
		return err
	}

	return masterImpersonatedClient.Projects().Delete(projectInternalName, &metav1.DeleteOptions{})
}

// Get returns the project with the given name
func (p *ProjectProvider) Get(userInfo *provider.UserInfo, projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticapiv1.Project, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}
	if options == nil {
		options = &provider.ProjectGetOptions{IncludeUninitialized: true}
	}
	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	project, err := masterImpersonatedClient.Projects().Get(projectInternalName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if !options.IncludeUninitialized && project.Status.Phase != kubermaticapiv1.ProjectActive {
		return nil, kerrors.NewServiceUnavailable("Project is not initialized yet")
	}

	return project, nil
}

// GetUnsecured returns the project with the given name
// This function is unsafe in a sense that it uses privileged account to get project with the given name
func (p *PrivilegedProjectProvider) GetUnsecured(projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticapiv1.Project, error) {
	if options == nil {
		options = &provider.ProjectGetOptions{IncludeUninitialized: true}
	}
	project, err := p.clientPrivileged.Get(projectInternalName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if !options.IncludeUninitialized && project.Status.Phase != kubermaticapiv1.ProjectActive {
		return nil, kerrors.NewServiceUnavailable("Project is not initialized yet")
	}
	return project, nil
}

// DeleteUnsecured deletes any given project
// This function is unsafe in a sense that it uses privileged account to delete project with the given name
//
// Note:
// Before deletion project's status.phase is set to ProjectTerminating
func (p *PrivilegedProjectProvider) DeleteUnsecured(projectInternalName string) error {
	existingProject, err := p.clientPrivileged.Get(projectInternalName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	existingProject.Status.Phase = kubermaticapiv1.ProjectTerminating
	if _, err := p.clientPrivileged.Update(existingProject); err != nil {
		return err
	}

	return p.clientPrivileged.Delete(projectInternalName, &metav1.DeleteOptions{})
}

// UpdateUnsecured update a specific project and returns the updated project
// This function is unsafe in a sense that it uses privileged account to update the project
func (p *PrivilegedProjectProvider) UpdateUnsecured(project *kubermaticapiv1.Project) (*kubermaticapiv1.Project, error) {
	return p.clientPrivileged.Update(project)
}

// List gets a list of projects, by default it returns all resources.
// If you want to filter the result please set ProjectListOptions
//
// Note that the list is taken from the cache
func (p *ProjectProvider) List(options *provider.ProjectListOptions) ([]*kubermaticapiv1.Project, error) {
	if options == nil {
		options = &provider.ProjectListOptions{}
	}
	projects, err := p.projectLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	ret := []*kubermaticapiv1.Project{}
	for _, project := range projects {
		if len(options.ProjectName) > 0 && project.Spec.Name != options.ProjectName {
			continue
		}
		if len(options.OwnerUID) > 0 {
			owners := project.GetOwnerReferences()
			for _, owner := range owners {
				if owner.UID == options.OwnerUID {
					ret = append(ret, project.DeepCopy())
					continue
				}
			}
			continue
		}

		ret = append(ret, project.DeepCopy())
	}

	// Filter out restricted labels
	for i, project := range ret {
		project.Labels = label.FilterLabels(label.ClusterResourceType, project.Labels)
		ret[i] = project
	}

	return ret, nil
}
