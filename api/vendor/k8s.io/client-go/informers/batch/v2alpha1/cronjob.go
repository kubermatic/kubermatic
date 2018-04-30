/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by informer-gen. DO NOT EDIT.

package v2alpha1

import (
	time "time"

	batch_v2alpha1 "k8s.io/api/batch/v2alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	kubernetes "k8s.io/client-go/kubernetes"
	v2alpha1 "k8s.io/client-go/listers/batch/v2alpha1"
	cache "k8s.io/client-go/tools/cache"
)

// CronJobInformer provides access to a shared informer and lister for
// CronJobs.
type CronJobInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v2alpha1.CronJobLister
}

type cronJobInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewCronJobInformer constructs a new informer for CronJob type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewCronJobInformer(client kubernetes.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredCronJobInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredCronJobInformer constructs a new informer for CronJob type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredCronJobInformer(client kubernetes.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BatchV2alpha1().CronJobs(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.BatchV2alpha1().CronJobs(namespace).Watch(options)
			},
		},
		&batch_v2alpha1.CronJob{},
		resyncPeriod,
		indexers,
	)
}

func (f *cronJobInformer) defaultInformer(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredCronJobInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *cronJobInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&batch_v2alpha1.CronJob{}, f.defaultInformer)
}

func (f *cronJobInformer) Lister() v2alpha1.CronJobLister {
	return v2alpha1.NewCronJobLister(f.Informer().GetIndexer())
}
