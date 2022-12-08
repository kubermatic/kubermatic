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
	"time"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
)

const (
	openStackFlavor                    = "m1.small"
	openStackFloatingIPPool            = "ext-net"
	openStackInstanceReadyCheckPeriod  = 5 * time.Second
	openStackInstanceReadyCheckTimeout = 2 * time.Minute
)

type openStackScenario struct {
	baseScenario
}

func (s *openStackScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.OpenStack.KKPDatacenter,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Domain:         secrets.OpenStack.Domain,
				Project:        secrets.OpenStack.Project,
				ProjectID:      secrets.OpenStack.ProjectID,
				Username:       secrets.OpenStack.Username,
				Password:       secrets.OpenStack.Password,
				FloatingIPPool: openStackFloatingIPPool,
			},
		},
		Version: s.version,
	}
}

func (s *openStackScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	cloudProviderSpec := provider.NewOpenstackConfig().
		WithFlavor(openStackFlavor).
		WithInstanceReadyCheckPeriod(openStackInstanceReadyCheckPeriod).
		WithInstanceReadyCheckTimeout(openStackInstanceReadyCheckTimeout).
		Build()

	md, err := s.createMachineDeployment(cluster, num, cloudProviderSpec)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
