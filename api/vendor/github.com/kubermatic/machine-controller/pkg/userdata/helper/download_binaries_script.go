package helper

import (
	"bytes"
	"fmt"
	"text/template"
)

const (
	downloadBinariesTpl = `#setup some common directories
mkdir -p /opt/bin/
mkdir -p /var/lib/calico
mkdir -p /etc/kubernetes/manifests
mkdir -p /etc/cni/net.d
mkdir -p /opt/cni/bin

# docker
if [ ! -f /opt/bin/docker ]; then
    # TODO: Support newer versions. Make sure to validate if the newer containerd version has configuration changes
    # Newer versions of docker use containterd with a config file - containerd from 17.03 does not support that
    # we maybe need to manage both ways(flags & config) or wait until we deprecate Kubernetes 1.11 
    curl -L http://download.docker.com/linux/static/stable/x86_64/docker-17.03.2-ce.tgz | tar -xvzC /opt/ -f -
    mv /opt/docker/* /opt/bin/
    rm -rf /opt/docker
fi

# cni
if [ ! -f /opt/cni/bin/loopback ]; then
    curl -L https://github.com/containernetworking/plugins/releases/download/v0.6.0/cni-plugins-amd64-v0.6.0.tgz | tar -xvzC /opt/cni/bin -f -
fi

{{- if .DownloadKubelet }}
# kubelet
if [ ! -f /opt/bin/kubelet ]; then
    curl -Lfo /opt/bin/kubelet https://storage.googleapis.com/kubernetes-release/release/v{{ .KubeletVersion }}/bin/linux/amd64/kubelet
    chmod +x /opt/bin/kubelet
fi
{{- end }}

if [[ ! -x /opt/bin/health-monitor.sh ]]; then
    curl -Lfo /opt/bin/health-monitor.sh https://raw.githubusercontent.com/kubermatic/machine-controller/8b5b66e4910a6228dfaecccaa0a3b05ec4902f8e/pkg/userdata/scripts/health-monitor.sh
    chmod +x /opt/bin/health-monitor.sh
fi
`
)

// DownloadBinariesScript returns the script which is responsible to download all required binaries.
// Extracted into a dedicated function so we can use it to prepare custom images: TODO: Use it to prepare custom images...
func DownloadBinariesScript(kubeletVersion string, downloadKubelet bool) (string, error) {
	tmpl, err := template.New("download-binaries").Funcs(TxtFuncMap()).Parse(downloadBinariesTpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse download-binaries template: %v", err)
	}

	data := struct {
		KubeletVersion  string
		DownloadKubelet bool
	}{
		KubeletVersion:  kubeletVersion,
		DownloadKubelet: downloadKubelet,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute download-binaries template: %v", err)
	}

	return b.String(), nil
}
