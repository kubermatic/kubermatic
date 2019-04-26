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

package convert

import (
	"encoding/json"
	"fmt"
	"net"

	ctconfig "github.com/coreos/container-linux-config-transpiler/config"

	"github.com/kubermatic/machine-controller/pkg/userdata/plugin"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func NewIgnition(p plugin.Provider) *Ignition {
	return &Ignition{p: p}
}

type Ignition struct {
	p plugin.Provider
}

func (j *Ignition) UserData(
	spec clusterv1alpha1.MachineSpec,
	kubeconfig *clientcmdapi.Config,
	cloudConfig, cloudProviderName string,
	clusterDNSIPs []net.IP,
	externalCloudProvider bool,
) (string, error) {
	before, err := j.p.UserData(spec, kubeconfig, cloudConfig, cloudProviderName, clusterDNSIPs, externalCloudProvider)
	if err != nil {
		return "", err
	}

	return ToIgnition(before)
}

func ToIgnition(s string) (string, error) {
	// Convert to ignition
	cfg, ast, report := ctconfig.Parse([]byte(s))
	if len(report.Entries) > 0 {
		return "", fmt.Errorf("failed to validate coreos cloud config: %s", report.String())
	}

	ignCfg, report := ctconfig.Convert(cfg, "", ast)
	if len(report.Entries) > 0 {
		return "", fmt.Errorf("failed to convert container linux config to ignition: %s", report.String())
	}

	out, err := json.Marshal(ignCfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ignition config: %v", err)
	}

	return string(out), nil
}
