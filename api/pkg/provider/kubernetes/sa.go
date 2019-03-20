package kubernetes

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

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

	domain string
}

// CreateServiceAccount creates a new service account
func (p *ServiceAccountProvider) CreateServiceAccount(userInfo *provider.UserInfo, project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error) {
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

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	return masterImpersonatedClient.Users().Create(&user)
}

func (p *ServiceAccountProvider) GetServiceAccountByNameForProject(userInfo *provider.UserInfo, serviceAccountName, projectName string) (*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if len(serviceAccountName) == 0 || len(projectName) == 0 {
		return nil, kerrors.NewBadRequest("service account name and project name cannot be empty")
	}

	serviceAccounts, err := p.serviceAccountLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, sa := range serviceAccounts {
		if strings.HasPrefix(sa.Name, "serviceaccount") && sa.Spec.Name == serviceAccountName {
			for _, owner := range sa.GetOwnerReferences() {
				if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName &&
					owner.Name == projectName {
					return sa, nil
				}
			}
		}
	}

	return nil, kerrors.NewNotFound(schema.GroupResource{Resource: "ServiceAccount"}, serviceAccountName)
}
