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

package helper

import (
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func GetServerAddressFromKubeconfig(kubeconfig *clientcmdapi.Config) (string, error) {
	if len(kubeconfig.Clusters) != 1 {
		return "", fmt.Errorf("kubeconfig does not contain exactly one cluster, can not extract server address")
	}
	// Clusters is a map so we have to use range here
	for _, clusterConfig := range kubeconfig.Clusters {
		return strings.Replace(clusterConfig.Server, "https://", "", -1), nil
	}

	return "", fmt.Errorf("no server address found")

}

func GetCACert(kubeconfig *clientcmdapi.Config) (string, error) {
	if len(kubeconfig.Clusters) != 1 {
		return "", fmt.Errorf("kubeconfig does not contain exactly one cluster, can not extract server address")
	}
	// Clusters is a map so we have to use range here
	for _, clusterConfig := range kubeconfig.Clusters {
		return string(clusterConfig.CertificateAuthorityData), nil
	}

	return "", fmt.Errorf("no CACert found")
}

// StringifyKubeconfig marshals a kubeconfig to its text form
func StringifyKubeconfig(kubeconfig *clientcmdapi.Config) (string, error) {
	kubeconfigBytes, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error writing kubeconfig: %v", err)
	}

	return string(kubeconfigBytes), nil
}

// LoadKernelModules returns a script which is responsible for loading all required kernel modules
// The nf_conntrack_ipv4 module get removed in newer kernel versions
func LoadKernelModulesScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

modprobe ip_vs
modprobe ip_vs_rr
modprobe ip_vs_wrr
modprobe ip_vs_sh

if modinfo nf_conntrack_ipv4 &> /dev/null; then
  modprobe nf_conntrack_ipv4
else
  modprobe nf_conntrack
fi
`
}

// KernelSettings returns the list of kernel settings required for a kubernetes worker node
// inotify changes according to https://github.com/kubernetes/kubernetes/issues/10421 - better than letting the kubelet die
func KernelSettings() string {
	return `net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
kernel.panic_on_oops = 1
kernel.panic = 10
net.ipv4.ip_forward = 1
vm.overcommit_memory = 1
fs.inotify.max_user_watches = 1048576
`
}

// JournalDConfig returns the journal config preferable on every node
func JournalDConfig() string {
	// JournaldMaxUse defines the maximum space that journalD logs can occupy.
	// https://www.freedesktop.org/software/systemd/man/journald.conf.html#SystemMaxUse=
	return `[Journal]
SystemMaxUse=5G
`
}

type dockerConfig struct {
	ExecOpts           []string          `json:"exec-opts,omitempty"`
	StorageDriver      string            `json:"storage-driver,omitempty"`
	StorageOpts        []string          `json:"storage-opts,omitempty"`
	LogDriver          string            `json:"log-driver,omitempty"`
	LogOpts            map[string]string `json:"log-opts,omitempty"`
	InsecureRegistries []string          `json:"insecure-registries,omitempty"`
	RegistryMirrors    []string          `json:"registry-mirrors,omitempty"`
}

// DockerConfig returns the docker daemon.json.
func DockerConfig(insecureRegistries, registryMirrors []string) (string, error) {
	cfg := dockerConfig{
		ExecOpts:           []string{"native.cgroupdriver=systemd"},
		StorageDriver:      "overlay2",
		LogDriver:          "json-file",
		LogOpts:            map[string]string{"max-size": "100m"},
		InsecureRegistries: insecureRegistries,
		RegistryMirrors:    registryMirrors,
	}
	if insecureRegistries == nil {
		cfg.InsecureRegistries = []string{}
	}
	if registryMirrors == nil {
		cfg.RegistryMirrors = []string{}
	}

	b, err := json.Marshal(cfg)
	return string(b), err
}

func ProxyEnvironment(proxy, noProxy string) string {
	return fmt.Sprintf(`HTTP_PROXY=%s
http_proxy=%s
HTTPS_PROXY=%s
https_proxy=%s
NO_PROXY=%s
no_proxy=%s`, proxy, proxy, proxy, proxy, noProxy, noProxy)
}
