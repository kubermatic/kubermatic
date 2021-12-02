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

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"
)

func securityGroupName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

func reconcileSecurityGroup(ctx context.Context, clients *ClientSet, location string, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if cluster.Spec.Cloud.Azure.SecurityGroup == "" {
		cluster.Spec.Cloud.Azure.SecurityGroup = securityGroupName(cluster)
	}

	securityGroup, err := clients.SecurityGroups.Get(ctx, cluster.Spec.Cloud.Azure.ResourceGroup, cluster.Spec.Cloud.Azure.SecurityGroup, "")
	if err != nil && !isNotFound(securityGroup.Response) {
		return cluster, err
	}

	// if we found a security group, we can check for the ownership tag to determine
	// if the referenced security group is owned by this cluster and should be reconciled
	if !isNotFound(securityGroup.Response) && !hasOwnershipTag(securityGroup.Tags, cluster) {
		return cluster, nil
	}

	lowPort, highPort := kubermaticresources.NewTemplateDataBuilder().
		WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
		WithCluster(cluster).
		Build().
		NodePorts()

	target := targetSecurityGroup(cluster.Spec.Cloud, location, cluster.Name, lowPort, highPort)

	// check for attributes of the existing security group and return early if all values are already
	// as expected. Since there are a lot of pointers in the network.SecurityGroup struct, we need to
	// do a lot of "!= nil" checks so this does not panic.
	//
	// Attributes we check:
	// - defined security rules
	if securityGroup.SecurityGroupPropertiesFormat != nil && securityGroup.SecurityGroupPropertiesFormat.SecurityRules != nil &&
		compareSecurityRules(*securityGroup.SecurityGroupPropertiesFormat.SecurityRules, *target.SecurityGroupPropertiesFormat.SecurityRules) {
		return cluster, nil
	}

	if err := ensureSecurityGroup(ctx, clients, cluster.Spec.Cloud, target); err != nil {
		return cluster, err
	}

	return update(cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		updatedCluster.Spec.Cloud.Azure.SecurityGroup = cluster.Spec.Cloud.Azure.SecurityGroup
		kuberneteshelper.AddFinalizer(updatedCluster, FinalizerSecurityGroup)
	})
}

func targetSecurityGroup(cloud kubermaticv1.CloudSpec, location string, clusterName string, portRangeLow int, portRangeHigh int) *network.SecurityGroup {
	securityGroup := &network.SecurityGroup{
		Name:     to.StringPtr(cloud.Azure.SecurityGroup),
		Location: to.StringPtr(location),
		Tags: map[string]*string{
			clusterTagKey: to.StringPtr(clusterName),
		},
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			Subnets: &[]network.Subnet{
				{
					Name: to.StringPtr(cloud.Azure.SubnetName),
					ID:   to.StringPtr(assembleSubnetID(cloud)),
				},
			},
			// inbound
			SecurityRules: &[]network.SecurityRule{
				{
					Name: to.StringPtr("ssh_ingress"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 network.SecurityRuleProtocolTCP,
						SourceAddressPrefix:      to.StringPtr("*"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("22"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(100),
					},
				},
				{
					Name: to.StringPtr("inter_node_comm"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 "*",
						SourceAddressPrefix:      to.StringPtr("VirtualNetwork"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("VirtualNetwork"),
						DestinationPortRange:     to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(200),
					},
				},
				{
					Name: to.StringPtr("azure_load_balancer"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 "*",
						SourceAddressPrefix:      to.StringPtr("AzureLoadBalancer"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(300),
					},
				},
				{
					// Allow access to node ports from everywhere
					Name: to.StringPtr("node_ports_ingress"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionInbound,
						Protocol:                 network.SecurityRuleProtocolAsterisk,
						SourceAddressPrefix:      to.StringPtr("*"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr(fmt.Sprintf("%d-%d", portRangeLow, portRangeHigh)),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(400),
					},
				},
				// outbound
				{
					Name: to.StringPtr("outbound_allow_all"),
					SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
						Direction:                network.SecurityRuleDirectionOutbound,
						Protocol:                 "*",
						SourceAddressPrefix:      to.StringPtr("*"),
						SourcePortRange:          to.StringPtr("*"),
						DestinationAddressPrefix: to.StringPtr("*"),
						DestinationPortRange:     to.StringPtr("*"),
						Access:                   network.SecurityRuleAccessAllow,
						Priority:                 to.Int32Ptr(100),
					},
				},
			},
		},
	}

	updatedRules := append(*securityGroup.SecurityRules, tcpDenyAllRule(), udpDenyAllRule(), icmpAllowAllRule())
	securityGroup.SecurityRules = &updatedRules

	return securityGroup
}

// ensureSecurityGroup will create or update an Azure security group. The call is idempotent.
func ensureSecurityGroup(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec, sg *network.SecurityGroup) error {
	if sg == nil {
		return fmt.Errorf("invalid security group reference passed")
	}

	future, err := clients.SecurityGroups.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup, *sg)
	if err != nil {
		return fmt.Errorf("failed to create or update security group %q: %v", cloud.Azure.SecurityGroup, err)
	}

	if err := future.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return fmt.Errorf("failed to create or update security group %q: %v", cloud.Azure.SecurityGroup, err)
	}

	return nil
}

func deleteSecurityGroup(ctx context.Context, clients *ClientSet, cloud kubermaticv1.CloudSpec) error {
	future, err := clients.SecurityGroups.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.SecurityGroup)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletionRef(ctx, *clients.Autorest); err != nil {
		return err
	}

	return nil
}

func tcpDenyAllRule() network.SecurityRule {
	return network.SecurityRule{
		Name: to.StringPtr(denyAllTCPSecGroupRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Direction:                network.SecurityRuleDirectionInbound,
			Protocol:                 "TCP",
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationPortRange:     to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("*"),
			Access:                   network.SecurityRuleAccessDeny,
			Priority:                 to.Int32Ptr(800),
		},
	}
}

func udpDenyAllRule() network.SecurityRule {
	return network.SecurityRule{
		Name: to.StringPtr(denyAllUDPSecGroupRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Direction:                network.SecurityRuleDirectionInbound,
			Protocol:                 "UDP",
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationPortRange:     to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("*"),
			Access:                   network.SecurityRuleAccessDeny,
			Priority:                 to.Int32Ptr(801),
		},
	}
}

// Alright, so here's the deal. We need to allow ICMP, but on Azure it is not possible
// to specify ICMP as a protocol in a rule - only TCP or UDP.
// Therefore we're hacking around it by first blocking all incoming TCP and UDP
// and if these don't match, we have an "allow all" rule. Dirty, but the only way.
// See also: https://tinyurl.com/azure-allow-icmp
func icmpAllowAllRule() network.SecurityRule {
	return network.SecurityRule{
		Name: to.StringPtr(allowAllICMPSecGroupRuleName),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			Direction:                network.SecurityRuleDirectionInbound,
			Protocol:                 "*",
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("*"),
			DestinationPortRange:     to.StringPtr("*"),
			Access:                   network.SecurityRuleAccessAllow,
			Priority:                 to.Int32Ptr(900),
		},
	}
}

func compareSecurityRules(a []network.SecurityRule, b []network.SecurityRule) bool {
	if len(a) != len(b) {
		return false
	}

	for i, rule := range a {
		ruleB := b[i]
		if *rule.Name != *ruleB.Name || rule.SecurityRulePropertiesFormat.Direction != ruleB.SecurityRulePropertiesFormat.Direction ||
			rule.SecurityRulePropertiesFormat.Protocol != ruleB.SecurityRulePropertiesFormat.Protocol ||
			rule.SecurityRulePropertiesFormat.Access != ruleB.SecurityRulePropertiesFormat.Access ||
			!isEqualStringPtr(rule.SecurityRulePropertiesFormat.SourceAddressPrefix, ruleB.SecurityRulePropertiesFormat.SourceAddressPrefix) ||
			!isEqualStringPtr(rule.SecurityRulePropertiesFormat.SourcePortRange, ruleB.SecurityRulePropertiesFormat.SourcePortRange) ||
			!isEqualStringPtr(rule.SecurityRulePropertiesFormat.DestinationPortRange, ruleB.SecurityRulePropertiesFormat.DestinationPortRange) ||
			!isEqualStringPtr(rule.SecurityRulePropertiesFormat.DestinationAddressPrefix, ruleB.SecurityRulePropertiesFormat.DestinationAddressPrefix) ||
			!isEqualInt32Ptr(rule.SecurityRulePropertiesFormat.Priority, ruleB.SecurityRulePropertiesFormat.Priority) {
			return false
		}
	}

	return true
}

func isEqualStringPtr(s1 *string, s2 *string) bool {
	return s1 == nil && s2 == nil || (s1 != nil && s2 != nil && *s1 == *s2)
}

func isEqualInt32Ptr(s1 *int32, s2 *int32) bool {
	return s1 == nil && s2 == nil || (s1 != nil && s2 != nil && *s1 == *s2)
}
