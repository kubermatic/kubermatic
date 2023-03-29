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
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/provider/cloud/vmwareclouddirector"
	"k8c.io/kubermatic/v3/pkg/resources"
)

// Sourced from: https://raw.githubusercontent.com/vmware/cloud-director-named-disk-csi-driver/1.2.0/manifests/vcloud-csi-config.yaml
const (
	cloudConfigCSITpl = `vcd:
  host: {{ .Host }}
  org: {{ .Organization}}
  vdc: {{ .VDC }}
  vAppName: {{ .VApp }}
clusterid: {{ .ClusterID }}`
)

type cloudConfig struct {
	Host         string
	Organization string
	VDC          string
	VApp         string
	ClusterID    string
}

// ToString renders the cloud configuration as string.
func (cc *cloudConfig) toString() (string, error) {
	tpl, err := template.New("cloud-config").Funcs(sprig.TxtFuncMap()).Parse(cloudConfigCSITpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud config template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, cc); err != nil {
		return "", fmt.Errorf("failed to execute cloud config template: %w", err)
	}

	return buf.String(), nil
}

func GetVMwareCloudDirectorCSIConfig(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) (string, error) {
	vAppName := cluster.Spec.Cloud.VMwareCloudDirector.VApp
	if vAppName == "" {
		vAppName = fmt.Sprintf(vmwareclouddirector.ResourceNamePattern, cluster.Name)
	}

	// host shouldn't have the `/api` suffix.
	host := strings.TrimSuffix(dc.Spec.VMwareCloudDirector.URL, "/api")

	cc := cloudConfig{
		Host:         host,
		Organization: credentials.VMwareCloudDirector.Organization,
		VDC:          credentials.VMwareCloudDirector.VDC,
		VApp:         vAppName,
		ClusterID:    cluster.Name,
	}

	return cc.toString()
}
