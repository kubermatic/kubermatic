package kubernetes

import (
	"errors"

	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	restclient "k8s.io/client-go/rest"
)

// kubermaticImpersonationClient gives kubermatic client set that uses user impersonation
type kubermaticImpersonationClient func(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error)

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

// ProjectProvider represents a data structure that knows how to manage projects
type ProjectProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient kubermaticImpersonationClient

	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged kubermaticclientv1.ProjectInterface

	// projectLister local cache that stores projects objects
	projectLister kubermaticv1lister.ProjectLister
}

// New creates a brand new project in the system with the given name
//
// Note:
// a user cannot own more than one project with the given name
// since we get the list of the current projects from a cache (lister) there is a small time window
// during which a user can create more that one project with the given name.
func (p *ProjectProvider) New(user *kubermaticapiv1.User, projectName string) (*kubermaticapiv1.Project, error) {
	if user == nil {
		return nil, errors.New("a user is missing but required")
	}

	alreadyExistingProjects, err := p.List(&provider.ProjectListOptions{ProjectName: projectName, OwnerUID: user.UID})
	if err != nil {
		return nil, err
	}
	if len(alreadyExistingProjects) > 0 {
		return nil, kerrors.NewAlreadyExists(schema.GroupResource{Group: kubermaticapiv1.SchemeGroupVersion.Group, Resource: kubermaticapiv1.ProjectResourceName}, projectName)
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
			Name: rand.String(10),
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
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
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
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
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
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
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
					ret = append(ret, project)
					continue
				}
			}
			continue
		}

		ret = append(ret, project)
	}
	return ret, nil
}

// NewKubermaticImpersonationClient creates a new default impersonation client
// that knows how to create KubermaticV1Interface client for a impersonated user
//
// Note:
// It is usually not desirable to create many RESTClient thus in the future we might
// consider storing RESTClients in a pool for the given group name
func NewKubermaticImpersonationClient(cfg *restclient.Config) *DefaultKubermaticImpersonationClient {
	return &DefaultKubermaticImpersonationClient{cfg}
}

// DefaultKubermaticImpersonationClient knows how to create impersonated client set
type DefaultKubermaticImpersonationClient struct {
	cfg *restclient.Config
}

// CreateImpersonatedClientSet actually creates impersonated kubermatic client set for the given user.
func (d *DefaultKubermaticImpersonationClient) CreateImpersonatedClientSet(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error) {
	config := *d.cfg
	config.Impersonate = impCfg
	return kubermaticclientv1.NewForConfig(&config)
}
