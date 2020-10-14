/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package openshift

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

func (r *Reconciler) getOSData(ctx context.Context, cluster *kubermaticv1.Cluster, versions kubermatic.Versions) (*openshiftData, error) {
	seed, err := r.seedGetter()
	if err != nil {
		return nil, fmt.Errorf("failed to get seed: %v", err)
	}

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("couldn't find dc %s", cluster.Spec.Cloud.DatacenterName)
	}

	supportsFailureDomainZoneAntiAffinity, err := resources.SupportsFailureDomainZoneAntiAffinity(ctx, r.Client)
	if err != nil {
		return nil, err
	}

	return &openshiftData{
		cluster:             cluster,
		client:              r.Client,
		dc:                  &datacenter,
		overwriteRegistry:   r.overwriteRegistry,
		nodeAccessNetwork:   r.nodeAccessNetwork,
		oidc:                r.oidc,
		etcdDiskSize:        r.etcdDiskSize,
		kubermaticImage:     r.kubermaticImage,
		etcdLauncherImage:   r.etcdLauncherImage,
		dnatControllerImage: r.dnatControllerImage,
		externalURL:         r.externalURL,
		seed:                seed.DeepCopy(),
		versions:            versions,

		supportsFailureDomainZoneAntiAffinity: supportsFailureDomainZoneAntiAffinity,
	}, nil
}

func (r *Reconciler) reconcileResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	osData, err := r.getOSData(ctx, cluster, r.versions)
	if err != nil {
		return fmt.Errorf("failed to get osData: %v", err)
	}

	if err := r.services(ctx, osData); err != nil {
		return fmt.Errorf("failed to reconcile Services: %v", err)
	}

	if err := r.address(ctx, osData.Cluster(), osData.Seed()); err != nil {
		return fmt.Errorf("failed to reconcile the cluster address: %v", err)
	}

	if err := r.secrets(ctx, osData); err != nil {
		return fmt.Errorf("failed to reconcile Secrets: %v", err)
	}

	if err := r.ensureServiceAccounts(ctx, osData.Cluster()); err != nil {
		return err
	}

	if err := r.ensureRoles(ctx, osData.Cluster()); err != nil {
		return err
	}

	if err := r.ensureRoleBindings(ctx, osData.Cluster()); err != nil {
		return err
	}

	if err := r.statefulSets(ctx, osData); err != nil {
		return fmt.Errorf("failed to reconcile StatefulSets: %v", err)
	}

	if err := r.etcdBackupConfigs(ctx, osData.Cluster(), osData); err != nil {
		return err
	}

	// Wait until the cloud provider infra is ready before attempting
	// to render the cloud-config
	// TODO: Model resource deployment as a DAG so we don't need hacks
	// like this combined with tribal knowledge and "someone is noticing this
	// isn't working correctly"
	// https://github.com/kubermatic/kubermatic/issues/2948
	// We can just return and don't need a RequeueAfter because the cluster object
	// will get updated when the cloud infra health status changes
	if osData.Cluster().Status.ExtendedHealth.CloudProviderInfrastructure != kubermaticv1.HealthStatusUp {
		return nil
	}

	if err := r.configMaps(ctx, osData); err != nil {
		return fmt.Errorf("failed to reconcile ConfigMaps: %v", err)
	}

	if err := r.deployments(ctx, osData); err != nil {
		return fmt.Errorf("failed to reconcile Deployments: %v", err)
	}

	if osData.Cluster().Spec.ExposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
		if err := nodeportproxy.EnsureResources(ctx, r.Client, osData); err != nil {
			return fmt.Errorf("failed to ensure NodePortProxy resources: %v", err)
		}
	}

	if err := r.cronJobs(ctx, osData); err != nil {
		return fmt.Errorf("failed to reconcile CronJobs: %v", err)
	}

	if err := r.podDisruptionBudgets(ctx, osData); err != nil {
		return fmt.Errorf("failed to reconcile PodDisruptionBudgets: %v", err)
	}

	if err := r.verticalPodAutoscalers(ctx, osData); err != nil {
		return fmt.Errorf("failed to reconcile VerticalPodAutoscalers: %v", err)
	}

	return nil
}
