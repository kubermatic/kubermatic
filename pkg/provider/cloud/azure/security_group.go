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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticresources "k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/utils/ptr"
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

	// if we found a security group (no error), we can check for the ownership tag to determine
	// if the referenced security group is owned by this cluster and should be reconciled. We return
	// early in this condition if the security group is not owned by us.
	if err == nil && !hasOwnershipTag(securityGroup.Tags, cluster) {
		return update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
			updatedCluster.Spec.Cloud.Azure.SecurityGroup = cluster.Spec.Cloud.Azure.SecurityGroup
		})
	}

	lowPort, highPort := kubermaticresources.NewTemplateDataBuilder().
		WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
		WithCluster(cluster).
		Build().
		NodePorts()
	nodePortsAllowedIPRanges := kubermaticresources.GetNodePortsAllowedIPRanges(cluster, cluster.Spec.Cloud.Azure.NodePortsAllowedIPRanges, cluster.Spec.Cloud.Azure.NodePortsAllowedIPRange, nil)

	target := targetSecurityGroup(cluster.Spec.Cloud, location, cluster.Name, lowPort, highPort, nodePortsAllowedIPRanges.GetIPv4CIDRs(), nodePortsAllowedIPRanges.GetIPv6CIDRs())

	// check for attributes of the existing security group and return early if all values are already
	// as expected. Since there are a lot of pointers in the network.SecurityGroup struct, we need to
	// do a lot of "!= nil" checks so this does not panic.
	//
	// Attributes we check:
	// - defined security rules
	if securityGroup.Properties == nil || securityGroup.Properties.SecurityRules == nil || !compareSecurityRules(securityGroup.Properties.SecurityRules, target.Properties.SecurityRules) {
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
		Name:     ptr.To(cloud.Azure.SecurityGroup),
		Location: ptr.To(location),
		Tags: map[string]*string{
			clusterTagKey: ptr.To(clusterName),
		},
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			Subnets: []*armnetwork.Subnet{
				{
					Name: ptr.To(cloud.Azure.SubnetName),
					ID:   ptr.To(assembleSubnetID(cloud)),
				},
			},
			// inbound
			SecurityRules: []*armnetwork.SecurityRule{
				{
					Name: ptr.To("ssh_ingress"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Direction:                &inbound,
						Protocol:                 &tcp,
						SourceAddressPrefix:      ptr.To("*"),
						SourcePortRange:          ptr.To("*"),
						DestinationAddressPrefix: ptr.To("*"),
						DestinationPortRange:     ptr.To("22"),
						Access:                   &allow,
						Priority:                 ptr.To[int32](100),
					},
				},
				// outbound
				{
					Name: ptr.To("outbound_allow_all"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Direction:                &outbound,
						Protocol:                 &all,
						SourceAddressPrefix:      ptr.To("*"),
						SourcePortRange:          ptr.To("*"),
						DestinationAddressPrefix: ptr.To("*"),
						DestinationPortRange:     ptr.To("*"),
						Access:                   &allow,
						Priority:                 ptr.To[int32](100),
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

	securityGroup.Properties.SecurityRules = append(securityGroup.Properties.SecurityRules, icmpAllowAllRule())

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
		prefixes = append(prefixes, ptr.To(prefix))
	}

	rule := &armnetwork.SecurityRule{
		Name: ptr.To(name),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Direction:                &inbound,
			Protocol:                 &all,
			SourcePortRange:          ptr.To("*"),
			DestinationAddressPrefix: ptr.To("*"),
			DestinationPortRange:     ptr.To(fmt.Sprintf("%d-%d", portRangeLow, portRangeHigh)),
			Access:                   &allow,
			Priority:                 ptr.To[int32](priority),
		},
	}

	if len(nodePortsAllowedIPRanges) == 1 {
		rule.Properties.SourceAddressPrefix = ptr.To(nodePortsAllowedIPRanges[0])
	} else {
		rule.Properties.SourceAddressPrefixes = prefixes
	}

	return rule
}

func icmpAllowAllRule() *armnetwork.SecurityRule {
	inbound := armnetwork.SecurityRuleDirectionInbound
	icmp := armnetwork.SecurityRuleProtocolIcmp
	allow := armnetwork.SecurityRuleAccessAllow

	return &armnetwork.SecurityRule{
		Name: ptr.To(allowAllICMPSecGroupRuleName),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Direction:                &inbound,
			Protocol:                 &icmp,
			SourceAddressPrefix:      ptr.To("*"),
			SourcePortRange:          ptr.To("*"),
			DestinationAddressPrefix: ptr.To("*"),
			DestinationPortRange:     ptr.To("*"),
			Access:                   &allow,
			Priority:                 ptr.To[int32](800),
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
