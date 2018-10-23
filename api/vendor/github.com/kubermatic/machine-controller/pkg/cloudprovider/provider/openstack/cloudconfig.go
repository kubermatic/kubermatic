package openstack

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
domain-name = {{ .Global.DomainName | iniEscape }}
region      = {{ .Global.Region | iniEscape }}

[LoadBalancer]
manage-security-groups = {{ .LoadBalancer.ManageSecurityGroups }}

[BlockStorage]
ignore-volume-az  = {{ .BlockStorage.IgnoreVolumeAZ }}
trust-device-path = {{ .BlockStorage.TrustDevicePath }}
bs-version        = {{ .BlockStorage.BSVersion | iniEscape }}
`
)

type LoadBalancerOpts struct {
	ManageSecurityGroups bool `gcfg:"manage-security-groups"`
}

type BlockStorageOpts struct {
	BSVersion       string `gcfg:"bs-version"`
	TrustDevicePath bool   `gcfg:"trust-device-path"`
	IgnoreVolumeAZ  bool   `gcfg:"ignore-volume-az"`
}

type GlobalOpts struct {
	AuthURL    string `gcfg:"auth-url"`
	Username   string
	Password   string
	TenantName string `gcfg:"tenant-name"`
	DomainName string `gcfg:"domain-name"`
	Region     string
}

// CloudConfig is used to read and store information from the cloud configuration file
type CloudConfig struct {
	Global       GlobalOpts
	LoadBalancer LoadBalancerOpts
	BlockStorage BlockStorageOpts
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
