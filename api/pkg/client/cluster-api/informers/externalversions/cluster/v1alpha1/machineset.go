package v1alpha1

import (
	time "time"

	versioned "github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/clientset/versioned"
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kubermatic/kubermatic/api/pkg/client/cluster-api/listers/cluster/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	cluster_v1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// MachineSetInformer provides access to a shared informer and lister for
// MachineSets.
type MachineSetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.MachineSetLister
}

type machineSetInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewMachineSetInformer constructs a new informer for MachineSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMachineSetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredMachineSetInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredMachineSetInformer constructs a new informer for MachineSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredMachineSetInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ClusterV1alpha1().MachineSets(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ClusterV1alpha1().MachineSets(namespace).Watch(options)
			},
		},
		&cluster_v1alpha1.MachineSet{},
		resyncPeriod,
		indexers,
	)
}

func (f *machineSetInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredMachineSetInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *machineSetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&cluster_v1alpha1.MachineSet{}, f.defaultInformer)
}

func (f *machineSetInformer) Lister() v1alpha1.MachineSetLister {
	return v1alpha1.NewMachineSetLister(f.Informer().GetIndexer())
}
