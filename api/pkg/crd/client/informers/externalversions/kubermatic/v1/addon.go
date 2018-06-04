package v1

import (
	time "time"

	versioned "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/internalinterfaces"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermatic_v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// AddonInformer provides access to a shared informer and lister for
// Addons.
type AddonInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.AddonLister
}

type addonInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewAddonInformer constructs a new informer for Addon type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAddonInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAddonInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredAddonInformer constructs a new informer for Addon type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAddonInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubermaticV1().Addons(namespace).List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubermaticV1().Addons(namespace).Watch(options)
			},
		},
		&kubermatic_v1.Addon{},
		resyncPeriod,
		indexers,
	)
}

func (f *addonInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAddonInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *addonInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&kubermatic_v1.Addon{}, f.defaultInformer)
}

func (f *addonInformer) Lister() v1.AddonLister {
	return v1.NewAddonLister(f.Informer().GetIndexer())
}
