package v1

import (
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/prometheus/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ServiceMonitorLister helps list ServiceMonitors.
type ServiceMonitorLister interface {
	// List lists all ServiceMonitors in the indexer.
	List(selector labels.Selector) (ret []*v1.ServiceMonitor, err error)
	// ServiceMonitors returns an object that can list and get ServiceMonitors.
	ServiceMonitors(namespace string) ServiceMonitorNamespaceLister
	ServiceMonitorListerExpansion
}

// serviceMonitorLister implements the ServiceMonitorLister interface.
type serviceMonitorLister struct {
	indexer cache.Indexer
}

// NewServiceMonitorLister returns a new ServiceMonitorLister.
func NewServiceMonitorLister(indexer cache.Indexer) ServiceMonitorLister {
	return &serviceMonitorLister{indexer: indexer}
}

// List lists all ServiceMonitors in the indexer.
func (s *serviceMonitorLister) List(selector labels.Selector) (ret []*v1.ServiceMonitor, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ServiceMonitor))
	})
	return ret, err
}

// ServiceMonitors returns an object that can list and get ServiceMonitors.
func (s *serviceMonitorLister) ServiceMonitors(namespace string) ServiceMonitorNamespaceLister {
	return serviceMonitorNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ServiceMonitorNamespaceLister helps list and get ServiceMonitors.
type ServiceMonitorNamespaceLister interface {
	// List lists all ServiceMonitors in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.ServiceMonitor, err error)
	// Get retrieves the ServiceMonitor from the indexer for a given namespace and name.
	Get(name string) (*v1.ServiceMonitor, error)
	ServiceMonitorNamespaceListerExpansion
}

// serviceMonitorNamespaceLister implements the ServiceMonitorNamespaceLister
// interface.
type serviceMonitorNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ServiceMonitors in the indexer for a given namespace.
func (s serviceMonitorNamespaceLister) List(selector labels.Selector) (ret []*v1.ServiceMonitor, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.ServiceMonitor))
	})
	return ret, err
}

// Get retrieves the ServiceMonitor from the indexer for a given namespace and name.
func (s serviceMonitorNamespaceLister) Get(name string) (*v1.ServiceMonitor, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("servicemonitor"), name)
	}
	return obj.(*v1.ServiceMonitor), nil
}
