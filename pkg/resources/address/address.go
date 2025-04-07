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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	utilnet "k8s.io/utils/net"
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

	frontProxyLBServiceIP := ""
	frontProxyLBServiceHostname := ""
	if m.cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		frontProxyLoadBalancerService := &corev1.Service{}
		nn := types.NamespacedName{Namespace: m.cluster.Status.NamespaceName, Name: resources.FrontLoadBalancerServiceName}
		if err := m.client.Get(ctx, nn, frontProxyLoadBalancerService); err != nil {
			return nil, fmt.Errorf("failed to get the front-loadbalancer service: %w", err)
		}
		frontProxyLBServiceIP, frontProxyLBServiceHostname = m.getFrontProxyLBServiceData(frontProxyLoadBalancerService)
	}

	// External Name
	externalName := ""
	if m.cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		if frontProxyLBServiceIP != "" {
			externalName = frontProxyLBServiceIP
		} else {
			externalName = frontProxyLBServiceHostname
		}
	} else {
		externalName = fmt.Sprintf("%s.%s.%s", m.cluster.Name, subdomain, m.externalURL)
	}

	if m.cluster.Status.Address.ExternalName != externalName {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Status.Address.ExternalName = externalName
		})
		m.log.Debugw("Set external name for cluster", "externalName", externalName)
	}

	// Internal name
	internalName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.ApiserverServiceName, m.cluster.Status.NamespaceName)
	if m.cluster.Status.Address.InternalName != internalName {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Status.Address.InternalName = internalName
		})
		m.log.Debugw("Set internal name for cluster", "internalName", internalName)
	}

	// IP
	ip := ""
	// When using the Tunneling expose strategy we disable KAS endpoints
	// reconciliation, and we reconcile them with the agent IPs in the user
	// controller manager.
	switch m.cluster.Spec.ExposeStrategy {
	case kubermaticv1.ExposeStrategyLoadBalancer:
		if frontProxyLBServiceIP != "" {
			ip = frontProxyLBServiceIP
		} else if frontProxyLBServiceHostname != "" {
			var err error
			// Always lookup IP address, in case it changes
			ip, err = m.getExternalIP(frontProxyLBServiceHostname)
			if err != nil {
				return nil, err
			}
		}
	case kubermaticv1.ExposeStrategyNodePort:
		fallthrough
	case kubermaticv1.ExposeStrategyTunneling:
		var err error
		// Always lookup IP address, in case it changes (IP's on AWS LB's change)
		ip, err = m.getExternalIP(externalName)
		if err != nil {
			return nil, err
		}
	}
	if m.cluster.Status.Address.IP != ip {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Status.Address.IP = ip
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

	if service.Spec.Type == corev1.ServiceTypeLoadBalancer {
		apiServerHostName := ""
		apiServerIP := ""
		for _, ingress := range service.Status.LoadBalancer.Ingress {
			if ingress.Hostname != "" {
				apiServerHostName = ingress.Hostname
				break
			}

			// Picks the first available IP address.
			if len(ingress.IP) > 0 && apiServerIP == "" {
				apiServerIP = ingress.IP
			}
		}

		// Hostname has precedence over IP.
		if apiServerHostName != "" {
			if m.cluster.Status.Address.APIServerExternalAddress != fmt.Sprintf("https://%s", apiServerHostName) {
				modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
					c.Status.Address.APIServerExternalAddress = fmt.Sprintf("https://%s", apiServerHostName)
				})
				m.log.Debugw("Set APIServer hostname for cluster", "hostname", apiServerHostName)
			}
		} else if apiServerIP != "" {
			if m.cluster.Status.Address.APIServerExternalAddress != fmt.Sprintf("https://%s", apiServerIP) {
				modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
					c.Status.Address.APIServerExternalAddress = fmt.Sprintf("https://%s", apiServerIP)
				})
				m.log.Debugw("Set APIServer IP for cluster", "ip", apiServerIP)
			}
		}
	}

	// Port
	var port = service.Spec.Ports[0].TargetPort.IntVal
	if m.cluster.Spec.ExposeStrategy != kubermaticv1.ExposeStrategyTunneling {
		port = service.Spec.Ports[0].NodePort
	}

	// Use the nodeport value for KAS secure port when strategy is NodePort or
	// LoadBalancer. This is because the same service will be accessed both
	// locally and passing from nodeport proxy.
	if m.cluster.Status.Address.Port != port {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Status.Address.Port = port
		})
		m.log.Debugw("Set port for cluster", "port", port)
	}

	// URL
	url := fmt.Sprintf("https://%s", net.JoinHostPort(externalName, fmt.Sprintf("%d", port)))
	if m.cluster.Status.Address.URL != url {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Status.Address.URL = url
		})
		m.log.Debugw("Set URL for cluster", "url", url)
	}

	return modifiers, nil
}

func (m *ModifiersBuilder) getFrontProxyLBServiceData(frontProxyLoadBalancerService *corev1.Service) (string, string) {
	//  frontProxyLBServiceIP is set according to below priority
	// 1. First public IPv4 from the status list
	// 2. First private IPv4 from the status list
	// 3. First public IPv6 from the status list
	// 4. First private IPv6 from the status list
	// 5. Default IP as per configured spec if status is not populated
	serviceIP := ""
	serviceHostname := ""
	var publicIPv4, privateIPv4, privateIPv6, publicIPv6 []string

	for _, ingress := range frontProxyLoadBalancerService.Status.LoadBalancer.Ingress {
		if ingress.Hostname != "" {
			serviceHostname = ingress.Hostname
		}

		if len(ingress.IP) == 0 {
			continue
		}

		if utilnet.IsIPv4String(ingress.IP) {
			if !net.ParseIP(ingress.IP).IsPrivate() {
				publicIPv4 = append(publicIPv4, ingress.IP)
			} else {
				privateIPv4 = append(privateIPv4, ingress.IP)
			}
		} else if utilnet.IsIPv6String(ingress.IP) {
			if !net.ParseIP(ingress.IP).IsPrivate() {
				publicIPv6 = append(publicIPv6, ingress.IP)
			} else {
				privateIPv6 = append(privateIPv6, ingress.IP)
			}
		}
	}

	switch {
	case len(publicIPv4) > 0:
		serviceIP = publicIPv4[0]
	case len(privateIPv4) > 0:
		serviceIP = privateIPv4[0]
	case len(publicIPv6) > 0:
		serviceIP = publicIPv6[0]
	case len(privateIPv6) > 0:
		serviceIP = privateIPv6[0]
	}

	m.log.Debugw("From the ingress values in LB status, the following values will be used", "ip", serviceIP, "hostname", serviceHostname)

	// default in case the implementation doesn't populate the status
	if len(frontProxyLoadBalancerService.Status.LoadBalancer.Ingress) == 0 {
		serviceIP = frontProxyLoadBalancerService.Spec.LoadBalancerIP
	}

	return serviceIP, serviceHostname
}

func (m *ModifiersBuilder) getExternalIP(hostname string) (string, error) {
	resolvedIPs, err := m.lookupFunction(hostname)
	if err != nil {
		return "", fmt.Errorf("failed to lookup ip for %s: %w", hostname, err)
	}
	ipList := sets.New[string]()
	for _, ip := range resolvedIPs {
		if ip.To4() != nil {
			ipList.Insert(ip.String())
		}
	}

	// If no IPv4 address was found, look for IPv6 addresses.
	if len(ipList) == 0 {
		for _, ip := range resolvedIPs {
			if ip.To16() != nil && len(ip.To16()) == net.IPv6len {
				ipList.Insert(ip.String())
			}
		}
	}

	ips := sets.List(ipList)
	if len(ips) == 0 {
		return "", fmt.Errorf("no ip addresses found for %s: %w", hostname, err)
	}
	// Return the first IP address.
	if len(ips) > 1 {
		m.log.Debugw("Lookup returned multiple IP addresses. Picking the first one after sorting", "hostname", hostname, "foundAddresses", ips, "pickedAddress", ips[0])
	}

	// If the current cluster external address is found among the resolved IPs, the same IP will be returned, as there's no
	// need to change the user cluster external endpoint. Otherwise, the cluster control plane components may enter an
	// infinite reconciliation loop
	if ipList.Has(m.cluster.Status.Address.IP) {
		return m.cluster.Status.Address.IP, nil
	}

	return ips[0], nil
}
