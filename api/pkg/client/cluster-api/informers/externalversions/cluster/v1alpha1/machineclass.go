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

// MachineClassInformer provides access to a shared informer and lister for
// MachineClasses.
type MachineClassInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.MachineClassLister
}

type machineClassInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewMachineClassInformer constructs a new informer for MachineClass type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMachineClassInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredMachineClassInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredMachineClassInformer constructs a new informer for MachineClass type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredMachineClassInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ClusterV1alpha1().MachineClasses(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.ClusterV1alpha1().MachineClasses(namespace).Watch(options)
			},
		},
		&cluster_v1alpha1.MachineClass{},
		resyncPeriod,
		indexers,
	)
}

func (f *machineClassInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredMachineClassInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *machineClassInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&cluster_v1alpha1.MachineClass{}, f.defaultInformer)
}

func (f *machineClassInformer) Lister() v1alpha1.MachineClassLister {
	return v1alpha1.NewMachineClassLister(f.Informer().GetIndexer())
}
