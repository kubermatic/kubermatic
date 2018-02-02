#!/bin/bash

# See https://kubernetes.io/docs/setup/independent/high-availability/#install-cni-network
set -eu pipefail

source ./config.sh

./install-prerequistes.sh

# Generate PKI
cfssl gencert -initca ca-csr.json | cfssljson -bare ca -

# Generate etcd client CA.
cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=client client.json | cfssljson -bare client

for ((i = 0; i < ${#ETCD_HOSTNAMES[@]}; i++)); do
        echo "Server ${i}"
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo mkdir -p /etc/kubernetes/pki/etcd"
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "mkdir -p ~/etc/kubernetes/pki/etcd"
        scp ca.pem ca-key.pem client.pem client-key.pem ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo cp -R ~/etc/kubernetes/pki/etcd/* /etc/kubernetes/pki/etcd/; sudo chown -R root:root /etc/kubernetes/pki/etcd"
done

for ((i = 0; i < ${#ETCD_HOSTNAMES[@]}; i++)); do
        echo "Server ${i}"
        cfssl print-defaults csr > "config${i}.json"
        sed -i '0,/CN/{s/example\.net/'"${ETCD_HOSTNAMES[$i]}"'/}' "config${i}.json"
        sed -i 's/www\.example\.net/'"${ETCD_PRIVATE_IPS[$i]}"'/' "config${i}.json"
        sed -i 's/example\.net/'"${ETCD_HOSTNAMES[$i]}"'/' "config${i}.json"

        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=server "config${i}.json" | cfssljson -bare "server${i}"
        cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile=peer "config${i}.json" | cfssljson -bare "peer${i}"
        scp "./config${i}.json" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/config.json
        scp "./peer${i}.csr" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/peer.csr
        scp "./peer${i}-key.pem" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/peer-key.pem
        scp "./peer${i}.pem" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/peer.pem
        scp "./server${i}.csr" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/server.csr
        scp "./server${i}-key.pem" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/server-key.pem
        scp "./server${i}.pem" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/server.pem
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo cp -R ~/etc/kubernetes/pki/etcd/* /etc/kubernetes/pki/etcd/; sudo chown -R root:root /etc/kubernetes/pki/etcd"
done

# Build etcd ring ie. etcd0=https://<etcd0-ip-address>:2380,etcd1=https://<etcd1-ip-address>:2380,etcd2=https://<etcd2-ip-address>:2380
ETCD_RING=""
for ((i = 0; i < ${#ETCD_HOSTNAMES[@]}; i++)); do
        ETCD_RING+=${ETCD_HOSTNAMES[$i]}"=https://"${ETCD_PRIVATE_IPS[$i]}":2380,"
done
ETCD_RING="$(echo ${ETCD_RING} | sed 's/[,]*$//')"
echo $ETCD_RING

NEW_ETCD_CLUSTER_TOKEN=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
export ETCD_VERSION="v3.1.10" # Suggested version for Kubernetes 1.9
curl -LO https://github.com/coreos/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz
for ((i = 0; i < ${#ETCD_HOSTNAMES[@]}; i++)); do
        scp "./etcd-${ETCD_VERSION}-linux-amd64.tar.gz" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo tar -xzvf ~/etcd-${ETCD_VERSION}-linux-amd64.tar.gz --strip-components=1 -C /usr/local/bin/"
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo rm -rf ~/etcd-${ETCD_VERSION}-linux-amd64*"
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo sh -c \"echo '' > /etc/etcd.env && echo PEER_NAME=${ETCD_HOSTNAMES[$i]} >> /etc/etcd.env && echo PRIVATE_IP=${ETCD_PRIVATE_IPS[$i]} >> /etc/etcd.env\""
        cat >etcd${i}.service <<EOL
[Unit]
Description=etcd
Documentation=https://github.com/coreos/etcd
Conflicts=etcd.service
Conflicts=etcd2.service

[Service]
EnvironmentFile=/etc/etcd.env
Type=notify
Restart=always
RestartSec=5s
LimitNOFILE=40000
TimeoutStartSec=0

ExecStart=/usr/local/bin/etcd --name ${ETCD_HOSTNAMES[$i]} \\
    --data-dir /var/lib/etcd \\
    --listen-client-urls https://${ETCD_PRIVATE_IPS[$i]}:2379 \\
    --advertise-client-urls https://${ETCD_PRIVATE_IPS[$i]}:2379 \\
    --listen-peer-urls https://${ETCD_PRIVATE_IPS[$i]}:2380 \\
    --initial-advertise-peer-urls https://${ETCD_PRIVATE_IPS[$i]}:2380 \\
    --cert-file=/etc/kubernetes/pki/etcd/server.pem \\
    --key-file=/etc/kubernetes/pki/etcd/server-key.pem \\
    --client-cert-auth \\
    --trusted-ca-file=/etc/kubernetes/pki/etcd/ca.pem \\
    --peer-cert-file=/etc/kubernetes/pki/etcd/peer.pem \\
    --peer-key-file=/etc/kubernetes/pki/etcd/peer-key.pem \\
    --peer-client-cert-auth \\
    --peer-trusted-ca-file=/etc/kubernetes/pki/etcd/ca.pem \\
    --initial-cluster ${ETCD_RING} \\
    --initial-cluster-token ${NEW_ETCD_CLUSTER_TOKEN} \\
    --initial-cluster-state new

[Install]
WantedBy=multi-user.target
EOL
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "mkdir -p ~/etc/systemd/system/"
        scp "etcd${i}.service" ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]}:~/etc/systemd/system/etcd.service
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo cp ~/etc/systemd/system/etcd.service /etc/systemd/system/etcd.service; sudo chown root:root /etc/systemd/system/etcd.service"
        # Stop and cleanup exsisting etcd Server
        ssh ${DEFAULT_LOGIN_USER}@${ETCD_PUBLIC_IPS[$i]} "sudo systemctl daemon-reload; sudo systemctl reset-failed; sudo systemctl start --no-block etcd; sudo systemctl enable etcd"
done

# Cloud Provider prerequistes
cat > ./10-hostname.conf <<EOF
[Service]
Environment="KUBELET_EXTRA_ARGS= --cloud-provider=${CLOUD_PROVIDER_FLAG} --cloud-config=/etc/kubernetes/cloud-config"
EOF

for ((i = 0; i < ${#MASTER_HOSTNAMES[@]}; i++)); do
        echo "Inject Kubelet Master Server ${i}"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo mkdir -p /etc/systemd/system/kubelet.service.d/"
        scp ./10-hostname.conf ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]}:~/10-hostname.conf
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo cp ~/10-hostname.conf /etc/systemd/system/kubelet.service.d/; sudo chown root:root /etc/systemd/system/kubelet.service.d/10-hostname.conf"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo systemctl daemon-reload"
done

for ((i = 0; i < ${#WORKER_PUBLIC_IPS[@]}; i++)); do
        echo "Inject Kubelet Master Server ${i}"
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo mkdir -p /etc/systemd/system/kubelet.service.d/"
        scp ./10-hostname.conf ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]}:~/10-hostname.conf
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo cp ~/10-hostname.conf /etc/systemd/system/kubelet.service.d/; sudo chown root:root /etc/systemd/system/kubelet.service.d/10-hostname.conf"
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo systemctl daemon-reload"
done

# Master Server Setup
for ((i = 0; i < ${#MASTER_HOSTNAMES[@]}; i++)); do
        echo "Master Server ${i}"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo mkdir -p /etc/kubernetes/pki/etcd"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "mkdir -p ~/etc/kubernetes/pki/etcd"
        scp ca.pem client.pem client-key.pem ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/etcd/
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo cp -R ~/etc/kubernetes/pki/etcd/* ~/etc/kubernetes/pki/etcd/; sudo chown root:root /etc/kubernetes/pki/etcd"
done

ETCD_RING_YAML=""
for ((i = 0; i < ${#ETCD_HOSTNAMES[@]}; i++)); do
        ETCD_RING_YAML+="  - \"https://${ETCD_PRIVATE_IPS[$i]}:2379\""$'\n'
done
echo "$ETCD_RING_YAML"

SANS_RING_YAML=""
for ((i = 0; i < ${#MASTER_LOAD_BALANCER_ADDRS[@]}; i++)); do
        SANS_RING_YAML+="- \"${MASTER_LOAD_BALANCER_ADDRS[$i]}\""$'\n'
done
for ((i = 0; i < ${#MASTER_PUBLIC_IPS[@]}; i++)); do
        SANS_RING_YAML+="- \"${MASTER_PUBLIC_IPS[$i]}\""$'\n'
done
for ((i = 0; i < ${#MASTER_PRIVATE_IPS[@]}; i++)); do
        SANS_RING_YAML+="- \"${MASTER_PRIVATE_IPS[$i]}\""$'\n'
done
echo "$SANS_RING_YAML"

scp install-kubeadm-ubuntu.sh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]}:~/etc/kubernetes/install-kubeadm-ubuntu.sh
ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "sudo cp -R ~/etc/kubernetes/* /etc/kubernetes/; sudo chown root:root /etc/kubernetes"
ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "sudo bash /etc/kubernetes/install-kubeadm-ubuntu.sh"
KUBEADM_TOKEN="$(ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "kubeadm token generate")"

cat >kubeadm-config0.yaml <<EOL
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
cloudProvider: "${CLOUD_PROVIDER_FLAG}"
kubernetesVersion: ${KUBERNETES_VERSION}
token: "${KUBEADM_TOKEN}"
tokenTTL: "0"
api:
  advertiseAddress: "${MASTER_PRIVATE_IPS[0]}"
etcd:
  endpoints:
${ETCD_RING_YAML}
  caFile: /etc/kubernetes/pki/etcd/ca.pem
  certFile: /etc/kubernetes/pki/etcd/client.pem
  keyFile: /etc/kubernetes/pki/etcd/client-key.pem
networking:
  podSubnet: "${POD_SUBNET}"
apiServerCertSANs:
${SANS_RING_YAML}
apiServerExtraArgs:
  #endpoint-reconciler-type=lease
  apiserver-count: "${#MASTER_HOSTNAMES[@]}"
  cloud-config: /etc/kubernetes/cloud-config
controllerManagerExtraArgs:
  cloud-config: /etc/kubernetes/cloud-config
EOL
scp kubeadm-config0.yaml ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]}:~/etc/kubernetes/kubeadm-config.yaml
ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "sudo cp -R ~/etc/kubernetes/* /etc/kubernetes/; sudo chown root:root /etc/kubernetes"

scp ${CLOUD_CONFIG_FILE} ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]}:~/etc/kubernetes/cloud-config
ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "sudo cp -R ~/etc/kubernetes/* /etc/kubernetes/; sudo chown root:root /etc/kubernetes"

ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "sudo kubeadm init --config=/etc/kubernetes/kubeadm-config.yaml"

# Copy generated certificates back to our machine
mkdir -p apiserver0pki || true

ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "mkdir -p ~/apiserver0pki"
ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "sudo cp -R /etc/kubernetes/pki/* ~/apiserver0pki/; sudo chown -R ${DEFAULT_LOGIN_USER}:${DEFAULT_LOGIN_USER} ~/apiserver0pki"
scp -r ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]}:~/apiserver0pki/* apiserver0pki
rm -rf ./apiserver0pki/etcd
for ((i = 1; i < ${#MASTER_HOSTNAMES[@]}; i++)); do
        echo "Copy CA to new Master Servers ${i}"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo mkdir -p /etc/kubernetes/pki/"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "mkdir -p ~/etc/kubernetes/pki/"
        scp ./apiserver0pki/* ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]}:~/etc/kubernetes/pki/
        cat >kubeadm-config${i}.yaml <<EOL
apiVersion: kubeadm.k8s.io/v1alpha1
kind: MasterConfiguration
cloudProvider: "${CLOUD_PROVIDER_FLAG}"
kubernetesVersion: ${KUBERNETES_VERSION}
token: "${KUBEADM_TOKEN}"
tokenTTL: "0"
api:
  advertiseAddress: "${MASTER_PRIVATE_IPS[$i]}"
etcd:
  endpoints:
${ETCD_RING_YAML}
  caFile: /etc/kubernetes/pki/etcd/ca.pem
  certFile: /etc/kubernetes/pki/etcd/client.pem
  keyFile: /etc/kubernetes/pki/etcd/client-key.pem
networking:
  podSubnet: "${POD_SUBNET}"
apiServerCertSANs:
${SANS_RING_YAML}
apiServerExtraArgs:
  #endpoint-reconciler-type=lease
  apiserver-count: "${#MASTER_HOSTNAMES[@]}"
  cloud-config: /etc/kubernetes/cloud-config
controllerManagerExtraArgs:
  cloud-config: /etc/kubernetes/cloud-config
EOL
        scp ./kubeadm-config${i}.yaml ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]}:~/etc/kubernetes/kubeadm-config.yaml
        scp ./install-kubeadm-ubuntu.sh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]}:~/etc/kubernetes/install-kubeadm-ubuntu.sh
        scp ${CLOUD_CONFIG_FILE} ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]}:~/etc/kubernetes/cloud-config
        scp ${CLOUD_CONFIG_FILE} ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]}:~/etc/kubernetes/cloud-config
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo cp -R ~/etc/kubernetes/* /etc/kubernetes/; sudo chown root:root /etc/kubernetes"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "rm -rf ~/apiserver0pki"

        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo bash /etc/kubernetes/install-kubeadm-ubuntu.sh"
        ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo kubeadm init --config=/etc/kubernetes/kubeadm-config.yaml"
done

ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "sudo cp /etc/kubernetes/admin.conf ~/admin.conf; sudo chown -R ${DEFAULT_LOGIN_USER}:${DEFAULT_LOGIN_USER} ~/admin.conf"
scp ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]}:~/admin.conf kubeconfig
ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[0]} "rm ~/admin.conf"
sed -i -e 's/'"${MASTER_PRIVATE_IPS[0]}"'/'"${MASTER_LOAD_BALANCER_ADDRS[0]}"'/g' kubeconfig
# Wait for LB to be ready.
for (( i = 0; i < 10; i++ )); do
    kubectl apply -f https://raw.githubusercontent.com/projectcalico/canal/master/k8s-install/1.7/rbac.yaml --kubeconfig=kubeconfig && \
    kubectl apply -f https://raw.githubusercontent.com/projectcalico/canal/master/k8s-install/1.7/canal.yaml --kubeconfig=kubeconfig && \
    break || sleep 20;
done

# Switch to LB
echo "Switch Proxy to LB addr"
kubectl get configmap -n kube-system kube-proxy -o yaml --kubeconfig=kubeconfig > kube-proxy.yaml
sed -i -e 's#server:.*#server: https://'"${MASTER_LOAD_BALANCER_ADDRS[0]}"':6443#g' kube-proxy.yaml
kubectl apply -f kube-proxy.yaml --force --kubeconfig=kubeconfig
# restart all kube-proxy pods to ensure that they load the new configmap
kubectl delete pod -n kube-system -l k8s-app=kube-proxy --kubeconfig=kubeconfig

# Setup Workers
for ((i = 0; i < ${#WORKER_PUBLIC_IPS[@]}; i++)); do
        echo "Create Worker $i"
        TOKEN=$(ssh ${DEFAULT_LOGIN_USER}@${MASTER_PUBLIC_IPS[$i]} "sudo kubeadm token create --print-join-command")

        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "mkdir -p ~/etc/kubernetes"
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo mkdir -p /etc/kubernetes"
        scp ./install-kubeadm-ubuntu.sh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]}:~/etc/kubernetes/install-kubeadm-ubuntu.sh
        scp ${CLOUD_CONFIG_FILE} ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]}:~/etc/kubernetes/cloud-config
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo cp ~/etc/kubernetes/* /etc/kubernetes/; sudo chown -R root:root /etc/kubernetes/"

        scp ./10-hostname.conf ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]}:~/10-hostname.conf
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo mkdir -p /etc/systemd/system/kubelet.service.d/"
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo cp ~/10-hostname.conf /etc/systemd/system/kubelet.service.d/; sudo chown -R root:root /etc/systemd/system/kubelet.service.d/"

        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo bash /etc/kubernetes/install-kubeadm-ubuntu.sh"
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo ${TOKEN}"
        # # TODO(realfake) On all workers:
        for (( retry_count = 0; retry_count < 10; retry_count++ )); do ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo sed -i 's#server:.*#server: https://'"${MASTER_LOAD_BALANCER_ADDRS[0]}"':6443#g' /etc/kubernetes/kubelet.conf" && break || sleep 10; done
        ssh ${DEFAULT_LOGIN_USER}@${WORKER_PUBLIC_IPS[$i]} "sudo systemctl restart kubelet"
done
