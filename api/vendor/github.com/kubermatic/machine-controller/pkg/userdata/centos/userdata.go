package centos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"text/template"

	"github.com/Masterminds/semver"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	machinetemplate "github.com/kubermatic/machine-controller/pkg/template"
	"github.com/kubermatic/machine-controller/pkg/userdata/cloud"
	userdatahelper "github.com/kubermatic/machine-controller/pkg/userdata/helper"

	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

func getConfig(r runtime.RawExtension) (*Config, error) {
	p := Config{}
	if len(r.Raw) == 0 {
		return &p, nil
	}
	if err := json.Unmarshal(r.Raw, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Config TODO
type Config struct {
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// Provider is a pkg/userdata.Provider implementation
type Provider struct{}

// UserData renders user-data template
func (p Provider) UserData(
	spec clusterv1alpha1.MachineSpec,
	kubeconfig *clientcmdapi.Config,
	ccProvider cloud.ConfigProvider,
	clusterDNSIPs []net.IP,
) (string, error) {

	tmpl, err := template.New("user-data").Funcs(machinetemplate.TxtFuncMap()).Parse(ctTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse user-data template: %v", err)
	}

	kubeletVersion, err := semver.NewVersion(spec.Versions.Kubelet)
	if err != nil {
		return "", fmt.Errorf("invalid kubelet version: '%v'", err)
	}

	cpConfig, cpName, err := ccProvider.GetCloudConfig(spec)
	if err != nil {
		return "", fmt.Errorf("failed to get cloud config: %v", err)
	}

	pconfig, err := providerconfig.GetConfig(spec.ProviderConfig)
	if err != nil {
		return "", fmt.Errorf("failed to get provider config: %v", err)
	}

	if pconfig.OverwriteCloudConfig != nil {
		cpConfig = *pconfig.OverwriteCloudConfig
	}

	if pconfig.Network != nil {
		return "", errors.New("static IP config is not supported with CentOS")
	}

	osConfig, err := getConfig(pconfig.OperatingSystemSpec)
	if err != nil {
		return "", fmt.Errorf("failed to parse OperatingSystemSpec: '%v'", err)
	}

	bootstrapToken, err := userdatahelper.GetTokenFromKubeconfig(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error extracting token: %v", err)
	}

	kubeadmCACertHash, err := userdatahelper.GetKubeadmCACertHash(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error extracting kubeadm cacert hash: %v", err)
	}

	serverAddr, err := userdatahelper.GetServerAddressFromKubeconfig(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error extracting server address from kubeconfig: %v", err)
	}

	data := struct {
		MachineSpec       clusterv1alpha1.MachineSpec
		ProviderConfig    *providerconfig.Config
		OSConfig          *Config
		BoostrapToken     string
		CloudProvider     string
		CloudConfig       string
		KubeletVersion    string
		ClusterDNSIPs     []net.IP
		KubeadmCACertHash string
		ServerAddr        string
		JournaldMaxSize   string
	}{
		MachineSpec:       spec,
		ProviderConfig:    pconfig,
		OSConfig:          osConfig,
		BoostrapToken:     bootstrapToken,
		CloudProvider:     cpName,
		CloudConfig:       cpConfig,
		KubeletVersion:    kubeletVersion.String(),
		ClusterDNSIPs:     clusterDNSIPs,
		KubeadmCACertHash: kubeadmCACertHash,
		ServerAddr:        serverAddr,
		JournaldMaxSize:   userdatahelper.JournaldMaxUse,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}
	return b.String(), nil
}

const ctTemplate = `#cloud-config
hostname: {{ .MachineSpec.Name }}

{{- if .OSConfig.DistUpgradeOnBoot }}
package_upgrade: true
package_reboot_if_required: true
{{- end }}

ssh_pwauth: no

{{- if ne (len .ProviderConfig.SSHPublicKeys) 0 }}
ssh_authorized_keys:
{{- range .ProviderConfig.SSHPublicKeys }}
  - "{{ . }}"
{{- end }}
{{- end }}

write_files:
- path: "/etc/systemd/journald.conf.d/max_disk_use.conf"
  content: |
    [Journal]
    SystemMaxUse={{ .JournaldMaxSize }}

- path: "/etc/sysctl.d/k8s.conf"
  content: |
    net.bridge.bridge-nf-call-ip6tables = 1
    net.bridge.bridge-nf-call-iptables = 1
    kernel.panic_on_oops = 1
    kernel.panic = 10
    vm.overcommit_memory = 1

- path: "/etc/yum.repos.d/kubernetes.repo"
  content: |
    [kubernetes]
    name=Kubernetes
    baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-$basearch
    enabled=1
    gpgcheck=1
    repo_gpgcheck=1
    gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg

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

- path: "/etc/sysconfig/kubelet-overwrite"
  content: |
    KUBELET_DNS_ARGS=
    KUBELET_EXTRA_ARGS=--authentication-token-webhook=true \
      {{- if .CloudProvider }}
      --cloud-provider={{ .CloudProvider }} \
      --cloud-config=/etc/kubernetes/cloud-config \
      {{- end}}
      --hostname-override={{ .MachineSpec.Name }} \
      --read-only-port=0 \
      --protect-kernel-defaults=true \
      --cluster-dns={{ ipSliceToCommaSeparatedString .ClusterDNSIPs }} \
      --cluster-domain=cluster.local

{{- if semverCompare "<1.11.0" .KubeletVersion }}
- path: "/etc/systemd/system/kubelet.service.d/20-extra.conf"
  content: |
    [Service]
    EnvironmentFile=/etc/sysconfig/kubelet
{{- end }}

- path: "/etc/kubernetes/cloud-config"
  content: |
{{ if ne .CloudConfig "" }}{{ .CloudConfig | indent 4 }}{{ end }}

- path: "/usr/local/bin/setup"
  permissions: "0777"
  content: |
    #!/bin/bash
    set -xeuo pipefail
    setenforce 0 || true
    sysctl --system

    yum install -y docker-1.13.1 \
      kubelet-{{ .KubeletVersion }} \
      kubeadm-{{ .KubeletVersion }} \
      ebtables \
      ethtool \
      nfs-utils \
      bash-completion \
      sudo

    cp /etc/sysconfig/kubelet-overwrite /etc/sysconfig/kubelet

    systemctl enable --now docker
    systemctl enable --now kubelet

    kubeadm join \
      --token {{ .BoostrapToken }} \
      --discovery-token-ca-cert-hash sha256:{{ .KubeadmCACertHash }} \
      {{- if semverCompare ">=1.9.X" .KubeletVersion }}
      --ignore-preflight-errors=CRI \
      {{- end }}
      {{ .ServerAddr }}

- path: "/usr/local/bin/supervise.sh"
  permissions: "0777"
  content: |
    #!/bin/bash
    set -xeuo pipefail
    while ! "$@"; do
      sleep 1
    done

- path: "/etc/systemd/system/setup.service"
  content: |
    [Install]
    WantedBy=multi-user.target

    [Unit]
    Requires=network-online.target
    After=network-online.target

    [Service]
    Type=oneshot
    RemainAfterExit=true
    ExecStart=/usr/local/bin/supervise.sh /usr/local/bin/setup

runcmd:
- systemctl enable --now setup.service
`
