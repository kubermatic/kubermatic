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

package vmwareclouddirector

import (
	"encoding/json"
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/vmwareclouddirector"
	"k8c.io/kubermatic/v2/pkg/resources"
)

type VCDConfig struct {
	Host         string `yaml:"host"`
	Organization string `yaml:"org"`
	VDC          string `yaml:"vdc"`
	VAppName     string `yaml:"vAppName"`
}

type CloudConfig struct {
	VCD       VCDConfig `yaml:"vcd"`
	ClusterID string    `yaml:"clusterid"`
}

func GetVMwareCloudDirectorCSIConfig(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) CloudConfig {
	vAppName := cluster.Spec.Cloud.VMwareCloudDirector.VApp
	if vAppName == "" {
		vAppName = fmt.Sprintf(vmwareclouddirector.ResourceNamePattern, cluster.Name)
	}

	// host shouldn't have the `/api` suffix.
	host := strings.TrimSuffix(dc.Spec.VMwareCloudDirector.URL, "/api")

	return CloudConfig{
		VCD: VCDConfig{
			Organization: credentials.VMwareCloudDirector.Organization,
			VDC:          credentials.VMwareCloudDirector.VDC,
			VAppName:     vAppName,
			Host:         host,
		},
		ClusterID: cluster.Name,
	}
}

// ToString renders the cloud configuration as string.
func (cc *CloudConfig) ToString() (string, error) {
	b, err := json.Marshal(cc)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return string(b), nil
}
