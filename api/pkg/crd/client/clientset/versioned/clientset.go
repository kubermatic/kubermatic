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

package versioned

import (
	glog "github.com/golang/glog"
	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/etcdoperator/v1beta2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	EtcdoperatorV1beta2() etcdoperatorv1beta2.EtcdoperatorV1beta2Interface
	// Deprecated: please explicitly pick a version if possible.
	Etcdoperator() etcdoperatorv1beta2.EtcdoperatorV1beta2Interface
	KubermaticV1() kubermaticv1.KubermaticV1Interface
	// Deprecated: please explicitly pick a version if possible.
	Kubermatic() kubermaticv1.KubermaticV1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	etcdoperatorV1beta2 *etcdoperatorv1beta2.EtcdoperatorV1beta2Client
	kubermaticV1        *kubermaticv1.KubermaticV1Client
}

// EtcdoperatorV1beta2 retrieves the EtcdoperatorV1beta2Client
func (c *Clientset) EtcdoperatorV1beta2() etcdoperatorv1beta2.EtcdoperatorV1beta2Interface {
	return c.etcdoperatorV1beta2
}

// Deprecated: Etcdoperator retrieves the default version of EtcdoperatorClient.
// Please explicitly pick a version.
func (c *Clientset) Etcdoperator() etcdoperatorv1beta2.EtcdoperatorV1beta2Interface {
	return c.etcdoperatorV1beta2
}

// KubermaticV1 retrieves the KubermaticV1Client
func (c *Clientset) KubermaticV1() kubermaticv1.KubermaticV1Interface {
	return c.kubermaticV1
}

// Deprecated: Kubermatic retrieves the default version of KubermaticClient.
// Please explicitly pick a version.
func (c *Clientset) Kubermatic() kubermaticv1.KubermaticV1Interface {
	return c.kubermaticV1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.etcdoperatorV1beta2, err = etcdoperatorv1beta2.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.kubermaticV1, err = kubermaticv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		glog.Errorf("failed to create the DiscoveryClient: %v", err)
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.etcdoperatorV1beta2 = etcdoperatorv1beta2.NewForConfigOrDie(c)
	cs.kubermaticV1 = kubermaticv1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.etcdoperatorV1beta2 = etcdoperatorv1beta2.New(c)
	cs.kubermaticV1 = kubermaticv1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
