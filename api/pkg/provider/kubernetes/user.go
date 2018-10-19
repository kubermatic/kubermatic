package kubernetes

import (
	"crypto/sha256"
	"fmt"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

// UserByID returns a user by the given ID
func (p *UserProvider) UserByID(id string) (*kubermaticv1.User, error) {
	return p.client.KubermaticV1().Users().Get(id, v1.GetOptions{})
}

// UserByEmail returns a user by the given email
func (p *UserProvider) UserByEmail(email string) (*kubermaticv1.User, error) {
	users, err := p.userLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

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

// CreateUser creates a new user.
//
// Note that:
// The name of the newly created resource will be unique and it is derived from the user's email address (sha256(email)
// This prevents creating multiple resources for the same user with the same email address.
//
// In the beginning I was considering to hex-encode the email address as it will produce a unique output because the email address in unique.
// The only issue I have found with this approach is that the length can get quite long quite fast.
// Thus decided to use sha256 as it produces fixed output and the hash collisions are very, very, very, very rare.
func (p *UserProvider) CreateUser(id, name, email string) (*kubermaticv1.User, error) {
	if len(id) == 0 || len(name) == 0 || len(email) == 0 {
		return nil, kerrors.NewBadRequest("Email, ID and Name cannot be empty when creating a new user resource")
	}

	uniqueObjectName := fmt.Sprintf("%x", sha256.Sum256([]byte(email)))

	user := kubermaticv1.User{}
	user.Name = uniqueObjectName
	user.Spec.Email = email
	user.Spec.Name = name
	user.Spec.ID = id
	user.Spec.Projects = []kubermaticv1.ProjectGroup{}

	return p.client.KubermaticV1().Users().Create(&user)
}
