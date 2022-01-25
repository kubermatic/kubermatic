// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	versioned "k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned"
	internalinterfaces "k8c.io/kubermatic/v2/pkg/crd/client/informers/externalversions/internalinterfaces"
	v1 "k8c.io/kubermatic/v2/pkg/crd/client/listers/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// EtcdRestoreInformer provides access to a shared informer and lister for
// EtcdRestores.
type EtcdRestoreInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.EtcdRestoreLister
}

type etcdRestoreInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewEtcdRestoreInformer constructs a new informer for EtcdRestore type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewEtcdRestoreInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredEtcdRestoreInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredEtcdRestoreInformer constructs a new informer for EtcdRestore type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredEtcdRestoreInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubermaticV1().EtcdRestores(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubermaticV1().EtcdRestores(namespace).Watch(context.TODO(), options)
			},
		},
		&kubermaticv1.EtcdRestore{},
		resyncPeriod,
		indexers,
	)
}

func (f *etcdRestoreInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredEtcdRestoreInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *etcdRestoreInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&kubermaticv1.EtcdRestore{}, f.defaultInformer)
}

func (f *etcdRestoreInformer) Lister() v1.EtcdRestoreLister {
	return v1.NewEtcdRestoreLister(f.Informer().GetIndexer())
}
