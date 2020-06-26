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
	"fmt"
	"net"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func SyncClusterAddress(ctx context.Context,
	log *zap.SugaredLogger,
	cluster *kubermaticv1.Cluster,
	client ctrlruntimeclient.Client,
	externalURL string,
	seed *kubermaticv1.Seed) ([]func(*kubermaticv1.Cluster), error) {
	var modifiers []func(*kubermaticv1.Cluster)

	subdomain := seed.Name
	if seed.Spec.SeedDNSOverwrite != "" {
		subdomain = seed.Spec.SeedDNSOverwrite
	}

	frontProxyLoadBalancerServiceIP := ""
	if cluster.Spec.ExposeStrategy == corev1.ServiceTypeLoadBalancer {
		frontProxyLoadBalancerService := &corev1.Service{}
		nn := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.FrontLoadBalancerServiceName}
		if err := client.Get(ctx, nn, frontProxyLoadBalancerService); err != nil {
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
	if cluster.Spec.ExposeStrategy == corev1.ServiceTypeLoadBalancer {
		externalName = frontProxyLoadBalancerServiceIP
	} else {
		externalName = fmt.Sprintf("%s.%s.%s", cluster.Name, subdomain, externalURL)
	}

	if cluster.Address.ExternalName != externalName {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.ExternalName = externalName
		})
		log.Debugw("Set external name for cluster", "externalName", externalName)
	}

	// Internal name
	internalName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.ApiserverExternalServiceName, cluster.Status.NamespaceName)
	if cluster.Address.InternalName != internalName {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.InternalName = internalName
		})
		log.Debugw("Set internal name for cluster", "internalName", internalName)
	}

	// IP
	ip := ""
	if cluster.Spec.ExposeStrategy == corev1.ServiceTypeLoadBalancer {
		ip = frontProxyLoadBalancerServiceIP
	} else {
		var err error
		// Always lookup IP address, in case it changes (IP's on AWS LB's change)
		ip, err = getExternalIPv4(log, externalName)
		if err != nil {
			return nil, err
		}
	}
	if cluster.Address.IP != ip {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.IP = ip
		})
		log.Debugw("Set IP for cluster", "ip", ip)
	}

	service := &corev1.Service{}
	serviceKey := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.ApiserverExternalServiceName}
	if err := client.Get(ctx, serviceKey, service); err != nil {
		return nil, err
	}
	if len(service.Spec.Ports) < 1 {
		return nil, fmt.Errorf("service %q has no port configured", serviceKey.String())
	}

	// Port
	port := service.Spec.Ports[0].NodePort
	if cluster.Address.Port != port {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.Port = port
		})
		log.Debugw("Set port for cluster", "port", port)
	}

	// URL
	url := fmt.Sprintf("https://%s:%d", externalName, port)
	if cluster.Address.URL != url {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.URL = url
		})
		log.Debugw("Set URL for cluster", "url", url)
	}

	if cluster.IsOpenshift() {
		openshiftConsoleCallBackURL := fmt.Sprintf("https://%s/api/v1/projects/%s/dc/%s/clusters/%s/openshift/console/proxy/auth/callback",
			externalURL, cluster.Labels[kubermaticv1.ProjectIDLabelKey], seed.Name, cluster.Name)
		if cluster.Address.OpenshiftConsoleCallBack != openshiftConsoleCallBackURL {
			modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
				c.Address.OpenshiftConsoleCallBack = openshiftConsoleCallBackURL
			})
		}
	}

	return modifiers, nil
}

func getExternalIPv4(log *zap.SugaredLogger, hostname string) (string, error) {
	resolvedIPs, err := net.LookupIP(hostname)
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
		log.Debugw("Lookup returned multiple ipv4 addresses. Picking the first one after sorting", "hostname", hostname, "foundAddresses", ips, "pickedAddress", ips[0])
	}
	return ips[0], nil
}
