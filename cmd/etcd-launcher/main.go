/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"go.etcd.io/etcd/v3/clientv3"
	"go.etcd.io/etcd/v3/etcdserver/api/v3rpc/rpctypes"
	"go.etcd.io/etcd/v3/etcdserver/etcdserverpb"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultClusterSize       = 3
	defaultEtcdctlAPIVersion = "3"
	etcdCommandPath          = "/usr/local/bin/etcd"
)

type config struct {
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
	config      *config
	client      *clientv3.Client
	localClient *clientv3.Client
}

func main() {
	// normal workflow, we don't do migrations anymore
	e := &etcdCluster{}
	err := e.parseConfigFlags()
	if err != nil {
		log.Fatalf("failed to get launcher configuration: %v", err)
	}

	logOpts := kubermaticlog.NewDefaultOptions()
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	// here we find the cluster state
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalw("failed to get in cluster config", zap.Error(err))
	}
	clusterClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		log.Fatalw("failed to create cluster client", zap.Error(err))
	}

	k8cCluster := kubermaticv1.Cluster{}
	if err := clusterClient.Get(context.Background(), types.NamespacedName{Name: strings.ReplaceAll(e.config.namespace, "cluster-", ""), Namespace: ""}, &k8cCluster); err != nil {
		log.Fatalw("failed to get cluster", zap.Error(err))
	}
	initialMembers := initialMemberList(e.config.clusterSize, e.config.namespace)

	e.config.initialState = "new"
	// check if the etcd cluster is initialized succcessfully.
	if k8cCluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEtcdClusterInitialized, corev1.ConditionTrue) {
		e.config.initialState = "existing"
	}

	log.Info("initializing etcd..")
	log.Infof("initial-state: %s", e.config.initialState)
	log.Infof("initial-cluster: %s", strings.Join(initialMembers, ","))

	if _, err := os.Stat(etcdCommandPath); os.IsNotExist(err) {
		log.Fatalw("can't find command", "command-path", etcdCommandPath, zap.Error(err))
	}
	// setup and start etcd command
	cmd := exec.Command(etcdCommandPath, etcdCmd(e.config)...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Infof("starting etcd command: %s", cmd.String())
	if err = cmd.Start(); err != nil {
		log.Fatalf("failed to start etcd: %v", err)
	}

	if err = wait.Poll(1*time.Second, 30*time.Second, e.isHealthy); err != nil {
		log.Fatalf("manager thread failed to connect to cluster: %v", err)
	}

	isMemeber, err := e.isClusterMember(e.config.podName)
	if err != nil {
		log.Fatalf("failed to check cluster membership: %v", err)
	}

	if isMemeber {
		for { // reconcile dead members
			containsUnwantedMembers, err := e.containsUnwantedMembers()
			if err != nil {
				log.Warnw("failed to list memebers ", zap.Error(err))
				time.Sleep(10 * time.Second)
				continue
			}
			// we only need to reconcile if we have members that we shouldn't have
			if !containsUnwantedMembers {
				log.Info("cluster members reconciled..")
				break
			}
			// to avoide race conditions, we will run only on the cluster leader
			leader, err := e.isLeader()
			if err != nil || !leader {
				log.Warnw("failed to remove member, error occurred or didn't get the current leader", zap.Error(err))
				time.Sleep(10 * time.Second)
				continue
			}
			if err := e.removeDeadMembers(log); err != nil {
				log.Warnw("failed to remove member", zap.Error(err))
				continue
			}
		}
	} else { // new etcd member, need to join the cluster
		log.Info("pod is not a cluster member, trying to join..")
		// remove possibly stale member data dir..
		log.Info("removing possibly stale data dir")
		_ = os.RemoveAll(path.Join(e.config.dataDir, "member"))
		// join the cluster

		if _, err := e.client.MemberAdd(context.Background(), []string{fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.config.podName, e.config.namespace)}); err != nil {
			log.Fatalf("failed to join cluster: %v", err)
		}
		log.Info("joined etcd cluster succcessfully.")
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

func (e *etcdCluster) parseConfigFlags() error {
	config := &config{}

	flag.StringVar(&config.namespace, "namespace", "", "namespace of the user cluster")
	flag.IntVar(&config.clusterSize, "etcd-cluster-size", defaultClusterSize, "number of replicas in the etcd cluster")
	flag.StringVar(&config.podName, "pod-name", "", "name of this etcd pod")
	flag.StringVar(&config.podIP, "pod-ip", "", "IP address of this etcd pod")
	flag.StringVar(&config.etcdctlAPIVersion, "api-version", defaultEtcdctlAPIVersion, "etcdctl API version")
	flag.StringVar(&config.token, "token", "", "etcd database token")
	flag.BoolVar(&config.enableCorruptionCheck, "enable-corruption-check", false, "enable etcd experimental corruption check")
	flag.Parse()

	if config.namespace == "" {
		return errors.New("-namespace is not set")
	}

	if config.clusterSize < defaultClusterSize {
		return fmt.Errorf("-etcd-cluster-size is smaller than %d", defaultClusterSize)
	}

	if config.podName == "" {
		return errors.New("-pod-name is not set")
	}

	if config.podIP == "" {
		return errors.New("-pod-ip is not set")
	}

	if config.etcdctlAPIVersion != "2" && config.etcdctlAPIVersion != "3" {
		return errors.New("-api-version is either 2 or 3")
	}

	if config.token == "" {
		return errors.New("-token is not set")
	}

	config.dataDir = fmt.Sprintf("/var/run/etcd/pod_%s/", config.podName)

	e.config = config
	return nil
}

func etcdCmd(config *config) []string {
	cmd := []string{
		fmt.Sprintf("--name=%s", config.podName),
		fmt.Sprintf("--data-dir=%s", config.dataDir),
		fmt.Sprintf("--initial-cluster=%s", strings.Join(initialMemberList(config.clusterSize, config.namespace), ",")),
		fmt.Sprintf("--initial-cluster-token=%s", config.token),
		fmt.Sprintf("--initial-cluster-state=%s", config.initialState),
		fmt.Sprintf("--advertise-client-urls=https://%s.etcd.%s.svc.cluster.local:2379,https://%s:2379", config.podName, config.namespace, config.podIP),
		fmt.Sprintf("--listen-client-urls=https://%s:2379,https://127.0.0.1:2379", config.podIP),
		fmt.Sprintf("--listen-metrics-urls=http://%s:2378,http://127.0.0.1:2378", config.podIP),
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

func (e *etcdCluster) containsUnwantedMembers() (bool, error) {
	members, err := e.listMembers()
	if err != nil {
		return false, err
	}
	expectedMembers := initialMemberList(e.config.clusterSize, e.config.namespace)
membersLoop:
	for _, member := range members {
		for _, expectedMember := range expectedMembers {
			if len(member.GetPeerURLs()) == 1 && expectedMember == (member.GetName()+"="+member.GetPeerURLs()[0]) {
				continue membersLoop
			}
		}
		return true, nil
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
	defer cancel()
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
	defer cancel()
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
	for i := 0; i < 10; i++ {
		resp, err := e.localClient.Status(context.Background(), e.endpoint())
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

func (e *etcdCluster) removeDeadMembers(log *zap.SugaredLogger) error {
	members, err := e.listMembers()
	if err != nil {
		return err
	}
	for _, member := range members {
		if member.Name == e.config.podName {
			continue
		}

		if err = wait.Poll(1*time.Second, 30*time.Second, func() (bool, error) {
			return e.isEndpointHealthy(member.PeerURLs[0])
		}); err != nil {
			log.Infow("member is not responding, removing from cluster", "member-name", member.Name)
			_, err = e.client.MemberRemove(context.Background(), member.ID)
			return err
		}
	}
	return nil
}
