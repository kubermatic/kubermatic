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

package scenarios

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/machine/operatingsystem"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/net"
	"k8c.io/machine-controller/sdk/providerconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

type Scenario interface {
	// these are all satisfied by the BaseScenario
	CloudProvider() kubermaticv1.ProviderType
	OperatingSystem() providerconfig.OperatingSystem
	ClusterVersion() semver.Semver
	Datacenter() *kubermaticv1.Datacenter
	Name() string
	Log(log *zap.SugaredLogger) *zap.SugaredLogger
	NamedLog(log *zap.SugaredLogger) *zap.SugaredLogger
	IsValid() error

	Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec
	MachineDeployments(ctx context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error)
}

type BaseScenario struct {
	cloudProvider    kubermaticv1.ProviderType
	operatingSystem  providerconfig.OperatingSystem
	clusterVersion   semver.Semver
	dualstackEnabled bool
	datacenter       *kubermaticv1.Datacenter
}

func (s *BaseScenario) WithCloudProvider(provider kubermaticv1.ProviderType) *BaseScenario {
	s.cloudProvider = provider
	return s
}

func (s *BaseScenario) WithOperatingSystem(os providerconfig.OperatingSystem) *BaseScenario {
	s.operatingSystem = os
	return s
}

func (s *BaseScenario) WithClusterVersion(version semver.Semver) *BaseScenario {
	s.clusterVersion = version
	return s
}

func (s *BaseScenario) WithDualstackEnabled(enabled bool) *BaseScenario {
	s.dualstackEnabled = enabled
	return s
}

func (s *BaseScenario) WithDatacenter(dc *kubermaticv1.Datacenter) *BaseScenario {
	s.datacenter = dc
	return s
}

func (s *BaseScenario) CloudProvider() kubermaticv1.ProviderType {
	return s.cloudProvider
}

func (s *BaseScenario) OperatingSystem() providerconfig.OperatingSystem {
	return s.operatingSystem
}

func (s *BaseScenario) ClusterVersion() semver.Semver {
	return s.clusterVersion
}

func (s *BaseScenario) Datacenter() *kubermaticv1.Datacenter {
	return s.datacenter
}

func (s *BaseScenario) Log(log *zap.SugaredLogger) *zap.SugaredLogger {
	return log.With(
		"provider", s.cloudProvider,
		"os", s.operatingSystem,
		"version", s.clusterVersion.String(),
	)
}

func (s *BaseScenario) NamedLog(log *zap.SugaredLogger) *zap.SugaredLogger {
	return log.With("scenario", s.Name())
}

func (s *BaseScenario) Name() string {
	return fmt.Sprintf("%s-%s-%s", s.cloudProvider, s.operatingSystem, s.clusterVersion.String())
}

func (s *BaseScenario) IsValid() error {
	return nil
}

func (s *BaseScenario) CreateMachineDeployment(cluster *kubermaticv1.Cluster, replicas int, cloudProviderSpec interface{}, sshPubKeys []string, secrets types.Secrets) (clusterv1alpha1.MachineDeployment, error) {
	replicas32 := int32(replicas)

	osSpec, err := s.GetOperatingSystemSpec(secrets)
	if err != nil {
		return clusterv1alpha1.MachineDeployment{}, err
	}

	networkConfig := &providerconfig.NetworkConfig{}
	if s.dualstackEnabled {
		networkConfig.IPFamily = net.IPFamilyIPv4IPv6
	}

	providerSpec, err := machine.NewBuilder().
		WithDatacenter(s.datacenter).
		WithCluster(cluster).
		WithOperatingSystemSpec(osSpec).
		WithCloudProviderSpec(cloudProviderSpec).
		WithNetworkConfig(networkConfig).
		AddSSHPublicKey(sshPubKeys...).
		BuildProviderSpec()
	if err != nil {
		return clusterv1alpha1.MachineDeployment{}, err
	}

	machineLabels := map[string]string{
		"machine": "md-" + utilrand.String(5),
	}

	return clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-" + utilrand.String(5),
			Namespace: metav1.NamespaceSystem,
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Replicas: &replicas32,
			Selector: metav1.LabelSelector{
				MatchLabels: machineLabels,
			},
			Template: clusterv1alpha1.MachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: machineLabels,
				},
				Spec: clusterv1alpha1.MachineSpec{
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: s.clusterVersion.String(),
					},
					ProviderSpec: *providerSpec,
				},
			},
		},
	}, nil
}

func (s *BaseScenario) GetOperatingSystemSpec(secrets types.Secrets) (interface{}, error) {
	// inject RHEL credentials when needed
	if s.operatingSystem == providerconfig.OperatingSystemRHEL {
		return operatingsystem.NewRHELSpecBuilder(s.cloudProvider).
			SetSubscriptionDetails(
				secrets.RHEL.SubscriptionUser,
				secrets.RHEL.SubscriptionPassword,
				secrets.RHEL.OfflineToken,
			).
			Build(), nil
	}

	return operatingsystem.DefaultSpec(s.operatingSystem, s.cloudProvider)
}
