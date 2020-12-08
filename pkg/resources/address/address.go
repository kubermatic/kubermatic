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

package address

import (
	"context"
	"errors"
	"fmt"
	"net"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type lookupFunction func(host string) ([]net.IP, error)

type ModifiersBuilder struct {
	log         *zap.SugaredLogger
	client      ctrlruntimeclient.Client
	cluster     *kubermaticv1.Cluster
	seed        *kubermaticv1.Seed
	externalURL string
	// used to ease unit tests
	lookupFunction lookupFunction
}

func NewModifiersBuilder(log *zap.SugaredLogger) *ModifiersBuilder {
	return &ModifiersBuilder{
		log:            log,
		lookupFunction: net.LookupIP,
	}
}

func (m *ModifiersBuilder) Client(c ctrlruntimeclient.Client) *ModifiersBuilder {
	m.client = c
	return m
}

func (m *ModifiersBuilder) Cluster(c *kubermaticv1.Cluster) *ModifiersBuilder {
	m.cluster = c
	return m
}

func (m *ModifiersBuilder) Seed(s *kubermaticv1.Seed) *ModifiersBuilder {
	m.seed = s
	return m
}

func (m *ModifiersBuilder) ExternalURL(e string) *ModifiersBuilder {
	m.externalURL = e
	return m
}

func (m *ModifiersBuilder) lookupFunc(l lookupFunction) *ModifiersBuilder {
	m.lookupFunction = l
	return m
}

func (m *ModifiersBuilder) Build(ctx context.Context) ([]func(*kubermaticv1.Cluster), error) {
	var modifiers []func(*kubermaticv1.Cluster)
	if m.seed == nil {
		return modifiers, errors.New("providing seed is mandatory for building address modifiers")
	}
	if m.cluster == nil {
		return modifiers, errors.New("providing cluster is mandatory for building address modifiers")
	}
	if m.client == nil {
		return modifiers, errors.New("providing client is mandatory for building address modifiers")
	}

	subdomain := m.seed.Name
	if m.seed.Spec.SeedDNSOverwrite != "" {
		subdomain = m.seed.Spec.SeedDNSOverwrite
	}

	frontProxyLoadBalancerServiceIP := ""
	if m.cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		frontProxyLoadBalancerService := &corev1.Service{}
		nn := types.NamespacedName{Namespace: m.cluster.Status.NamespaceName, Name: resources.FrontLoadBalancerServiceName}
		if err := m.client.Get(ctx, nn, frontProxyLoadBalancerService); err != nil {
			return nil, fmt.Errorf("failed to get the front-loadbalancer service: %v", err)
		}
		// Use this as default in case the implementation doesn't populate the status
		frontProxyLoadBalancerServiceIP = frontProxyLoadBalancerService.Spec.LoadBalancerIP
		// Supposively there is only one if not..Good luck
		for _, ingress := range frontProxyLoadBalancerService.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				frontProxyLoadBalancerServiceIP = ingress.IP
			}
		}
	}

	// External Name
	externalName := ""
	if m.cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		externalName = frontProxyLoadBalancerServiceIP
	} else {
		externalName = fmt.Sprintf("%s.%s.%s", m.cluster.Name, subdomain, m.externalURL)
	}

	if m.cluster.Address.ExternalName != externalName {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.ExternalName = externalName
		})
		m.log.Debugw("Set external name for cluster", "externalName", externalName)
	}

	// Internal name
	internalName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.ApiserverServiceName, m.cluster.Status.NamespaceName)
	if m.cluster.Address.InternalName != internalName {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.InternalName = internalName
		})
		m.log.Debugw("Set internal name for cluster", "internalName", internalName)
	}

	// IP
	ip := ""
	if m.cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		ip = frontProxyLoadBalancerServiceIP
	} else {
		var err error
		// Always lookup IP address, in case it changes (IP's on AWS LB's change)
		ip, err = m.getExternalIPv4(externalName)
		if err != nil {
			return nil, err
		}
	}
	if m.cluster.Address.IP != ip {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.IP = ip
		})
		m.log.Debugw("Set IP for cluster", "ip", ip)
	}

	service := &corev1.Service{}
	serviceKey := types.NamespacedName{Namespace: m.cluster.Status.NamespaceName, Name: resources.ApiserverServiceName}
	if err := m.client.Get(ctx, serviceKey, service); err != nil {
		return nil, err
	}
	if len(service.Spec.Ports) < 1 {
		return nil, fmt.Errorf("service %q has no port configured", serviceKey.String())
	}

	// Port
	port := service.Spec.Ports[0].NodePort
	if m.cluster.Address.Port != port {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.Port = port
		})
		m.log.Debugw("Set port for cluster", "port", port)
	}

	// URL
	url := fmt.Sprintf("https://%s:%d", externalName, port)
	if m.cluster.Address.URL != url {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.URL = url
		})
		m.log.Debugw("Set URL for cluster", "url", url)
	}

	if m.cluster.IsOpenshift() {
		openshiftConsoleCallBackURL := fmt.Sprintf("https://%s/api/v1/projects/%s/dc/%s/clusters/%s/openshift/console/proxy/auth/callback",
			m.externalURL, m.cluster.Labels[kubermaticv1.ProjectIDLabelKey], m.seed.Name, m.cluster.Name)
		if m.cluster.Address.OpenshiftConsoleCallBack != openshiftConsoleCallBackURL {
			modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
				c.Address.OpenshiftConsoleCallBack = openshiftConsoleCallBackURL
			})
		}
	}

	return modifiers, nil
}

func (m *ModifiersBuilder) getExternalIPv4(hostname string) (string, error) {
	resolvedIPs, err := m.lookupFunction(hostname)
	if err != nil {
		return "", fmt.Errorf("failed to lookup ip for %s: %v", hostname, err)
	}
	ipList := sets.NewString()
	for _, ip := range resolvedIPs {
		if ip.To4() != nil {
			ipList.Insert(ip.String())
		}
	}
	ips := ipList.List()
	if len(ips) == 0 {
		return "", fmt.Errorf("no ip addresses found for %s: %v", hostname, err)
	}

	//Just one ipv4
	if len(ips) > 1 {
		m.log.Debugw("Lookup returned multiple ipv4 addresses. Picking the first one after sorting", "hostname", hostname, "foundAddresses", ips, "pickedAddress", ips[0])
	}
	return ips[0], nil
}
