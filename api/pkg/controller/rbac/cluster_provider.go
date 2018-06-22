package rbac

import (
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"

	"fmt"

	kuberinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
)

const (
	masterProviderName = "master"
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
