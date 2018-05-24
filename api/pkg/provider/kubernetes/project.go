package kubernetes

import (
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

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
func NewProjectProvider(createImpersonatedClient kubermaticImpersonationClient, projectLister kubermaticv1lister.ProjectLister) (*ProjectProvider, error) {
	kubermaticClient, err := createImpersonatedClient(restclient.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	return &ProjectProvider{
		createImpersonatedClient: createImpersonatedClient,
		clientPrivileged:         kubermaticClient.Projects(),
		projectLister:            projectLister,
	}, nil
}

// ProjectProvider represents a data structure that knows how to manage projects
type ProjectProvider struct {
	// createImpersonatedClient is used as a ground for impersonation
	createImpersonatedClient kubermaticImpersonationClient

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
	projects, err := p.projectLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, project := range projects {
		owners := project.GetOwnerReferences()
		for _, owner := range owners {
			if owner.UID == user.UID && project.Spec.Name == projectName {
				return nil, kerrors.NewAlreadyExists(schema.GroupResource{Group: project.GroupVersionKind().Group, Resource: kubermaticapiv1.ProjectPlural}, projectName)
			}
		}
	}

	project := &kubermaticapiv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       "User",
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

// Delete deletes the given project as the given user
func (p *ProjectProvider) Delete(user *kubermaticapiv1.User, projectInternalName string) error {
	groupName, err := user.GroupForProject(projectInternalName)
	if err != nil {
		return kerrors.NewForbidden(schema.GroupResource{}, projectInternalName, err)
	}
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: user.Spec.Email,
		Groups:   []string{groupName},
	}

	impersonatedClient, err := p.createImpersonatedClient(impersonationCfg)
	if err != nil {
		return err
	}

	return impersonatedClient.Projects().Delete(projectInternalName, &metav1.DeleteOptions{})
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
