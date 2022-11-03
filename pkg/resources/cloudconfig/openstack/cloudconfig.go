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

package openstack

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/kubermatic/machine-controller/pkg/ini"
)

// use-octavia is enabled by default in CCM since v1.17.0, and disabled by
// default with the in-tree cloud provider.
// https://v1-18.docs.kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancer
const (
	cloudConfigTpl = `[Global]
auth-url    = {{ .Global.AuthURL | iniEscape }}
{{- if .Global.ApplicationCredentialID }}
application-credential-id     = {{ .Global.ApplicationCredentialID | iniEscape }}
application-credential-secret = {{ .Global.ApplicationCredentialSecret | iniEscape }}
{{- else }}
username    = {{ .Global.Username | iniEscape }}
password    = {{ .Global.Password | iniEscape }}
tenant-name = {{ .Global.ProjectName | iniEscape }}
tenant-id   = {{ .Global.ProjectID | iniEscape }}
domain-name = {{ .Global.DomainName | iniEscape }}
{{- end }}
region      = {{ .Global.Region | iniEscape }}

[LoadBalancer]
lb-version = {{ default "v2" .LoadBalancer.LBVersion | iniEscape }}
subnet-id = {{ .LoadBalancer.SubnetID | iniEscape }}
floating-network-id = {{ .LoadBalancer.FloatingNetworkID | iniEscape }}
lb-method = {{ default "ROUND_ROBIN" .LoadBalancer.LBMethod | iniEscape }}
lb-provider = {{ .LoadBalancer.LBProvider | iniEscape }}
{{- if .LoadBalancer.UseOctavia }}
use-octavia = {{ .LoadBalancer.UseOctavia | Bool }}
{{- end }}
{{- if .LoadBalancer.EnableIngressHostname }}
enable-ingress-hostname = {{ .LoadBalancer.EnableIngressHostname | Bool }}
{{- if .LoadBalancer.IngressHostnameSuffix }}
ingress-hostname-suffix = {{ .LoadBalancer.IngressHostnameSuffix | strPtr | iniEscape }}
{{- end }}
{{- end }}

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
	UseOctavia           *bool        `gcfg:"use-octavia"`

	EnableIngressHostname *bool   `gcfg:"enable-ingress-hostname"`
	IngressHostnameSuffix *string `gcfg:"ingress-hostname-suffix"`
}

type BlockStorageOpts struct {
	BSVersion             string `gcfg:"bs-version"`
	TrustDevicePath       bool   `gcfg:"trust-device-path"`
	IgnoreVolumeAZ        bool   `gcfg:"ignore-volume-az"`
	NodeVolumeAttachLimit uint   `gcfg:"node-volume-attach-limit"`
}

type GlobalOpts struct {
	AuthURL                     string `gcfg:"auth-url"`
	Username                    string
	Password                    string
	ApplicationCredentialID     string `gcfg:"application-credential-id"`
	ApplicationCredentialSecret string `gcfg:"application-credential-secret"`

	// project name formerly known as tenant name.
	// it serialized as tenant-name because openstack CCM reads only tenant-name. In CCM, internally project and tenant
	// are stored into tenant-name.
	ProjectName string `gcfg:"tenant-name"`

	// project id formerly known as tenant id.
	// serialized as tenant-id for same reason as ProjectName
	ProjectID  string `gcfg:"tenant-id"`
	DomainName string `gcfg:"domain-name"`
	Region     string
}

// CloudConfig is used to read and store information from the cloud configuration file.
type CloudConfig struct {
	Global       GlobalOpts
	LoadBalancer LoadBalancerOpts
	BlockStorage BlockStorageOpts
	Version      string
}

func CloudConfigToString(c *CloudConfig) (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape
	funcMap["Bool"] = func(b *bool) string { return strconv.FormatBool(*b) }
	funcMap["strPtr"] = func(s *string) string { return *s }

	tpl, err := template.New("cloud-config").Funcs(funcMap).Parse(cloudConfigTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud config template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := tpl.Execute(buf, c); err != nil {
		return "", fmt.Errorf("failed to execute cloud config template: %w", err)
	}

	return buf.String(), nil
}
