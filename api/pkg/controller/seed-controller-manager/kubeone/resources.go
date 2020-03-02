package kubeone

import (
	"context"
	"fmt"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/grpcserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/kubeone"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	"github.com/pkg/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) getClusterTemplateData(ctx context.Context, client client.Client, cluster *kubermaticv1.Cluster) (*resources.TemplateData, error) {
	seed, err := r.seedGetter()

	datacenter, found := seed.Spec.Datacenters[cluster.Spec.Cloud.DatacenterName]
	if !found {
		return nil, fmt.Errorf("failed to get datacenter %s", cluster.Spec.Cloud.DatacenterName)
	}

	supportsFailureDomainZoneAntiAffinity, err := controllerutil.SupportsFailureDomainZoneAntiAffinity(ctx, r.Client)
	if err != nil {
		return nil, err
	}

	return resources.NewTemplateData(
		ctx,
		client,
		cluster,
		&datacenter,
		seed.DeepCopy(),
		r.overwriteRegistry,
		r.nodePortRange,
		r.nodeAccessNetwork,
		r.etcdDiskSize,
		r.monitoringScrapeAnnotationPrefix,
		r.inClusterPrometheusRulesFile,
		r.inClusterPrometheusDisableDefaultRules,
		r.inClusterPrometheusDisableDefaultScrapingConfigs,
		r.inClusterPrometheusScrapingConfigsFile,
		r.oidcCAFile,
		r.oidcIssuerURL,
		r.oidcIssuerClientID,
		r.nodeLocalDNSCacheEnabled,
		r.kubermaticImage,
		r.dnatControllerImage,
		supportsFailureDomainZoneAntiAffinity,
	), nil
}

func GetConfigMapCreators(data *resources.TemplateData) []reconciling.NamedConfigMapCreatorGetter {
	return []reconciling.NamedConfigMapCreatorGetter{
		kubeone.ConfigMapCreator(data),
	}
}

func (r *Reconciler) ensureConfigMaps(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetConfigMapCreators(data)

	objModifier := reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster))

	if err := reconciling.ReconcileConfigMaps(ctx, creators, cluster.Status.NamespaceName, r.Client, objModifier); err != nil {
		return errors.Wrapf(err, "failed to ensure that the ConfigMap exists")
	}

	return nil

}

func (r *Reconciler) ensureDeployments(ctx context.Context, cluster *kubermaticv1.Cluster, data *resources.TemplateData) error {
	creators := GetDeploymentCreators(data)

	return reconciling.ReconcileDeployments(ctx, creators, cluster.Status.NamespaceName, r.Client, reconciling.OwnerRefWrapper(resources.GetClusterRef(cluster)))
}

func GetDeploymentCreators(data *resources.TemplateData) []reconciling.NamedDeploymentCreatorGetter {
	return []reconciling.NamedDeploymentCreatorGetter{
		grpcserver.DeploymentCreator(data),
	}
}
