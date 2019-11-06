// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	time "time"

	versioned "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/internalinterfaces"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// KubermaticSettingInformer provides access to a shared informer and lister for
// KubermaticSettings.
type KubermaticSettingInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.KubermaticSettingLister
}

type kubermaticSettingInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewKubermaticSettingInformer constructs a new informer for KubermaticSetting type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewKubermaticSettingInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredKubermaticSettingInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredKubermaticSettingInformer constructs a new informer for KubermaticSetting type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredKubermaticSettingInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubermaticV1().KubermaticSettings().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubermaticV1().KubermaticSettings().Watch(options)
			},
		},
		&kubermaticv1.KubermaticSetting{},
		resyncPeriod,
		indexers,
	)
}

func (f *kubermaticSettingInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredKubermaticSettingInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *kubermaticSettingInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&kubermaticv1.KubermaticSetting{}, f.defaultInformer)
}

func (f *kubermaticSettingInformer) Lister() v1.KubermaticSettingLister {
	return v1.NewKubermaticSettingLister(f.Informer().GetIndexer())
}
