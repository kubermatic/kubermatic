package v1

import (
	time "time"

	versioned "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	internalinterfaces "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/internalinterfaces"
	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/prometheus/v1"
	prometheus_v1 "github.com/kubermatic/kubermatic/api/pkg/crd/prometheus/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ServiceMonitorInformer provides access to a shared informer and lister for
// ServiceMonitors.
type ServiceMonitorInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ServiceMonitorLister
}

type serviceMonitorInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewServiceMonitorInformer constructs a new informer for ServiceMonitor type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewServiceMonitorInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredServiceMonitorInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredServiceMonitorInformer constructs a new informer for ServiceMonitor type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredServiceMonitorInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MonitoringV1().ServiceMonitors(namespace).List(options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MonitoringV1().ServiceMonitors(namespace).Watch(options)
			},
		},
		&prometheus_v1.ServiceMonitor{},
		resyncPeriod,
		indexers,
	)
}

func (f *serviceMonitorInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredServiceMonitorInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *serviceMonitorInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&prometheus_v1.ServiceMonitor{}, f.defaultInformer)
}

func (f *serviceMonitorInformer) Lister() v1.ServiceMonitorLister {
	return v1.NewServiceMonitorLister(f.Informer().GetIndexer())
}
