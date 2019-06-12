package address

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/glog"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func SyncClusterAddress(ctx context.Context,
	cluster *kubermaticv1.Cluster,
	client ctrlruntimeclient.Client,
	externalURL, seedDCName string,
	nodeDCs map[string]provider.DatacenterMeta) ([]func(*kubermaticv1.Cluster), error) {
	var modifiers []func(*kubermaticv1.Cluster)

	nodeDc, found := nodeDCs[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("unknown node dataceter set '%s'", cluster.Spec.Cloud.DatacenterName)
	}
	if nodeDc.SeedDNSOverwrite != nil && *nodeDc.SeedDNSOverwrite != "" {
		seedDCName = *nodeDc.SeedDNSOverwrite
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
		externalName = fmt.Sprintf("%s.%s.%s", cluster.Name, seedDCName, externalURL)
	}

	if cluster.Address.ExternalName != externalName {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.ExternalName = externalName
		})
		glog.V(2).Infof("Set external name for cluster %s to %q", cluster.Name, externalName)
	}

	// Internal name
	internalName := fmt.Sprintf("%s.%s.svc.cluster.local.", resources.ApiserverExternalServiceName, cluster.Status.NamespaceName)
	if cluster.Address.InternalName != internalName {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.InternalName = internalName
		})
		glog.V(2).Infof("Set internal name for cluster %s to '%s'", cluster.Name, internalName)
	}

	// IP
	ip := ""
	if cluster.Spec.ExposeStrategy == corev1.ServiceTypeLoadBalancer {
		ip = frontProxyLoadBalancerServiceIP
	} else {
		var err error
		// Always lookup IP address, in case it changes (IP's on AWS LB's change)
		ip, err = getExternalIPv4(externalName)
		if err != nil {
			return nil, err
		}
	}
	if cluster.Address.IP != ip {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.IP = ip
		})
		glog.V(2).Infof("Set IP for cluster %s to '%s'", cluster.Name, ip)
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
		glog.V(2).Infof("Set port for cluster %s to %d", cluster.Name, port)
	}

	// URL
	url := fmt.Sprintf("https://%s:%d", externalName, port)
	if cluster.Address.URL != url {
		modifiers = append(modifiers, func(c *kubermaticv1.Cluster) {
			c.Address.URL = url
		})
		glog.V(2).Infof("Set URL for cluster %s to '%s'", cluster.Name, url)
	}

	return modifiers, nil
}

func getExternalIPv4(hostname string) (string, error) {
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
		glog.V(4).Infof("lookup of %s returned multiple ipv4 addresses (%v). Picking the first one after sorting: %s", hostname, ips, ips[0])
	}
	return ips[0], nil
}
