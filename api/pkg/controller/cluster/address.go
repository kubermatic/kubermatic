package cluster

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/types"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

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
		glog.V(6).Infof("lookup of %s returned multiple ipv4 addresses (%v). Picking the first one after sorting: %s", hostname, ips, ips[0])
	}
	return ips[0], nil
}

// syncAddress will set the all address relevant fields on the cluster
func (r *Reconciler) syncAddress(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	var err error
	//TODO(mrIncompetent): The token should be moved out of Address. But maybe we rather implement another auth-handling? Like openid-connect?
	if cluster.Address.AdminToken == "" {
		// Generate token according to https://kubernetes.io/docs/admin/bootstrap-tokens/#token-format
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Address.AdminToken = kubernetes.GenerateToken()
		})
		if err != nil {
			return err
		}
		glog.V(4).Infof("Created admin token for cluster %s", cluster.Name)
	}

	nodeDc, found := r.dcs[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return fmt.Errorf("unknown node dataceter set '%s'", cluster.Spec.Cloud.DatacenterName)
	}
	seedDCName := r.dc
	if nodeDc.SeedDNSOverwrite != nil && *nodeDc.SeedDNSOverwrite != "" {
		seedDCName = *nodeDc.SeedDNSOverwrite
	}

	externalName := fmt.Sprintf("%s.%s.%s", cluster.Name, seedDCName, r.externalURL)
	if cluster.Address.ExternalName != externalName {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Address.ExternalName = externalName
		})
		if err != nil {
			return err
		}
		glog.V(4).Infof("Set external name for cluster %s to '%s'", cluster.Name, cluster.Address.ExternalName)
	}

	// Always lookup IP address, c case it changes (IP's on AWS LB's change)
	ip, err := getExternalIPv4(cluster.Address.ExternalName)
	if err != nil {
		return err
	}
	if cluster.Address.IP != ip {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Address.IP = ip
		})
		if err != nil {
			return err
		}
		glog.V(4).Infof("Set IP for cluster %s to '%s'", cluster.Name, cluster.Address.IP)
	}

	// We fetch the Apiserver service as its a NodePort and we'll take the first NodePort (so far we only have one)
	service := &corev1.Service{}
	serviceKey := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.ApiserverExternalServiceName}
	if err := r.Get(ctx, serviceKey, service); err != nil {
		if errors.IsNotFound(err) {
			glog.V(6).Infof("Skipping URL setting for cluster %s as no external apiserver service exists yet. Will retry later", cluster.Name)
			return nil
		}
		return err
	}

	url := fmt.Sprintf("https://%s:%d", cluster.Address.ExternalName, int(service.Spec.Ports[0].NodePort))
	if cluster.Address.URL != url {
		err = r.updateCluster(ctx, cluster, func(c *kubermaticv1.Cluster) {
			c.Address.URL = url
		})
		if err != nil {
			return err
		}
		glog.V(4).Infof("Set URL for cluster %s to '%s'", cluster.Name, cluster.Address.URL)
	}

	return nil
}
