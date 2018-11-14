package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// MachineClassLister helps list MachineClasses.
type MachineClassLister interface {
	// List lists all MachineClasses in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.MachineClass, err error)
	// MachineClasses returns an object that can list and get MachineClasses.
	MachineClasses(namespace string) MachineClassNamespaceLister
	MachineClassListerExpansion
}

// machineClassLister implements the MachineClassLister interface.
type machineClassLister struct {
	indexer cache.Indexer
}

// NewMachineClassLister returns a new MachineClassLister.
func NewMachineClassLister(indexer cache.Indexer) MachineClassLister {
	return &machineClassLister{indexer: indexer}
}

// List lists all MachineClasses in the indexer.
func (s *machineClassLister) List(selector labels.Selector) (ret []*v1alpha1.MachineClass, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MachineClass))
	})
	return ret, err
}

// MachineClasses returns an object that can list and get MachineClasses.
func (s *machineClassLister) MachineClasses(namespace string) MachineClassNamespaceLister {
	return machineClassNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MachineClassNamespaceLister helps list and get MachineClasses.
type MachineClassNamespaceLister interface {
	// List lists all MachineClasses in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.MachineClass, err error)
	// Get retrieves the MachineClass from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.MachineClass, error)
	MachineClassNamespaceListerExpansion
}

// machineClassNamespaceLister implements the MachineClassNamespaceLister
// interface.
type machineClassNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all MachineClasses in the indexer for a given namespace.
func (s machineClassNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.MachineClass, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MachineClass))
	})
	return ret, err
}

// Get retrieves the MachineClass from the indexer for a given namespace and name.
func (s machineClassNamespaceLister) Get(name string) (*v1alpha1.MachineClass, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("machineclass"), name)
	}
	return obj.(*v1alpha1.MachineClass), nil
}
