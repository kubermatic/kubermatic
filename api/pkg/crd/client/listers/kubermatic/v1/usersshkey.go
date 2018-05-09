package v1

import (
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// UserSSHKeyLister helps list UserSSHKeies.
type UserSSHKeyLister interface {
	// List lists all UserSSHKeies in the indexer.
	List(selector labels.Selector) (ret []*v1.UserSSHKey, err error)
	// Get retrieves the UserSSHKey from the index for a given name.
	Get(name string) (*v1.UserSSHKey, error)
	UserSSHKeyListerExpansion
}

// userSSHKeyLister implements the UserSSHKeyLister interface.
type userSSHKeyLister struct {
	indexer cache.Indexer
}

// NewUserSSHKeyLister returns a new UserSSHKeyLister.
func NewUserSSHKeyLister(indexer cache.Indexer) UserSSHKeyLister {
	return &userSSHKeyLister{indexer: indexer}
}

// List lists all UserSSHKeies in the indexer.
func (s *userSSHKeyLister) List(selector labels.Selector) (ret []*v1.UserSSHKey, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.UserSSHKey))
	})
	return ret, err
}

// Get retrieves the UserSSHKey from the index for a given name.
func (s *userSSHKeyLister) Get(name string) (*v1.UserSSHKey, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("usersshkey"), name)
	}
	return obj.(*v1.UserSSHKey), nil
}
