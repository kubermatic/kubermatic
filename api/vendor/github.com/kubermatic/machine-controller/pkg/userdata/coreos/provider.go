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
// UserData plugin for CoreOS.
//

package coreos

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/semver"

	"github.com/kubermatic/machine-controller/pkg/apis/plugin"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	userdatahelper "github.com/kubermatic/machine-controller/pkg/userdata/helper"
)

// Provider is a pkg/userdata/plugin.Provider implementation.
type Provider struct{}

// UserData renders user-data template to string.
func (p Provider) UserData(req plugin.UserDataRequest) (string, error) {

	tmpl, err := template.New("user-data").Funcs(userdatahelper.TxtFuncMap()).Parse(userDataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse user-data template: %v", err)
	}

	kubeletVersion, err := semver.NewVersion(req.MachineSpec.Versions.Kubelet)
	if err != nil {
		return "", fmt.Errorf("invalid kubelet version: %v", err)
	}

	pconfig, err := providerconfigtypes.GetConfig(req.MachineSpec.ProviderSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get provider config: %v", err)
	}

	if pconfig.OverwriteCloudConfig != nil {
		req.CloudConfig = *pconfig.OverwriteCloudConfig
	}

	coreosConfig, err := LoadConfig(pconfig.OperatingSystemSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get coreos config from provider config: %v", err)
	}

	kubeconfigString, err := userdatahelper.StringifyKubeconfig(req.Kubeconfig)
	if err != nil {
		return "", err
	}

	kubernetesCACert, err := userdatahelper.GetCACert(req.Kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error extracting cacert: %v", err)
	}

	// We need to reconfigure rkt to allow insecure registries in case the hyperkube image comes from an insecure registry
	var insecureHyperkubeImage bool
	for _, registry := range req.InsecureRegistries {
		if strings.Contains(req.HyperkubeImage, registry) {
			insecureHyperkubeImage = true
		}
	}

	if coreosConfig.DisableAutoUpdate {
		coreosConfig.DisableLocksmithD = true
		coreosConfig.DisableUpdateEngine = true
	}

	data := struct {
		plugin.UserDataRequest
		ProviderSpec           *providerconfigtypes.Config
		CoreOSConfig           *Config
		Kubeconfig             string
		KubernetesCACert       string
		KubeletVersion         string
		InsecureHyperkubeImage bool
	}{
		UserDataRequest:        req,
		ProviderSpec:           pconfig,
		CoreOSConfig:           coreosConfig,
		Kubeconfig:             kubeconfigString,
		KubernetesCACert:       kubernetesCACert,
		KubeletVersion:         kubeletVersion.String(),
		InsecureHyperkubeImage: insecureHyperkubeImage,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}
	return userdatahelper.CleanupTemplateOutput(b.String())
}

// UserData template.
const userDataTemplate = `passwd:
{{- if ne (len .ProviderSpec.SSHPublicKeys) 0 }}
  users:
    - name: core
      ssh_authorized_keys:
        {{range .ProviderSpec.SSHPublicKeys}}- {{.}}
        {{end}}
{{- end }}

{{- if .ProviderSpec.Network }}
networkd:
  units:
    - name: static-nic.network
      contents: |
        [Match]
        # Because of difficulty predicting specific NIC names on different cloud providers,
        # we only support static addressing on VSphere. There should be a single NIC attached
        # that we will match by name prefix 'en' which denotes ethernet devices.
        Name=en*

        [Network]
        DHCP=no
        Address={{ .ProviderSpec.Network.CIDR }}
        Gateway={{ .ProviderSpec.Network.Gateway }}
        {{range .ProviderSpec.Network.DNS.Servers}}DNS={{.}}
        {{end}}
{{- end }}

systemd:
  units:
{{- if .CoreOSConfig.DisableUpdateEngine }}
    - name: update-engine.service
      mask: true
{{- end }}
{{- if .CoreOSConfig.DisableLocksmithD }}
    - name: locksmithd.service
      mask: true
{{- end }}
    - name: docker.service
      enabled: true

{{- if .HTTPProxy }}
    - name: update-engine.service
      dropins:
        - name: 50-proxy.conf
          contents: |
            [Service]
            Environment=ALL_PROXY={{ .HTTPProxy }}
{{- end }}

    - name: download-healthcheck-script.service
      enabled: true
      contents: |
        [Unit]
        Requires=network-online.target
        After=network-online.target
        [Service]
        Type=oneshot
        EnvironmentFile=-/etc/environment
        ExecStart=/opt/bin/download.sh
        [Install]
        WantedBy=multi-user.target

    - name: docker-healthcheck.service
      enabled: true
      dropins:
      - name: 40-docker.conf
        contents: |
          [Unit]
          Requires=download-healthcheck-script.service
          After=download-healthcheck-script.service
      contents: |
{{ containerRuntimeHealthCheckSystemdUnit | indent 10 }}

    - name: kubelet-healthcheck.service
      enabled: true
      dropins:
      - name: 40-docker.conf
        contents: |
          [Unit]
          Requires=download-healthcheck-script.service
          After=download-healthcheck-script.service
      contents: |
{{ kubeletHealthCheckSystemdUnit | indent 10 }}

    - name: kubelet.service
      enabled: true
      contents: |
        [Unit]
        Description=Kubernetes Kubelet
        Requires=docker.service
        After=docker.service
        [Service]
        TimeoutStartSec=5min
        CPUAccounting=true
        MemoryAccounting=true
        EnvironmentFile=-/etc/environment
{{- if .HTTPProxy }}
        Environment=KUBELET_IMAGE=docker://{{ .HyperkubeImage }}:v{{ .KubeletVersion }}
{{- else }}
        Environment=KUBELET_IMAGE=docker://k8s.gcr.io/hyperkube-amd64:v{{ .KubeletVersion }}
{{- end }}
        Environment="RKT_RUN_ARGS=--uuid-file-save=/var/cache/kubelet-pod.uuid \
          --inherit-env \
          --insecure-options=image{{if .InsecureHyperkubeImage }},http{{ end }} \
          --volume=resolv,kind=host,source=/etc/resolv.conf \
          --mount volume=resolv,target=/etc/resolv.conf \
          --volume cni-bin,kind=host,source=/opt/cni/bin \
          --mount volume=cni-bin,target=/opt/cni/bin \
          --volume cni-conf,kind=host,source=/etc/cni/net.d \
          --mount volume=cni-conf,target=/etc/cni/net.d \
          --volume etc-kubernetes,kind=host,source=/etc/kubernetes \
          --mount volume=etc-kubernetes,target=/etc/kubernetes \
          --volume var-log,kind=host,source=/var/log \
          --mount volume=var-log,target=/var/log \
          --volume var-lib-calico,kind=host,source=/var/lib/calico \
          --mount volume=var-lib-calico,target=/var/lib/calico"
        ExecStartPre=/bin/mkdir -p /var/lib/calico
        ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests
        ExecStartPre=/bin/mkdir -p /etc/cni/net.d
        ExecStartPre=/bin/mkdir -p /opt/cni/bin
        ExecStartPre=-/usr/bin/rkt rm --uuid-file=/var/cache/kubelet-pod.uuid
        ExecStartPre=-/bin/rm -rf /var/lib/rkt/cas/tmp/
        ExecStartPre=/bin/bash /opt/load-kernel-modules.sh
        ExecStart=/usr/lib/coreos/kubelet-wrapper \
{{ if semverCompare ">=1.17.0" .KubeletVersion }}{{ print "          kubelet \\\n" }}{{ end -}}
{{ kubeletFlags .KubeletVersion .CloudProviderName .MachineSpec.Name .DNSIPs .ExternalCloudProvider .PauseImage .MachineSpec.Taints | indent 10 }}
        ExecStop=-/usr/bin/rkt stop --uuid-file=/var/cache/kubelet-pod.uuid
        Restart=always
        RestartSec=10
        [Install]
        WantedBy=multi-user.target

    - name: docker.service
      enabled: true
      dropins:
      - name: 10-environment.conf
        contents: |
          [Service]
          EnvironmentFile=-/etc/environment

storage:
  files:
{{- if .HTTPProxy }}
    - path: /etc/environment
      filesystem: root
      mode: 0644
      contents:
        inline: |
{{ proxyEnvironment .HTTPProxy .NoProxy | indent 10 }}
{{- end }}

    - path: "/etc/systemd/journald.conf.d/max_disk_use.conf"
      filesystem: root
      mode: 0644
      contents:
        inline: |
{{ journalDConfig | indent 10 }}
    
    - path: "/etc/kubernetes/kubelet.conf"
      filesystem: root
      mode: 0644
      contents:
        inline: |
{{ kubeletConfiguration "cluster.local" .DNSIPs | indent 10 }}

    - path: /opt/load-kernel-modules.sh
      filesystem: root
      mode: 0755
      contents:
        inline: |
{{ kernelModulesScript | indent 10 }}

    - path: /etc/sysctl.d/k8s.conf
      filesystem: root
      mode: 0644
      contents:
        inline: |
{{ kernelSettings | indent 10 }}

    - path: /proc/sys/kernel/panic_on_oops
      filesystem: root
      mode: 0644
      contents:
        inline: |
          1

    - path: /proc/sys/kernel/panic
      filesystem: root
      mode: 0644
      contents:
        inline: |
          10

    - path: /proc/sys/vm/overcommit_memory
      filesystem: root
      mode: 0644
      contents:
        inline: |
          1

    - path: /etc/kubernetes/bootstrap-kubelet.conf
      filesystem: root
      mode: 0400
      contents:
        inline: |
{{ .Kubeconfig | indent 10 }}

    - path: /etc/kubernetes/cloud-config
      filesystem: root
      mode: 0400
      contents:
        inline: |
{{ .CloudConfig | indent 10 }}

    - path: /etc/kubernetes/pki/ca.crt
      filesystem: root
      mode: 0644
      contents:
        inline: |
{{ .KubernetesCACert | indent 10 }}
{{ if ne .CloudProviderName "aws" }}
    - path: /etc/hostname
      filesystem: root
      mode: 0600
      contents:
        inline: '{{ .MachineSpec.Name }}'
{{- end }}

    - path: /etc/ssh/sshd_config
      filesystem: root
      mode: 0600
      user:
        id: 0
      group:
        id: 0
      contents:
        inline: |
          # Use most defaults for sshd configuration.
          Subsystem sftp internal-sftp
          ClientAliveInterval 180
          UseDNS no
          UsePAM yes
          PrintLastLog no # handled by PAM
          PrintMotd no # handled by PAM
          PasswordAuthentication no
          ChallengeResponseAuthentication no

    - path: /etc/docker/daemon.json
      filesystem: root
      mode: 0644
      contents:
        inline: |
{{ dockerConfig .InsecureRegistries .RegistryMirrors | indent 10 }}

    - path: /opt/bin/download.sh
      filesystem: root
      mode: 0755
      contents:
        inline: |
          #!/bin/bash
          set -xeuo pipefail
{{ downloadBinariesScript .KubeletVersion false | indent 10 }}`
