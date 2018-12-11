package vsphere

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/kubermatic/machine-controller/pkg/ini"

	"github.com/Masterminds/sprig"
)

const (
	cloudConfigTpl = `[Global]
user              = {{ .Global.User | iniEscape }}
password          = {{ .Global.Password | iniEscape }}
port              = {{ .Global.VCenterPort | iniEscape }}
insecure-flag     = {{ .Global.InsecureFlag }}
working-dir       = {{ .Global.WorkingDir | iniEscape }}
datacenter        = {{ .Global.Datacenter | iniEscape }}
default-datastore = {{ .Global.DefaultDatastore | iniEscape }}
server            = {{ .Global.VCenterIP | iniEscape }}

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

type WorkspaceOpts struct {
	VCenterIP        string `gcfg:"server"`
	Datacenter       string `gcfg:"datacenter"`
	Folder           string `gcfg:"folder"`
	DefaultDatastore string `gcfg:"default-datastore"`
	ResourcePoolPath string `gcfg:"resourcepool-path"`
}

type DiskOpts struct {
	SCSIControllerType string `dcfg:"scsicontrollertype"`
}

type GlobalOpts struct {
	User             string `gcfg:"user"`
	Password         string `gcfg:"password"`
	InsecureFlag     bool   `gcfg:"insecure-flag"`
	VCenterPort      string `gcfg:"port"`
	WorkingDir       string `gcfg:"working-dir"`
	Datacenter       string `gcfg:"datacenter"`
	DefaultDatastore string `gcfg:"default-datastore"`
	VCenterIP        string `gcfg:"server"`
}

type VirtualCenterConfig struct {
	User        string `gcfg:"user"`
	Password    string `gcfg:"password"`
	VCenterPort string `gcfg:"port"`
	Datacenters string `gcfg:"datacenters"`
}

// CloudConfig is used to read and store information from the cloud configuration file
type CloudConfig struct {
	Global    GlobalOpts
	Disk      DiskOpts
	Workspace WorkspaceOpts

	VirtualCenter map[string]*VirtualCenterConfig
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
