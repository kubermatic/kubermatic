package aws

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/kubermatic/machine-controller/pkg/ini"

	"github.com/Masterminds/sprig"
)

const (
	cloudConfigTpl = `[global]
Zone={{ .Global.Zone | iniEscape }}
VPC={{ .Global.VPC | iniEscape }}
SubnetID={{ .Global.SubnetID | iniEscape }}
RouteTableID={{ .Global.RouteTableID | iniEscape }}
RoleARN={{ .Global.RoleARN | iniEscape }}
KubernetesClusterID={{ .Global.KubernetesClusterID | iniEscape }}
DisableSecurityGroupIngress={{ .Global.DisableSecurityGroupIngress }}
ElbSecurityGroup={{ .Global.ElbSecurityGroup | iniEscape }}
DisableStrictZoneCheck={{ .Global.DisableStrictZoneCheck }}
`
)

type CloudConfig struct {
	Global GlobalOpts
}

type GlobalOpts struct {
	Zone                        string
	VPC                         string
	SubnetID                    string
	RouteTableID                string
	RoleARN                     string
	KubernetesClusterTag        string
	KubernetesClusterID         string
	ElbSecurityGroup            string
	DisableSecurityGroupIngress bool
	DisableStrictZoneCheck      bool
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
