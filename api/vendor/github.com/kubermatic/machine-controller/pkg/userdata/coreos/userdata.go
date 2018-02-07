package coreos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"text/template"

	"github.com/Masterminds/semver"
	ctconfig "github.com/coreos/container-linux-config-transpiler/config"
	machinesv1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	machinetemplate "github.com/kubermatic/machine-controller/pkg/template"
	"github.com/kubermatic/machine-controller/pkg/userdata/cloud"
	"k8s.io/apimachinery/pkg/runtime"
)

type Provider struct{}

type Config struct {
	DisableAutoUpdate bool `json:"disableAutoUpdate"`
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

func (p Provider) SupportedContainerRuntimes() (runtimes []machinesv1alpha1.ContainerRuntimeInfo) {
	return []machinesv1alpha1.ContainerRuntimeInfo{
		{
			Name:    "docker",
			Version: "1.12",
		},
		{
			Name:    "docker",
			Version: "1.12.6",
		},
		{
			Name:    "docker",
			Version: "17.09",
		},
		{
			Name:    "docker",
			Version: "17.09.0",
		},
	}
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

	coreosConfig, err := getConfig(pconfig.OperatingSystemSpec)
	if err != nil {
		return "", fmt.Errorf("failed to get coreos config from provider config: %v", err)
	}

	data := struct {
		MachineSpec       machinesv1alpha1.MachineSpec
		ProviderConfig    *providerconfig.Config
		CoreOSConfig      *Config
		Kubeconfig        string
		CloudProvider     string
		CloudConfig       string
		HyperkubeImageTag string
		ClusterDNSIPs     []net.IP
	}{
		MachineSpec:       spec,
		ProviderConfig:    pconfig,
		CoreOSConfig:      coreosConfig,
		Kubeconfig:        kubeconfig,
		CloudProvider:     cpName,
		CloudConfig:       cpConfig,
		HyperkubeImageTag: fmt.Sprintf("v%s_coreos.0", kubeletVersion.String()),
		ClusterDNSIPs:     clusterDNSIPs,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}

	// Convert to ignition
	cfg, ast, report := ctconfig.Parse(b.Bytes())
	if len(report.Entries) > 0 {
		return "", fmt.Errorf("failed to validate coreos cloud config: %s", report.String())
	}

	ignCfg, report := ctconfig.Convert(cfg, "", ast)
	if len(report.Entries) > 0 {
		return "", fmt.Errorf("failed to convert container linux config to ignition: %s", report.String())
	}

	out, err := json.MarshalIndent(ignCfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal ignition config: %v", err)
	}

	return string(out), nil
}

const ctTemplate = `
passwd:
  users:
    - name: core
      ssh_authorized_keys:
        {{range .ProviderConfig.SSHPublicKeys}}- {{.}}
        {{end}}

systemd:
  units:
{{ if .CoreOSConfig.DisableAutoUpdate }}
    - name: update-engine.service
      mask: true
    - name: locksmithd.service
      mask: true
{{ end }}
    - name: docker.service
      enabled: true

    - name: kubelet.service
      enabled: true
      dropins:
      - name: 40-docker.conf
        contents: |
          [Unit]
          Requires=docker.service
          After=docker.service
      contents: |
        [Unit]
        Description=Kubernetes Kubelet
        Requires=docker.service
        After=docker.service
        [Service]
        TimeoutStartSec=5min
        Environment=KUBELET_IMAGE_TAG={{ .HyperkubeImageTag }}
        Environment="RKT_RUN_ARGS=--uuid-file-save=/var/cache/kubelet-pod.uuid \
          --volume=resolv,kind=host,source=/etc/resolv.conf \
          --mount volume=resolv,target=/etc/resolv.conf \
          --volume cni-bin,kind=host,source=/opt/cni/bin \
          --mount volume=cni-bin,target=/opt/cni/bin \
          --volume cni-conf,kind=host,source=/etc/cni/net.d \
          --mount volume=cni-conf,target=/etc/cni/net.d \
          --volume etc-kubernetes,kind=host,source=/etc/kubernetes \
          --mount volume=etc-kubernetes,target=/etc/kubernetes \
          --volume var-log,kind=host,source=/var/log \
          --mount volume=var-log,target=/var/log"
        ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests
        ExecStartPre=/bin/mkdir -p /etc/cni/net.d
        ExecStartPre=/bin/mkdir -p /opt/cni/bin
        ExecStartPre=-/usr/bin/rkt rm --uuid-file=/var/cache/kubelet-pod.uuid
        ExecStart=/usr/lib/coreos/kubelet-wrapper \
          --container-runtime=docker \
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
        ExecStop=-/usr/bin/rkt stop --uuid-file=/var/cache/kubelet-pod.uuid
        Restart=always
        RestartSec=10
        [Install]
        WantedBy=multi-user.target

storage:
  files:
    - path: /etc/kubernetes/bootstrap.kubeconfig
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

{{- if contains "1.12" .MachineSpec.Versions.ContainerRuntime.Version }}
    - path: /etc/coreos/docker-1.12
      mode: 0644
      filesystem: root
      contents:
        inline: |
          yes
{{ end }}
`
