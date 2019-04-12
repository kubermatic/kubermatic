package kubernetes

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

const ServiceAccountLabelGroup = "initialGroup"

// NewServiceAccountProvider returns a service account provider
func NewServiceAccountProvider(createMasterImpersonatedClient kubermaticImpersonationClient, serviceAccountLister kubermaticv1lister.UserLister, domain string) *ServiceAccountProvider {
	return &ServiceAccountProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		serviceAccountLister:           serviceAccountLister,
		domain:                         domain,
	}
}

// ServiceAccountProvider manages service account resources
type ServiceAccountProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient kubermaticImpersonationClient

	serviceAccountLister kubermaticv1lister.UserLister

	// domain name on which the server is deployed
	domain string
}

// Create creates a new service account
func (p *ServiceAccountProvider) Create(userInfo *provider.UserInfo, project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error) {
	if project == nil {
		return nil, kerrors.NewBadRequest("Project cannot be nil")
	}
	if len(name) == 0 || len(group) == 0 {
		return nil, kerrors.NewBadRequest("Service account name and group cannot be empty when creating a new SA resource")
	}

	uniqueID := rand.String(10)
	uniqueName := fmt.Sprintf("serviceaccount-%s", uniqueID)

	user := kubermaticv1.User{}
	user.Name = uniqueName
	user.Spec.Email = fmt.Sprintf("%s@%s", uniqueName, p.domain)
	user.Spec.Name = name
	user.Spec.ID = uniqueID
	user.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ProjectKindName,
			UID:        project.GetUID(),
			Name:       project.Name,
		},
	}
	user.Labels = map[string]string{ServiceAccountLabelGroup: group}
	user.Spec.Projects = []kubermaticv1.ProjectGroup{}

	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	return masterImpersonatedClient.Users().Create(&user)
}

// List gets service accounts for the project
func (p *ServiceAccountProvider) List(userInfo *provider.UserInfo, project *kubermaticv1.Project, options *provider.ServiceAccountListOptions) ([]*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if project == nil {
		return nil, kerrors.NewBadRequest("project cannot be nil")
	}
	if options == nil {
		options = &provider.ServiceAccountListOptions{}
	}

	serviceAccounts, err := p.serviceAccountLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	resultList := make([]*kubermaticv1.User, 0)
	for _, sa := range serviceAccounts {
		if strings.HasPrefix(sa.Name, "serviceaccount") {
			for _, owner := range sa.GetOwnerReferences() {
				if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName && owner.Name == project.Name {
					resultList = append(resultList, sa)
				}
			}
		}
	}

	// Note:
	// After we get the list of SA we try to get at least one item using unprivileged account to see if the user have read access
	if len(resultList) > 0 {

		masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
		if err != nil {
			return nil, err
		}

		saToGet := resultList[0]
		_, err = masterImpersonatedClient.Users().Get(saToGet.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}

	}
	if len(options.ServiceAccountName) == 0 {
		return resultList, nil
	}

	filteredList := make([]*kubermaticv1.User, 0)
	for _, sa := range resultList {
		if sa.Spec.Name == options.ServiceAccountName {
			filteredList = append(filteredList, sa)
			break
		}
	}

	return filteredList, nil
}

// Get method returns service account with given name
func (p *ServiceAccountProvider) Get(userInfo *provider.UserInfo, name string) (*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if len(name) == 0 {
		return nil, kerrors.NewBadRequest("service account name cannot be empty")
	}

	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	serviceAccount, err := masterImpersonatedClient.Users().Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return serviceAccount, nil
}

// Update simply updates the given service account
func (p *ServiceAccountProvider) Update(userInfo *provider.UserInfo, serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if serviceAccount == nil {
		return nil, kerrors.NewBadRequest("service account name cannot be nil")
	}

	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	return masterImpersonatedClient.Users().Update(serviceAccount)
}

// Delete simply deletes the given service account
func (p *ServiceAccountProvider) Delete(userInfo *provider.UserInfo, name string) error {
	if userInfo == nil {
		return kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if len(name) == 0 {
		return kerrors.NewBadRequest("service account name cannot be empty")
	}

	masterImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}

	return masterImpersonatedClient.Users().Delete(name, &metav1.DeleteOptions{})
}
