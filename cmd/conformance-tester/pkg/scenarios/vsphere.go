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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
)

type vSphereScenario struct {
	baseScenario

	customFolder     bool
	datastoreCluster bool
}

func (s *vSphereScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	spec := &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.VSphere.KKPDatacenter,
			VSphere: &kubermaticv1.VSphereCloudSpec{
				Username:  secrets.VSphere.Username,
				Password:  secrets.VSphere.Password,
				Datastore: s.datacenter.Spec.VSphere.DefaultDatastore,
			},
		},
		Version: s.version,
	}

	if s.customFolder {
		spec.Cloud.VSphere.Folder = fmt.Sprintf("%s/custom_folder_test", s.datacenter.Spec.VSphere.RootPath)
	}

	if s.datastoreCluster {
		spec.Cloud.VSphere.DatastoreCluster = "dsc-1"
		spec.Cloud.VSphere.Datastore = ""
	}

	return spec
}

func (s *vSphereScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	cloudProviderSpec := provider.NewVSphereConfig().
		WithCPUs(2).
		WithMemoryMB(4096).
		WithDiskSizeGB(10).
		Build()

	md, err := s.createMachineDeployment(cluster, num, cloudProviderSpec)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
