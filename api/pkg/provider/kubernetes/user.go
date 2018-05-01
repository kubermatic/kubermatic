package kubernetes

import (
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/labels"
)

const (
	userLabelKey      = "user"
	userEmailLabelKey = "email"
	userIDLabelKey    = "id"
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
	selector := labels.SelectorFromSet(map[string]string{userEmailLabelKey: kubernetes.ToLabelValue(email)})
	users, err := p.userLister.List(selector)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, provider.ErrNotFound
	}
	return users[0], err
}

// CreateUser creates a user
func (p *UserProvider) CreateUser(id, name, email string) (*kubermaticv1.User, error) {
	user := kubermaticv1.User{}
	user.Labels = map[string]string{
		userEmailLabelKey: kubernetes.ToLabelValue(email),
		userIDLabelKey:    kubernetes.ToLabelValue(id),
	}
	user.GenerateName = "user-"
	user.Spec.Email = email
	user.Spec.Name = name
	user.Spec.ID = id
	user.Spec.Groups = []string{}

	return p.client.KubermaticV1().Users().Create(&user)
}
