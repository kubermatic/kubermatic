package rbac

import (
	"fmt"

	"github.com/golang/glog"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1listers "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	MasterProviderPrefix = "master"
	SeedProviderPrefix   = "seed"
)

// ClusterProvider holds set of clients that allow for communication with the cluster and
// that are required to properly generate RBAC for resources in that particular cluster
type ClusterProvider struct {
	providerName              string
	kubeClient                kubernetes.Interface
	kubeInformerFactory       kuberinformers.SharedInformerFactory
	kubermaticClient          kubermaticclientset.Interface
	kubermaticInformerFactory externalversions.SharedInformerFactory

	rbacClusterRoleLister        rbaclister.ClusterRoleLister
	rbacClusterRoleBindingLister rbaclister.ClusterRoleBindingLister

	clusterResourceLister kubermaticv1listers.ClusterLister
}

// NewClusterProvider creates a brand new ClusterProvider
//
// Note:
// This method will create and register Listers for RBAC Roles and Bindings
func NewClusterProvider(providerName string, kubeClient kubernetes.Interface, kubeInformerFactory kuberinformers.SharedInformerFactory, kubermaticClient kubermaticclientset.Interface, kubermaticInformerFactory externalversions.SharedInformerFactory) *ClusterProvider {
	cp := &ClusterProvider{
		providerName:              providerName,
		kubeClient:                kubeClient,
		kubeInformerFactory:       kubeInformerFactory,
		kubermaticClient:          kubermaticClient,
		kubermaticInformerFactory: kubermaticInformerFactory,
	}

	cp.rbacClusterRoleLister = cp.kubeInformerFactory.Rbac().V1().ClusterRoles().Lister()
	cp.rbacClusterRoleBindingLister = cp.kubeInformerFactory.Rbac().V1().ClusterRoleBindings().Lister()

	return cp
}

// StartInformers starts shared informers factories
func (p *ClusterProvider) StartInformers(stopCh <-chan struct{}) {
	p.kubeInformerFactory.Start(stopCh)
	p.kubermaticInformerFactory.Start(stopCh)
}

// WaitForCachesToSync waits for all started informers' cache until they are synced.
func (p *ClusterProvider) WaitForCachesToSync(stopCh <-chan struct{}) error {
	infSyncStatus := p.kubermaticInformerFactory.WaitForCacheSync(stopCh)
	for informerType, informerSynced := range infSyncStatus {
		if !informerSynced {
			return fmt.Errorf("Unable to sync caches for seed cluster provider %s for informer %v", p.providerName, informerType)
		}
	}

	infKubeSyncStatus := p.kubeInformerFactory.WaitForCacheSync(stopCh)
	for informerType, informerSynced := range infKubeSyncStatus {
		if !informerSynced {
			return fmt.Errorf("Unable to sync caches for seed cluster provider %s for informer %v", p.providerName, informerType)
		}
	}
	return nil
}

// AddIndexerFor adds Lister for the given resource
// Note: this method creates Lister for some resources, for example "cluster" resources
func (p *ClusterProvider) AddIndexerFor(indexer cache.Indexer, gvr schema.GroupVersionResource) {
	if gvr.Resource == kubermaticv1.ClusterResourceName {
		p.clusterResourceLister = kubermaticv1listers.NewClusterLister(indexer)
		glog.V(4).Infof("creating a lister for resource %q for provider %q", gvr.String(), p.providerName)
	}
}
