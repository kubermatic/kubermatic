package v1alpha1

import (
	v1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/addons/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// AddonLister helps list Addons.
type AddonLister interface {
	// List lists all Addons in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.Addon, err error)
	// Get retrieves the Addon from the index for a given name.
	Get(name string) (*v1alpha1.Addon, error)
	AddonListerExpansion
}

// addonLister implements the AddonLister interface.
type addonLister struct {
	indexer cache.Indexer
}

// NewAddonLister returns a new AddonLister.
func NewAddonLister(indexer cache.Indexer) AddonLister {
	return &addonLister{indexer: indexer}
}

// List lists all Addons in the indexer.
func (s *addonLister) List(selector labels.Selector) (ret []*v1alpha1.Addon, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Addon))
	})
	return ret, err
}

// Get retrieves the Addon from the index for a given name.
func (s *addonLister) Get(name string) (*v1alpha1.Addon, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("addon"), name)
	}
	return obj.(*v1alpha1.Addon), nil
}
