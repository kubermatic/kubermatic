// This file was automatically generated by informer-gen

package v1

import (
	versioned "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/informers/externalversions/internalinterfaces"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/listers/kubermatic/v1"
	kubermatic_v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
	time "time"
)

// UserSSHKeyInformer provides access to a shared informer and lister for
// UserSSHKeies.
type UserSSHKeyInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.UserSSHKeyLister
}

type userSSHKeyInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

// NewUserSSHKeyInformer constructs a new informer for UserSSHKey type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewUserSSHKeyInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.KubermaticV1().UserSSHKeies().List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.KubermaticV1().UserSSHKeies().Watch(options)
			},
		},
		&kubermatic_v1.UserSSHKey{},
		resyncPeriod,
		indexers,
	)
}

func defaultUserSSHKeyInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewUserSSHKeyInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (f *userSSHKeyInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&kubermatic_v1.UserSSHKey{}, defaultUserSSHKeyInformer)
}

func (f *userSSHKeyInformer) Lister() v1.UserSSHKeyLister {
	return v1.NewUserSSHKeyLister(f.Informer().GetIndexer())
}
