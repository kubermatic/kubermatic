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
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/api/compute/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func reconcileFirewallRules(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, svc *compute.Service, projectID string) error {
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

	// allow traffic within the same cluster
	if !kuberneteshelper.HasFinalizer(cluster, firewallSelfCleanupFinalizer) {
		_, err := firewallService.Insert(projectID, &compute.Firewall{
			Name:    selfRuleName,
			Network: cluster.Spec.Cloud.GCP.Network,
			Allowed: []*compute.FirewallAllowed{
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
			},
			TargetTags: []string{tag},
			SourceTags: []string{tag},
		}).Do()
		// we ignore a Google API "already exists" error
		if err != nil && !isHTTPError(err, http.StatusConflict) {
			return fmt.Errorf("failed to create firewall rule %s: %w", selfRuleName, err)
		}

		cluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
			kuberneteshelper.AddFinalizer(cluster, firewallSelfCleanupFinalizer)
		})
		if err != nil {
			return fmt.Errorf("failed to add %s finalizer: %w", firewallSelfCleanupFinalizer, err)
		}
	}

	_, err := firewallService.Insert(projectID, &compute.Firewall{
		Name:    icmpRuleName,
		Network: cluster.Spec.Cloud.GCP.Network,
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "icmp",
			},
		},
		TargetTags: []string{tag},
	}).Do()
	// we ignore a Google API "already exists" error
	if err != nil && !isHTTPError(err, http.StatusConflict) {
		return fmt.Errorf("failed to create firewall rule %s: %w", icmpRuleName, err)
	}

	newCluster, err := update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(cluster, firewallICMPCleanupFinalizer)
	})
	if err != nil {
		return fmt.Errorf("failed to add %s finalizer: %w", firewallICMPCleanupFinalizer, err)
	}
	*cluster = *newCluster

	// open nodePorts for TCP and UDP
	_, err = firewallService.Insert(projectID, &compute.Firewall{
		Name:    nodePortRuleName,
		Network: cluster.Spec.Cloud.GCP.Network,
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      []string{fmt.Sprintf("%d-%d", nodePortRangeLow, nodePortRangeHigh)},
			},
			{
				IPProtocol: "udp",
				Ports:      []string{fmt.Sprintf("%d-%d", nodePortRangeLow, nodePortRangeHigh)},
			},
		},
		TargetTags:   []string{tag},
		SourceRanges: []string{nodePortsAllowedIPRange},
	}).Do()
	// we ignore a Google API "already exists" error
	if err != nil && !isHTTPError(err, http.StatusConflict) {
		return fmt.Errorf("failed to create firewall rule %s: %w", nodePortRuleName, err)
	}

	newCluster, err = update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		kuberneteshelper.AddFinalizer(cluster, firewallNodePortCleanupFinalizer)
	})
	if err != nil {
		return fmt.Errorf("failed to add %s finalizer: %w", firewallNodePortCleanupFinalizer, err)
	}
	*cluster = *newCluster

	return err
}

func deleteFirewallRules(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater, log *zap.SugaredLogger, svc *compute.Service, projectID string) (*kubermaticv1.Cluster, error) {
	firewallService := compute.NewFirewallsService(svc)

	selfRuleName := fmt.Sprintf("firewall-%s-self", cluster.Name)
	icmpRuleName := fmt.Sprintf("firewall-%s-icmp", cluster.Name)
	nodePortRuleName := fmt.Sprintf("firewall-%s-nodeport", cluster.Name)

	if kuberneteshelper.HasFinalizer(cluster, firewallSelfCleanupFinalizer) {
		_, err := firewallService.Delete(projectID, selfRuleName).Do()
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
		_, err := firewallService.Delete(projectID, icmpRuleName).Do()
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
		_, err := firewallService.Delete(projectID, nodePortRuleName).Do()
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
		err := cleanUnusedRoutes(cluster, log, svc, projectID)
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
