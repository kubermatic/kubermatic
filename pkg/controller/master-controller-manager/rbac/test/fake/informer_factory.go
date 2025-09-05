/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package fake

import (
	"context"
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type SharedInformerFactory struct {
	kubeinformers.SharedInformerFactory
}

// NewFakeSharedInformerFactory returns a new factory.
func NewFakeSharedInformerFactory(kubeClient kubernetes.Interface, namespace string) *SharedInformerFactory {
	f := kubeinformers.NewFilteredSharedInformerFactory(kubeClient, time.Minute*5, namespace, nil)
	factory := &SharedInformerFactory{SharedInformerFactory: f}
	return factory
}

// AddFakeClusterRoleInformer adds a dummy informer that returns items from clusterRoleIndexer.
func (f *SharedInformerFactory) AddFakeClusterRoleInformer(clusterRoleIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.ClusterRole{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: clusterRoleIndexer}
	})
}

// AddFakeClusterRoleBindingInformer adds a dummy informer that returns items from clusterRoleBindingIndexer.
func (f *SharedInformerFactory) AddFakeClusterRoleBindingInformer(clusterRoleBindingIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.ClusterRoleBinding{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: clusterRoleBindingIndexer}
	})
}

// AddFakeRoleInformer adds a dummy informer that returns items from roleIndexer.
func (f *SharedInformerFactory) AddFakeRoleInformer(roleIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.Role{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: roleIndexer}
	})
}

// AddFakeRoleBindingInformer adds a dummy informer that returns items from roleBindingIndexer.
func (f *SharedInformerFactory) AddFakeRoleBindingInformer(roleBindingIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.RoleBinding{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: roleBindingIndexer}
	})
}

type dummySharedIndexInformer struct {
	indexer cache.Indexer
}

var _ cache.SharedIndexInformer = &dummySharedIndexInformer{}

func (i *dummySharedIndexInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	panic("implement me")
}

func (i *dummySharedIndexInformer) AddEventHandlerWithOptions(handler cache.ResourceEventHandler, options cache.HandlerOptions) (cache.ResourceEventHandlerRegistration, error) {
	panic("implement me")
}

func (i *dummySharedIndexInformer) RemoveEventHandler(handler cache.ResourceEventHandlerRegistration) error {
	panic("implement me")
}

func (i *dummySharedIndexInformer) SetWatchErrorHandler(handler cache.WatchErrorHandler) error {
	panic("implement me")
}

func (i *dummySharedIndexInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) (cache.ResourceEventHandlerRegistration, error) {
	panic("implement me")
}

func (i *dummySharedIndexInformer) GetStore() cache.Store {
	panic("implement me")
}

func (i *dummySharedIndexInformer) GetController() cache.Controller {
	panic("implement me")
}

func (i *dummySharedIndexInformer) Run(stopCh <-chan struct{}) {
	panic("implement me")
}

func (i *dummySharedIndexInformer) HasSynced() bool {
	panic("implement me")
}

func (i *dummySharedIndexInformer) IsStopped() bool {
	panic("implement me")
}

func (i *dummySharedIndexInformer) LastSyncResourceVersion() string {
	panic("implement me")
}

func (i *dummySharedIndexInformer) AddIndexers(indexers cache.Indexers) error {
	panic("implement me")
}

func (i *dummySharedIndexInformer) RunWithContext(ctx context.Context) {
	panic("implement me")
}

func (i *dummySharedIndexInformer) SetTransform(handler cache.TransformFunc) error {
	return nil
}

func (i *dummySharedIndexInformer) SetWatchErrorHandlerWithContext(handler cache.WatchErrorHandlerWithContext) error {
	panic("implement me")
}

func (i *dummySharedIndexInformer) GetIndexer() cache.Indexer {
	return i.indexer
}
