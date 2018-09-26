package kubernetes

import (
	"strings"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// UserLabelKey defines the label key for the user -> cluster relation
	UserLabelKey = "user"
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

// UserByEmail returns a user by the given email
func (p *UserProvider) UserByEmail(email string) (*kubermaticv1.User, error) {
	users, err := p.userLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if strings.ToLower(user.Spec.Email) == strings.ToLower(email) {
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
		if strings.ToLower(user.Spec.Email) == strings.ToLower(email) {
			return user.DeepCopy(), nil
		}
	}

	return nil, provider.ErrNotFound
}

// CreateUser creates a user
func (p *UserProvider) CreateUser(id, name, email string) (*kubermaticv1.User, error) {
	user := kubermaticv1.User{}
	user.GenerateName = "user-"
	user.Spec.Email = email
	user.Spec.Name = name
	user.Spec.ID = id
	user.Spec.Projects = []kubermaticv1.ProjectGroup{}

	return p.client.KubermaticV1().Users().Create(&user)
}

// Update updates the given user
func (p *UserProvider) Update(user *kubermaticv1.User) (*kubermaticv1.User, error) {
	return p.client.KubermaticV1().Users().Update(user)
}
