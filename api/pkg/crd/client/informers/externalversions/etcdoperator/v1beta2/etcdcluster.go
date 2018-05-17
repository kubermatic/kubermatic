package v1beta2

import (
	time "time"

	versioned "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/internalinterfaces"
	v1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/etcdoperator/v1beta2"
	etcdoperator_v1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// EtcdClusterInformer provides access to a shared informer and lister for
// EtcdClusters.
type EtcdClusterInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta2.EtcdClusterLister
}

type etcdClusterInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewEtcdClusterInformer constructs a new informer for EtcdCluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewEtcdClusterInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredEtcdClusterInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredEtcdClusterInformer constructs a new informer for EtcdCluster type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredEtcdClusterInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.EtcdV1beta2().EtcdClusters(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.EtcdV1beta2().EtcdClusters(namespace).Watch(options)
			},
		},
		&etcdoperator_v1beta2.EtcdCluster{},
		resyncPeriod,
		indexers,
	)
}

func (f *etcdClusterInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredEtcdClusterInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *etcdClusterInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&etcdoperator_v1beta2.EtcdCluster{}, f.defaultInformer)
}

func (f *etcdClusterInformer) Lister() v1beta2.EtcdClusterLister {
	return v1beta2.NewEtcdClusterLister(f.Informer().GetIndexer())
}
