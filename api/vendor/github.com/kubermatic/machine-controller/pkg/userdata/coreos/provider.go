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
	"net"
	"text/template"

	"github.com/Masterminds/semver"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	userdatahelper "github.com/kubermatic/machine-controller/pkg/userdata/helper"
)

// Provider is a pkg/userdata/plugin.Provider implementation.
type Provider struct{}

// UserData renders user-data template to string.
func (p Provider) UserData(
	spec clusterv1alpha1.MachineSpec,
	kubeconfig *clientcmdapi.Config,
	cloudConfig string,
	cloudProviderName string,
	clusterDNSIPs []net.IP,
	externalCloudProvider bool,
) (string, error) {

	tmpl, err := template.New("user-data").Funcs(userdatahelper.TxtFuncMap()).Parse(userDataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse user-data template: %v", err)
	}

	kubeletVersion, err := semver.NewVersion(spec.Versions.Kubelet)
	if err != nil {
		return "", fmt.Errorf("invalid kubelet version: %v", err)
	}

	pconfig, err := providerconfig.GetConfig(spec.ProviderSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get provider config: %v", err)
	}

	if pconfig.OverwriteCloudConfig != nil {
		cloudConfig = *pconfig.OverwriteCloudConfig
	}

	coreosConfig, err := LoadConfig(pconfig.OperatingSystemSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get coreos config from provider config: %v", err)
	}

	kubeconfigString, err := userdatahelper.StringifyKubeconfig(kubeconfig)
	if err != nil {
		return "", err
	}

	kubernetesCACert, err := userdatahelper.GetCACert(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error extracting cacert: %v", err)
	}

	data := struct {
		MachineSpec       clusterv1alpha1.MachineSpec
		ProviderSpec      *providerconfig.Config
		CoreOSConfig      *Config
		Kubeconfig        string
		CloudProvider     string
		CloudConfig       string
		HyperkubeImageTag string
		ClusterDNSIPs     []net.IP
		KubernetesCACert  string
		KubeletVersion    string
		IsExternal        bool
	}{
		MachineSpec:       spec,
		ProviderSpec:      pconfig,
		CoreOSConfig:      coreosConfig,
		Kubeconfig:        kubeconfigString,
		CloudProvider:     cloudProviderName,
		CloudConfig:       cloudConfig,
		HyperkubeImageTag: fmt.Sprintf("v%s", kubeletVersion.String()),
		ClusterDNSIPs:     clusterDNSIPs,
		KubernetesCACert:  kubernetesCACert,
		KubeletVersion:    kubeletVersion.String(),
		IsExternal:        externalCloudProvider,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}

	return b.String(), nil
}

// UserData template.
const userDataTemplate = `
passwd:
  users:
    - name: core
      ssh_authorized_keys:
        {{range .ProviderSpec.SSHPublicKeys}}- {{.}}
        {{end}}

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
{{- if .CoreOSConfig.DisableAutoUpdate }}
    - name: update-engine.service
      mask: true
    - name: locksmithd.service
      mask: true
{{ end }}
    - name: docker.service
      enabled: true

    - name: download-healthcheck-script.service
      enabled: true
      contents: |
        [Unit]
        Requires=network-online.target
        After=network-online.target
        [Service]
        Type=oneshot
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
        Environment=KUBELET_IMAGE=docker://k8s.gcr.io/hyperkube-amd64:{{ .HyperkubeImageTag }}
        Environment="RKT_RUN_ARGS=--uuid-file-save=/var/cache/kubelet-pod.uuid \
          --insecure-options=image \
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
        ExecStart=/usr/lib/coreos/kubelet-wrapper \
{{ kubeletFlags .KubeletVersion .CloudProvider .MachineSpec.Name .ClusterDNSIPs .IsExternal | indent 10 }}
        ExecStop=-/usr/bin/rkt stop --uuid-file=/var/cache/kubelet-pod.uuid
        Restart=always
        RestartSec=10
        [Install]
        WantedBy=multi-user.target

storage:
  files:
    - path: "/etc/systemd/journald.conf.d/max_disk_use.conf"
      filesystem: root
      mode: 0644
      contents:
        inline: |
{{ journalDConfig | indent 10 }}

    - path: /etc/modules-load.d/k8s.conf
      filesystem: root
      mode: 0644
      contents:
        inline: |
{{ kernelModules | indent 10 }}

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

{{ if ne .CloudProvider "aws" }}
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

    - path: /etc/systemd/system/docker.service.d/10-storage.conf
      filesystem: root
      mode: 0644
      contents:
        inline: |
          [Service]
          Environment=DOCKER_OPTS=--storage-driver=overlay2

    - path: /opt/bin/download.sh
      filesystem: root
      mode: 0755
      contents:
        inline: |
          #!/bin/bash
          set -xeuo pipefail
{{ downloadBinariesScript .KubeletVersion false | indent 10 }}
`
