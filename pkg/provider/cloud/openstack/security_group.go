/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package openstack

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gophercloud/gophercloud"
	ossecuritygroups "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/groups"
	ossecuritygrouprules "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/net"
)

func getSecurityGroups(netClient *gophercloud.ServiceClient, opts ossecuritygroups.ListOpts) ([]ossecuritygroups.SecGroup, error) {
	page, err := ossecuritygroups.List(netClient, opts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list security groups: %w", err)
	}
	secGroups, err := ossecuritygroups.ExtractGroups(page)
	if err != nil {
		return nil, fmt.Errorf("failed to extract security groups: %w", err)
	}
	return secGroups, nil
}

func validateSecurityGroupExists(netClient *gophercloud.ServiceClient, securityGroup string) error {
	results, err := getSecurityGroups(netClient, ossecuritygroups.ListOpts{Name: securityGroup})
	if err != nil {
		return fmt.Errorf("failed to get security group: %w", err)
	}
	if len(results) == 0 {
		return fmt.Errorf("specified security group %s not found", securityGroup)
	}
	return nil
}

func deleteSecurityGroup(netClient *gophercloud.ServiceClient, sgName string) error {
	results, err := getSecurityGroups(netClient, ossecuritygroups.ListOpts{Name: sgName})
	if err != nil {
		return fmt.Errorf("failed to get security group: %w", err)
	}

	for _, sg := range results {
		res := ossecuritygroups.Delete(netClient, sg.ID)
		if res.Err != nil {
			return res.Err
		}
		if err := res.ExtractErr(); err != nil {
			return err
		}
	}

	return nil
}

type securityGroupSpec struct {
	name           string
	ipv4Rules      bool
	ipv6Rules      bool
	nodePortsCIDRs kubermaticv1.NetworkRanges
	lowPort        int
	highPort       int
}

func ensureSecurityGroup(netClient *gophercloud.ServiceClient, req securityGroupSpec) (string, error) {
	secGroups, err := getSecurityGroups(netClient, ossecuritygroups.ListOpts{Name: req.name})
	if err != nil {
		return "", fmt.Errorf("failed to get security groups: %w", err)
	}

	var securityGroupID string
	switch len(secGroups) {
	case 0:
		gres := ossecuritygroups.Create(netClient, ossecuritygroups.CreateOpts{
			Name:        req.name,
			Description: "Contains security rules for the Kubernetes worker nodes",
		})
		if gres.Err != nil {
			return "", gres.Err
		}
		g, err := gres.Extract()
		if err != nil {
			return "", err
		}
		securityGroupID = g.ID
	case 1:
		securityGroupID = secGroups[0].ID
	default:
		return "", fmt.Errorf("there are already %d security groups with name %q, dont know which one to use",
			len(secGroups), req.name)
	}

	var rules []ossecuritygrouprules.CreateOpts

	if req.ipv4Rules {
		rules = append(rules, []ossecuritygrouprules.CreateOpts{
			{
				// Allows ipv4 traffic within this group
				Direction:     ossecuritygrouprules.DirIngress,
				EtherType:     ossecuritygrouprules.EtherType4,
				SecGroupID:    securityGroupID,
				RemoteGroupID: securityGroupID,
			},
			{
				// Allows ssh from external
				Direction:    ossecuritygrouprules.DirIngress,
				EtherType:    ossecuritygrouprules.EtherType4,
				SecGroupID:   securityGroupID,
				PortRangeMin: provider.DefaultSSHPort,
				PortRangeMax: provider.DefaultSSHPort,
				Protocol:     ossecuritygrouprules.ProtocolTCP,
			},
			{
				// Allows ICMP traffic
				Direction:  ossecuritygrouprules.DirIngress,
				EtherType:  ossecuritygrouprules.EtherType4,
				SecGroupID: securityGroupID,
				Protocol:   ossecuritygrouprules.ProtocolICMP,
			},
		}...)
	}

	if req.ipv6Rules {
		rules = append(rules, []ossecuritygrouprules.CreateOpts{
			{
				// Allows ipv6 traffic within this group
				Direction:     ossecuritygrouprules.DirIngress,
				EtherType:     ossecuritygrouprules.EtherType6,
				SecGroupID:    securityGroupID,
				RemoteGroupID: securityGroupID,
			},
			{
				// Allows ssh from external
				Direction:    ossecuritygrouprules.DirIngress,
				EtherType:    ossecuritygrouprules.EtherType6,
				SecGroupID:   securityGroupID,
				PortRangeMin: provider.DefaultSSHPort,
				PortRangeMax: provider.DefaultSSHPort,
				Protocol:     ossecuritygrouprules.ProtocolTCP,
			},
			{
				// Allows ICMPv6 traffic
				Direction:  ossecuritygrouprules.DirIngress,
				EtherType:  ossecuritygrouprules.EtherType6,
				SecGroupID: securityGroupID,
				Protocol:   ossecuritygrouprules.ProtocolIPv6ICMP,
			},
		}...)
	}

	for _, cidr := range req.nodePortsCIDRs.CIDRBlocks {
		tcp := ossecuritygrouprules.CreateOpts{
			// Allows TCP traffic to nodePorts from external
			Direction:      ossecuritygrouprules.DirIngress,
			SecGroupID:     securityGroupID,
			PortRangeMin:   req.lowPort,
			PortRangeMax:   req.highPort,
			Protocol:       ossecuritygrouprules.ProtocolTCP,
			RemoteIPPrefix: cidr,
		}
		udp := ossecuritygrouprules.CreateOpts{
			// Allows UDP traffic to nodePorts from external
			Direction:      ossecuritygrouprules.DirIngress,
			SecGroupID:     securityGroupID,
			PortRangeMin:   req.lowPort,
			PortRangeMax:   req.highPort,
			Protocol:       ossecuritygrouprules.ProtocolUDP,
			RemoteIPPrefix: cidr,
		}
		if net.IsIPv4CIDRString(cidr) {
			tcp.EtherType = ossecuritygrouprules.EtherType4
			udp.EtherType = ossecuritygrouprules.EtherType4
		} else {
			tcp.EtherType = ossecuritygrouprules.EtherType6
			udp.EtherType = ossecuritygrouprules.EtherType6
		}
		rules = append(rules, tcp, udp)
	}

	for _, opts := range rules {
		if err := ensureSecurityGroupRule(netClient, opts); err != nil {
			return "", err
		}
	}

	return req.name, nil
}

func ensureSecurityGroupRule(netClient *gophercloud.ServiceClient, opts ossecuritygrouprules.CreateOpts) error {
	res := ossecuritygrouprules.Create(netClient, opts)
	if res.Err != nil {
		var unexpected gophercloud.ErrUnexpectedResponseCode
		if errors.As(res.Err, &unexpected) && unexpected.Actual == http.StatusConflict {
			// already exists
			return nil
		}

		if errors.As(res.Err, &gophercloud.ErrDefault400{}) && opts.Protocol == ossecuritygrouprules.ProtocolIPv6ICMP {
			// workaround for old versions of Openstack with different protocol name,
			// from before https://review.opendev.org/#/c/252155/
			opts.Protocol = "icmpv6"

			return ensureSecurityGroupRule(netClient, opts)
		}

		return res.Err
	}

	_, err := res.Extract()

	return err
}
