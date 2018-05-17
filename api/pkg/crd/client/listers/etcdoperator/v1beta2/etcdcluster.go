package v1beta2

import (
	v1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// EtcdClusterLister helps list EtcdClusters.
type EtcdClusterLister interface {
	// List lists all EtcdClusters in the indexer.
	List(selector labels.Selector) (ret []*v1beta2.EtcdCluster, err error)
	// EtcdClusters returns an object that can list and get EtcdClusters.
	EtcdClusters(namespace string) EtcdClusterNamespaceLister
	EtcdClusterListerExpansion
}

// etcdClusterLister implements the EtcdClusterLister interface.
type etcdClusterLister struct {
	indexer cache.Indexer
}

// NewEtcdClusterLister returns a new EtcdClusterLister.
func NewEtcdClusterLister(indexer cache.Indexer) EtcdClusterLister {
	return &etcdClusterLister{indexer: indexer}
}

// List lists all EtcdClusters in the indexer.
func (s *etcdClusterLister) List(selector labels.Selector) (ret []*v1beta2.EtcdCluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta2.EtcdCluster))
	})
	return ret, err
}

// EtcdClusters returns an object that can list and get EtcdClusters.
func (s *etcdClusterLister) EtcdClusters(namespace string) EtcdClusterNamespaceLister {
	return etcdClusterNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// EtcdClusterNamespaceLister helps list and get EtcdClusters.
type EtcdClusterNamespaceLister interface {
	// List lists all EtcdClusters in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1beta2.EtcdCluster, err error)
	// Get retrieves the EtcdCluster from the indexer for a given namespace and name.
	Get(name string) (*v1beta2.EtcdCluster, error)
	EtcdClusterNamespaceListerExpansion
}

// etcdClusterNamespaceLister implements the EtcdClusterNamespaceLister
// interface.
type etcdClusterNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all EtcdClusters in the indexer for a given namespace.
func (s etcdClusterNamespaceLister) List(selector labels.Selector) (ret []*v1beta2.EtcdCluster, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1beta2.EtcdCluster))
	})
	return ret, err
}

// Get retrieves the EtcdCluster from the indexer for a given namespace and name.
func (s etcdClusterNamespaceLister) Get(name string) (*v1beta2.EtcdCluster, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1beta2.Resource("etcdcluster"), name)
	}
	return obj.(*v1beta2.EtcdCluster), nil
}
