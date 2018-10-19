package helper

import (
	"bytes"
	"fmt"
	"net"
	"text/template"
)

const (
	kubeletFlagsTpl = `--bootstrap-kubeconfig=/etc/kubernetes/bootstrap-kubelet.conf \
--kubeconfig=/etc/kubernetes/kubelet.conf \
--pod-manifest-path=/etc/kubernetes/manifests \
--allow-privileged=true \
--network-plugin=cni \
--cni-conf-dir=/etc/cni/net.d \
--cni-bin-dir=/opt/cni/bin \
--authorization-mode=Webhook \
--client-ca-file=/etc/kubernetes/pki/ca.crt \
{{- if semverCompare "<1.12.0-0" .KubeletVersion }}
--cadvisor-port=0 \
{{- end }}
--rotate-certificates=true \
--cert-dir=/etc/kubernetes/pki \
--authentication-token-webhook=true \
{{- if .CloudProvider }}
--cloud-provider={{ .CloudProvider }} \
--cloud-config=/etc/kubernetes/cloud-config \
{{- end }}
{{- if .Hostname }}
--hostname-override={{ .Hostname }} \
{{- end }}
--read-only-port=0 \
--exit-on-lock-contention \
--lock-file=/tmp/kubelet.lock \
--anonymous-auth=false \
--protect-kernel-defaults=true \
--cluster-dns={{ .ClusterDNSIPs | join "," }} \
--cluster-domain=cluster.local`

	kubeletSystemdUnitTpl = `[Unit]
After=docker.service
Requires=docker.service

Description=kubelet: The Kubernetes Node Agent
Documentation=https://kubernetes.io/docs/home/

[Service]
Restart=always
StartLimitInterval=0
RestartSec=10

Environment="PATH=/opt/bin:/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin/"

ExecStart=/opt/bin/kubelet $KUBELET_EXTRA_ARGS \
{{ kubeletFlags .KubeletVersion .CloudProvider .Hostname .ClusterDNSIPs | indent 2 }}

[Install]
WantedBy=multi-user.target`
)

// KubeletSystemdUnit returns the systemd unit for the kubelet
func KubeletSystemdUnit(kubeletVersion, cloudProvider, hostname string, dnsIPs []net.IP) (string, error) {
	tmpl, err := template.New("kubelet-systemd-unit").Funcs(TxtFuncMap()).Parse(kubeletSystemdUnitTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse kubelet-systemd-unit template: %v", err)
	}

	data := struct {
		KubeletVersion string
		CloudProvider  string
		Hostname       string
		ClusterDNSIPs  []net.IP
	}{
		KubeletVersion: kubeletVersion,
		CloudProvider:  cloudProvider,
		Hostname:       hostname,
		ClusterDNSIPs:  dnsIPs,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute kubelet-systemd-unit template: %v", err)
	}

	return string(b.String()), nil
}

// KubeletFlags returns the kubelet flags
func KubeletFlags(version, cloudProvider, hostname string, dnsIPs []net.IP) (string, error) {
	tmpl, err := template.New("kubelet-flags").Funcs(TxtFuncMap()).Parse(kubeletFlagsTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse kubelet-flags template: %v", err)
	}

	data := struct {
		CloudProvider  string
		Hostname       string
		ClusterDNSIPs  []net.IP
		KubeletVersion string
	}{
		CloudProvider:  cloudProvider,
		Hostname:       hostname,
		ClusterDNSIPs:  dnsIPs,
		KubeletVersion: version,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute kubelet-flags template: %v", err)
	}

	return string(b.String()), nil
}

func KubeletHealthCheckSystemdUnit() string {
	return `[Unit]
Requires=kubelet.service
After=kubelet.service

[Service]
ExecStart=/opt/bin/health-monitor.sh kubelet

[Install]
WantedBy=multi-user.target
`
}

func ContainerRuntimeHealthCheckSystemdUnit() string {
	return `[Unit]
Requires=docker.service
After=docker.service

[Service]
ExecStart=/opt/bin/health-monitor.sh container-runtime

[Install]
WantedBy=multi-user.target
`
}
