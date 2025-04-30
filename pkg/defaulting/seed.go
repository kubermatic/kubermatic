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

package defaulting

import (
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

const (
	// DefaultBackupInterval defines the default interval used to create backups.
	DefaultBackupInterval = "20m"

	// DefaultMeteringStorageSize is the default size for the metering Prometheus PVC.
	DefaultMeteringStorageSize = "100Gi"
	// DefaultMeteringRetentionDays is the default number of days for which the metering Prometheus
	// should keep data.
	DefaultMeteringRetentionDays = 90
)

// DefaultSeed fills in missing values in the Seed's spec by copying them from the global
// defaults in the KubermaticConfiguration (in which some fields might already be deprecated,
// as we move configuration down into the Seeds). This function assumes that the config has
// already been defaulted.
func DefaultSeed(seed *kubermaticv1.Seed, config *kubermaticv1.KubermaticConfiguration, logger *zap.SugaredLogger) (*kubermaticv1.Seed, error) {
	logger = logger.With("seed", seed.Name)
	logger.Debug("Applying defaults to Seed")

	seedCopy := seed.DeepCopy()

	seedCopy.SetDefaults()

	if seedCopy.Spec.ExposeStrategy == "" {
		seedCopy.Spec.ExposeStrategy = config.Spec.ExposeStrategy
	}

	if seedCopy.Spec.Metering != nil {
		if seedCopy.Spec.Metering.RetentionDays <= 0 {
			seedCopy.Spec.Metering.RetentionDays = DefaultMeteringRetentionDays
		}

		if seedCopy.Spec.Metering.StorageSize == "" {
			seedCopy.Spec.Metering.StorageSize = DefaultMeteringStorageSize
		}
	}

	if err := defaultDockerRepo(&seedCopy.Spec.NodeportProxy.Envoy.DockerRepository, DefaultEnvoyDockerRepository, "nodeportProxy.envoy.dockerRepository", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultDockerRepo(&seedCopy.Spec.NodeportProxy.EnvoyManager.DockerRepository, DefaultNodeportProxyDockerRepository, "nodeportProxy.envoyManager.dockerRepository", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultDockerRepo(&seedCopy.Spec.NodeportProxy.Updater.DockerRepository, DefaultNodeportProxyDockerRepository, "nodeportProxy.updater.dockerRepository", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultResources(&seedCopy.Spec.NodeportProxy.Envoy.Resources, DefaultNodeportProxyEnvoyResources, "nodeportProxy.envoy.resources", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultResources(&seedCopy.Spec.NodeportProxy.EnvoyManager.Resources, DefaultNodeportProxyEnvoyManagerResources, "nodeportProxy.envoyManager.resources", logger); err != nil {
		return seedCopy, err
	}

	if err := defaultResources(&seedCopy.Spec.NodeportProxy.Updater.Resources, DefaultNodeportProxyUpdaterResources, "nodeportProxy.updater.resources", logger); err != nil {
		return seedCopy, err
	}

	if len(seedCopy.Spec.NodeportProxy.Envoy.LoadBalancerService.Annotations) == 0 {
		seedCopy.Spec.NodeportProxy.Envoy.LoadBalancerService.Annotations = DefaultNodeportProxyServiceAnnotations
		logger.Debugw("Defaulting field", "field", "nodeportProxy.envoy.loadBalancerService.annotations", "value", seedCopy.Spec.NodeportProxy.Annotations)
	}

	// apply settings from the KubermaticConfiguration to the Seed, in case they are not set there;
	// over time, we move pretty much all of this into the Seed, but this code copies the still existing,
	// deprecated fields over.
	settings := &seedCopy.Spec.DefaultComponentSettings

	if settings.Apiserver.Replicas == nil {
		settings.Apiserver.Replicas = config.Spec.UserCluster.APIServerReplicas
	}

	if settings.Apiserver.NodePortRange == "" {
		settings.Apiserver.NodePortRange = config.Spec.UserCluster.NodePortRange
	}

	if settings.Apiserver.EndpointReconcilingDisabled == nil && config.Spec.UserCluster.DisableAPIServerEndpointReconciling {
		settings.Apiserver.EndpointReconcilingDisabled = &config.Spec.UserCluster.DisableAPIServerEndpointReconciling
	}

	if settings.ControllerManager.Replicas == nil {
		settings.ControllerManager.Replicas = ptr.To[int32](DefaultControllerManagerReplicas)
	}

	if settings.Scheduler.Replicas == nil {
		settings.Scheduler.Replicas = ptr.To[int32](DefaultSchedulerReplicas)
	}

	if settings.Etcd.DiskSize == nil {
		etcdDiskSize, err := resource.ParseQuantity(config.Spec.UserCluster.EtcdVolumeSize)
		if err != nil {
			return seedCopy, fmt.Errorf("failed to parse spec.userCluster.etcdVolumeSize %q in KubermaticConfiguration: %w", config.Spec.UserCluster.EtcdVolumeSize, err)
		}
		settings.Etcd.DiskSize = &etcdDiskSize
	}

	if settings.Etcd.ClusterSize == nil {
		settings.Etcd.ClusterSize = ptr.To[int32](kubermaticv1.DefaultEtcdClusterSize)
	}

	return seedCopy, nil
}
