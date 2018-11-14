package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// MachineDeploymentLister helps list MachineDeployments.
type MachineDeploymentLister interface {
	// List lists all MachineDeployments in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.MachineDeployment, err error)
	// MachineDeployments returns an object that can list and get MachineDeployments.
	MachineDeployments(namespace string) MachineDeploymentNamespaceLister
	MachineDeploymentListerExpansion
}

// machineDeploymentLister implements the MachineDeploymentLister interface.
type machineDeploymentLister struct {
	indexer cache.Indexer
}

// NewMachineDeploymentLister returns a new MachineDeploymentLister.
func NewMachineDeploymentLister(indexer cache.Indexer) MachineDeploymentLister {
	return &machineDeploymentLister{indexer: indexer}
}

// List lists all MachineDeployments in the indexer.
func (s *machineDeploymentLister) List(selector labels.Selector) (ret []*v1alpha1.MachineDeployment, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MachineDeployment))
	})
	return ret, err
}

// MachineDeployments returns an object that can list and get MachineDeployments.
func (s *machineDeploymentLister) MachineDeployments(namespace string) MachineDeploymentNamespaceLister {
	return machineDeploymentNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MachineDeploymentNamespaceLister helps list and get MachineDeployments.
type MachineDeploymentNamespaceLister interface {
	// List lists all MachineDeployments in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.MachineDeployment, err error)
	// Get retrieves the MachineDeployment from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.MachineDeployment, error)
	MachineDeploymentNamespaceListerExpansion
}

// machineDeploymentNamespaceLister implements the MachineDeploymentNamespaceLister
// interface.
type machineDeploymentNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all MachineDeployments in the indexer for a given namespace.
func (s machineDeploymentNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.MachineDeployment, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MachineDeployment))
	})
	return ret, err
}

// Get retrieves the MachineDeployment from the indexer for a given namespace and name.
func (s machineDeploymentNamespaceLister) Get(name string) (*v1alpha1.MachineDeployment, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("machinedeployment"), name)
	}
	return obj.(*v1alpha1.MachineDeployment), nil
}
