/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package azure

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/utils/pointer"
)

func securityGroupName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileSecurityGroup(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Azure.SecurityGroup == "" {
		cluster.Spec.Cloud.Azure.SecurityGroup = securityGroupName(cluster)
	}

	securityGroup, err := clients.SecurityGroups.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup, cluster.Spec.Cloud.Azure.SecurityGroup, nil)
	if err != nil && !isNotFound(err) {
		return nil, err
	}

	// if we found a security group, we can check for the ownership tag to determine
	// if the referenced security group is owned by this cluster and should be reconciled
	if !isNotFound(err) && !hasOwnershipTag(securityGroup.Tags, cluster) {
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.SecurityGroup = cluster.Spec.Cloud.Azure.SecurityGroup
		})
	}

	lowPort, highPort := kubermaticresources.NewTemplateDataBuilder().
		WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
		WithCluster(cluster).
		Build().
		NodePorts()
	nodePortsAllowedIPRanges := kubermaticresources.GetNodePortsAllowedIPRanges(cluster, cluster.Spec.Cloud.Azure.NodePortsAllowedIPRanges, cluster.Spec.Cloud.Azure.NodePortsAllowedIPRange)

	target := targetSecurityGroup(cluster.Spec.Cloud, location, cluster.Name, lowPort, highPort, nodePortsAllowedIPRanges.GetIPv4CIDRs(), nodePortsAllowedIPRanges.GetIPv6CIDRs())

	// check for attributes of the existing security group and return early if all values are already
	// as expected. Since there are a lot of pointers in the network.SecurityGroup struct, we need to
	// do a lot of "!= nil" checks so this does not panic.
	//
	// Attributes we check:
	// - defined security rules
	if !(securityGroup.Properties != nil && securityGroup.Properties.SecurityRules != nil &&
		compareSecurityRules(securityGroup.Properties.SecurityRules, target.Properties.SecurityRules)) {
		if err := ensureSecurityGroup(ctx, clients, cluster.Spec.Cloud, target); err != nil {
			return cluster, err
		}
	}

	return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.SecurityGroup = cluster.Spec.Cloud.Azure.SecurityGroup
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerSecurityGroup)
	})
}

func targetSecurityGroup(cloud kubermaticv1.CloudSpec, location string, clusterName string, portRangeLow int, portRangeHigh int,
	nodePortsIPv4CIDRs []string, nodePortsIPv6CIDRs []string) *armnetwork.SecurityGroup {
	inbound := armnetwork.SecurityRuleDirectionInbound
	outbound := armnetwork.SecurityRuleDirectionOutbound
	all := armnetwork.SecurityRuleProtocolAsterisk
	allow := armnetwork.SecurityRuleAccessAllow
	tcp := armnetwork.SecurityRuleProtocolTCP

	securityGroup := &armnetwork.SecurityGroup{
		Name:     pointer.String(cloud.Azure.SecurityGroup),
		Location: pointer.String(location),
		Tags: map[string]*string{
			clusterTagKey: pointer.String(clusterName),
		},
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			Subnets: []*armnetwork.Subnet{
				{
					Name: pointer.String(cloud.Azure.SubnetName),
					ID:   pointer.String(assembleSubnetID(cloud)),
				},
			},
			// inbound
			SecurityRules: []*armnetwork.SecurityRule{
				{
					Name: pointer.String("ssh_ingress"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Direction:                &inbound,
						Protocol:                 &tcp,
						SourceAddressPrefix:      pointer.String("*"),
						SourcePortRange:          pointer.String("*"),
						DestinationAddressPrefix: pointer.String("*"),
						DestinationPortRange:     pointer.String("22"),
						Access:                   &allow,
						Priority:                 pointer.Int32(100),
					},
				},
				{
					Name: pointer.String("inter_node_comm"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Direction:                &inbound,
						Protocol:                 &all,
						SourceAddressPrefix:      pointer.String("VirtualNetwork"),
						SourcePortRange:          pointer.String("*"),
						DestinationAddressPrefix: pointer.String("VirtualNetwork"),
						DestinationPortRange:     pointer.String("*"),
						Access:                   &allow,
						Priority:                 pointer.Int32(200),
					},
				},
				{
					Name: pointer.String("azure_load_balancer"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Direction:                &inbound,
						Protocol:                 &all,
						SourceAddressPrefix:      pointer.String("AzureLoadBalancer"),
						SourcePortRange:          pointer.String("*"),
						DestinationAddressPrefix: pointer.String("*"),
						DestinationPortRange:     pointer.String("*"),
						Access:                   &allow,
						Priority:                 pointer.Int32(300),
					},
				},
				// outbound
				{
					Name: pointer.String("outbound_allow_all"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Direction:                &outbound,
						Protocol:                 &all,
						SourceAddressPrefix:      pointer.String("*"),
						SourcePortRange:          pointer.String("*"),
						DestinationAddressPrefix: pointer.String("*"),
						DestinationPortRange:     pointer.String("*"),
						Access:                   &allow,
						Priority:                 pointer.Int32(100),
					},
				},
			},
		},
	}

	if len(nodePortsIPv4CIDRs) > 0 {
		securityGroup.Properties.SecurityRules = append(securityGroup.Properties.SecurityRules, nodePortsAllowedIPRangesRule("node_ports_ingress", 400, portRangeLow, portRangeHigh, nodePortsIPv4CIDRs))
	}

	if len(nodePortsIPv6CIDRs) > 0 {
		securityGroup.Properties.SecurityRules = append(securityGroup.Properties.SecurityRules, nodePortsAllowedIPRangesRule("node_ports_ingress_ipv6", 401, portRangeLow, portRangeHigh, nodePortsIPv6CIDRs))
	}

	securityGroup.Properties.SecurityRules = append(securityGroup.Properties.SecurityRules, tcpDenyAllRule(), udpDenyAllRule(), icmpAllowAllRule())

	return securityGroup
}

// ensureSecurityGroup will create or update an Azure security group. The call is idempotent.
func ensureSecurityGroup(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec, sg *armnetwork.SecurityGroup) error {
	if sg == nil {
		return fmt.Errorf("invalid security group reference passed")
	}

	future, err := clients.SecurityGroups.BeginCreateOrUpdate(ctx, cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, *sg, nil)
	if err != nil {
		return fmt.Errorf("failed to create or update security group %q: %w", cloud.Azure.SecurityGroup, err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}

func deleteSecurityGroup(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	future, err := clients.SecurityGroups.BeginDelete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, nil)
	if err != nil {
		return ignoreNotFound(err)
	}

	_, err = future.PollUntilDone(ctx, &runtime.PollUntilDoneOptions{
		Frequency: 5 * time.Second,
	})

	return err
}

// nodePortsAllowedIPRangesRule returns a security rule to allow access to node ports from provided IP ranges.
func nodePortsAllowedIPRangesRule(name string, priority int32, portRangeLow int, portRangeHigh int, nodePortsAllowedIPRanges []string) *armnetwork.SecurityRule {
	inbound := armnetwork.SecurityRuleDirectionInbound
	all := armnetwork.SecurityRuleProtocolAsterisk
	allow := armnetwork.SecurityRuleAccessAllow

	prefixes := []*string{}
	for _, prefix := range nodePortsAllowedIPRanges {
		prefixes = append(prefixes, pointer.String(prefix))
	}

	rule := &armnetwork.SecurityRule{
		Name: pointer.String(name),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Direction:                &inbound,
			Protocol:                 &all,
			SourcePortRange:          pointer.String("*"),
			DestinationAddressPrefix: pointer.String("*"),
			DestinationPortRange:     pointer.String(fmt.Sprintf("%d-%d", portRangeLow, portRangeHigh)),
			Access:                   &allow,
			Priority:                 pointer.Int32(priority),
		},
	}

	if len(nodePortsAllowedIPRanges) == 1 {
		rule.Properties.SourceAddressPrefix = pointer.String(nodePortsAllowedIPRanges[0])
	} else {
		rule.Properties.SourceAddressPrefixes = prefixes
	}

	return rule
}

func tcpDenyAllRule() *armnetwork.SecurityRule {
	inbound := armnetwork.SecurityRuleDirectionInbound
	tcp := armnetwork.SecurityRuleProtocolTCP
	deny := armnetwork.SecurityRuleAccessDeny

	return &armnetwork.SecurityRule{
		Name: pointer.String(denyAllTCPSecGroupRuleName),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Direction:                &inbound,
			Protocol:                 &tcp,
			SourceAddressPrefix:      pointer.String("*"),
			SourcePortRange:          pointer.String("*"),
			DestinationPortRange:     pointer.String("*"),
			DestinationAddressPrefix: pointer.String("*"),
			Access:                   &deny,
			Priority:                 pointer.Int32(800),
		},
	}
}

func udpDenyAllRule() *armnetwork.SecurityRule {
	inbound := armnetwork.SecurityRuleDirectionInbound
	udp := armnetwork.SecurityRuleProtocolUDP
	deny := armnetwork.SecurityRuleAccessDeny

	return &armnetwork.SecurityRule{
		Name: pointer.String(denyAllUDPSecGroupRuleName),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Direction:                &inbound,
			Protocol:                 &udp,
			SourceAddressPrefix:      pointer.String("*"),
			SourcePortRange:          pointer.String("*"),
			DestinationPortRange:     pointer.String("*"),
			DestinationAddressPrefix: pointer.String("*"),
			Access:                   &deny,
			Priority:                 pointer.Int32(801),
		},
	}
}

// Alright, so here's the deal. We need to allow ICMP, but on Azure it is not possible
// to specify ICMP as a protocol in a rule - only TCP or UDP.
// Therefore we're hacking around it by first blocking all incoming TCP and UDP
// and if these don't match, we have an "allow all" rule. Dirty, but the only way.
// See also: https://tinyurl.com/azure-allow-icmp
func icmpAllowAllRule() *armnetwork.SecurityRule {
	inbound := armnetwork.SecurityRuleDirectionInbound
	all := armnetwork.SecurityRuleProtocolAsterisk
	allow := armnetwork.SecurityRuleAccessAllow

	return &armnetwork.SecurityRule{
		Name: pointer.String(allowAllICMPSecGroupRuleName),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Direction:                &inbound,
			Protocol:                 &all,
			SourceAddressPrefix:      pointer.String("*"),
			SourcePortRange:          pointer.String("*"),
			DestinationAddressPrefix: pointer.String("*"),
			DestinationPortRange:     pointer.String("*"),
			Access:                   &allow,
			Priority:                 pointer.Int32(900),
		},
	}
}

func compareSecurityRules(a []*armnetwork.SecurityRule, b []*armnetwork.SecurityRule) bool {
	if len(a) != len(b) {
		return false
	}

	for i, rule := range a {
		ruleB := b[i]
		if *rule.Name != *ruleB.Name || rule.Properties.Direction != ruleB.Properties.Direction ||
			rule.Properties.Protocol != ruleB.Properties.Protocol ||
			rule.Properties.Access != ruleB.Properties.Access ||
			!isEqualString(rule.Properties.SourceAddressPrefix, ruleB.Properties.SourceAddressPrefix) ||
			!isEqualString(rule.Properties.SourcePortRange, ruleB.Properties.SourcePortRange) ||
			!isEqualString(rule.Properties.DestinationPortRange, ruleB.Properties.DestinationPortRange) ||
			!isEqualString(rule.Properties.DestinationAddressPrefix, ruleB.Properties.DestinationAddressPrefix) ||
			!isEqualInt32(rule.Properties.Priority, ruleB.Properties.Priority) {
			return false
		}
	}

	return true
}

func isEqualString(s1 *string, s2 *string) bool {
	return s1 == nil && s2 == nil || (s1 != nil && s2 != nil && *s1 == *s2)
}

func isEqualInt32(s1 *int32, s2 *int32) bool {
	return s1 == nil && s2 == nil || (s1 != nil && s2 != nil && *s1 == *s2)
}
