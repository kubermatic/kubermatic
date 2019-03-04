package rbac

import (
	"fmt"
	"sync"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// InformerProvider allows for storing shared informer factories for the given namespaces
// additionally it provides method for starting and waiting for all registered factories
type InformerProvider interface {
	// KubeInformerFactoryFor registeres a shared informer factory for the given namespace
	KubeInformerFactoryFor(namespace string) kubeinformers.SharedInformerFactory
	// StartInformers starts all registered factories
	StartInformers(stopCh <-chan struct{})
	// WaitForCachesToSync waits until caches from all factories are synced
	WaitForCachesToSync(stopCh <-chan struct{}) error
}

// informerProvider simply holds namespaced factories
type informerProvider struct {
	kubeClient    kubernetes.Interface
	kubeInformers map[string]kubeinformers.SharedInformerFactory

	resync  time.Duration
	lock    *sync.Mutex
	started bool
}

// NewInformerProvider creates a new provider that
func NewInformerProvider(kubeClient kubernetes.Interface, resync time.Duration) *informerProvider {
	return &informerProvider{kubeClient: kubeClient, resync: resync, kubeInformers: map[string]kubeinformers.SharedInformerFactory{}, lock: &sync.Mutex{}}
}

// KubeInformerFactoryFor registeres a shared informer factory for the given namespace
func (p *informerProvider) KubeInformerFactoryFor(namespace string) kubeinformers.SharedInformerFactory {
	p.lock.Lock()
	defer p.lock.Unlock()

	if informer, ok := p.kubeInformers[namespace]; ok {
		return informer
	}

	if p.started {
		// this is a programmer error
		panic("Please register a factory before starting the provider call to StartInformers method")
	}

	p.kubeInformers[namespace] = kubeinformers.NewFilteredSharedInformerFactory(p.kubeClient, p.resync, namespace, nil)
	return p.kubeInformers[namespace]
}

// StartInformers starts all registered factories
func (p *informerProvider) StartInformers(stopCh <-chan struct{}) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, informer := range p.kubeInformers {
		informer.Start(stopCh)
	}
	p.started = true
}

// WaitForCachesToSync waits until caches from all factories are synced
func (p *informerProvider) WaitForCachesToSync(stopCh <-chan struct{}) error {
	for _, informer := range p.kubeInformers {
		infKubeSyncStatus := informer.WaitForCacheSync(stopCh)
		for informerType, informerSynced := range infKubeSyncStatus {
			if !informerSynced {
				return fmt.Errorf("Unable to sync caches for for informer %v", informerType)
			}
		}
	}

	return nil
}
