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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func reconcileFirewallRules(ctx context.Context, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, svc *compute.Service, projectID string) error {
	// Retrieve nodePort range from cluster
	nodePortRangeLow, nodePortRangeHigh := resources.NewTemplateDataBuilder().
		WithNodePortRange(cluster.Spec.ComponentsOverride.Apiserver.NodePortRange).
		WithCluster(cluster).
		Build().
		NodePorts()

	nodePortsAllowedIPRange := cluster.Spec.Cloud.GCP.NodePortsAllowedIPRange
	if nodePortsAllowedIPRange == "" {
		nodePortsAllowedIPRange = "0.0.0.0/0"
	}

	firewallService := compute.NewFirewallsService(svc)
	tag := fmt.Sprintf("kubernetes-cluster-%s", cluster.Name)
	selfRuleName := fmt.Sprintf("firewall-%s-self", cluster.Name)
	icmpRuleName := fmt.Sprintf("firewall-%s-icmp", cluster.Name)
	nodePortRuleName := fmt.Sprintf("firewall-%s-nodeport", cluster.Name)

	//
	var allowedProtocols = []*compute.FirewallAllowed{
		{
			IPProtocol: "tcp",
		},
		{
			IPProtocol: "udp",
		},
		{
			IPProtocol: "icmp",
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
	err := createOrPatchFirewall(ctx, firewallService, projectID, selfRuleName, tag, tag, allowedProtocols, "", update, cluster, firewallSelfCleanupFinalizer)
	if err != nil {
		return err
	}

	allowedProtocols = []*compute.FirewallAllowed{
		{
			IPProtocol: "icmp",
		},
	}
	err = createOrPatchFirewall(ctx, firewallService, projectID, icmpRuleName, tag, "", allowedProtocols, "0.0.0.0/0", update, cluster, firewallICMPCleanupFinalizer)
	if err != nil {
		return err
	}

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
	err = createOrPatchFirewall(ctx, firewallService, projectID, nodePortRuleName, tag, "", allowedProtocols, nodePortsAllowedIPRange, update, cluster, firewallNodePortCleanupFinalizer)
	if err != nil {
		return err
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
	allowedIPRange string,
	update provider.ClusterUpdater,
	cluster *kubermaticv1.Cluster,
	finalizer string) error {
	firewall := &compute.Firewall{
		Name:       firewallName,
		Network:    cluster.Spec.Cloud.GCP.Network,
		TargetTags: []string{targetTag},
		Allowed:    protocols,
	}
	if sourceTag != "" {
		firewall.SourceTags = []string{sourceTag}
	}
	if allowedIPRange != "" {
		firewall.SourceRanges = []string{allowedIPRange}
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
		newCluster, err := update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
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

	selfRuleName := fmt.Sprintf("firewall-%s-self", cluster.Name)
	icmpRuleName := fmt.Sprintf("firewall-%s-icmp", cluster.Name)
	nodePortRuleName := fmt.Sprintf("firewall-%s-nodeport", cluster.Name)

	if kuberneteshelper.HasFinalizer(cluster, firewallSelfCleanupFinalizer) {
		_, err := firewallService.Delete(projectID, selfRuleName).Context(ctx).Do()
		// we ignore a Google API "not found" error
		if err != nil && !isHTTPError(err, http.StatusNotFound) {
			return nil, fmt.Errorf("failed to delete firewall rule %s: %w", selfRuleName, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, firewallSelfCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %w", firewallSelfCleanupFinalizer, err)
		}
	}

	if kuberneteshelper.HasFinalizer(cluster, firewallICMPCleanupFinalizer) {
		_, err := firewallService.Delete(projectID, icmpRuleName).Context(ctx).Do()
		// we ignore a Google API "not found" error
		if err != nil && !isHTTPError(err, http.StatusNotFound) {
			return nil, fmt.Errorf("failed to delete firewall rule %s: %w", icmpRuleName, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, firewallICMPCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %w", firewallICMPCleanupFinalizer, err)
		}
	}

	// remove the nodeport firewall rule
	if kuberneteshelper.HasFinalizer(cluster, firewallNodePortCleanupFinalizer) {
		_, err := firewallService.Delete(projectID, nodePortRuleName).Context(ctx).Do()
		// we ignore a Google API "not found" error
		if err != nil && !isHTTPError(err, http.StatusNotFound) {
			return nil, fmt.Errorf("failed to delete firewall rule %s: %w", nodePortRuleName, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
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
		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.RemoveFinalizer(cluster, routesCleanupFinalizer)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove %s finalizer: %w", routesCleanupFinalizer, err)
		}
	}

	return cluster, nil
}
