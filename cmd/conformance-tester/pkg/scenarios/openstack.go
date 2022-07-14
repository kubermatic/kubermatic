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
	"errors"
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	openstacktypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

const (
	openStackFlavor                    = "m1.small"
	openStackDatacenter                = "syseleven-dbl1"
	openStackFloatingIPPool            = "ext-net"
	openStackInstanceReadyCheckPeriod  = "5s"
	openStackInstanceReadyCheckTimeout = "2m"
)

func openStackImage(os providerconfig.OperatingSystem) string {
	return fmt.Sprintf("kubermatic-e2e-%s", string(os))
}

// GetOpenStackScenarios returns a matrix of (version x operating system).
func GetOpenStackScenarios(versions []*semver.Semver) []Scenario {
	var scenarios []Scenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &openStackScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}

	return scenarios
}

type openStackScenario struct {
	version *semver.Semver
	osSpec  apimodels.OperatingSystemSpec
}

func (s *openStackScenario) Name() string {
	return fmt.Sprintf("openstack-%s-%s", getOSNameFromSpec(s.osSpec), s.version.String())
}

func (s *openStackScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: openStackDatacenter,
					Openstack: &apimodels.OpenstackCloudSpec{
						Domain:         secrets.OpenStack.Domain,
						Project:        secrets.OpenStack.Project,
						ProjectID:      secrets.OpenStack.ProjectID,
						Username:       secrets.OpenStack.Username,
						Password:       secrets.OpenStack.Password,
						FloatingIPPool: openStackFloatingIPPool,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *openStackScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: openStackDatacenter,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Domain:         secrets.OpenStack.Domain,
				Project:        secrets.OpenStack.Project,
				ProjectID:      secrets.OpenStack.ProjectID,
				Username:       secrets.OpenStack.Username,
				Password:       secrets.OpenStack.Password,
				FloatingIPPool: openStackFloatingIPPool,
			},
		},
		Version: *s.version,
	}
}

func (s *openStackScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets, _ *kubermaticv1.Datacenter) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.osSpec)
	flavor := openStackFlavor
	image := openStackImage(osName)
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Openstack: &apimodels.OpenstackNodeSpec{
							Flavor:                    &flavor,
							Image:                     &image,
							InstanceReadyCheckPeriod:  openStackInstanceReadyCheckPeriod,
							InstanceReadyCheckTimeout: openStackInstanceReadyCheckTimeout,
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.osSpec,
				},
			},
		},
	}, nil
}

func (s *openStackScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, _ *kubermaticv1.Datacenter) ([]clusterv1alpha1.MachineDeployment, error) {
	// See alibaba provider for more info on this.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	os := getOSNameFromSpec(s.osSpec)
	image := openStackImage(os)

	md, err := createMachineDeployment(num, s.version, os, s.osSpec, providerconfig.CloudProviderOpenstack, openstacktypes.RawConfig{
		Flavor:                    providerconfig.ConfigVarString{Value: openStackFlavor},
		Image:                     providerconfig.ConfigVarString{Value: image},
		InstanceReadyCheckPeriod:  providerconfig.ConfigVarString{Value: openStackInstanceReadyCheckPeriod},
		InstanceReadyCheckTimeout: providerconfig.ConfigVarString{Value: openStackInstanceReadyCheckTimeout},
	})
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *openStackScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
