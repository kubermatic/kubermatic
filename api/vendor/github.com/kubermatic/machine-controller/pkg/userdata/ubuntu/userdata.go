package ubuntu

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"text/template"

	"github.com/Masterminds/semver"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	machinesv1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	machinetemplate "github.com/kubermatic/machine-controller/pkg/template"
	"github.com/kubermatic/machine-controller/pkg/userdata/cloud"
	"k8s.io/apimachinery/pkg/runtime"
)

type Provider struct{}

type Config struct {
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

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

func (p Provider) SupportedContainerRuntimes() (runtimes []machinesv1alpha1.ContainerRuntimeInfo) {
	for _, ic := range dockerInstallCandidates {
		for _, v := range ic.versions {
			runtimes = append(runtimes, machinesv1alpha1.ContainerRuntimeInfo{Name: containerruntime.Docker, Version: v})
		}
	}

	for _, ic := range crioInstallCandidates {
		for _, v := range ic.versions {
			runtimes = append(runtimes, machinesv1alpha1.ContainerRuntimeInfo{Name: containerruntime.CRIO, Version: v})
		}
	}

	return runtimes
}

func (p Provider) UserData(spec machinesv1alpha1.MachineSpec, kubeconfig string, ccProvider cloud.ConfigProvider, clusterDNSIPs []net.IP) (string, error) {
	tmpl, err := template.New("user-data").Funcs(machinetemplate.TxtFuncMap()).Parse(ctTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse user-data template: %v", err)
	}

	kubeletVersion, err := semver.NewVersion(spec.Versions.Kubelet)
	if err != nil {
		return "", fmt.Errorf("invalid kubelet version: %v", err)
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

	var crPkg, crPkgVersion string
	if spec.Versions.ContainerRuntime.Name == containerruntime.Docker {
		crPkg, crPkgVersion, err = getDockerInstallCandidate(spec.Versions.ContainerRuntime.Version)
		if err != nil {
			return "", fmt.Errorf("failed to get docker install candidate for %s: %v", spec.Versions.ContainerRuntime.Version, err)
		}
	} else if spec.Versions.ContainerRuntime.Name == containerruntime.CRIO {
		crPkg, crPkgVersion, err = getCRIOInstallCandidate(spec.Versions.ContainerRuntime.Version)
		if err != nil {
			return "", fmt.Errorf("failed to get cri-o install candidate for %s: %v", spec.Versions.ContainerRuntime.Version, err)
		}
	} else {
		return "", fmt.Errorf("unknown container runtime selected '%s'", spec.Versions.ContainerRuntime.Name)
	}

	data := struct {
		MachineSpec         machinesv1alpha1.MachineSpec
		ProviderConfig      *providerconfig.Config
		OSConfig            *Config
		Kubeconfig          string
		CloudProvider       string
		CloudConfig         string
		CRAptPackage        string
		CRAptPackageVersion string
		KubeletDownloadURL  string
		ClusterDNSIPs       []net.IP
	}{
		MachineSpec:         spec,
		ProviderConfig:      pconfig,
		OSConfig:            osConfig,
		Kubeconfig:          kubeconfig,
		CloudProvider:       cpName,
		CloudConfig:         cpConfig,
		CRAptPackage:        crPkg,
		CRAptPackageVersion: crPkgVersion,
		KubeletDownloadURL:  fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/v%s/bin/linux/amd64/kubelet", kubeletVersion.String()),
		ClusterDNSIPs:       clusterDNSIPs,
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

package_update: true
{{- if .OSConfig.DistUpgradeOnBoot }}
package_upgrade: true
package_reboot_if_required: true
{{- end }}

ssh_authorized_keys:
{{- range .ProviderConfig.SSHPublicKeys }}
- "{{ . }}"
{{- end }}

write_files:
- path: "/etc/kubernetes/cloud-config"
  content: |
{{ if ne .CloudConfig "" }}{{ .CloudConfig | indent 4 }}{{ end }}

- path: "/etc/kubernetes/bootstrap.kubeconfig"
  content: |
{{ .Kubeconfig | indent 4 }}

- path: "/etc/kubernetes/download.sh"
  permissions: '0777'
  content: |
    #!/bin/bash
    set -xeuo pipefail
    mkdir -p /opt/bin /opt/cni/bin /etc/cni/net.d /var/run/kubernetes /var/lib/kubelet /etc/kubernetes/manifests /var/log/containers
    if [ ! -f /opt/bin/kubelet ]; then
      curl -L -o /opt/bin/kubelet {{ .KubeletDownloadURL }}
      chmod +x /opt/bin/kubelet
    fi
    if [ ! -f /opt/cni/bin/bridge ]; then
      curl -L -o /opt/cni.tgz https://storage.googleapis.com/cni-plugins/cni-plugins-amd64-v0.6.0.tgz
      mkdir -p /opt/cni/bin/
      tar -xzf /opt/cni.tgz -C /opt/cni/bin/
    fi

- path: "/etc/systemd/system/kubelet.service"
  content: |
    [Unit]
    Description=Kubelet
    Requires=network.target
{{- if eq .MachineSpec.Versions.ContainerRuntime.Name "docker" }}
    Requires=docker.service
    After=docker.service
{{- end }}
{{- if eq .MachineSpec.Versions.ContainerRuntime.Name "cri-o" }}
    Requires=crio.service
    After=crio.service
{{- end }}
    After=network.target

    [Service]
    Restart=always
    RestartSec=10
    StartLimitInterval=600
    StartLimitBurst=50
    TimeoutStartSec=5min
    Environment="PATH=/sbin:/bin:/usr/sbin:/usr/bin:/opt/bin"
    ExecStartPre=/etc/kubernetes/download.sh
    ExecStart=/opt/bin/kubelet \
{{- if eq .MachineSpec.Versions.ContainerRuntime.Name "docker" }}
      --container-runtime=docker \
{{- end }}
{{- if eq .MachineSpec.Versions.ContainerRuntime.Name "cri-o" }}
      --container-runtime=remote \
      --container-runtime-endpoint=unix:///var/run/crio/crio.sock \
      --cgroup-driver="systemd" \
{{- end }}
      --allow-privileged=true \
      --cni-bin-dir=/opt/cni/bin \
      --cni-conf-dir=/etc/cni/net.d \
      --cluster-dns={{ ipSliceToCommaSeparatedString .ClusterDNSIPs }} \
      --cluster-domain=cluster.local \
      --network-plugin=cni \
      {{- if .CloudProvider }}
      --cloud-provider={{ .CloudProvider }} \
      --cloud-config=/etc/kubernetes/cloud-config \
      {{- end }}
      --cert-dir=/etc/kubernetes/ \
      --pod-manifest-path=/etc/kubernetes/manifests \
      --resolv-conf=/etc/resolv.conf \
      --rotate-certificates=true \
      --kubeconfig=/etc/kubernetes/kubeconfig \
      --bootstrap-kubeconfig=/etc/kubernetes/bootstrap.kubeconfig \
      --lock-file=/var/run/lock/kubelet.lock \
      --exit-on-lock-contention

    [Install]
    WantedBy=multi-user.target

{{- if eq .MachineSpec.Versions.ContainerRuntime.Name "cri-o" }}

- path: "/etc/sysconfig/crio-network"
  content: |
    CRIO_NETWORK_OPTIONS="--registry=docker.io"
{{- end }}

runcmd:
{{- if eq .MachineSpec.Versions.ContainerRuntime.Name "cri-o" }}
- systemctl enable crio
- systemctl start crio
{{- end }}
- systemctl enable kubelet
- systemctl start kubelet

apt:
  sources:
{{- if eq .MachineSpec.Versions.ContainerRuntime.Name "cri-o" }}
    cri-o:
      source: "ppa:projectatomic/ppa"
{{- end }}
{{- if eq .MachineSpec.Versions.ContainerRuntime.Name "docker" }}
    docker:
      source: deb [arch=amd64] https://download.docker.com/linux/ubuntu $RELEASE stable
      key: |
        -----BEGIN PGP PUBLIC KEY BLOCK-----

        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
        YT90qFF93M3v01BbxP+EIY2/9tiIPbrd
        =0YYh
        -----END PGP PUBLIC KEY BLOCK-----
{{- end }}

# install dependencies for cloud-init via bootcmd...
bootcmd:
- "sudo apt-get update && sudo apt-get install -y software-properties-common gdisk eatmydata"

packages:
- "curl"
- "ca-certificates"
- "ceph-common"
- "cifs-utils"
- "conntrack"
- "e2fsprogs"
- "ebtables"
- "ethtool"
- "git"
- "glusterfs-client"
- "iptables"
- "jq"
- "kmod"
- "openssh-client"
- "nfs-common"
- "socat"
- "util-linux"
{{- if .CRAptPackage }}
{{- if ne .CRAptPackageVersion "" }}
- ["{{ .CRAptPackage }}", "{{ .CRAptPackageVersion }}"]
{{- else }}
- "{{ .CRAptPackage }}"
{{- end }}{{ end }}
`
