package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
	"go.etcd.io/etcd/etcdserver/etcdserverpb"
	"golang.org/x/sync/errgroup"
)

const (
	initialStateEnvName      = "INITIAL_STATE"
	initialClusterEnvName    = "INITIAL_CLUSTER"
	defaultClusterSize       = 3
	defaultEtcdctlAPIVersion = "3"
)

var (
	eg errgroup.Group
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
	initialState          string
}

type etcdCluster struct {
	config      *envConfig
	client      *clientv3.Client
	localClient *clientv3.Client
}

func main() {
	// normal workflow, we don't do migrations anymore
	e := &etcdCluster{}
	err := e.getConfigFromEnv()
	if err != nil {
		log.Fatalf("failed to get launcher configuration: %v", err)
	}

	initialMembers := initialMemberList(e.config.clusterSize, e.config.namespace)
	// not required, will leave it for now.
	os.Setenv(initialStateEnvName, e.config.initialState)
	os.Setenv(initialClusterEnvName, strings.Join(initialMembers, ","))

	log.Print("initializing etcd..")
	log.Printf("initial-state: %s", os.Getenv(initialStateEnvName))
	log.Printf("initial-cluster: %s", os.Getenv(initialClusterEnvName))

	// setup and start etcd command
	cmd := exec.Command("/usr/local/bin/etcd", etcdCmd(e.config)...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err = cmd.Start(); err != nil {
		log.Fatalf("failed to start etcd: %v", err)
	}

	// if existing, we need to reconcile
	var healthy bool
	for i := 0; i < 5; i++ {
		if healthy, err = e.isHealthy(); healthy {
			break
		}
	}
	if err != nil {
		log.Fatalf("manager thread failed to connect to cluster: %v", err)
	}

	isMemeber, err := e.isClusterMember(e.config.podName)
	if err != nil {
		log.Fatalf("failed to check cluster membership: %v", err)
	}

	if isMemeber {
		for { // reconcile dead members
			members, err := e.listMembers()
			if err != nil {
				time.Sleep(10 * time.Second)
				continue
			}
			// we only need to reconcile if we have more members than we should
			if len(members) <= e.config.clusterSize {
				break
			}
			// to avoide race conditions, we will run only on the cluster leader
			leader, err := e.isLeader()
			if err != nil || !leader {
				time.Sleep(10 * time.Second)
				continue
			}
			if err := e.removeDeadMembers(); err != nil {
				continue
			}
		}
	} else { // new etcd member, need to join hte cluster
		// remove possibly stale member data dir..
		_ = os.RemoveAll(path.Join(e.config.dataDir, "member"))
		// join the cluster
		if _, err := e.client.MemberAdd(context.Background(), []string{fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.config.podName, e.config.namespace)}); err != nil {
			log.Fatalf("failed to join cluster: %v", err)
		}
	}

	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func initialMemberList(n int, namespace string) []string {
	members := []string{}
	for i := 0; i < n; i++ {
		members = append(members, fmt.Sprintf("etcd-%d=http://etcd-%d.etcd.%s.svc.cluster.local:2380", i, i, namespace))
	}
	return members
}

func clientEndpoints(n int, namespace string) []string {
	endpoints := []string{}
	for i := 0; i < n; i++ {
		endpoints = append(endpoints, fmt.Sprintf("etcd-%d.etcd.%s.svc.cluster.local:2380", i, namespace))
	}
	return endpoints
}

func (e *etcdCluster) endpoint() string {
	return fmt.Sprintf("%s.etcd.%s.svc.cluster.local:2380", e.config.podName, e.config.namespace)
}

func (e *etcdCluster) getConfigFromEnv() error {
	var err error
	var ok bool
	config := &envConfig{}

	if config.namespace, ok = os.LookupEnv("NAMESPACE"); !ok || config.namespace == "" {
		return errors.New("NAMESPACE is not set")
	}

	config.clusterSize = defaultClusterSize
	if s := os.Getenv("ECTD_CLUSTER_SIZE"); s != "" {
		if config.clusterSize, err = strconv.Atoi(s); err != nil {
			return fmt.Errorf("failed to read ECTD_CLUSTER_SIZE: %v", err)
		}
		if config.clusterSize < defaultClusterSize {
			return fmt.Errorf("ECTD_CLUSTER_SIZE is smaller then %d", defaultClusterSize)
		}
	}

	if config.podName, ok = os.LookupEnv("POD_NAME"); !ok || config.podName == "" {
		return errors.New("POD_NAME is not set")
	}

	if config.podIP, ok = os.LookupEnv("POD_IP"); !ok || config.podIP == "" {
		return errors.New("POD_IP is not set")
	}

	config.etcdctlAPIVersion = defaultEtcdctlAPIVersion
	if v := os.Getenv("ETCDCTL_API"); v == "2" || v == "3" {
		config.etcdctlAPIVersion = v
	}

	if config.token, ok = os.LookupEnv("TOKEN"); !ok || config.token == "" {
		return errors.New("TOEKN is not set")
	}

	if c := os.Getenv("ENABLE_CORRUPTION_CHECK"); strings.ToLower(c) == "true" {
		config.enableCorruptionCheck = true
	}

	if config.initialState, ok = os.LookupEnv(initialStateEnvName); !ok || config.initialState == "" {
		return errors.New("INITIAL_STATE is not set")
	}
	config.dataDir = fmt.Sprintf("/var/run/etcd/pod_%s/", config.podName)

	e.config = config
	return nil

}

func etcdCmd(config *envConfig) []string {
	cmd := []string{
		fmt.Sprintf("--name=%s", config.podName),
		fmt.Sprintf("--data-dir=%s", config.dataDir),
		fmt.Sprintf("--initial-cluster=%s", strings.Join(initialMemberList(config.clusterSize, config.namespace), ",")),
		fmt.Sprintf("--initial-cluster-token=%s", config.token),
		fmt.Sprintf("--initial-cluster-state=%s", config.initialState),
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

func (e *etcdCluster) getClient() error {
	if e.client != nil {
		return nil
	}
	endpoints := clientEndpoints(e.config.clusterSize, e.config.namespace)
	var err error
	e.client, err = e.getClientWithEndpoints(endpoints)
	return err
}

func (e *etcdCluster) getLocalClient() error {
	if e.localClient != nil {
		return nil
	}
	var err error
	e.localClient, err = e.getClientWithEndpoints([]string{e.endpoint()})
	return err
}

func (e *etcdCluster) getClientWithEndpoints(eps []string) (*clientv3.Client, error) {
	var err error
	for i := 0; i < 5; i++ {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   eps,
			DialTimeout: 2 * time.Second,
		})
		if err == nil && cli != nil {
			return cli, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("failed to establish client connection: %v", err)

}

func (e *etcdCluster) listMembers() ([]*etcdserverpb.Member, error) {
	if e.client == nil {
		return nil, fmt.Errorf("can't find cluster client")
	}
	resp, err := e.client.MemberList(context.Background())
	if err != nil {
		return nil, err
	}
	return resp.Members, err
}

func (e *etcdCluster) isClusterMember(name string) (bool, error) {
	members, err := e.listMembers()
	if err != nil {
		return false, err
	}
	if len(members) == 0 {
		return false, nil
	}

	for _, member := range members {
		if member.Name == name || member.PeerURLs[0] == fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.config.podName, e.config.namespace) {
			return true, nil
		}
	}
	return false, nil
}

func (e *etcdCluster) isHealthy() (bool, error) {
	if err := e.getClient(); err != nil {
		return false, err
	}
	// just get a key from etcd, this is how `etcdctl endpoint health` works!
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err := e.client.Get(ctx, "healthy")
	cancel()
	if err != nil && err != rpctypes.ErrPermissionDenied {
		return false, nil
	}
	return true, nil
}

func (e *etcdCluster) isEndpointHealthy(endpoint string) (bool, error) {
	client, err := e.getClientWithEndpoints([]string{endpoint})
	if err != nil {
		return false, err
	}
	// just get a key from etcd, this is how `etcdctl endpoint health` works!
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = client.Get(ctx, "healthy")
	cancel()
	if err != nil && err != rpctypes.ErrPermissionDenied {
		return false, nil
	}
	return true, nil

}

func (e *etcdCluster) isLeader() (bool, error) {
	var err error
	if err = e.getLocalClient(); err != nil {
		return false, err
	}
	var resp *clientv3.StatusResponse
	for i := 0; i < 10; i++ {
		resp, err = e.localClient.Status(context.Background(), e.endpoint())
		if err != nil || resp.Leader == 0 {
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.Header.MemberId == resp.Leader {
			return true, nil
		}
	}
	return false, nil
}

func (e *etcdCluster) removeDeadMembers() error {
	members, err := e.listMembers()
	if err != nil {
		return err
	}
	for _, member := range members {
		if member.Name == e.config.podName {
			continue
		}
		healthy, err := e.isEndpointHealthy(member.PeerURLs[0])
		if err != nil {
			continue
		}
		if !healthy {
			if _, err := e.client.MemberRemove(context.Background(), member.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
