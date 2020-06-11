package openshift

import (
	"context"
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/nodeportproxy"

	corev1 "k8s.io/api/core/v1"
)

func (r *Reconciler) getOSData(ctx context.Context, cluster *kubermaticv1.Cluster) (*openshiftData, error) {
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
		cluster:                               cluster,
		client:                                r.Client,
		dc:                                    &datacenter,
		overwriteRegistry:                     r.overwriteRegistry,
		nodeAccessNetwork:                     r.nodeAccessNetwork,
		oidc:                                  r.oidc,
		etcdDiskSize:                          r.etcdDiskSize,
		kubermaticImage:                       r.kubermaticImage,
		dnatControllerImage:                   r.dnatControllerImage,
		supportsFailureDomainZoneAntiAffinity: supportsFailureDomainZoneAntiAffinity,
		externalURL:                           r.externalURL,
		seed:                                  seed.DeepCopy(),
	}, nil
}

func (r *Reconciler) reconcileResources(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	osData, err := r.getOSData(ctx, cluster)
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

	if osData.Cluster().Spec.ExposeStrategy == corev1.ServiceTypeLoadBalancer {
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
