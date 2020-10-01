export MASTER_ENDPOINT="https://etcd-0.etcd.cluster-lg69pmx8wf.svc.cluster.local:2379"

export INITIAL_STATE="new"
export INITIAL_CLUSTER="etcd-0=http://etcd-0.etcd.cluster-lg69pmx8wf.svc.cluster.local:2380,etcd-1=http://etcd-1.etcd.cluster-lg69pmx8wf.svc.cluster.local:2380,etcd-2=http://etcd-2.etcd.cluster-lg69pmx8wf.svc.cluster.local:2380"

echo "initial-state: ${INITIAL_STATE}"
echo "initial-cluster: ${INITIAL_CLUSTER}"

exec /usr/local/bin/etcd \
    --name=${POD_NAME} \
    --data-dir="/var/run/etcd/pod_${POD_NAME}/" \
    --initial-cluster=${INITIAL_CLUSTER} \
    --initial-cluster-token="lg69pmx8wf" \
    --initial-cluster-state=${INITIAL_STATE} \
    --advertise-client-urls "https://${POD_NAME}.etcd.cluster-lg69pmx8wf.svc.cluster.local:2379,https://${POD_IP}:2379" \
    --listen-client-urls "https://${POD_IP}:2379,https://127.0.0.1:2379" \
    --listen-peer-urls "http://${POD_IP}:2380" \
    --listen-metrics-urls "http://${POD_IP}:2378,http://127.0.0.1:2378" \
    --initial-advertise-peer-urls "http://${POD_NAME}.etcd.cluster-lg69pmx8wf.svc.cluster.local:2380" \
    --trusted-ca-file /etc/etcd/pki/ca/ca.crt \
    --client-cert-auth \
    --cert-file /etc/etcd/pki/tls/etcd-tls.crt \
    --key-file /etc/etcd/pki/tls/etcd-tls.key \
    --experimental-initial-corrupt-check=true \
    --experimental-corrupt-check-time=10m \
    --auto-compaction-retention=8
