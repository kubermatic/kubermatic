package fake

import (
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type SharedInformerFactory struct {
	kubeinformers.SharedInformerFactory
}

// NewFakeSharedInformerFactory returns a new factory
func NewFakeSharedInformerFactory(kubeClient kubernetes.Interface, namespace string) *SharedInformerFactory {
	f := kubeinformers.NewFilteredSharedInformerFactory(kubeClient, time.Minute*5, namespace, nil)
	factory := &SharedInformerFactory{SharedInformerFactory: f}
	return factory
}

// AddFakeClusterRoleInformer adds a dummy informer that returns items from clusterRoleIndexer
func (f *SharedInformerFactory) AddFakeClusterRoleInformer(clusterRoleIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.ClusterRole{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: clusterRoleIndexer}
	})
}

// AddFakeClusterRoleBindingInformer adds a dummy informer that returns items from clusterRoleBindingIndexer
func (f *SharedInformerFactory) AddFakeClusterRoleBindingInformer(clusterRoleBindingIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.ClusterRoleBinding{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: clusterRoleBindingIndexer}
	})
}

// AddFakeRoleInformer adds a dummy informer that returns items from roleIndexer
func (f *SharedInformerFactory) AddFakeRoleInformer(roleIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.Role{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: roleIndexer}
	})
}

// AddFakeRoleBindingInformer adds a dummy informer that returns items from roleBindingIndexer
func (f *SharedInformerFactory) AddFakeRoleBindingInformer(roleBindingIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.RoleBinding{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: roleBindingIndexer}
	})
}

type dummySharedIndexInformer struct {
	indexer cache.Indexer
}

func (i *dummySharedIndexInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	panic("implement me")
}

func (i *dummySharedIndexInformer) AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) {
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

func (i *dummySharedIndexInformer) LastSyncResourceVersion() string {
	panic("implement me")
}

func (i *dummySharedIndexInformer) AddIndexers(indexers cache.Indexers) error {
	panic("implement me")
}

func (i *dummySharedIndexInformer) GetIndexer() cache.Indexer {
	return i.indexer
}
