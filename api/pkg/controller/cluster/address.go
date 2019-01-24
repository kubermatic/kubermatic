package cluster

import (
	"fmt"
	"net"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
)

func getExternalIPv4Set(hostname string) (sets.String, error) {
	resolvedIPs, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup ip for %s: %v", hostname, err)
	}
	ipList := sets.NewString()
	for _, ip := range resolvedIPs {
		if ip.To4() != nil {
			ipList.Insert(ip.String())
		}
	}

	return ipList, nil
}

// syncAddress will set the all address relevant fields on the cluster
func (cc *Controller) syncAddress(c *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	var err error
	//TODO(mrIncompetent): The token should be moved out of Address. But maybe we rather implement another auth-handling? Like openid-connect?
	if c.Address.AdminToken == "" {
		// Generate token according to https://kubernetes.io/docs/admin/bootstrap-tokens/#token-format
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Address.AdminToken = kubernetes.GenerateToken()
		})
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof("Created admin token for cluster %s", c.Name)
	}

	nodeDc, found := cc.dcs[c.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("unknown node dataceter set '%s'", c.Spec.Cloud.DatacenterName)
	}
	seedDCName := cc.dc
	if nodeDc.SeedDNSOverwrite != nil && *nodeDc.SeedDNSOverwrite != "" {
		seedDCName = *nodeDc.SeedDNSOverwrite
	}

	externalName := fmt.Sprintf("%s.%s.%s", c.Name, seedDCName, cc.externalURL)
	if c.Address.ExternalName != externalName {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Address.ExternalName = externalName
		})
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof("Set external name for cluster %s to '%s'", c.Name, c.Address.ExternalName)
	}

	// Always lookup IP address, c case it changes (IP's on AWS LB's change)
	ipSet, err := getExternalIPv4Set(c.Address.ExternalName)
	if err != nil {
		return nil, err
	}

	ips := ipSet.List()
	if len(ips) == 0 {
		return nil, fmt.Errorf("failed to get IP for cluster. No IP's found. Check if DNS has been configured correctly")
	}
	ip := ips[0]

	if c.Address.IP != ip {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Address.IP = ip
		})
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof("Set IP for cluster %s to '%s'", c.Name, c.Address.IP)
	}

	// We fetch the Apiserver service as its a NodePort and we'll take the first NodePort (so far we only have one)
	s, err := cc.serviceLister.Services(c.Status.NamespaceName).Get(resources.ApiserverExternalServiceName)
	if err != nil {
		// A not found is fine. This happens on the first sync of a cluster, as this function gets called before the service gets created.
		if errors.IsNotFound(err) {
			glog.V(6).Infof("Skipping URL setting for cluster %s as no external apiserver service exists yet. Will retry later", c.Name)
			return c, nil
		}
		return nil, err
	}

	url := fmt.Sprintf("https://%s:%d", c.Address.ExternalName, int(s.Spec.Ports[0].NodePort))
	if c.Address.URL != url {
		c, err = cc.updateCluster(c.Name, func(c *kubermaticv1.Cluster) {
			c.Address.URL = url
		})
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof("Set URL for cluster %s to '%s'", c.Name, c.Address.URL)
	}

	return c, nil
}
