/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package vmwareclouddirector

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/vmwareclouddirector"
	"k8c.io/kubermatic/v2/pkg/resources"
)

// Sourced from: https://raw.githubusercontent.com/vmware/cloud-director-named-disk-csi-driver/1.6.0/manifests/vcloud-csi-config.yaml

type CloudConfig struct {
	VCD       VCDConfig `yaml:"vcd"`
	ClusterID string    `yaml:"clusterid"`
}

func ForCluster(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) CloudConfig {
	vAppName := cluster.Spec.Cloud.VMwareCloudDirector.VApp
	if vAppName == "" {
		vAppName = fmt.Sprintf(vmwareclouddirector.ResourceNamePattern, cluster.Name)
	}

	// host shouldn't have the `/api` suffix.
	host := strings.TrimSuffix(dc.Spec.VMwareCloudDirector.URL, "/api")

	return CloudConfig{
		VCD: VCDConfig{
			Host:         host,
			Organization: credentials.VMwareCloudDirector.Organization,
			VDC:          credentials.VMwareCloudDirector.VDC,
			VApp:         vAppName,
		},
		ClusterID: cluster.Name,
	}
}

func (c *CloudConfig) String() (string, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	return string(b), nil
}

type VCDConfig struct {
	Host         string `yaml:"host"`
	Organization string `yaml:"org"`
	VDC          string `yaml:"vdc"`
	VApp         string `yaml:"vAppName"`
}
