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

package rbac

import (
	"fmt"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	"github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1listers "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

// ClusterProvider holds set of clients that allow for communication with the cluster and
// that are required to properly generate RBAC for resources in that particular cluster
type ClusterProvider struct {
	providerName              string
	kubeClient                kubernetes.Interface
	kubeInformerProvider      InformerProvider
	kubermaticClient          kubermaticclientset.Interface
	kubermaticInformerFactory externalversions.SharedInformerFactory

	clusterResourceLister kubermaticv1listers.ClusterLister
}

// NewClusterProvider creates a brand new ClusterProvider
//
// Note:
// This method will create and register Listers for RBAC Roles and Bindings
func NewClusterProvider(providerName string, kubeClient kubernetes.Interface, kubeInformerProvider InformerProvider, kubermaticClient kubermaticclientset.Interface, kubermaticInformerFactory externalversions.SharedInformerFactory) *ClusterProvider {
	cp := &ClusterProvider{
		providerName:              providerName,
		kubeClient:                kubeClient,
		kubeInformerProvider:      kubeInformerProvider,
		kubermaticClient:          kubermaticClient,
		kubermaticInformerFactory: kubermaticInformerFactory,
	}

	// registering Listers for RBAC Cluster Roles and Bindings
	klog.V(4).Infof("registering ClusterRoles and ClusterRoleBindings informers in all namespaces for provider %s", providerName)
	_ = cp.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoles().Lister()
	_ = cp.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().ClusterRoleBindings().Lister()

	_ = cp.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().Roles().Lister()
	_ = cp.kubeInformerProvider.KubeInformerFactoryFor(metav1.NamespaceAll).Rbac().V1().RoleBindings().Lister()

	return cp
}

// StartInformers starts shared informers factories
func (p *ClusterProvider) StartInformers(stopCh <-chan struct{}) {
	p.kubeInformerProvider.StartInformers(stopCh)
	p.kubermaticInformerFactory.Start(stopCh)
}

// WaitForCachesToSync waits for all started informers' cache until they are synced.
func (p *ClusterProvider) WaitForCachesToSync(stopCh <-chan struct{}) error {
	infSyncStatus := p.kubermaticInformerFactory.WaitForCacheSync(stopCh)
	for informerType, informerSynced := range infSyncStatus {
		if !informerSynced {
			return fmt.Errorf("unable to sync caches for seed cluster provider %s for informer %v", p.providerName, informerType)
		}
	}

	if err := p.kubeInformerProvider.WaitForCachesToSync(stopCh); err != nil {
		return fmt.Errorf("unable to sync caches for kubermatic provider for cluster provider %s due to %v", p.providerName, err)
	}
	return nil
}

// AddIndexerFor adds Lister for the given resource
// Note: this method creates Lister for some resources, for example "cluster" resources
//
// TODO: try rm this since we have InformerProvider
func (p *ClusterProvider) AddIndexerFor(indexer cache.Indexer, gvr schema.GroupVersionResource) {
	if gvr.Resource == kubermaticv1.ClusterResourceName {
		p.clusterResourceLister = kubermaticv1listers.NewClusterLister(indexer)
		klog.V(4).Infof("creating a lister for resource %q for provider %q", gvr.String(), p.providerName)
	}
}
