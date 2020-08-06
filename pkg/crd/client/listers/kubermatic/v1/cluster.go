// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ClusterLister helps list Clusters.
type ClusterLister interface {
	// List lists all Clusters in the indexer.
	List(selector labels.Selector) (ret []*v1.Cluster, err error)
	// Get retrieves the Cluster from the index for a given name.
	Get(name string) (*v1.Cluster, error)
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
func (s *clusterLister) List(selector labels.Selector) (ret []*v1.Cluster, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.Cluster))
	})
	return ret, err
}

// Get retrieves the Cluster from the index for a given name.
func (s *clusterLister) Get(name string) (*v1.Cluster, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("cluster"), name)
	}
	return obj.(*v1.Cluster), nil
}
