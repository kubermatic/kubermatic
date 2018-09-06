package kubernetes

import (
	"strings"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	userLabelKey = "user"
)

// NewUserProvider returns a user provider
func NewUserProvider(client kubermaticclientset.Interface, userLister kubermaticv1lister.UserLister) *UserProvider {
	return &UserProvider{
		client:     client,
		userLister: userLister,
	}
}

// UserProvider manages user resources
type UserProvider struct {
	client     kubermaticclientset.Interface
	userLister kubermaticv1lister.UserLister
}

// ListByProject returns a list of users by the given project name
func (p *UserProvider) ListByProject(projectName string) ([]*kubermaticv1.User, error) {
	userList, err := p.userLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	projectUsers := []*kubermaticv1.User{}
	for _, user := range userList {
		for _, project := range user.Spec.Projects {
			if project.Name == projectName {
				projectUsers = append(projectUsers, user.DeepCopy())
				break
			}
		}
	}

	return projectUsers, nil
}

// UserByEmail returns a user by the given email
func (p *UserProvider) UserByEmail(email string) (*kubermaticv1.User, error) {
	users, err := p.userLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	email = normalizeText(email)
	for _, user := range users {
		if user.Spec.Email == email {
			return user.DeepCopy(), nil
		}
	}

	// In case we could not find the user from the lister, we get all users from the API
	// This ensures we don't run into issues with an outdated cache & create the same user twice
	// This part will be called when a new user does the first request & the user does not exist yet as resource.
	userList, err := p.client.KubermaticV1().Users().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, user := range userList.Items {
		if user.Spec.Email == email {
			return user.DeepCopy(), nil
		}
	}

	return nil, provider.ErrNotFound
}

// CreateUser creates a user
func (p *UserProvider) CreateUser(id, name, email string) (*kubermaticv1.User, error) {
	if len(id) == 0 || len(name) == 0 || len(email) == 0 {
		return nil, kerrors.NewBadRequest("Email, ID and Name cannot be empty when creating a new user resource")
	}

	user := kubermaticv1.User{}
	user.GenerateName = "user-"
	user.Spec.Email = email
	user.Spec.Name = name
	user.Spec.ID = id
	user.Spec.Projects = []kubermaticv1.ProjectGroup{}
	normalizedUser := normalizeUser(user)

	return p.client.KubermaticV1().Users().Create(&normalizedUser)
}

// Update updates the given user
func (p *UserProvider) Update(user *kubermaticv1.User) (*kubermaticv1.User, error) {
	normalizedUser := normalizeUser(*user)
	return p.client.KubermaticV1().Users().Update(&normalizedUser)
}

func normalizeText(text string) string {
	return strings.TrimSpace(text)
}

func normalizeUser(user kubermaticv1.User) kubermaticv1.User {
	user.Spec.Email = normalizeText(user.Spec.Email)
	user.Spec.Name = normalizeText(user.Spec.Name)
	user.Spec.ID = normalizeText(user.Spec.ID)
	return user
}
