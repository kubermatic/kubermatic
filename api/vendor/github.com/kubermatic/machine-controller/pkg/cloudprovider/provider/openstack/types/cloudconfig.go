/*
Copyright 2019 The Machine Controller Authors.

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

package types

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/kubermatic/machine-controller/pkg/ini"

	"github.com/Masterminds/sprig"
)

const (
	cloudConfigTpl = `[Global]
auth-url    = {{ .Global.AuthURL | iniEscape }}
username    = {{ .Global.Username | iniEscape }}
password    = {{ .Global.Password | iniEscape }}
tenant-name = {{ .Global.TenantName | iniEscape }}
tenant-id   = {{ .Global.TenantID | iniEscape }}
domain-name = {{ .Global.DomainName | iniEscape }}
region      = {{ .Global.Region | iniEscape }}

[LoadBalancer]
lb-version = {{ default "v2" .LoadBalancer.LBVersion | iniEscape }}
subnet-id = {{ .LoadBalancer.SubnetID | iniEscape }}
floating-network-id = {{ .LoadBalancer.FloatingNetworkID | iniEscape }}
lb-method = {{ default "ROUND_ROBIN" .LoadBalancer.LBMethod | iniEscape }}
lb-provider = {{ .LoadBalancer.LBProvider | iniEscape }}

{{- if .LoadBalancer.CreateMonitor }}
create-monitor = {{ .LoadBalancer.CreateMonitor }}
monitor-delay = {{ .LoadBalancer.MonitorDelay }}
monitor-timeout = {{ .LoadBalancer.MonitorTimeout }}
monitor-max-retries = {{ .LoadBalancer.MonitorMaxRetries }}
{{- end}}
{{- if semverCompare "~1.9.10 || ~1.10.6 || ~1.11.1 || >=1.12.*" .Version }}
manage-security-groups = {{ .LoadBalancer.ManageSecurityGroups }}
{{- end }}

[BlockStorage]
{{- if semverCompare ">=1.9" .Version }}
ignore-volume-az  = {{ .BlockStorage.IgnoreVolumeAZ }}
{{- end }}
trust-device-path = {{ .BlockStorage.TrustDevicePath }}
bs-version        = {{ default "auto" .BlockStorage.BSVersion | iniEscape }}
{{- if .BlockStorage.NodeVolumeAttachLimit }}
node-volume-attach-limit = {{ .BlockStorage.NodeVolumeAttachLimit }}
{{- end }}
`
)

type LoadBalancerOpts struct {
	LBVersion            string       `gcfg:"lb-version"`
	SubnetID             string       `gcfg:"subnet-id"`
	FloatingNetworkID    string       `gcfg:"floating-network-id"`
	LBMethod             string       `gcfg:"lb-method"`
	LBProvider           string       `gcfg:"lb-provider"`
	CreateMonitor        bool         `gcfg:"create-monitor"`
	MonitorDelay         ini.Duration `gcfg:"monitor-delay"`
	MonitorTimeout       ini.Duration `gcfg:"monitor-timeout"`
	MonitorMaxRetries    uint         `gcfg:"monitor-max-retries"`
	ManageSecurityGroups bool         `gcfg:"manage-security-groups"`
}

type BlockStorageOpts struct {
	BSVersion             string `gcfg:"bs-version"`
	TrustDevicePath       bool   `gcfg:"trust-device-path"`
	IgnoreVolumeAZ        bool   `gcfg:"ignore-volume-az"`
	NodeVolumeAttachLimit uint   `gcfg:"node-volume-attach-limit"`
}

type GlobalOpts struct {
	AuthURL    string `gcfg:"auth-url"`
	Username   string
	Password   string
	TenantName string `gcfg:"tenant-name"`
	TenantID   string `gcfg:"tenant-id"`
	DomainName string `gcfg:"domain-name"`
	Region     string
}

// CloudConfig is used to read and store information from the cloud configuration file
type CloudConfig struct {
	Global       GlobalOpts
	LoadBalancer LoadBalancerOpts
	BlockStorage BlockStorageOpts
	Version      string
}

func CloudConfigToString(c *CloudConfig) (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape

	tpl, err := template.New("cloud-config").Funcs(funcMap).Parse(cloudConfigTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud config template: %v", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, c); err != nil {
		return "", fmt.Errorf("failed to execute cloud config template: %v", err)
	}

	return buf.String(), nil
}
