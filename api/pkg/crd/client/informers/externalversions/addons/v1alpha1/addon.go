package v1alpha1

import (
	time "time"

	addons_v1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/addons/v1alpha1"
	versioned "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/addons/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// AddonInformer provides access to a shared informer and lister for
// Addons.
type AddonInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.AddonLister
}

type addonInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewAddonInformer constructs a new informer for Addon type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewAddonInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredAddonInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredAddonInformer constructs a new informer for Addon type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredAddonInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AddonsV1alpha1().Addons().List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.AddonsV1alpha1().Addons().Watch(options)
			},
		},
		&addons_v1alpha1.Addon{},
		resyncPeriod,
		indexers,
	)
}

func (f *addonInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredAddonInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *addonInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&addons_v1alpha1.Addon{}, f.defaultInformer)
}

func (f *addonInformer) Lister() v1alpha1.AddonLister {
	return v1alpha1.NewAddonLister(f.Informer().GetIndexer())
}
