// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// MLAAdminSettingLister helps list MLAAdminSettings.
// All objects returned here must be treated as read-only.
type MLAAdminSettingLister interface {
	// List lists all MLAAdminSettings in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.MLAAdminSetting, err error)
	// MLAAdminSettings returns an object that can list and get MLAAdminSettings.
	MLAAdminSettings(namespace string) MLAAdminSettingNamespaceLister
	MLAAdminSettingListerExpansion
}

// mLAAdminSettingLister implements the MLAAdminSettingLister interface.
type mLAAdminSettingLister struct {
	indexer cache.Indexer
}

// NewMLAAdminSettingLister returns a new MLAAdminSettingLister.
func NewMLAAdminSettingLister(indexer cache.Indexer) MLAAdminSettingLister {
	return &mLAAdminSettingLister{indexer: indexer}
}

// List lists all MLAAdminSettings in the indexer.
func (s *mLAAdminSettingLister) List(selector labels.Selector) (ret []*v1.MLAAdminSetting, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.MLAAdminSetting))
	})
	return ret, err
}

// MLAAdminSettings returns an object that can list and get MLAAdminSettings.
func (s *mLAAdminSettingLister) MLAAdminSettings(namespace string) MLAAdminSettingNamespaceLister {
	return mLAAdminSettingNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MLAAdminSettingNamespaceLister helps list and get MLAAdminSettings.
// All objects returned here must be treated as read-only.
type MLAAdminSettingNamespaceLister interface {
	// List lists all MLAAdminSettings in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.MLAAdminSetting, err error)
	// Get retrieves the MLAAdminSetting from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.MLAAdminSetting, error)
	MLAAdminSettingNamespaceListerExpansion
}

// mLAAdminSettingNamespaceLister implements the MLAAdminSettingNamespaceLister
// interface.
type mLAAdminSettingNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all MLAAdminSettings in the indexer for a given namespace.
func (s mLAAdminSettingNamespaceLister) List(selector labels.Selector) (ret []*v1.MLAAdminSetting, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.MLAAdminSetting))
	})
	return ret, err
}

// Get retrieves the MLAAdminSetting from the indexer for a given namespace and name.
func (s mLAAdminSettingNamespaceLister) Get(name string) (*v1.MLAAdminSetting, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("mlaadminsetting"), name)
	}
	return obj.(*v1.MLAAdminSetting), nil
}
