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
	"strings"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	vspheretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

const (
	vSphereDatacenter = "vsphere-ger"
)

// GetVSphereScenarios returns a matrix of (version x operating system).
func GetVSphereScenarios(scenarioOptions []string, versions []*semver.Semver) []Scenario {
	var (
		customFolder     bool
		datastoreCluster bool
	)

	for _, opt := range scenarioOptions {
		switch opt {
		case "custom-folder":
			customFolder = true
		case "datastore-cluster":
			datastoreCluster = true
		}
	}

	var scenarios []Scenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
			customFolder:     customFolder,
			datastoreCluster: datastoreCluster,
		})
		// CentOS
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
			customFolder:     customFolder,
			datastoreCluster: datastoreCluster,
		})
	}

	return scenarios
}

type vSphereScenario struct {
	version          *semver.Semver
	osSpec           apimodels.OperatingSystemSpec
	customFolder     bool
	datastoreCluster bool
}

func (s *vSphereScenario) Name() string {
	return fmt.Sprintf("vsphere-%s-%s", getOSNameFromSpec(s.osSpec), strings.ReplaceAll(s.version.String(), ".", "-"))
}

func (s *vSphereScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	spec := &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: vSphereDatacenter,
					Vsphere: &apimodels.VSphereCloudSpec{
						Username:  secrets.VSphere.Username,
						Password:  secrets.VSphere.Password,
						Datastore: secrets.VSphere.Datastore,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}

	if s.customFolder {
		spec.Cluster.Spec.Cloud.Vsphere.Folder = "/dc-1/vm/Kubermatic-dev/custom_folder_test"
	}

	if s.datastoreCluster {
		spec.Cluster.Spec.Cloud.Vsphere.DatastoreCluster = "dsc-1"
		spec.Cluster.Spec.Cloud.Vsphere.Datastore = ""
	}

	return spec
}

func (s *vSphereScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	spec := &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: vSphereDatacenter,
			VSphere: &kubermaticv1.VSphereCloudSpec{
				Username:  secrets.VSphere.Username,
				Password:  secrets.VSphere.Password,
				Datastore: secrets.VSphere.Datastore,
			},
		},
		Version: *s.version,
	}

	if s.customFolder {
		spec.Cloud.VSphere.Folder = "/dc-1/vm/Kubermatic-dev/custom_folder_test"
	}

	if s.datastoreCluster {
		spec.Cloud.VSphere.DatastoreCluster = "dsc-1"
		spec.Cloud.VSphere.Datastore = ""
	}

	return spec
}

func (s *vSphereScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets, datacenter *kubermaticv1.Datacenter) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.osSpec)
	replicas := int32(num)
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Vsphere: &apimodels.VSphereNodeSpec{
							Template: datacenter.Spec.VSphere.Templates[osName],
							CPUs:     2,
							Memory:   4096,
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

func (s *vSphereScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.Datacenter) ([]clusterv1alpha1.MachineDeployment, error) {
	// See alibaba provider for more info on this.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	os := getOSNameFromSpec(s.osSpec)
	template := datacenter.Spec.VSphere.Templates[os]

	md, err := createMachineDeployment(num, s.version, os, s.osSpec, providerconfig.CloudProviderVsphere, vspheretypes.RawConfig{
		TemplateVMName: providerconfig.ConfigVarString{Value: template},
		CPUs:           2,
		MemoryMB:       4096,
	})
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *vSphereScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
