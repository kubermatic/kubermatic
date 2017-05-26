package nanny

var kubeletTemplate = `[Unit]
Description=Kubernetes Kubelet

[Service]
Restart=always
RestartSec=10
Environment="PATH=/opt/bin:/usr/bin:/usr/sbin:$PATH"
ExecStartPre=/usr/bin/mkdir -p /var/lib/kubelet /etc/kubernetes/manifests
ExecStartPre=/usr/bin/curl -L -o /var/lib/kubelet/kubelet https://storage.googleapis.com/kubernetes-release/release/v1.5.4/bin/linux/amd64/kubelet
ExecStartPre=/usr/bin/chmod +x /var/lib/kubelet/kubelet
ExecStartPre=/usr/bin/mkdir -p /opt/bin
ExecStartPre=/usr/bin/curl -L -o /opt/bin/socat https://s3-eu-west-1.amazonaws.com/kubermatic/coreos/socat
ExecStartPre=/usr/bin/chmod +x /opt/bin/socat
ExecStart=/var/lib/kubelet/kubelet \
  --address=0.0.0.0 \
  --kubeconfig=/var/run/kubelet/kubeconfig \
  --require-kubeconfig \
  --cluster-dns=10.10.10.10 \
  --cluster-domain=cluster.local \
  --allow-privileged=true \
  --hostname-override="{{.Name}}" \
  --network-plugin=cni

[Install]
WantedBy=multi-user.target`
