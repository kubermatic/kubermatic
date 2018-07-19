package centos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"text/template"

	"github.com/Masterminds/semver"
	"github.com/kubermatic/machine-controller/pkg/containerruntime"
	machinesv1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	machinetemplate "github.com/kubermatic/machine-controller/pkg/template"
	"github.com/kubermatic/machine-controller/pkg/userdata/cloud"
	userdatahelper "github.com/kubermatic/machine-controller/pkg/userdata/helper"
	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Provider struct{}

type Config struct {
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

type packageCompatibilityMatrix struct {
	versions []string
	pkg      string
}

var dockerInstallCandidates = []packageCompatibilityMatrix{
	{
		versions: []string{"1.13", "1.13.1"},
		pkg:      "docker-1.13.1",
	},
}

func (p Provider) SupportedContainerRuntimes() (runtimes []machinesv1alpha1.ContainerRuntimeInfo) {
	for _, installCandidate := range dockerInstallCandidates {
		for _, v := range installCandidate.versions {
			runtimes = append(runtimes, machinesv1alpha1.ContainerRuntimeInfo{Name: containerruntime.Docker, Version: v})
		}
	}
	return runtimes
}

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

func getDockerPackageName(version string) (string, error) {
	for _, installCandidate := range dockerInstallCandidates {
		for _, v := range installCandidate.versions {
			if v == version {
				return installCandidate.pkg, nil
			}
		}
	}
	return "", fmt.Errorf("no package found for version '%s'", version)
}

func (p Provider) UserData(spec machinesv1alpha1.MachineSpec, kubeconfig *clientcmdapi.Config, ccProvider cloud.ConfigProvider, clusterDNSIPs []net.IP) (string, error) {
	tmpl, err := template.New("user-data").Funcs(machinetemplate.TxtFuncMap()).Parse(ctTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse user-data template: %v", err)
	}

	semverKubeletVersion, err := semver.NewVersion(spec.Versions.Kubelet)
	if err != nil {
		return "", fmt.Errorf("invalid kubelet version: '%v'", err)
	}
	kubeletVersion := semverKubeletVersion.String()

	dockerPackageName, err := getDockerPackageName(spec.Versions.ContainerRuntime.Version)
	if err != nil {
		return "", fmt.Errorf("error getting Docker package name: '%v'", err)
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
		MachineSpec       machinesv1alpha1.MachineSpec
		ProviderConfig    *providerconfig.Config
		OSConfig          *Config
		BoostrapToken     string
		CloudProvider     string
		CloudConfig       string
		KubeletVersion    string
		DockerPackageName string
		ClusterDNSIPs     []net.IP
		KubeadmCACertHash string
		ServerAddr        string
	}{
		MachineSpec:       spec,
		ProviderConfig:    pconfig,
		OSConfig:          osConfig,
		BoostrapToken:     bootstrapToken,
		CloudProvider:     cpName,
		CloudConfig:       cpConfig,
		KubeletVersion:    kubeletVersion,
		DockerPackageName: dockerPackageName,
		ClusterDNSIPs:     clusterDNSIPs,
		KubeadmCACertHash: kubeadmCACertHash,
		ServerAddr:        serverAddr,
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

{{ if ne (len .ProviderConfig.SSHPublicKeys) 0 }}
ssh_authorized_keys:
{{- range .ProviderConfig.SSHPublicKeys }}
  - "{{ . }}"
{{- end }}
{{- end }}

write_files:
- path: "/etc/sysctl.d/k8s.conf"
  content: |
    net.bridge.bridge-nf-call-ip6tables = 1
    net.bridge.bridge-nf-call-iptables = 1

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
    KUBELET_EXTRA_ARGS={{ if .CloudProvider }}--cloud-provider={{ .CloudProvider }} --cloud-config=/etc/kubernetes/cloud-config{{ end}} \
      --authentication-token-webhook=true --hostname-override={{ .MachineSpec.Name }} --read-only-port 0 \
      --cluster-dns={{ ipSliceToCommaSeparatedString .ClusterDNSIPs }} --cluster-domain=cluster.local
{{ if semverCompare "<1.11.0" .KubeletVersion }}
- path: "/etc/systemd/system/kubelet.service.d/20-extra.conf"
  content: |
    [Service]
    EnvironmentFile=/etc/sysconfig/kubelet
{{ end }}


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

    # There is a dependency issue in the rpm repo for 1.8, if the cni package is not explicitly
    # specified, installation of the kube packages fails
    export CNI_PKG=''
    {{- if semverCompare "=1.8.X" .KubeletVersion }}
    export CNI_PKG='kubernetes-cni-0.5.1-1'
    {{- end }}

    yum install -y {{ .DockerPackageName }} \
      kubelet-{{ .KubeletVersion }} \
      kubeadm-{{ .KubeletVersion }} \
      ebtables \
      ethtool \
      nfs-utils \
      bash-completion \
      sudo \
      ${CNI_PKG}

    cp /etc/sysconfig/kubelet-overwrite /etc/sysconfig/kubelet

    systemctl enable --now docker
    systemctl enable --now kubelet

    kubeadm join \
      --token {{ .BoostrapToken }} \
      --discovery-token-ca-cert-hash sha256:{{ .KubeadmCACertHash }} \
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
