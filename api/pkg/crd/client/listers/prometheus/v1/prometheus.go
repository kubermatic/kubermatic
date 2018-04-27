package v1

import (
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/prometheus/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// PrometheusLister helps list Prometheuses.
type PrometheusLister interface {
	// List lists all Prometheuses in the indexer.
	List(selector labels.Selector) (ret []*v1.Prometheus, err error)
	// Prometheuses returns an object that can list and get Prometheuses.
	Prometheuses(namespace string) PrometheusNamespaceLister
	PrometheusListerExpansion
}

// prometheusLister implements the PrometheusLister interface.
type prometheusLister struct {
	indexer cache.Indexer
}

// NewPrometheusLister returns a new PrometheusLister.
func NewPrometheusLister(indexer cache.Indexer) PrometheusLister {
	return &prometheusLister{indexer: indexer}
}

// List lists all Prometheuses in the indexer.
func (s *prometheusLister) List(selector labels.Selector) (ret []*v1.Prometheus, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Prometheus))
	})
	return ret, err
}

// Prometheuses returns an object that can list and get Prometheuses.
func (s *prometheusLister) Prometheuses(namespace string) PrometheusNamespaceLister {
	return prometheusNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// PrometheusNamespaceLister helps list and get Prometheuses.
type PrometheusNamespaceLister interface {
	// List lists all Prometheuses in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1.Prometheus, err error)
	// Get retrieves the Prometheus from the indexer for a given namespace and name.
	Get(name string) (*v1.Prometheus, error)
	PrometheusNamespaceListerExpansion
}

// prometheusNamespaceLister implements the PrometheusNamespaceLister
// interface.
type prometheusNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Prometheuses in the indexer for a given namespace.
func (s prometheusNamespaceLister) List(selector labels.Selector) (ret []*v1.Prometheus, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Prometheus))
	})
	return ret, err
}

// Get retrieves the Prometheus from the indexer for a given namespace and name.
func (s prometheusNamespaceLister) Get(name string) (*v1.Prometheus, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("prometheus"), name)
	}
	return obj.(*v1.Prometheus), nil
}
