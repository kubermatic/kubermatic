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

//
// Google Cloud Provider for the Machine Controller
//

package types

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/kubermatic/machine-controller/pkg/ini"

	"github.com/Masterminds/sprig"
)

// cloudConfigTemplate renders the cloud-config in gcfg format. All
// fields are optional, that's why containing the ifs and the explicit newlines.
const cloudConfigTemplate = "[global]\n" +
	"project-id = {{ .Global.ProjectID | iniEscape }}\n" +
	"local-zone = {{ .Global.LocalZone | iniEscape }}\n" +
	"network-name = {{ .Global.NetworkName | iniEscape }}\n" +
	"subnetwork-name = {{ .Global.SubnetworkName | iniEscape }}\n" +
	"token-url = {{ .Global.TokenURL | iniEscape }}\n" +
	"multizone = {{ .Global.MultiZone }}\n" +
	"regional = {{ .Global.Regional }}\n" +
	"{{ range .Global.NodeTags }}node-tags = {{ . | iniEscape }}\n{{end}}"

// GlobalOpts contains the values of the global section of the cloud configuration.
type GlobalOpts struct {
	ProjectID        string
	LocalZone        string
	NetworkName      string
	SubnetworkName   string
	TokenURL         string
	MultiZone        bool
	Regional         bool
	NodeTags         []string
	RHSMOfflineToken string
}

// CloudConfig contains only the section global.
type CloudConfig struct {
	Global GlobalOpts
}

// AsString renders the cloud configuration as string.
func (cc *CloudConfig) AsString() (string, error) {
	funcMap := sprig.TxtFuncMap()
	funcMap["iniEscape"] = ini.Escape

	tmpl, err := template.New("cloud-config").Funcs(funcMap).Parse(cloudConfigTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud config template: %v", err)
	}

	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, cc); err != nil {
		return "", fmt.Errorf("failed to execute cloud config template: %v", err)
	}

	return buf.String(), nil
}
