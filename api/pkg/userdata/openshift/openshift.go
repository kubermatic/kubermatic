package openshift

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
	cloudConfig string,
	cloudProviderName string,
	clusterDNSIPs []net.IP,
	_ bool) (string, error) {

	tmpl, err := template.New("user-data").Funcs(userdatahelper.TxtFuncMap()).Parse(ctTemplate)
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

	osConfig, err := getConfig(pconfig.OperatingSystemSpec)
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
	}{
		MachineSpec:      spec,
		ProviderSpec:     pconfig,
		OSConfig:         osConfig,
		CloudProvider:    cloudProviderName,
		CloudConfig:      cloudConfig,
		KubeletVersion:   kubeletVersion.String(),
		ClusterDNSIPs:    clusterDNSIPs,
		ServerAddr:       serverAddr,
		Kubeconfig:       kubeconfigString,
		KubernetesCACert: kubernetesCACert,
	}
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}
	return b.String(), nil
}

const ctTemplate = `#cloud-config
{{ if ne .CloudProvider "aws" }}
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
 - path: /etc/systemd/system/origin-node.service
  content: |
    [Unit]
    Description=OpenShift Node
    After=docker.service
    After=chronyd.service
    After=ntpd.service
    Wants=docker.service
    Documentation=https://github.com/openshift/origin
    Wants=dnsmasq.service
    After=dnsmasq.service
     [Service]
    Type=notify
    EnvironmentFile=/etc/sysconfig/origin-node
    ExecStart=/usr/local/bin/openshift-node
    LimitNOFILE=65536
    LimitCORE=infinity
    WorkingDirectory=/var/lib/origin/
    SyslogIdentifier=origin-node
    Restart=always
    RestartSec=5s
    TimeoutStartSec=300
    OOMScoreAdjust=-999
     [Install]
    WantedBy=multi-user.target
 - path: "/etc/dnsmasq.d/origin-dns.conf"
  content: |
    no-resolv
    domain-needed
    no-negcache
    max-cache-ttl=1
    enable-dbus
    dns-forward-max=10000
    cache-size=10000
    bind-dynamic
    min-port=1024
    except-interface=lo
 - path: "/etc/NetworkManager/dispatcher.d/99-origin-dns.sh"
  permissions: "0755"
  content: |
    #!/bin/bash -x
    # -*- mode: sh; sh-indentation: 2 -*-
     # This NetworkManager dispatcher script replicates the functionality of
    # NetworkManager's dns=dnsmasq  however, rather than hardcoding the listening
    # address and /etc/resolv.conf to 127.0.0.1 it pulls the IP address from the
    # interface that owns the default route. This enables us to then configure pods
    # to use this IP address as their only resolver, where as using 127.0.0.1 inside
    # a pod would fail.
    #
    # To use this,
    # - If this host is also a master, reconfigure master dnsConfig to listen on
    #   8053 to avoid conflicts on port 53 and open port 8053 in the firewall
    # - Drop this script in /etc/NetworkManager/dispatcher.d/
    # - systemctl restart NetworkManager
    #
    # Test it:
    # host kubernetes.default.svc.cluster.local
    # host google.com
    #
    # TODO: I think this would be easy to add as a config option in NetworkManager
    # natively, look at hacking that up
     cd /etc/sysconfig/network-scripts
    . ./network-functions
     [ -f ../network ] && . ../network
     if [[ $2 =~ ^(up|dhcp4-change|dhcp6-change)$ ]]; then
    	# If the origin-upstream-dns config file changed we need to restart
    	NEEDS_RESTART=0
    	UPSTREAM_DNS='/etc/dnsmasq.d/origin-upstream-dns.conf'
    	# We'll regenerate the dnsmasq origin config in a temp file first
    	UPSTREAM_DNS_TMP=$(mktemp)
    	UPSTREAM_DNS_TMP_SORTED=$(mktemp)
    	CURRENT_UPSTREAM_DNS_SORTED=$(mktemp)
    	NEW_RESOLV_CONF=$(mktemp)
    	NEW_NODE_RESOLV_CONF=$(mktemp)
     	######################################################################
    	# couldn't find an existing method to determine if the interface owns the
    	# default route
    	def_route=$(/sbin/ip route list match 0.0.0.0/0 | awk '{print $3 }')
    	def_route_int=$(/sbin/ip route get to ${def_route} | awk -F 'dev' '{print $2}' | head -n1 | awk '{print $1}')
    	def_route_ip=$(/sbin/ip route get to ${def_route}  | awk -F 'src' '{print $2}' | head -n1 | awk '{print $1}')
    	if [[ ${DEVICE_IFACE} == ${def_route_int} ]]; then
    		if [ ! -f /etc/dnsmasq.d/origin-dns.conf ]; then
    			cat << EOF > /etc/dnsmasq.d/origin-dns.conf
    no-resolv
    domain-needed
    server=/cluster.local/172.30.0.1
    server=/30.172.in-addr.arpa/172.30.0.1
    enable-dbus
    dns-forward-max=5000
    cache-size=5000
    min-port=1024
    EOF
    			# New config file, must restart
    			NEEDS_RESTART=1
    		fi
     		# If network manager doesn't know about the nameservers then the best
    		# we can do is grab them from /etc/resolv.conf but only if we've got no
    		# watermark
    		if ! grep -q '99-origin-dns.sh' /etc/resolv.conf; then
    			if [[ -z "${IP4_NAMESERVERS}" || "${IP4_NAMESERVERS}" == "${def_route_ip}" ]]; then
    						IP4_NAMESERVERS=$(grep '^nameserver[[:blank:]]' /etc/resolv.conf | awk '{ print $2 }')
    			fi
    			######################################################################
    			# Write out default nameservers for /etc/dnsmasq.d/origin-upstream-dns.conf
    			# and /etc/origin/node/resolv.conf in their respective formats
    			for ns in ${IP4_NAMESERVERS}; do
    				if [[ ! -z $ns ]]; then
    					echo "server=${ns}" >> $UPSTREAM_DNS_TMP
    					echo "nameserver ${ns}" >> $NEW_NODE_RESOLV_CONF
    				fi
    			done
    			# Sort it in case DNS servers arrived in a different order
    			sort $UPSTREAM_DNS_TMP > $UPSTREAM_DNS_TMP_SORTED
    			sort $UPSTREAM_DNS > $CURRENT_UPSTREAM_DNS_SORTED
    			# Compare to the current config file (sorted)
    			NEW_DNS_SUM=$(md5sum ${UPSTREAM_DNS_TMP_SORTED} | awk '{print $1}')
    			CURRENT_DNS_SUM=$(md5sum ${CURRENT_UPSTREAM_DNS_SORTED} | awk '{print $1}')
    			if [ "${NEW_DNS_SUM}" != "${CURRENT_DNS_SUM}" ]; then
    				# DNS has changed, copy the temp file to the proper location (-Z
    				# sets default selinux context) and set the restart flag
    				cp -Z $UPSTREAM_DNS_TMP $UPSTREAM_DNS
    				NEEDS_RESTART=1
    			fi
    			# compare /etc/origin/node/resolv.conf checksum and replace it if different
    			NEW_NODE_RESOLV_CONF_MD5=$(md5sum ${NEW_NODE_RESOLV_CONF})
    			OLD_NODE_RESOLV_CONF_MD5=$(md5sum /etc/origin/node/resolv.conf)
    			if [ "${NEW_NODE_RESOLV_CONF_MD5}" != "${OLD_NODE_RESOLV_CONF_MD5}" ]; then
    				cp -Z $NEW_NODE_RESOLV_CONF /etc/origin/node/resolv.conf
    			fi
    		fi
     		if ! $(systemctl -q is-active dnsmasq.service); then
    			NEEDS_RESTART=1
    		fi
     		######################################################################
    		if [ "${NEEDS_RESTART}" -eq "1" ]; then
    			systemctl restart dnsmasq
    		fi
     		# Only if dnsmasq is running properly make it our only nameserver and place
    		# a watermark on /etc/resolv.conf
    		if $(systemctl -q is-active dnsmasq.service); then
    			if ! grep -q '99-origin-dns.sh' /etc/resolv.conf; then
    					echo "# nameserver updated by /etc/NetworkManager/dispatcher.d/99-origin-dns.sh" >> ${NEW_RESOLV_CONF}
    			fi
    			sed -e '/^nameserver.*$/d' /etc/resolv.conf >> ${NEW_RESOLV_CONF}
    			echo "nameserver "${def_route_ip}"" >> ${NEW_RESOLV_CONF}
    			if ! grep -qw search ${NEW_RESOLV_CONF}; then
    				echo 'search cluster.local' >> ${NEW_RESOLV_CONF}
    			elif ! grep -q 'search cluster.local' ${NEW_RESOLV_CONF}; then
    				# cluster.local should be in first three DNS names so that glibc resolver would work
    				sed -i -e 's/^search[[:blank:]]\(.\+\)\( cluster\.local\)\{0,1\}$/search cluster.local \1/' ${NEW_RESOLV_CONF}
    			fi
    			cp -Z ${NEW_RESOLV_CONF} /etc/resolv.conf
    		fi
    	fi
     	# Clean up after yourself
    	rm -f $UPSTREAM_DNS_TMP $UPSTREAM_DNS_TMP_SORTED $CURRENT_UPSTREAM_DNS_SORTED $NEW_RESOLV_CONF
    fi
 - path: "/opt/bin/setup"
  permissions: "0777"
  content: |
    #!/bin/bash
    set -xeuo pipefail
     # As we added some modules and don't want to reboot, restart the service
    systemctl restart systemd-modules-load.service
    sysctl --system
     # Create workdir for origin-node and load its unitfile
    mkdir -p /var/lib/origin && systemctl daemon-reload
     {{ if ne .CloudProvider "aws" }}
    # The normal way of setting it via cloud-init is broken:
    # https://bugs.launchpad.net/cloud-init/+bug/1662542
    hostnamectl set-hostname {{ .MachineSpec.Name }}
    {{ end }}
     mkdir -p /etc/dnsmasq.d
    if ! [[ -e /etc/dnsmasq.d/origin-upstream-dns.conf ]]; then
      cat /etc/resolv.conf |grep '^nameserver'|awk '{print $2}'|xargs -n 1 -I ^ echo 'server=^' >> /etc/dnsmasq.d/origin-upstream-dns.conf
    fi
     systemctl stop firewalld && systemctl mask firewalld
     yum install -y centos-release-openshift-origin311
    yum install -y \
      docker \
      dnsmasq \
      origin-clients \
      origin-hyperkube \
      origin-node \
      ebtables \
      ethtool \
      nfs-utils \
      bash-completion \
      sudo \
      socat \
      wget \
      curl \
      NetworkManager \
      ipvsadm{{ if eq .CloudProvider "vsphere" }} \
      open-vm-tools{{ end }}
     mkdir -p /etc/origin/node/pods
     {{- if eq .CloudProvider "vsphere" }}
    systemctl enable --now vmtoolsd.service
    {{ end }}
    systemctl enable --now NetworkManager
    systemctl enable --now origin-node.service
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
 - path: "/etc/origin/node/bootstrap.kubeconfig"
  content: |
{{ .Kubeconfig | indent 4 }}
 - path: "/etc/sysconfig/origin-node"
  content: |
    OPTIONS=
    DEBUG_LOGLEVEL=2
    IMAGE_VERSION=v3.11
    KUBECONFIG=/etc/origin/node/bootstrap.kubeconfig
    BOOTSTRAP_CONFIG_NAME=node-config-compute
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
 - path: "/usr/local/bin/openshift-node"
  permissions: "0755"
  content: |
    #!/bin/sh
     # This launches the Kubelet by converting the node configuration into kube flags.
     set -euo pipefail
     if ! [[ -f /etc/origin/node/client-ca.crt ]]; then
    	if [[ -f /etc/origin/node/bootstrap.kubeconfig ]]; then
    		oc config --config=/etc/origin/node/bootstrap.kubeconfig view --raw --minify -o go-template='{{ "{{index .clusters 0 \"cluster\" \"certificate-authority-data\" }}" }}' | base64 -d - > /etc/origin/node/client-ca.crt
    	fi
    fi
    config=/etc/origin/node/bootstrap-node-config.yaml
    # TODO: remove when dynamic kubelet config is delivered
    if [[ -f /etc/origin/node/node-config.yaml ]]; then
    	config=/etc/origin/node/node-config.yaml
    fi
    flags=$( /usr/bin/openshift-node-config "--config=${config}" )
    eval "exec /usr/bin/hyperkube kubelet --v=${DEBUG_LOGLEVEL:-2} ${flags}"
 - path: "/etc/origin/node/client-ca.crt"
  content: |
{{ .KubernetesCACert | indent 4 }}
 - path: "/etc/origin/node/bootstrap-node-config.yaml"
  permissions: "0644"
  content: |
    kind: NodeConfig
    apiVersion: v1
    authConfig:
      authenticationCacheSize: 1000
      authenticationCacheTTL: 5m
      authorizationCacheSize: 1000
      authorizationCacheTTL: 5m
    dnsBindAddress: "127.0.0.1:53"
    dnsDomain: cluster.local
    dnsIP: 0.0.0.0
    dnsNameservers: null
    dnsRecursiveResolvConf: /etc/origin/node/resolv.conf
    dockerConfig:
      dockerShimRootDirectory: /var/lib/dockershim
      dockerShimSocket: /var/run/dockershim.sock
      execHandlerName: native
    enableUnidling: true
    imageConfig:
      format: "docker.io/openshift/origin-${component}:${version}"
      latest: false
    iptablesSyncPeriod: "30s"
    kubeletArguments:
      pod-manifest-path:
      - /etc/origin/node/pods
      bootstrap-kubeconfig:
      - /etc/origin/node/bootstrap.kubeconfig
      feature-gates:
      - RotateKubeletClientCertificate=true,RotateKubeletServerCertificate=true
      rotate-certificates:
      - "true"
      cert-dir:
      - /etc/origin/node/certificates
      enable-controller-attach-detach:
      - 'true'
    masterClientConnectionOverrides:
      acceptContentTypes: application/vnd.kubernetes.protobuf,application/json
      burst: 40
      contentType: application/vnd.kubernetes.protobuf
      qps: 20
    masterKubeConfig: node.kubeconfig
    networkConfig:
      mtu: 1450
      networkPluginName: redhat/openshift-ovs-subnet
    servingInfo:
      bindAddress: 0.0.0.0:10250
      bindNetwork: tcp4
      clientCA: client-ca.crt
    proxyArguments:
      cluster-cidr:
        - 10.128.0.0/14
    volumeConfig:
      localQuota:
        perFSGroup: null
    volumeDirectory: /var/lib/origin/openshift.local.volumes
 runcmd:
- systemctl enable --now setup.service
`
