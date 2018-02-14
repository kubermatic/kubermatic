/*
Copyright 2017 The Kubernetes Authors.

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

// This file was automatically generated by informer-gen

package v1beta2

import (
	apps_v1beta2 "k8s.io/api/apps/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	internalinterfaces "k8s.io/client-go/informers/internalinterfaces"
	kubernetes "k8s.io/client-go/kubernetes"
	v1beta2 "k8s.io/client-go/listers/apps/v1beta2"
	cache "k8s.io/client-go/tools/cache"
	time "time"
)

// StatefulSetInformer provides access to a shared informer and lister for
// StatefulSets.
type StatefulSetInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1beta2.StatefulSetLister
}

type statefulSetInformer struct {
	factory internalinterfaces.SharedInformerFactory
}

// NewStatefulSetInformer constructs a new informer for StatefulSet type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewStatefulSetInformer(client kubernetes.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return client.AppsV1beta2().StatefulSets(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return client.AppsV1beta2().StatefulSets(namespace).Watch(options)
			},
		},
		&apps_v1beta2.StatefulSet{},
		resyncPeriod,
		indexers,
	)
}

func defaultStatefulSetInformer(client kubernetes.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewStatefulSetInformer(client, v1.NamespaceAll, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
}

func (f *statefulSetInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apps_v1beta2.StatefulSet{}, defaultStatefulSetInformer)
}

func (f *statefulSetInformer) Lister() v1beta2.StatefulSetLister {
	return v1beta2.NewStatefulSetLister(f.Informer().GetIndexer())
}
