package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
)

const (
	initialStateEnvName      = "INITIAL_STATE"
	initialClusterEnvName    = "INITIAL_CLUSTER"
	defaultClusterSize       = 3
	defaultEtcdctlAPIVersion = "3"
)

type envConfig struct {
	namespace             string
	clusterSize           int
	podName               string
	podIP                 string
	etcdctlAPIVersion     string
	dataDir               string
	token                 string
	enableCorruptionCheck bool
}

func main() {
	// normal workflow, we don't do migrations anymore
	config, err := getConfigFromEnv()
	if err != nil {
		log.Fatalf("failed to get launcher configuration: %v", err)
	}

	// not required, will leave it for now.
	os.Setenv(initialStateEnvName, "new")
	os.Setenv(initialClusterEnvName, initialMemberList(config.clusterSize, config.namespace))

	log.Print("initializing etcd..")
	log.Printf("initial-state: %s", os.Getenv(initialStateEnvName))
	log.Printf("initial-cluster: %s", os.Getenv(initialClusterEnvName))

	err = syscall.Exec(
		"/usr/local/bin/etcd",
		etcdCmd(config),
		os.Environ())
	if err != nil {
		log.Fatal(err)
	}

}

func initialMemberList(n int, namespace string) string {
	members := []string{}
	for i := 0; i < n; i++ {
		members = append(members, fmt.Sprintf("etcd-%d=http://etcd-%d.etcd.%s.svc.cluster.local:2380", i, i, namespace))
	}
	return strings.Join(members, ",")
}

func getConfigFromEnv() (*envConfig, error) {
	var err error
	var ok bool

	config := &envConfig{}
	if config.namespace, ok = os.LookupEnv("NAMESPACE"); !ok || config.namespace == "" {
		return nil, errors.New("NAMESPACE is not set")
	}

	config.clusterSize = defaultClusterSize
	if s := os.Getenv("ECTD_CLUSTER_SIZE"); s != "" {
		if config.clusterSize, err = strconv.Atoi(s); err != nil {
			return nil, fmt.Errorf("failed to read ECTD_CLUSTER_SIZE: %v", err)
		}
		if config.clusterSize > defaultClusterSize {
			return nil, fmt.Errorf("ECTD_CLUSTER_SIZE is smaller then %d", defaultClusterSize)
		}
	}

	if config.podName, ok = os.LookupEnv("POD_NAME"); !ok || config.podName == "" {
		return nil, errors.New("POD_NAME is not set")
	}

	if config.podIP, ok = os.LookupEnv("POD_IP"); !ok || config.podIP == "" {
		return nil, errors.New("POD_IP is not set")
	}

	config.etcdctlAPIVersion = defaultEtcdctlAPIVersion
	if v := os.Getenv("ETCDCTL_API"); v == "2" || v == "3" {
		config.etcdctlAPIVersion = v
	}

	if config.token, ok = os.LookupEnv("TOKEN"); !ok || config.token == "" {
		return nil, errors.New("TOEKN is not set")
	}

	if c := os.Getenv("ENABLE_CORRUPTION_CHECK"); strings.ToLower(c) == "true" {
		config.enableCorruptionCheck = true
	}
	config.dataDir = fmt.Sprintf("/var/run/etcd/pod_%s/", config.podName)

	return config, nil
}

func etcdCmd(config *envConfig) []string {
	cmd := []string{
		"etcd",
		fmt.Sprintf("--name=%s", config.podName),
		fmt.Sprintf("--data-dir=%s", config.dataDir),
		fmt.Sprintf("--initial-cluster=%s", initialMemberList(config.clusterSize, config.namespace)),
		fmt.Sprintf("--initial-cluster-token=%s", config.token),
		"--initial-cluster-state=new",
		fmt.Sprintf("--advertise-client-urls=https://%s.etcd.%s.svc.cluster.local:2379,https://%s:2379", config.podName, config.namespace, config.podIP),
		fmt.Sprintf("--listen-client-urls=https://%s:2379,https://127.0.0.1:2379", config.podIP),
		fmt.Sprintf("--listen-peer-urls=http://%s:2380", config.podIP),
		fmt.Sprintf("--initial-advertise-peer-urls=http://%s.etcd.%s.svc.cluster.local:2380", config.podName, config.namespace),
		"--trusted-ca-file=/etc/etcd/pki/ca/ca.crt",
		"--client-cert-auth",
		"--cert-file=/etc/etcd/pki/tls/etcd-tls.crt",
		"--key-file=/etc/etcd/pki/tls/etcd-tls.key",
		"--auto-compaction-retention=8",
	}

	if config.enableCorruptionCheck {
		cmd = append(cmd, []string{
			"--experimental-initial-corrupt-check=true",
			"--experimental-corrupt-check-time=10m",
		}...)
	}

	return cmd
}
