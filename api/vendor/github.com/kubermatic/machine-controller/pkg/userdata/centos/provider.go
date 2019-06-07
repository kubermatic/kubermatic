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
// UserData plugin for CentOS.
//

package centos

import (
	"bytes"
	"errors"
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
		return "", fmt.Errorf("invalid kubelet version: '%v'", err)
	}

	pconfig, err := providerconfig.GetConfig(spec.ProviderSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get provider config: %v", err)
	}

	if pconfig.OverwriteCloudConfig != nil {
		cloudConfig = *pconfig.OverwriteCloudConfig
	}

	if pconfig.Network != nil {
		return "", errors.New("static IP config is not supported with CentOS")
	}

	centosConfig, err := LoadConfig(pconfig.OperatingSystemSpec)
	if err != nil {
		return "", fmt.Errorf("failed to parse OperatingSystemSpec: '%v'", err)
	}

	serverAddr, err := userdatahelper.GetServerAddressFromKubeconfig(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error extracting server address from kubeconfig: %v", err)
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
		MachineSpec      clusterv1alpha1.MachineSpec
		ProviderSpec     *providerconfig.Config
		OSConfig         *Config
		CloudProvider    string
		CloudConfig      string
		KubeletVersion   string
		ClusterDNSIPs    []net.IP
		ServerAddr       string
		Kubeconfig       string
		KubernetesCACert string
		IsExternal       bool
	}{
		MachineSpec:      spec,
		ProviderSpec:     pconfig,
		OSConfig:         centosConfig,
		CloudProvider:    cloudProviderName,
		CloudConfig:      cloudConfig,
		KubeletVersion:   kubeletVersion.String(),
		ClusterDNSIPs:    clusterDNSIPs,
		ServerAddr:       serverAddr,
		Kubeconfig:       kubeconfigString,
		KubernetesCACert: kubernetesCACert,
		IsExternal:       externalCloudProvider,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}
	return userdatahelper.CleanupTemplateOutput(b.String())
}

// UserData template.
const userDataTemplate = `#cloud-config
{{ if ne .CloudProvider "aws" }}
hostname: {{ .MachineSpec.Name }}
{{- /* Never set the hostname on AWS nodes. Kubernetes(kube-proxy) requires the hostname to be the private dns name */}}
{{ end }}

{{- if .OSConfig.DistUpgradeOnBoot }}
package_upgrade: true
package_reboot_if_required: true
{{- end }}

ssh_pwauth: no

{{- if ne (len .ProviderSpec.SSHPublicKeys) 0 }}
ssh_authorized_keys:
{{- range .ProviderSpec.SSHPublicKeys }}
  - "{{ . }}"
{{- end }}
{{- end }}

write_files:
- path: "/etc/systemd/journald.conf.d/max_disk_use.conf"
  content: |
{{ journalDConfig | indent 4 }}

- path: "/etc/modules-load.d/k8s.conf"
  content: |
{{ kernelModules | indent 4 }}

- path: "/etc/sysctl.d/k8s.conf"
  content: |
{{ kernelSettings | indent 4 }}

- path: /etc/sysconfig/selinux
  content: |
    # This file controls the state of SELinux on the system.
    # SELINUX= can take one of these three values:
    #     enforcing - SELinux security policy is enforced.
    #     permissive - SELinux prints warnings instead of enforcing.
    #     disabled - No SELinux policy is loaded.
    SELINUX=permissive
    # SELINUXTYPE= can take one of three two values:
    #     targeted - Targeted processes are protected,
    #     minimum - Modification of targeted policy. Only selected processes are protected.
    #     mls - Multi Level Security protection.
    SELINUXTYPE=targeted

- path: "/opt/bin/setup"
  permissions: "0777"
  content: |
    #!/bin/bash
    set -xeuo pipefail

    setenforce 0 || true

{{- /* As we added some modules and don't want to reboot, restart the service */}}
    systemctl restart systemd-modules-load.service
    sysctl --system

{{- /* Make sure we always disable swap - Otherwise the kubelet won't start */}}
    cp /etc/fstab /etc/fstab.orig
    cat /etc/fstab.orig | awk '$3 ~ /^swap$/ && $1 !~ /^#/ {$0="# commented out by cloudinit\n#"$0} 1' > /etc/fstab.noswap
    mv /etc/fstab.noswap /etc/fstab
    swapoff -a
    {{ if ne .CloudProvider "aws" }}
{{- /*  The normal way of setting it via cloud-init is broken, see */}}
{{- /*  https://bugs.launchpad.net/cloud-init/+bug/1662542 */}}
    hostnamectl set-hostname {{ .MachineSpec.Name }}
    {{ end }}

    yum install -y docker-1.13.1 \
      ebtables \
      ethtool \
      nfs-utils \
      bash-completion \
      sudo \
      socat \
      wget \
      curl \
      ipvsadm{{ if eq .CloudProvider "vsphere" }} \
      open-vm-tools{{ end }}

{{ downloadBinariesScript .KubeletVersion true | indent 4 }}

    {{- if eq .CloudProvider "vsphere" }}
    systemctl enable --now vmtoolsd.service
    {{ end -}}
    systemctl enable --now docker
    systemctl enable --now kubelet
    systemctl enable --now --no-block kubelet-healthcheck.service
    systemctl enable --now --no-block docker-healthcheck.service

- path: "/opt/bin/supervise.sh"
  permissions: "0755"
  content: |
    #!/bin/bash
    set -xeuo pipefail
    while ! "$@"; do
      sleep 1
    done

- path: "/etc/systemd/system/kubelet.service"
  content: |
{{ kubeletSystemdUnit .KubeletVersion .CloudProvider .MachineSpec.Name .ClusterDNSIPs .IsExternal | indent 4 }}

- path: "/etc/systemd/system/kubelet.service.d/extras.conf"
  content: |
    [Service]
    Environment="KUBELET_EXTRA_ARGS=--cgroup-driver=systemd"

- path: "/etc/kubernetes/cloud-config"
  content: |
{{ .CloudConfig | indent 4 }}

- path: "/etc/kubernetes/bootstrap-kubelet.conf"
  content: |
{{ .Kubeconfig | indent 4 }}

- path: "/etc/kubernetes/pki/ca.crt"
  content: |
{{ .KubernetesCACert | indent 4 }}

- path: "/etc/systemd/system/setup.service"
  permissions: "0644"
  content: |
    [Install]
    WantedBy=multi-user.target

    [Unit]
    Requires=network-online.target
    After=network-online.target

    [Service]
    Type=oneshot
    RemainAfterExit=true
    ExecStart=/opt/bin/supervise.sh /opt/bin/setup

- path: "/etc/profile.d/opt-bin-path.sh"
  permissions: "0644"
  content: |
    export PATH="/opt/bin:$PATH"

- path: /etc/systemd/system/kubelet-healthcheck.service
  permissions: "0644"
  content: |
{{ kubeletHealthCheckSystemdUnit | indent 4 }}

- path: /etc/systemd/system/docker-healthcheck.service
  permissions: "0644"
  content: |
{{ containerRuntimeHealthCheckSystemdUnit | indent 4 }}

runcmd:
- systemctl enable --now setup.service
`
