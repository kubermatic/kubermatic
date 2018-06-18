package etcd

import (
	"testing"
)

func TestGetEtcdCommand(t *testing.T) {

	tests := []struct {
		name             string
		clusterName      string
		clusterNamespace string
		migrate          bool
		expected         string
	}{
		{
			name:             "test new cluster without migration",
			clusterName:      "lg69pmx8wf",
			clusterNamespace: "cluster-lg69pmx8wf",
			migrate:          false,
			expected:         noMigration,
		},
		{
			name:             "test existing cluster with migration",
			clusterName:      "62m9k9tqlm",
			clusterNamespace: "cluster-62m9k9tqlm",
			migrate:          true,
			expected:         migration,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args, err := getEtcdCommand(test.clusterName, test.clusterNamespace, test.migrate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(args) != 3 {
				t.Fatalf("got less arguments than expected. got %d expected %d", len(args), 3)
			}
			cmd := args[2]
			if cmd != test.expected {
				t.Errorf("expected \n%s \n\nGot:\n\n%s", test.expected, cmd)
			}
		})
	}
}

var (
	noMigration = `ETCDCTL_API=3
MASTER_ENDPOINT="http://etcd-0.etcd.cluster-lg69pmx8wf.svc.cluster.local:2379"


INITIAL_STATE="new"
INITIAL_CLUSTER="etcd-0=http://etcd-0.etcd.cluster-lg69pmx8wf.svc.cluster.local:2380,etcd-1=http://etcd-1.etcd.cluster-lg69pmx8wf.svc.cluster.local:2380,etcd-2=http://etcd-2.etcd.cluster-lg69pmx8wf.svc.cluster.local:2380"


echo ${INITIAL_STATE}
echo ${INITIAL_CLUSTER}

exec /usr/local/bin/etcd \
    --name=${POD_NAME} \
    --data-dir="/var/run/etcd/pod_${POD_NAME}/" \
    --heartbeat-interval=500 \
    --election-timeout=5000 \
    --initial-cluster=${INITIAL_CLUSTER} \
    --initial-cluster-token="lg69pmx8wf" \
    --initial-cluster-state=${INITIAL_STATE} \
    --advertise-client-urls http://${POD_NAME}.etcd.cluster-lg69pmx8wf.svc.cluster.local:2379 \
    --listen-client-urls http://0.0.0.0:2379 \
    --listen-peer-urls http://0.0.0.0:2380
`

	migration = `ETCDCTL_API=3
MASTER_ENDPOINT="http://etcd-0.etcd.cluster-62m9k9tqlm.svc.cluster.local:2379"


# If we're already initialized
if [ -d "/var/run/etcd/pod_${POD_NAME}/" ]; then
    INITIAL_STATE="existing"
    INITIAL_CLUSTER="etcd-0=http://etcd-0.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380,etcd-1=http://etcd-1.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380,etcd-2=http://etcd-2.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380"
else
    if [ "${POD_NAME}" = "etcd-0" ]; then
        echo "i'm etcd-0. I do the restore"
        etcdctl --endpoints http://etcd-cluster-client:2379 snapshot save snapshot.db
        etcdctl snapshot restore snapshot.db \
            --name etcd-0 \
            --data-dir="/var/run/etcd/pod_${POD_NAME}/" \
            --initial-cluster="etcd-0=http://etcd-0.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380" \
            --initial-cluster-token="62m9k9tqlm" \
            --initial-advertise-peer-urls http://etcd-0.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380
        INITIAL_STATE="new"
        INITIAL_CLUSTER="etcd-0=http://etcd-0.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380"
    fi

    if [ "${POD_NAME}" = "etcd-1" ]; then
        echo "i'm etcd-1. I join as new member as soon as etcd-0 comes up"
        etcdctl --endpoints ${MASTER_ENDPOINT} member add etcd-1 --peer-urls=http://etcd-1.etcd.cluster-62m9k9tqlm.svc.cluster.local:2379
        INITIAL_STATE="existing"
        INITIAL_CLUSTER="etcd-0=http://etcd-0.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380,etcd-1=http://etcd-1.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380"
    fi

    if [ "${POD_NAME}" = "etcd-2" ]; then
        echo "i'm etcd-2. I join as new member as soon as we have 2 existing & healthy members"
        until etcdctl --endpoints ${MASTER_ENDPOINT} member list | grep -q etcd-1; do sleep 1; echo "Waiting for etcd-1"; done
        INITIAL_STATE="existing"
        INITIAL_CLUSTER="etcd-0=http://etcd-0.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380,etcd-1=http://etcd-1.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380,etcd-2=http://etcd-2.etcd.cluster-62m9k9tqlm.svc.cluster.local:2380"
    fi
fi



echo ${INITIAL_STATE}
echo ${INITIAL_CLUSTER}

exec /usr/local/bin/etcd \
    --name=${POD_NAME} \
    --data-dir="/var/run/etcd/pod_${POD_NAME}/" \
    --heartbeat-interval=500 \
    --election-timeout=5000 \
    --initial-cluster=${INITIAL_CLUSTER} \
    --initial-cluster-token="62m9k9tqlm" \
    --initial-cluster-state=${INITIAL_STATE} \
    --advertise-client-urls http://${POD_NAME}.etcd.cluster-62m9k9tqlm.svc.cluster.local:2379 \
    --listen-client-urls http://0.0.0.0:2379 \
    --listen-peer-urls http://0.0.0.0:2380
`
)
