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

// UserSSHKeyInformer provides access to a shared informer and lister for
// UserSSHKeies.
type UserSSHKeyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.UserSSHKeyLister
}

type userSSHKeyInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewUserSSHKeyInformer constructs a new informer for UserSSHKey type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewUserSSHKeyInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredUserSSHKeyInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredUserSSHKeyInformer constructs a new informer for UserSSHKey type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredUserSSHKeyInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubermaticV1().UserSSHKeies().List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubermaticV1().UserSSHKeies().Watch(options)
			},
		},
		&kubermatic_v1.UserSSHKey{},
		resyncPeriod,
		indexers,
	)
}

func (f *userSSHKeyInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredUserSSHKeyInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *userSSHKeyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&kubermatic_v1.UserSSHKey{}, f.defaultInformer)
}

func (f *userSSHKeyInformer) Lister() v1.UserSSHKeyLister {
	return v1.NewUserSSHKeyLister(f.Informer().GetIndexer())
}
