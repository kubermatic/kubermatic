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
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/api/compute/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

// cleanUnusedRoutes finds and remove unused gcp routes.
func cleanUnusedRoutes(ctx context.Context, cluster *kubermaticv1.Cluster, log *zap.SugaredLogger, svc *compute.Service, projectID string) error {
	// filter routes on:
	// - name prefix for routes created by gcp cloud provider
	// - default tag for routes created by gcp cloud provider
	// - GCP network
	filterStr := fmt.Sprintf("(name eq \"%s\")(description eq \"%s\")(network eq \".*%s.*\")",
		k8sNodeRoutePrefixRegexp,
		k8sNodeRouteTag,
		networkURL(projectID, cluster.Spec.Cloud.GCP.Network))

	routesList, err := svc.Routes.List(projectID).Filter(filterStr).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to list GCP routes: %w", err)
	}
	logger := log.With("cluster", cluster.Name)
	for _, route := range routesList.Items {
		if isMyRoute, err := isClusterRoute(cluster, route); err != nil || !isMyRoute {
			if err != nil {
				logger.Warnf("failed to determine route [%s] CIDR", route.Name)
			}
			continue
		}
		if isNextHopNotFound(route) {
			logger.Infof("deleting unused GCP route [%s]", route.Name)
			if _, err := svc.Routes.Delete(projectID, route.Name).Context(ctx).Do(); err != nil && !isHTTPError(err, http.StatusNotFound) {
				return fmt.Errorf("failed to delete GCP route %s: %w", route.Name, err)
			}
		}
	}
	return nil
}

// networkURL checks the network name and retuen the network URL based on it.
func networkURL(project, network string) string {
	url, err := url.Parse(network)
	if err == nil && url.Host != "" {
		return network
	}
	return computeAPIEndpoint + strings.Join([]string{"projects", project, "global", "networks", path.Base(network)}, "/")
}

// isClusterRoute checks if the route CIDR is part of the Cluster CIDR.
func isClusterRoute(cluster *kubermaticv1.Cluster, route *compute.Route) (bool, error) {
	_, routeCIDR, err := net.ParseCIDR(route.DestRange)
	if err != nil {
		return false, fmt.Errorf("failed to parse route destination CIDR: %w", err)
	}
	// Not responsible if this route's CIDR is not within our clusterCIDR
	lastIP := make([]byte, len(routeCIDR.IP))
	for i := range lastIP {
		lastIP[i] = routeCIDR.IP[i] | ^routeCIDR.Mask[i]
	}

	// check across all cluster cidrs
	for _, clusterCIDRStr := range cluster.Spec.ClusterNetwork.Pods.CIDRBlocks {
		_, clusterCIDR, err := net.ParseCIDR(clusterCIDRStr)
		if err != nil {
			return false, fmt.Errorf("failed to parse cluster CIDR: %w", err)
		}
		if clusterCIDR.Contains(routeCIDR.IP) || clusterCIDR.Contains(lastIP) {
			return true, nil
		}
	}
	return false, nil
}

// isNextHopNotFound checks if the route has a NEXT_HOP_INSTANCE_NOT_FOUND warning.
func isNextHopNotFound(route *compute.Route) bool {
	for _, w := range route.Warnings {
		if w.Code == "NEXT_HOP_INSTANCE_NOT_FOUND" {
			return true
		}
	}
	return false
}
