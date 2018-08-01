package ubuntu

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

	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	machinesv1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	machinetemplate "github.com/kubermatic/machine-controller/pkg/template"
	"github.com/kubermatic/machine-controller/pkg/userdata/cloud"
	userdatahelper "github.com/kubermatic/machine-controller/pkg/userdata/helper"
)

var (
	NoInstallCandidateAvailableErr = errors.New("no install candidate available for the desired version")
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

// SupportedContainerRuntimes return list of container runtimes
func (p Provider) SupportedContainerRuntimes() (runtimes []machinesv1alpha1.ContainerRuntimeInfo) {
	for _, ic := range dockerInstallCandidates {
		for _, v := range ic.versions {
			runtimes = append(runtimes, machinesv1alpha1.ContainerRuntimeInfo{Name: containerruntime.Docker, Version: v})
		}
	}

	return runtimes
}

// UserData renders user-data template
func (p Provider) UserData(
	spec machinesv1alpha1.MachineSpec,
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
		return "", fmt.Errorf("invalid kubelet version: %v", err)
	}

	var kubeadmDropInFilename string
	if kubeletVersion.Minor() > 8 {
		kubeadmDropInFilename = "10-kubeadm.conf"
	} else {
		kubeadmDropInFilename = "kubeadm-10.conf"
	}

	cpConfig, cpName, err := ccProvider.GetCloudConfig(spec)
	if err != nil {
		return "", fmt.Errorf("failed to get cloud config: %v", err)
	}

	pconfig, err := providerconfig.GetConfig(spec.ProviderConfig)
	if err != nil {
		return "", fmt.Errorf("failed to get provider config: %v", err)
	}

	osConfig, err := getConfig(pconfig.OperatingSystemSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get ubuntu config from provider config: %v", err)
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

	var crPkg, crPkgVersion string
	if spec.Versions.ContainerRuntime.Name == containerruntime.Docker {
		crPkg, crPkgVersion, err = getDockerInstallCandidate(spec.Versions.ContainerRuntime.Version)
		if err != nil {
			return "", fmt.Errorf("failed to get docker install candidate for %s: %v", spec.Versions.ContainerRuntime.Version, err)
		}
	} else {
		return "", fmt.Errorf("unknown container runtime selected '%s'", spec.Versions.ContainerRuntime.Name)
	}

	data := struct {
		MachineSpec           machinesv1alpha1.MachineSpec
		ProviderConfig        *providerconfig.Config
		OSConfig              *Config
		BoostrapToken         string
		CloudProvider         string
		CloudConfig           string
		CRAptPackage          string
		CRAptPackageVersion   string
		KubernetesVersion     string
		KubeadmDropInFilename string
		ClusterDNSIPs         []net.IP
		KubeadmCACertHash     string
		ServerAddr            string
	}{
		MachineSpec:           spec,
		ProviderConfig:        pconfig,
		OSConfig:              osConfig,
		BoostrapToken:         bootstrapToken,
		CloudProvider:         cpName,
		CloudConfig:           cpConfig,
		CRAptPackage:          crPkg,
		CRAptPackageVersion:   crPkgVersion,
		KubernetesVersion:     kubeletVersion.String(),
		KubeadmDropInFilename: kubeadmDropInFilename,
		ClusterDNSIPs:         clusterDNSIPs,
		KubeadmCACertHash:     kubeadmCACertHash,
		ServerAddr:            serverAddr,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}

	return string(b.String()), nil
}

const ctTemplate = `#cloud-config
hostname: {{ .MachineSpec.Name }}

ssh_pwauth: no

ssh_authorized_keys:
{{- range .ProviderConfig.SSHPublicKeys }}
- "{{ . }}"
{{- end }}

write_files:
- path: "/etc/sysctl.d/k8s.conf"
  content: |
    net.bridge.bridge-nf-call-ip6tables = 1
    net.bridge.bridge-nf-call-iptables = 1
    kernel.panic_on_oops = 1
    kernel.panic = 10
    vm.overcommit_memory = 1

- path: "/etc/kubernetes/cloud-config"
  content: |
{{ if ne .CloudConfig "" }}{{ .CloudConfig | indent 4 }}{{ end }}

- path: "/etc/apt/sources.list.d/docker.list"
  permissions: "0644"
  content: deb [arch=amd64] https://download.docker.com/linux/ubuntu xenial stable

- path: "/etc/apt/sources.list.d/kubernetes.list"
  permissions: "0644"
  content: deb http://apt.kubernetes.io/ kubernetes-xenial main

- path: "/usr/local/bin/setup"
  permissions: "0755"
  content: |
    #!/bin/bash
    set -xeuo pipefail

    sysctl --system
    mkdir -p /opt/bin
    apt-key add /opt/docker.asc
    apt-key add /opt/kubernetes.asc
    apt-get update

    # If something failed during package installation but one of docker/kubeadm/kubelet was already installed
    # an apt-mark hold after the install won't do it, which is why we test here if the binaries exist and if
    # yes put them on hold
    set +e
    which docker && apt-mark hold docker docker-ce
    which kubelet && apt-mark hold kubelet
    which kubeadm && apt-mark hold kubeadm

    # When docker is started from within the apt installation it fails with a
    # 'no sockets found via socket activation: make sure the service was started by systemd'
    # Apparently the package is broken in a way that it gets started without its dependencies, manually starting
    # it works fine thought
    which docker && systemctl start docker
    set -e

    {{- if .OSConfig.DistUpgradeOnBoot }}
    DEBIAN_FRONTEND=noninteractive apt-get -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" dist-upgrade -y
    {{- end }}
    if [[ -e /var/run/reboot-required ]]; then
      reboot
    fi

    export CR_PKG=''
{{ if .CRAptPackage }}
  {{ if ne .CRAptPackageVersion "" }}
    export CR_PKG='{{ .CRAptPackage }}={{ .CRAptPackageVersion }}'
  {{ else }}
    export CR_PKG='{{ .CRAptPackage }}'
  {{ end }}
{{ end }}

    # There is a dependency issue in the rpm repo for 1.8, if the cni package is not explicitly
    # specified, installation of the kube packages fails
    export CNI_PKG=''
    {{- if semverCompare "=1.8.X" .KubernetesVersion }}
    export CNI_PKG='kubernetes-cni=0.5.1-00'
    {{- end }}

    DEBIAN_FRONTEND=noninteractive apt-get -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" install -y \
      curl \
      ca-certificates \
      ceph-common \
      cifs-utils \
      conntrack \
      e2fsprogs \
      ebtables \
      ethtool \
      glusterfs-client \
      iptables \
      jq \
      kmod \
      openssh-client \
      nfs-common \
      socat \
      util-linux \
      ${CR_PKG} \
      open-vm-tools \
      kubelet={{ .KubernetesVersion }}-00 \
      kubeadm={{ .KubernetesVersion }}-00 \
      ${CNI_PKG}

    cp /etc/default/kubelet-overwrite /etc/default/kubelet

    systemctl enable --now docker
    systemctl enable kubelet

    if ! [[ -e /etc/kubernetes/pki/ca.crt ]]; then
      kubeadm join \
        --token {{ .BoostrapToken }} \
        --discovery-token-ca-cert-hash sha256:{{ .KubeadmCACertHash }} \
        {{ .ServerAddr }}
    fi

- path: "/opt/kubernetes.asc"
  permissions: "0400"
  content: |
    -----BEGIN PGP PUBLIC KEY BLOCK-----
    
    mQENBFrBaNsBCADrF18KCbsZlo4NjAvVecTBCnp6WcBQJ5oSh7+E98jX9YznUCrN
    rgmeCcCMUvTDRDxfTaDJybaHugfba43nqhkbNpJ47YXsIa+YL6eEE9emSmQtjrSW
    IiY+2YJYwsDgsgckF3duqkb02OdBQlh6IbHPoXB6H//b1PgZYsomB+841XW1LSJP
    YlYbIrWfwDfQvtkFQI90r6NknVTQlpqQh5GLNWNYqRNrGQPmsB+NrUYrkl1nUt1L
    RGu+rCe4bSaSmNbwKMQKkROE4kTiB72DPk7zH4Lm0uo0YFFWG4qsMIuqEihJ/9KN
    X8GYBr+tWgyLooLlsdK3l+4dVqd8cjkJM1ExABEBAAG0QEdvb2dsZSBDbG91ZCBQ
    YWNrYWdlcyBBdXRvbWF0aWMgU2lnbmluZyBLZXkgPGdjLXRlYW1AZ29vZ2xlLmNv
    bT6JAT4EEwECACgFAlrBaNsCGy8FCQWjmoAGCwkIBwMCBhUIAgkKCwQWAgMBAh4B
    AheAAAoJEGoDCyG6B/T78e8H/1WH2LN/nVNhm5TS1VYJG8B+IW8zS4BqyozxC9iJ
    AJqZIVHXl8g8a/Hus8RfXR7cnYHcg8sjSaJfQhqO9RbKnffiuQgGrqwQxuC2jBa6
    M/QKzejTeP0Mgi67pyrLJNWrFI71RhritQZmzTZ2PoWxfv6b+Tv5v0rPaG+ut1J4
    7pn+kYgtUaKdsJz1umi6HzK6AacDf0C0CksJdKG7MOWsZcB4xeOxJYuy6NuO6Kcd
    Ez8/XyEUjIuIOlhYTd0hH8E/SEBbXXft7/VBQC5wNq40izPi+6WFK/e1O42DIpzQ
    749ogYQ1eodexPNhLzekKR3XhGrNXJ95r5KO10VrsLFNd8I=
    =TKuP
    -----END PGP PUBLIC KEY BLOCK-----

- path: "/opt/docker.asc"
  permissions: "0400"
  content: |
    -----BEGIN PGP PUBLIC KEY BLOCK-----

    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
    YT90qFF93M3v01BbxP+EIY2/9tiIPbrd
    =0YYh
    -----END PGP PUBLIC KEY BLOCK-----

- path: "/usr/local/bin/supervise.sh"
  permissions: "0755"
  content: |
    #!/bin/bash
    set -xeuo pipefail
    while ! "$@"; do
      sleep 1
    done

- path: "/etc/default/kubelet-overwrite"
  content: |
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
{{ if semverCompare "<1.11.0" .KubernetesVersion }}
- path: "/etc/systemd/system/kubelet.service.d/20-extra.conf"
  permissions: "0644"
  content: |
    [Service]
    EnvironmentFile=/etc/default/kubelet
{{ end }}

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
    ExecStart=/usr/local/bin/supervise.sh /usr/local/bin/setup

runcmd:
- systemctl enable --now setup.service
`
