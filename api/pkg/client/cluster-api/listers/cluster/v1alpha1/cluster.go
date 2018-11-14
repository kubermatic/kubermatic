package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// ClusterLister helps list Clusters.
type ClusterLister interface {
	// List lists all Clusters in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.Cluster, err error)
	// Clusters returns an object that can list and get Clusters.
	Clusters(namespace string) ClusterNamespaceLister
	ClusterListerExpansion
}

// clusterLister implements the ClusterLister interface.
type clusterLister struct {
	indexer cache.Indexer
}

// NewClusterLister returns a new ClusterLister.
func NewClusterLister(indexer cache.Indexer) ClusterLister {
	return &clusterLister{indexer: indexer}
}

// List lists all Clusters in the indexer.
func (s *clusterLister) List(selector labels.Selector) (ret []*v1alpha1.Cluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Cluster))
	})
	return ret, err
}

// Clusters returns an object that can list and get Clusters.
func (s *clusterLister) Clusters(namespace string) ClusterNamespaceLister {
	return clusterNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ClusterNamespaceLister helps list and get Clusters.
type ClusterNamespaceLister interface {
	// List lists all Clusters in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.Cluster, err error)
	// Get retrieves the Cluster from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.Cluster, error)
	ClusterNamespaceListerExpansion
}

// clusterNamespaceLister implements the ClusterNamespaceLister
// interface.
type clusterNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Clusters in the indexer for a given namespace.
func (s clusterNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Cluster, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Cluster))
	})
	return ret, err
}

// Get retrieves the Cluster from the indexer for a given namespace and name.
func (s clusterNamespaceLister) Get(name string) (*v1alpha1.Cluster, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("cluster"), name)
	}
	return obj.(*v1alpha1.Cluster), nil
}
