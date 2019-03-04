package fake

import (
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type fakeSharedInformerFactory struct {
	kubeinformers.SharedInformerFactory
}

// NewFakeSharedInformerFactory returns a new factory
func NewFakeSharedInformerFactory(kubeClient kubernetes.Interface, namespace string) *fakeSharedInformerFactory {
	f := kubeinformers.NewFilteredSharedInformerFactory(kubeClient, time.Minute*5, namespace, nil)
	factory := &fakeSharedInformerFactory{SharedInformerFactory: f}
	return factory
}

// AddFakeClusterRoleInformer adds a dummy informer that returns items from clusterRoleIndexer
func (f *fakeSharedInformerFactory) AddFakeClusterRoleInformer(clusterRoleIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.ClusterRole{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: clusterRoleIndexer}
	})
}

// AddFakeClusterRoleBindingInformer adds a dummy informer that returns items from clusterRoleBindingIndexer
func (f *fakeSharedInformerFactory) AddFakeClusterRoleBindingInformer(clusterRoleBindingIndexer cache.Indexer) {
	f.InformerFor(&rbacv1.ClusterRoleBinding{}, func(fakeKubeClient kubernetes.Interface, resync time.Duration) cache.SharedIndexInformer {
		return &dummySharedIndexInformer{indexer: clusterRoleBindingIndexer}
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
