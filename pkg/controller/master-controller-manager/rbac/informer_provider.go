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
	"sync"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// InformerProvider allows for storing shared informer factories for the given namespaces
// additionally it provides method for starting and waiting for all registered factories.
type InformerProvider interface {
	// KubeInformerFactoryFor registers a shared informer factory for the given namespace
	KubeInformerFactoryFor(namespace string) kubeinformers.SharedInformerFactory
	// StartInformers starts all registered factories
	StartInformers(stopCh <-chan struct{})
	// WaitForCachesToSync waits until caches from all factories are synced
	WaitForCachesToSync(stopCh <-chan struct{}) error
}

// InformerProviderImpl simply holds namespaced factories.
type InformerProviderImpl struct {
	kubeClient    kubernetes.Interface
	kubeInformers map[string]kubeinformers.SharedInformerFactory

	resync  time.Duration
	lock    *sync.Mutex
	started bool
}

// NewInformerProvider creates a new provider that.
func NewInformerProvider(kubeClient kubernetes.Interface, resync time.Duration) *InformerProviderImpl {
	return &InformerProviderImpl{kubeClient: kubeClient, resync: resync, kubeInformers: map[string]kubeinformers.SharedInformerFactory{}, lock: &sync.Mutex{}}
}

// KubeInformerFactoryFor registers a shared informer factory for the given namespace.
func (p *InformerProviderImpl) KubeInformerFactoryFor(namespace string) kubeinformers.SharedInformerFactory {
	p.lock.Lock()
	defer p.lock.Unlock()

	if informer, ok := p.kubeInformers[namespace]; ok {
		return informer
	}

	if p.started {
		// this is a programmer error
		panic("Please register a factory before starting the provider call to StartInformers method")
	}

	p.kubeInformers[namespace] = kubeinformers.NewSharedInformerFactoryWithOptions(p.kubeClient, p.resync, kubeinformers.WithNamespace(namespace))
	return p.kubeInformers[namespace]
}

// StartInformers starts all registered factories.
func (p *InformerProviderImpl) StartInformers(stopCh <-chan struct{}) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, informer := range p.kubeInformers {
		informer.Start(stopCh)
	}
	p.started = true
}

// WaitForCachesToSync waits until caches from all factories are synced.
func (p *InformerProviderImpl) WaitForCachesToSync(stopCh <-chan struct{}) error {
	for _, informer := range p.kubeInformers {
		infKubeSyncStatus := informer.WaitForCacheSync(stopCh)
		for informerType, informerSynced := range infKubeSyncStatus {
			if !informerSynced {
				return fmt.Errorf("unable to sync caches for for informer %v", informerType)
			}
		}
	}

	return nil
}
