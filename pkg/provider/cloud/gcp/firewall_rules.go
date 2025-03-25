/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package gcp

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/api/compute/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

const (
	selfRuleNamePattern         = "firewall-%s-self"
	icmpRuleNamePattern         = "firewall-%s-icmp"
	icmpIPv6RuleNamePattern     = "firewall-%s-icmp-ipv6"
	nodePortRuleNamePattern     = "firewall-%s-nodeport"
	nodePortIPv6RuleNamePattern = "firewall-%s-nodeport-ipv6"

	ipv6ICMPProtoNumber = "58" // IANA-assigned Internet Protocol Number for IPv6-ICMP
)

func reconcileFirewallRules(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, svc *compute.Service, projectID string) error {
	// Retrieve nodePort range from cluster
	nodePortRangeLow, nodePortRangeHigh := resources.NewTemplateDataBuilder().
		WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
		WithCluster(cluster).
		Build().
		NodePorts()

	firewallService := compute.NewFirewallsService(svc)
	tag := fmt.Sprintf("kubernetes-cluster-%s", cluster.Name)
	selfRuleName := fmt.Sprintf(selfRuleNamePattern, cluster.Name)
	icmpRuleName := fmt.Sprintf(icmpRuleNamePattern, cluster.Name)
	icmpIPv6RuleName := fmt.Sprintf(icmpIPv6RuleNamePattern, cluster.Name)
	nodePortRuleName := fmt.Sprintf(nodePortRuleNamePattern, cluster.Name)
	nodePortIPv6RuleName := fmt.Sprintf(nodePortIPv6RuleNamePattern, cluster.Name)

	ipv4Rules := cluster.IsIPv4Only() || cluster.IsDualStack()
	ipv6Rules := cluster.IsIPv6Only() || cluster.IsDualStack()

	// Allow all common IP protocols from within the cluster.
	var allowedProtocols = []*compute.FirewallAllowed{
		{
			IPProtocol: "tcp",
		},
		{
			IPProtocol: "udp",
		},
		{
			IPProtocol: "esp",
		},
		{
			IPProtocol: "ah",
		},
		{
			IPProtocol: "sctp",
		},
		{
			IPProtocol: "ipip",
		},
	}
	if ipv4Rules {
		allowedProtocols = append(allowedProtocols, &compute.FirewallAllowed{IPProtocol: "icmp"})
	}
	if ipv6Rules {
		allowedProtocols = append(allowedProtocols, &compute.FirewallAllowed{IPProtocol: ipv6ICMPProtoNumber})
	}
	err := createOrPatchFirewall(ctx, firewallService, projectID, selfRuleName, tag, tag, allowedProtocols, nil, update, cluster, firewallSelfCleanupFinalizer)
	if err != nil {
		return err
	}

	// Allow ICMP from anywhere.
	// Note that mixture of IPv4 and IPv6 in the same rule is not allowed by GCP,
	// so we need to create a separate rule for each IP family.
	if ipv4Rules {
		err = createOrPatchFirewall(ctx, firewallService, projectID, icmpRuleName, tag, "",
			[]*compute.FirewallAllowed{{IPProtocol: "icmp"}}, []string{resources.IPv4MatchAnyCIDR}, update, cluster, firewallICMPCleanupFinalizer)
		if err != nil {
			return err
		}
	}
	if ipv6Rules {
		err = createOrPatchFirewall(ctx, firewallService, projectID, icmpIPv6RuleName, tag, "",
			[]*compute.FirewallAllowed{{IPProtocol: ipv6ICMPProtoNumber}}, []string{resources.IPv6MatchAnyCIDR}, update, cluster, firewallICMPCleanupFinalizer)
		if err != nil {
			return err
		}
	}

	// Allow all ports from the NodePort range.
	// Note that mixture of IPv4 and IPv6 in the same rule is not allowed by GCP,
	// so we need to create a separate rule for each IP family.
	allowedProtocols = []*compute.FirewallAllowed{
		{
			IPProtocol: "tcp",
			Ports:      []string{fmt.Sprintf("%d-%d", nodePortRangeLow, nodePortRangeHigh)},
		},
		{
			IPProtocol: "udp",
			Ports:      []string{fmt.Sprintf("%d-%d", nodePortRangeLow, nodePortRangeHigh)},
		},
	}
	nodePortsAllowedIPRanges := resources.GetNodePortsAllowedIPRanges(cluster, cluster.Spec.Cloud.GCP.NodePortsAllowedIPRanges, cluster.Spec.Cloud.GCP.NodePortsAllowedIPRange, nil)
	nodePortsIPv4CIDRs := nodePortsAllowedIPRanges.GetIPv4CIDRs()
	nodePortsIPv6CIDRs := nodePortsAllowedIPRanges.GetIPv6CIDRs()
	if len(nodePortsIPv4CIDRs) > 0 {
		err = createOrPatchFirewall(ctx, firewallService, projectID, nodePortRuleName, tag, "",
			allowedProtocols, nodePortsIPv4CIDRs, update, cluster, firewallNodePortCleanupFinalizer)
		if err != nil {
			return err
		}
	}
	if len(nodePortsIPv6CIDRs) > 0 {
		err = createOrPatchFirewall(ctx, firewallService, projectID, nodePortIPv6RuleName, tag, "",
			allowedProtocols, nodePortsIPv6CIDRs, update, cluster, firewallNodePortCleanupFinalizer)
		if err != nil {
			return err
		}
	}

	return nil
}

func createOrPatchFirewall(ctx context.Context,
	firewallService *compute.FirewallsService,
	projectID string,
	firewallName string,
	targetTag string,
	sourceTag string,
	protocols []*compute.FirewallAllowed,
	allowedIPRanges []string,
	update provider.ClusterUpdater,
	cluster *kubermaticv1.Cluster,
	finalizer string) error {
	firewall := &compute.Firewall{
		Name:         firewallName,
		Network:      cluster.Spec.Cloud.GCP.Network,
		TargetTags:   []string{targetTag},
		Allowed:      protocols,
		SourceRanges: allowedIPRanges,
	}
	if sourceTag != "" {
		firewall.SourceTags = []string{sourceTag}
	}

	existingFirewall, err := firewallService.Get(projectID, firewallName).Context(ctx).Do()
	switch {
	case isHTTPError(err, http.StatusNotFound):
		if _, err = firewallService.Insert(projectID, firewall).Context(ctx).Do(); err != nil {
			return fmt.Errorf("failed to create new firewall %s for cluster %s, %w", firewallName, cluster.Name, err)
		}
	case err == nil:
		if !reflect.DeepEqual(existingFirewall.Allowed, firewall.Allowed) ||
			!strings.HasSuffix(existingFirewall.Network, firewall.Network) ||
			!reflect.DeepEqual(existingFirewall.TargetTags, firewall.TargetTags) ||
			!reflect.DeepEqual(existingFirewall.SourceTags, firewall.SourceTags) ||
			!reflect.DeepEqual(existingFirewall.SourceRanges, firewall.SourceRanges) {
			_, err = firewallService.Patch(projectID, firewallName, firewall).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("failed to patch firewall %s for cluster %s, %w", firewallName, cluster.Name, err)
			}
		}
	default:
		return fmt.Errorf("failed to get firewall %s for cluster %s, %w", firewallName, cluster.Name, err)
	}

	var toPatch = true
	for _, f := range cluster.Finalizers {
		if f == finalizer {
			toPatch = false
		}
	}
	if toPatch {
		newCluster, err := update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, finalizer)
		})
		if err != nil {
			return fmt.Errorf("failed to add %s finalizer: %w", finalizer, err)
		}
		*cluster = *newCluster
	}

	return nil
}

func deleteFirewallRules(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, log *zap.SugaredLogger, svc *compute.Service, projectID string) (*kubermaticv1.Cluster, error) {
	firewallService := compute.NewFirewallsService(svc)

	selfRuleName := fmt.Sprintf(selfRuleNamePattern, cluster.Name)
	icmpRuleName := fmt.Sprintf(icmpRuleNamePattern, cluster.Name)
	icmpIPv6RuleName := fmt.Sprintf(icmpIPv6RuleNamePattern, cluster.Name)
	nodePortRuleName := fmt.Sprintf(nodePortRuleNamePattern, cluster.Name)
	nodePortIPv6RuleName := fmt.Sprintf(nodePortIPv6RuleNamePattern, cluster.Name)

	ipv4Rules := cluster.IsIPv4Only() || cluster.IsDualStack()
	ipv6Rules := cluster.IsIPv6Only() || cluster.IsDualStack()

	if kuberneteshelper.HasFinalizer(cluster, firewallSelfCleanupFinalizer) {
		_, err := firewallService.Delete(projectID, selfRuleName).Context(ctx).Do()
		if err != nil && !isHTTPError(err, http.StatusNotFound) {
			return nil, fmt.Errorf("failed to delete firewall rule %s: %w", selfRuleName, err)
		}

		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, firewallSelfCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %w", firewallSelfCleanupFinalizer, err)
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, firewallICMPCleanupFinalizer) {
		if ipv4Rules {
			_, err := firewallService.Delete(projectID, icmpRuleName).Context(ctx).Do()
			if err != nil && !isHTTPError(err, http.StatusNotFound) {
				return nil, fmt.Errorf("failed to delete firewall rule %s: %w", icmpRuleName, err)
			}
		}
		if ipv6Rules {
			_, err := firewallService.Delete(projectID, icmpIPv6RuleName).Context(ctx).Do()
			if err != nil && !isHTTPError(err, http.StatusNotFound) {
				return nil, fmt.Errorf("failed to delete firewall rule %s: %w", icmpRuleName, err)
			}
		}

		var err error
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, firewallICMPCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %w", firewallICMPCleanupFinalizer, err)
		}
	}

	// remove the nodeport firewall rule
	if kuberneteshelper.HasFinalizer(cluster, firewallNodePortCleanupFinalizer) {
		if ipv4Rules {
			_, err := firewallService.Delete(projectID, nodePortRuleName).Context(ctx).Do()
			if err != nil && !isHTTPError(err, http.StatusNotFound) {
				return nil, fmt.Errorf("failed to delete firewall rule %s: %w", nodePortRuleName, err)
			}
		}
		if ipv6Rules {
			_, err := firewallService.Delete(projectID, nodePortIPv6RuleName).Context(ctx).Do()
			if err != nil && !isHTTPError(err, http.StatusNotFound) {
				return nil, fmt.Errorf("failed to delete firewall rule %s: %w", nodePortRuleName, err)
			}
		}

		var err error
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, firewallNodePortCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %w", firewallNodePortCleanupFinalizer, err)
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, routesCleanupFinalizer) {
		err := cleanUnusedRoutes(ctx, cluster, log, svc, projectID)
		if err != nil {
			return nil, err
		}
		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, routesCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %w", routesCleanupFinalizer, err)
		}
	}

	return cluster, nil
}
