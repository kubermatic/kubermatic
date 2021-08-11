/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package vsphere

import (
	"bytes"
	"fmt"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/kubermatic/machine-controller/pkg/ini"
)

const (
	cloudConfigCSITpl = `[Global]
user              = {{ .Global.User | iniEscape }}
password          = {{ .Global.Password | iniEscape }}
port              = {{ .Global.VCenterPort | iniEscape }}
insecure-flag     = {{ .Global.InsecureFlag }}
working-dir       = {{ .Global.WorkingDir | iniEscape }}
datacenter        = {{ .Global.Datacenter | iniEscape }}
datastore         = {{ .Global.DefaultDatastore | iniEscape }}
server            = {{ .Global.VCenterIP | iniEscape }}
cluster-id        = {{ .Global.ClusterID | iniEscape }}

[Disk]
scsicontrollertype = {{ .Disk.SCSIControllerType | iniEscape }}

[Workspace]
server            = {{ .Workspace.VCenterIP | iniEscape }}
datacenter        = {{ .Workspace.Datacenter | iniEscape }}
folder            = {{ .Workspace.Folder | iniEscape }}
default-datastore = {{ .Workspace.DefaultDatastore | iniEscape }}
resourcepool-path = {{ .Workspace.ResourcePoolPath | iniEscape }}

{{ range $name, $vc := .VirtualCenter }}
[VirtualCenter {{ $name | iniEscape }}]
user = {{ $vc.User | iniEscape }}
password = {{ $vc.Password | iniEscape }}
port = {{ $vc.VCenterPort }}
datacenters = {{ $vc.Datacenters | iniEscape }}
{{ end }}
`
)

func CloudConfigCSIToString(c *types.CloudConfig) (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape

	tpl, err := template.New("cloud-config").Funcs(funcMap).Parse(cloudConfigCSITpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud config template: %v", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, c); err != nil {
		return "", fmt.Errorf("failed to execute cloud config template: %v", err)
	}

	return buf.String(), nil
}
