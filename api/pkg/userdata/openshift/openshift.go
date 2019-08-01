package openshift

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"text/template"

	"github.com/Masterminds/semver"

	"github.com/kubermatic/machine-controller/pkg/apis/plugin"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	userdatahelper "github.com/kubermatic/machine-controller/pkg/userdata/helper"

	"k8s.io/apimachinery/pkg/runtime"
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

// Config contains CentOS specific settings. It's being used within the provider spec (Inside the MachineSpec)
type Config struct {
	DistUpgradeOnBoot bool `json:"distUpgradeOnBoot"`
}

// Provider is a pkg/userdata.Provider implementation
type Provider struct{}

// UserData renders a cloud-init script to provision a worker OpenShift node
// The content of this cloud-init comes from the OpenShift machine-config-operator: https://github.com/openshift/machine-config-operator/tree/release-4.1/templates/worker
func (p Provider) UserData(req plugin.UserDataRequest) (string, error) {

	tmpl, err := template.New("user-data").Funcs(userdatahelper.TxtFuncMap()).Parse(userdataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse user-data template: %v", err)
	}

	openShiftVersion, err := semver.NewVersion(req.MachineSpec.Versions.Kubelet)
	if err != nil {
		return "", fmt.Errorf("invalid kubelet version: '%v'", err)
	}

	pconfig, err := providerconfig.GetConfig(req.MachineSpec.ProviderSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get provider config: %v", err)
	}

	if pconfig.OverwriteCloudConfig != nil {
		req.CloudConfig = *pconfig.OverwriteCloudConfig
	}

	if pconfig.Network != nil {
		return "", errors.New("static IP config is not supported with CentOS")
	}

	osConfig, err := getConfig(pconfig.OperatingSystemSpec)
	if err != nil {
		return "", fmt.Errorf("failed to parse OperatingSystemSpec: '%v'", err)
	}

	serverAddr, err := userdatahelper.GetServerAddressFromKubeconfig(req.Kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error extracting server address from kubeconfig: %v", err)
	}

	kubeconfigString, err := userdatahelper.StringifyKubeconfig(req.Kubeconfig)
	if err != nil {
		return "", err
	}

	kubernetesCACert, err := userdatahelper.GetCACert(req.Kubeconfig)
	if err != nil {
		return "", fmt.Errorf("error extracting cacert: %v", err)
	}

	// The OpenShift 4 minor release is: Kubernetes minor - 12
	// We require it to download some tooling which follows the Kubernetes versioning
	kubernetesMinor := openShiftVersion.Minor() + 12

	data := struct {
		plugin.UserDataRequest
		ProviderSpec          *providerconfig.Config
		OSConfig              *Config
		OpenShiftVersion      string
		OpenShiftMinorVersion string
		ServerAddr            string
		Kubeconfig            string
		KubernetesCACert      string
		CRIORepo              string
		CRICtlURL             string
	}{
		UserDataRequest:       req,
		ProviderSpec:          pconfig,
		OSConfig:              osConfig,
		OpenShiftVersion:      openShiftVersion.String(),
		OpenShiftMinorVersion: fmt.Sprintf("%d.%d", openShiftVersion.Major(), openShiftVersion.Minor()),
		ServerAddr:            serverAddr,
		Kubeconfig:            kubeconfigString,
		KubernetesCACert:      kubernetesCACert,
		// There is a CRI-O release for every Kubernetes release.
		CRIORepo: fmt.Sprintf("https://cbs.centos.org/repos/paas7-crio-1%d-candidate/x86_64/os/", kubernetesMinor),
		// There is a crictl release for every Kubernetes release.
		CRICtlURL: fmt.Sprintf("https://github.com/kubernetes-sigs/cri-tools/releases/download/v1.%d.0/crictl-v1.%d.0-linux-amd64.tar.gz", kubernetesMinor, kubernetesMinor),
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}
	return b.String(), nil
}

const userdataTemplate = `#cloud-config
{{ if ne .CloudProviderName "aws" }}
hostname: {{ .MachineSpec.Name }}
# Never set the hostname on AWS nodes. Kubernetes(kube-proxy) requires the hostname to be the private dns name
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

- path: "/etc/sysctl.d/99-openshift.conf"
  content: |
    net.ipv4.ip_forward=1

- path: "/etc/yum.repos.d/crio.repo"
  content: |
    [crio]
    name=CRI-O
    baseurl={{ .CRIORepo }}
    enabled=1
    gpgcheck=0

- path: "/opt/bin/setup"
  permissions: "0777"
  content: |
    #!/bin/bash
    set -xeuo pipefail

    # TODO: Figure out why the hyperkube binary installation does not work with selinux enabled 
    setenforce 0 || true

    systemctl daemon-reload

    # As we added some modules and don't want to reboot, restart the service
    systemctl restart systemd-modules-load.service
    sysctl --system

    # crictl does not exist in the centos repos - Thus we directly fetch it from upstream
    curl -L {{ .CRICtlURL }} | tar -xvzC /usr/bin

    {{- if ne .CloudProviderName "aws" }}
    # The normal way of setting it via cloud-init is broken:
    # https://bugs.launchpad.net/cloud-init/+bug/1662542
    hostnamectl set-hostname {{ .MachineSpec.Name }}
    {{- end }}

    if systemctl is-active firewalld; then systemctl stop firewalld; fi;
    systemctl mask firewalld

    # Coming from the upstream ansible playbook
    # https://github.com/openshift/openshift-ansible/blob/release-4.1/roles/openshift_node/defaults/main.yml#L19
    yum install -y  \
      kernel \
      irqbalance \
      microcode_ctl \
      systemd \
      selinux-policy-targeted \
      setools-console \
      dracut-network \
      passwd \
      openssh-server \
      openssh-clients \
      podman \
      skopeo \
      runc \
      containernetworking-plugins \
      nfs-utils \
      NetworkManager \
      dnsmasq \
      lvm2 \
      iscsi-initiator-utils \
      sg3_utils \
      device-mapper-multipath \
      xfsprogs \
      e2fsprogs \
      mdadm \
      cryptsetup \
      chrony \
      logrotate \
      sssd \
      shadow-utils \
      sudo \
      coreutils \
      less \
      tar \
      xz \
      gzip \
      bzip2 \
      rsync \
      tmux \
      nmap-ncat \
      net-tools \
      bind-utils \
      strace \
      bash-completion \
      vim-minimal \
      nano \
      authconfig \
      policycoreutils-python \
      iptables-services \
      bridge-utils \
      biosdevname \
      container-storage-setup \
      cloud-utils-growpart \
      ceph-common \
      cri-o \
      podman \ {{- /* # We install podman to be able to fetch the hyperkube image from the image */}}
      glusterfs-fuse{{ if eq .CloudProviderName "vsphere" }} \
      open-vm-tools{{ end }}
    {{- if eq .CloudProviderName "vsphere" }}
    systemctl enable --now vmtoolsd.service
    {{ end }}

    {{- /* We copy hyperkube from the upstream image as those are not available otherwise */}}
    {{- /* TODO: Figure out how to handle the bugfix versions. The repo only has tags for minor versions. */}}
    {{- /* We might delay decision on how to proceed here until RedHat has a release strategy for OpenShift for Fedora or CentOS. */}}
    podman run \
      -v /usr/bin:/host/usr/bin \
      -ti quay.io/openshift/origin-hyperkube:{{ .OpenShiftMinorVersion }} \
      cp /usr/bin/hyperkube /host/usr/bin/hyperkube

    systemctl enable --now cri-o
    systemctl enable --now kubelet

- path: "/opt/bin/supervise.sh"
  permissions: "0755"
  content: |
    #!/bin/bash
    set -xeuo pipefail
    while ! "$@"; do
      sleep 1
    done

- path: "/etc/kubernetes/cloud-config"
  content: |
{{ .CloudConfig | indent 4 }}

- path: "/etc/kubernetes/kubeconfig"
  content: |
{{ .Kubeconfig | indent 4 }}

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

- path: "/etc/kubernetes/kubelet.conf"
  content: |
    kind: KubeletConfiguration
    apiVersion: kubelet.config.k8s.io/v1beta1
    cgroupDriver: systemd
    clusterDNS:
    {{- range .DNSIPs }}
      - "{{ . }}"
    {{- end }}
    clusterDomain: cluster.local
    maxPods: 250
    rotateCertificates: true
    runtimeRequestTimeout: 10m
    serializeImagePulls: false
    staticPodPath: /etc/kubernetes/manifests
    systemReserved:
      cpu: 500m
      memory: 500Mi
    featureGates:
      RotateKubeletServerCertificate: true
      ExperimentalCriticalPodAnnotation: true
      SupportPodPidsLimit: true
      LocalStorageCapacityIsolation: false
    serverTLSBootstrap: true

- path: "/etc/systemd/system/kubelet.service"
  content: |
    [Unit]
    Description=Kubernetes Kubelet
    Wants=rpc-statd.service
  
    [Service]
    Type=notify
    ExecStartPre=/bin/mkdir --parents /etc/kubernetes/manifests
    ExecStartPre=/bin/rm -f /var/lib/kubelet/cpu_manager_state
    EnvironmentFile=/etc/os-release
    EnvironmentFile=-/etc/kubernetes/kubelet-workaround
    EnvironmentFile=-/etc/kubernetes/kubelet-env
  
    ExecStart=/usr/bin/hyperkube \
        kubelet \
          --config=/etc/kubernetes/kubelet.conf \
          --bootstrap-kubeconfig=/etc/kubernetes/kubeconfig \
          --kubeconfig=/var/lib/kubelet/kubeconfig \
          --container-runtime=remote \
          --container-runtime-endpoint=/var/run/crio/crio.sock \
          --allow-privileged \
          --minimum-container-ttl-duration=6m0s \
          --volume-plugin-dir=/etc/kubernetes/kubelet-plugins/volume/exec \
          --client-ca-file=/etc/kubernetes/ca.crt \
          {{- if .CloudProviderName }}
          --cloud-provider={{ .CloudProviderName }} \
          --cloud-config=/etc/kubernetes/cloud-config \
          {{- end }}
          --anonymous-auth=false \
          --v=3 \
  
    Restart=always
    RestartSec=10
  
    [Install]
    WantedBy=multi-user.target

- path: "/etc/systemd/system.conf.d/kubelet-cgroups.conf"
  content: |
    # Turning on Accounting helps track down performance issues.
    [Manager]
    DefaultCPUAccounting=yes
    DefaultMemoryAccounting=yes
    DefaultBlockIOAccounting=yes

- path: "/etc/systemd/system/kubelet.service.d/10-crio.conf"
  content: |
    [Unit]
    After=crio.service
    Requires=crio.service

- path: "/etc/containers/registries.conf"
  content: |
    [registries.search]
    registries = ['docker.io']

    [registries.insecure]
    registries = []

    [registries.block]
    registries = []

- path: "/etc/containers/storage.conf"
  content: |
    [storage]
    driver = "overlay"
    runroot = "/var/run/containers/storage"
    graphroot = "/var/lib/containers/storage"
    [storage.options]
    additionalimagestores = [
    ]
    size = ""
    override_kernel_check = "true"
    [storage.options.thinpool]

- path: "/etc/kubernetes/ca.crt"
  content: |
{{ .KubernetesCACert | indent 4 }}

runcmd:
- systemctl enable --now setup.service
`
