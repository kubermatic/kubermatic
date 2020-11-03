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
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"go.etcd.io/etcd/v3/clientv3"
	"go.etcd.io/etcd/v3/etcdserver/api/v3rpc/rpctypes"
	"go.etcd.io/etcd/v3/etcdserver/etcdserverpb"
	"go.etcd.io/etcd/v3/pkg/transport"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"

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
	initialStateExisting     = "existing"
	initialStateNew          = "new"
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
	config *config
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

	e.config.initialState = initialStateNew
	// check if the etcd cluster is initialized successfully.
	if k8cCluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEtcdClusterInitialized, corev1.ConditionTrue) {
		e.config.initialState = initialStateExisting
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

	if err = wait.Poll(1*time.Second, 30*time.Second, e.isClusterHealthy); err != nil {
		log.Fatalf("manager thread failed to connect to cluster: %v", err)
	}

	isMemeber, err := e.isClusterMember(e.config.podName)
	if err != nil {
		log.Fatalf("failed to check cluster membership: %v", err)
	}
	if isMemeber {
		log.Infof("%s is a member", e.config.podName)
		for {
			// handle changes to peerURLs
			if err := e.updatePeerURL(); err != nil {
				log.Fatalf("failed to update peerURL: %v", err)
			}
			// reconcile dead members
			containsUnwantedMembers, err := e.containsUnwantedMembers()
			if err != nil {
				log.Warnw("failed to list members ", zap.Error(err))
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
		client, err := e.getClusterClient()
		if err != nil {
			log.Fatalf("can't find cluster client: %v", err)
		}
		defer client.Close()

		if _, err := client.MemberAdd(context.Background(), []string{fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.config.podName, e.config.namespace)}); err != nil {
			log.Fatalf("failed to join cluster: %v", err)
		}
		log.Info("joined etcd cluster successfully.")
	}

	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func (e *etcdCluster) updatePeerURL() error {
	members, err := e.listMembers()
	if err != nil {
		return err
	}
	for _, member := range members {
		peerURL, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return err
		}
		if member.Name == e.config.podName && peerURL.Scheme == "https" {
			client, err := e.getClusterClient()
			if err != nil {
				return err
			}
			defer client.Close()
			peerURL.Scheme = "http"
			_, err = client.MemberUpdate(context.Background(), member.ID, []string{peerURL.String()})
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func initialMemberList(n int, namespace string) []string {
	members := []string{}
	for i := 0; i < n; i++ {
		members = append(members, fmt.Sprintf("etcd-%d=http://etcd-%d.etcd.%s.svc.cluster.local:2380", i, i, namespace))
	}
	return members
}

func peerURLsList(n int, namespace string) []string {
	urls := []string{}
	for i := 0; i < n; i++ {
		urls = append(urls, fmt.Sprintf("etcd-%d.etcd.%s.svc.cluster.local:2380", i, namespace))
	}
	return urls
}

func clientEndpoints(n int, namespace string) []string {
	endpoints := []string{}
	for i := 0; i < n; i++ {
		endpoints = append(endpoints, fmt.Sprintf("https://etcd-%d.etcd.%s.svc.cluster.local:2379", i, namespace))
	}
	return endpoints
}

func (e *etcdCluster) endpoint() string {
	return "https://127.0.0.1:2379"
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
		"--client-cert-auth",
		fmt.Sprintf("--trusted-ca-file=%s", resources.EtcdTrustedCAFile),
		fmt.Sprintf("--cert-file=%s", resources.EtcdCertFile),
		fmt.Sprintf("--key-file=%s", resources.EtcdKetFile),
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

func (e *etcdCluster) getClusterClient() (*clientv3.Client, error) {
	endpoints := clientEndpoints(e.config.clusterSize, e.config.namespace)
	return e.getClientWithEndpoints(endpoints)
}

func (e *etcdCluster) getLocalClient() (*clientv3.Client, error) {
	return e.getClientWithEndpoints([]string{e.endpoint()})
}

func (e *etcdCluster) getClientWithEndpoints(eps []string) (*clientv3.Client, error) {
	var err error
	tlsInfo := transport.TLSInfo{
		CertFile:       resources.EtcdClientCertFile,
		KeyFile:        resources.EtcdClientKeyFile,
		TrustedCAFile:  resources.EtcdTrustedCAFile,
		ClientCertAuth: true,
	}
	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client TLS config: %v", err)
	}
	for i := 0; i < 5; i++ {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   eps,
			DialTimeout: 2 * time.Second,
			TLS:         tlsConfig,
		})
		if err == nil && cli != nil {
			return cli, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("failed to establish client connection: %v", err)

}

func (e *etcdCluster) listMembers() ([]*etcdserverpb.Member, error) {
	client, err := e.getClientWithEndpoints(clientEndpoints(e.config.clusterSize, e.config.namespace))
	if err != nil {
		return nil, fmt.Errorf("can't find cluster client: %v", err)
	}
	defer client.Close()

	resp, err := client.MemberList(context.Background())
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
		url, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return false, err
		}
		// if the member is not started yet, its name would be empty, in that case, we match for peerURL host.
		if member.Name == name || url.Host == fmt.Sprintf("%s.etcd.%s.svc.cluster.local:2380", e.config.podName, e.config.namespace) {
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
	expectedMembers := peerURLsList(e.config.clusterSize, e.config.namespace)
membersLoop:
	for _, member := range members {
		for _, expectedMember := range expectedMembers {
			if len(member.GetPeerURLs()) == 1 {
				peerURL, err := url.Parse(member.PeerURLs[0])
				if err != nil {
					return false, err
				}
				if expectedMember == peerURL.Host {
					continue membersLoop
				}
			}
		}
		return true, nil
	}
	return false, nil
}

func (e *etcdCluster) isClusterHealthy() (bool, error) {
	return e.isHealthyWithEndpoints(clientEndpoints(e.config.clusterSize, e.config.namespace))
}

func (e *etcdCluster) isHealthyWithEndpoints(endpoints []string) (bool, error) {
	client, err := e.getClientWithEndpoints(endpoints)
	if err != nil {
		return false, err
	}
	defer client.Close()
	// just get a key from etcd, this is how `etcdctl endpoint health` works!
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = client.Get(ctx, "healthy")
	defer cancel()
	if err != nil && err != rpctypes.ErrPermissionDenied {
		// silently swallow/drop transient errors
		return false, nil
	}
	return true, nil
}

func (e *etcdCluster) isLeader() (bool, error) {
	localClient, err := e.getLocalClient()
	if err != nil {
		return false, err
	}
	defer localClient.Close()

	for i := 0; i < 10; i++ {
		resp, err := localClient.Status(context.Background(), e.endpoint())
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

	client, err := e.getClusterClient()
	if err != nil {
		return fmt.Errorf("can't find cluster client: %v", err)
	}
	defer client.Close()

	for _, member := range members {
		if member.Name == e.config.podName {
			continue
		}
		if err = wait.Poll(1*time.Second, 30*time.Second, func() (bool, error) {
			// we use the cluster FQDN endpoint url here. Using the IP endpoint will
			// fail because the certificates don't include Pod IP addresses.
			return e.isHealthyWithEndpoints(member.ClientURLs[len(member.ClientURLs)-1:])
		}); err != nil {
			log.Infow("member is not responding, removing from cluster", "member-name", member.Name)
			_, err = client.MemberRemove(context.Background(), member.ID)
			return err
		}
	}
	return nil
}
