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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/util"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/machine/operatingsystem"
	"k8c.io/kubermatic/v2/pkg/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

type Scenario interface {
	// these are all satisfied by the baseScenario

	CloudProvider() kubermaticv1.ProviderType
	OperatingSystem() providerconfig.OperatingSystem
	ContainerRuntime() string
	DualstackEnabled() bool
	Version() semver.Semver
	Datacenter() *kubermaticv1.Datacenter
	Name() string
	Log(log *zap.SugaredLogger) *zap.SugaredLogger
	NamedLog(log *zap.SugaredLogger) *zap.SugaredLogger
	IsValid(opts *types.Options, log *zap.SugaredLogger) bool

	Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec
	MachineDeployments(ctx context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error)

	SetDualstackEnabled(bool)
}

type baseScenario struct {
	cloudProvider    kubermaticv1.ProviderType
	operatingSystem  providerconfig.OperatingSystem
	version          semver.Semver
	containerRuntime string
	dualstackEnabled bool
	datacenter       *kubermaticv1.Datacenter
}

func (s *baseScenario) CloudProvider() kubermaticv1.ProviderType {
	return s.cloudProvider
}

func (s *baseScenario) OperatingSystem() providerconfig.OperatingSystem {
	return s.operatingSystem
}

func (s *baseScenario) Version() semver.Semver {
	return s.version
}

func (s *baseScenario) ContainerRuntime() string {
	return s.containerRuntime
}

func (s *baseScenario) DualstackEnabled() bool {
	return s.dualstackEnabled
}

func (s *baseScenario) SetDualstackEnabled(enabled bool) {
	s.dualstackEnabled = enabled
}

func (s *baseScenario) Datacenter() *kubermaticv1.Datacenter {
	return s.datacenter
}

func (s *baseScenario) Log(log *zap.SugaredLogger) *zap.SugaredLogger {
	return log.With(
		"provider", s.cloudProvider,
		"os", s.operatingSystem,
		"version", s.version.String(),
		"cri", s.containerRuntime,
	)
}

func (s *baseScenario) NamedLog(log *zap.SugaredLogger) *zap.SugaredLogger {
	return log.With("scenario", s.Name())
}

func (s *baseScenario) Name() string {
	return fmt.Sprintf("%s-%s-%s-%s", s.cloudProvider, s.operatingSystem, s.containerRuntime, s.version.String())
}

func (s *baseScenario) IsValid(opts *types.Options, log *zap.SugaredLogger) bool {
	return true
}

func (s *baseScenario) createMachineDeployment(cluster *kubermaticv1.Cluster, replicas int, cloudProviderSpec interface{}) (clusterv1alpha1.MachineDeployment, error) {
	replicas32 := int32(replicas)

	osSpec, err := operatingsystem.DefaultSpec(s.operatingSystem, s.cloudProvider)
	if err != nil {
		return clusterv1alpha1.MachineDeployment{}, err
	}

	networkConfig := &providerconfig.NetworkConfig{}
	if s.dualstackEnabled {
		networkConfig.IPFamily = util.IPFamilyIPv4IPv6
	}

	providerSpec, err := machine.NewBuilder().
		WithDatacenter(s.datacenter).
		WithCluster(cluster).
		WithOperatingSystemSpec(osSpec).
		WithCloudProviderSpec(cloudProviderSpec).
		WithNetworkConfig(networkConfig).
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
						Kubelet: s.version.String(),
					},
					ProviderSpec: *providerSpec,
				},
			},
		},
	}, nil
}
