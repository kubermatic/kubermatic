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
	azuretypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/azure/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

const (
	azureVMSize     = "Standard_F2"
	azureDatacenter = "azure-westeurope"
)

// GetAzureScenarios returns a matrix of (version x operating system).
func GetAzureScenarios(versions []*semver.Semver) []Scenario {
	var scenarios []Scenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &azureScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &azureScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type azureScenario struct {
	version *semver.Semver
	osSpec  apimodels.OperatingSystemSpec
}

func (s *azureScenario) Name() string {
	return fmt.Sprintf("azure-%s-%s", getOSNameFromSpec(s.osSpec), s.version.String())
}

func (s *azureScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: azureDatacenter,
					Azure: &apimodels.AzureCloudSpec{
						ClientID:       secrets.Azure.ClientID,
						ClientSecret:   secrets.Azure.ClientSecret,
						SubscriptionID: secrets.Azure.SubscriptionID,
						TenantID:       secrets.Azure.TenantID,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *azureScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: azureDatacenter,
			Azure: &kubermaticv1.AzureCloudSpec{
				ClientID:       secrets.Azure.ClientID,
				ClientSecret:   secrets.Azure.ClientSecret,
				SubscriptionID: secrets.Azure.SubscriptionID,
				TenantID:       secrets.Azure.TenantID,
			},
		},
		Version: *s.version,
	}
}

func (s *azureScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets, _ *kubermaticv1.Datacenter) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)
	size := azureVMSize

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Azure: &apimodels.AzureNodeSpec{
							Size: &size,
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

func (s *azureScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, _ *kubermaticv1.Datacenter) ([]clusterv1alpha1.MachineDeployment, error) {
	// See alibaba provider for more info on this.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	config := azuretypes.RawConfig{
		VMSize: providerconfig.ConfigVarString{Value: azureVMSize},
	}

	config.Tags = map[string]string{}
	config.Tags["kKubernetesCluster"] = cluster.Name
	config.Tags["system-cluster"] = cluster.Name

	projectID, ok := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if ok {
		config.Tags["system-project"] = projectID
	}

	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderAzure, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *azureScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
