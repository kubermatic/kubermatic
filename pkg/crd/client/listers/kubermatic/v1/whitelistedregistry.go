// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// WhitelistedRegistryLister helps list WhitelistedRegistries.
// All objects returned here must be treated as read-only.
type WhitelistedRegistryLister interface {
	// List lists all WhitelistedRegistries in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.WhitelistedRegistry, err error)
	// Get retrieves the WhitelistedRegistry from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.WhitelistedRegistry, error)
	WhitelistedRegistryListerExpansion
}

// whitelistedRegistryLister implements the WhitelistedRegistryLister interface.
type whitelistedRegistryLister struct {
	indexer cache.Indexer
}

// NewWhitelistedRegistryLister returns a new WhitelistedRegistryLister.
func NewWhitelistedRegistryLister(indexer cache.Indexer) WhitelistedRegistryLister {
	return &whitelistedRegistryLister{indexer: indexer}
}

// List lists all WhitelistedRegistries in the indexer.
func (s *whitelistedRegistryLister) List(selector labels.Selector) (ret []*v1.WhitelistedRegistry, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.WhitelistedRegistry))
	})
	return ret, err
}

// Get retrieves the WhitelistedRegistry from the index for a given name.
func (s *whitelistedRegistryLister) Get(name string) (*v1.WhitelistedRegistry, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("whitelistedregistry"), name)
	}
	return obj.(*v1.WhitelistedRegistry), nil
}
