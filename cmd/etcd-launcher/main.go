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
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"go.etcd.io/etcd/v3/clientv3"
	"go.etcd.io/etcd/v3/etcdserver/etcdserverpb"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"

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

func main() {
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	cfg := &config{}
	err := cfg.parseConfigFlags()
	log := kubermaticlog.New(logOpts.Debug, logOpts.Format).Sugar()

	if err != nil {
		log.Fatalw("failed to get launcher configuration", zap.Error(err))
	}

	if _, err := os.Stat(etcdCommandPath); os.IsNotExist(err) {
		log.Fatalw("can't find etcd command", "binary", etcdCommandPath, zap.Error(err))
	}

	// here we find the cluster state
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalw("failed to get in cluster config", zap.Error(err))
	}
	clusterClient, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		log.Fatalw("failed to create cluster client", zap.Error(err))
	}

	ctx := context.Background()

	// fetch Kubermatic Cluster object
	clusterName := strings.ReplaceAll(cfg.namespace, "cluster-", "")
	k8cCluster := kubermaticv1.Cluster{}
	if err := clusterClient.Get(ctx, types.NamespacedName{Name: clusterName}, &k8cCluster); err != nil {
		log.Fatalw("failed to get cluster", "cluster", clusterName, zap.Error(err))
	}

	// check if the etcd cluster is initialized successfully
	cfg.initialState = initialStateNew
	if k8cCluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEtcdClusterInitialized, corev1.ConditionTrue) {
		cfg.initialState = initialStateExisting
	}

	log.Infow("initializing etcd...",
		"pod", cfg.podName,
		"state", cfg.initialState,
		"size", cfg.clusterSize,
		"namespace", cfg.namespace,
	)

	// setup and start etcd command
	cmd := exec.Command(etcdCommandPath, etcdCmd(cfg)...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	log.Infow("starting etcd...", "cmd", cmd.String())
	if err = cmd.Start(); err != nil {
		log.Fatalw("failed to start etcd", zap.Error(err))
	}

	// wait for etcd to accept connections and become healthy
	var etcdClient *etcd.Client

	timeout := 30 * time.Second
	endpoints := etcd.ClientEndpoints(cfg.clusterSize, cfg.namespace)
	log.Infof("waiting up to %v for etcd to become healthy...", timeout)

	if err := wait.Poll(1*time.Second, timeout, func() (bool, error) {
		var err error

		etcdClient, err = etcd.NewClient(endpoints, nil)
		if err != nil {
			return false, nil
		}

		return etcdClient.Healthy(ctx, nil)
	}); err != nil {
		log.Fatalw("failed to connect to etcd cluster", zap.Error(err))
	}

	// determine if this pod is already an etcd member
	isMember, err := etcdClient.IsClusterMember(ctx, cfg.podName)
	if err != nil {
		log.Fatalw("failed to check cluster membership", zap.Error(err))
	}

	if isMember {
		if err := reconcileClusterMembers(ctx, log, etcdClient, cfg); err != nil {
			log.Fatalw("failed to reconcile cluster status", zap.Error(err))
		}
	} else {
		if err := joinNewMember(ctx, log, etcdClient, cfg); err != nil {
			log.Fatalw("failed to join cluster", zap.Error(err))
		}
	}

	// we do not need this connection anymore
	if err := etcdClient.Close(); err != nil {
		log.Warnw("failed to close unused management connection", zap.Error(err))
	}

	// wait and sit until etcd exits
	log.Info("startup complete, regular operation from here on")
	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}

	log.Info("etcd exited")
}

func joinNewMember(ctx context.Context, log *zap.SugaredLogger, etcdClient *etcd.Client, cfg *config) error {
	log.Info("pod is not a cluster member, trying to join...")

	// remove possibly stale member data dir
	log.Info("removing possibly stale data dir...")
	if err := os.RemoveAll(path.Join(cfg.dataDir, "member")); err != nil {
		log.Warnw("failed to remove directory", zap.Error(err))
		// do not exit here
	}

	// join the cluster
	peerURL := fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:%d", cfg.podName, cfg.namespace, etcd.PeersPort)
	if err := etcdClient.MemberAdd(ctx, []string{peerURL}); err != nil {
		return fmt.Errorf("failed to join cluster: %v", err)
	}

	log.Info("joined etcd cluster successfully")
	return nil
}

func reconcileClusterMembers(ctx context.Context, log *zap.SugaredLogger, etcdClient *etcd.Client, cfg *config) error {
	log.Infof("%s is already a member", cfg.podName)

	for {
		// handle changes to peerURLs
		if err := updatePeerURL(ctx, etcdClient, cfg); err != nil {
			return fmt.Errorf("failed to update peerURL: %v", err)
		}

		// determine dead members
		unwantedMembers, err := hasUnwantedMembers(ctx, etcdClient, cfg)
		if err != nil {
			log.Warnw("failed to determine member status", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}

		// we only need to reconcile if we have members that we shouldn't have
		if !unwantedMembers {
			break
		}

		// to avoid race conditions, we will run only on the cluster leader
		leader, err := podIsLeader(ctx, log, cfg)
		if err != nil {
			log.Warnw("failed to determine leader status", zap.Error(err))
			time.Sleep(3 * time.Second)
			continue
		}

		if !leader {
			// There are unwanted members, but we are not the leader. This means
			// we need to wait for someone else to do the cleanup or for us to
			// become the leader. In any way, we should wait a short moment.
			time.Sleep(3 * time.Second)
			continue
		}

		if err := removeDeadMembers(ctx, log, etcdClient, cfg); err != nil {
			log.Warnw("failed to remove member", zap.Error(err))
		}
	}

	log.Info("cluster members reconciled successfully")

	return nil
}

func updatePeerURL(ctx context.Context, etcdClient *etcd.Client, cfg *config) error {
	self, err := getOwnMember(ctx, etcdClient, cfg.podName)
	if err != nil {
		return fmt.Errorf("failed to determine own member: %v", err)
	}

	peerURL, err := url.Parse(self.PeerURLs[0])
	if err != nil {
		return fmt.Errorf("failed to parse peer URL %q: %v", self.PeerURLs[0], err)
	}

	// ensure a non-HTTPS url
	if peerURL.Scheme == "https" {
		peerURL.Scheme = "http"
		if err := etcdClient.MemberUpdate(ctx, self.ID, []string{peerURL.String()}); err != nil {
			return fmt.Errorf("failed to update etcd member: %v", err)
		}
	}

	return nil
}

func hasUnwantedMembers(ctx context.Context, etcdClient *etcd.Client, cfg *config) (bool, error) {
	members, err := etcdClient.MemberList(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to list members: %v", err)
	}

	expectedPeers := etcd.PeerURLs(cfg.clusterSize, cfg.namespace)

	for _, member := range members {
		// we only want members with a single peer URL
		if len(member.PeerURLs) != 1 {
			return true, nil
		}

		peerURL, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return false, fmt.Errorf("failed to parse peer URL %q: %v", member.PeerURLs[0], err)
		}

		isExpected := false
		for _, expectedPeer := range expectedPeers {
			if expectedPeer == peerURL.Host {
				isExpected = true
				break
			}
		}

		if !isExpected {
			return true, nil
		}
	}

	return false, nil
}

func removeDeadMembers(ctx context.Context, log *zap.SugaredLogger, etcdClient *etcd.Client, cfg *config) error {
	members, err := etcdClient.MemberList(ctx)
	if err != nil {
		return fmt.Errorf("failed to list members: %v", err)
	}

	log.Warnw("finding and removing dead members...")

	for _, member := range members {
		// ignore the leader (as we only do this removal when we are the leader, this means we
		// ignore ourselves)
		if member.Name == cfg.podName {
			continue
		}

		if err = wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
			// We use the cluster FQDN endpoint url here (internally in Healthy()).
			// Using the IP endpoint will fail because the certificates don't include Pod IP addresses.
			return etcdClient.Healthy(ctx, member)
		}); err != nil {
			log.Warnw("member is not responding, removing from cluster...", "member", member.Name)
			if err := etcdClient.MemberRemove(ctx, member.ID); err != nil {
				return fmt.Errorf("failed to remove member: %v", err)
			}
		}

		log.Warnw("member is alive", "member", member.Name)
	}

	return nil
}

func podIsLeader(ctx context.Context, log *zap.SugaredLogger, cfg *config) (bool, error) {
	localhost := etcd.LocalEndpoint()

	var status *clientv3.StatusResponse
	if err := wait.PollImmediate(1*time.Second, 1*time.Minute, func() (bool, error) {
		localClient, err := etcd.NewClient([]string{localhost}, nil)
		if err != nil {
			log.Warnw(fmt.Sprintf("failed to create etcd client for %q", localhost), zap.Error(err))
			return false, nil
		}
		defer localClient.Close()

		status, err = localClient.Status(ctx, localhost)
		if err != nil {
			log.Warnw("failed to determine endpoint status", zap.Error(err))
			return false, nil
		}

		// member does not know about any leaders
		if status.Leader == 0 {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return false, fmt.Errorf("failed to determine leader status: %v", err)
	}

	return status.Header.MemberId == status.Leader, nil
}

// getOwnMember finds the etcd member that is running in this pod; returns
// nil when no such member could be found
func getOwnMember(ctx context.Context, etcdClient *etcd.Client, podName string) (*etcdserverpb.Member, error) {
	members, err := etcdClient.MemberList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %v", err)
	}

	for i, member := range members {
		if member.Name == podName {
			return members[i], nil
		}
	}

	return nil, nil
}

func (cfg *config) parseConfigFlags() error {
	flag.StringVar(&cfg.namespace, "namespace", "", "namespace of the user cluster")
	flag.IntVar(&cfg.clusterSize, "etcd-cluster-size", defaultClusterSize, "number of replicas in the etcd cluster")
	flag.StringVar(&cfg.podName, "pod-name", "", "name of this etcd pod")
	flag.StringVar(&cfg.podIP, "pod-ip", "", "IP address of this etcd pod")
	flag.StringVar(&cfg.etcdctlAPIVersion, "api-version", defaultEtcdctlAPIVersion, "etcdctl API version")
	flag.StringVar(&cfg.token, "token", "", "etcd database token")
	flag.BoolVar(&cfg.enableCorruptionCheck, "enable-corruption-check", false, "enable etcd experimental corruption check")
	flag.Parse()

	if cfg.namespace == "" {
		return errors.New("-namespace is not set")
	}

	if cfg.clusterSize < defaultClusterSize {
		return fmt.Errorf("-etcd-cluster-size is smaller than %d", defaultClusterSize)
	}

	if cfg.podName == "" {
		return errors.New("-pod-name is not set")
	}

	if cfg.podIP == "" {
		return errors.New("-pod-ip is not set")
	}

	if cfg.etcdctlAPIVersion != "2" && cfg.etcdctlAPIVersion != "3" {
		return errors.New("-api-version must be either 2 or 3")
	}

	if cfg.token == "" {
		return errors.New("-token is not set")
	}

	cfg.dataDir = fmt.Sprintf("/var/run/etcd/pod_%s/", cfg.podName)

	return nil
}

func etcdCmd(config *config) []string {
	cmd := []string{
		fmt.Sprintf("--name=%s", config.podName),
		fmt.Sprintf("--data-dir=%s", config.dataDir),
		fmt.Sprintf("--initial-cluster=%s", strings.Join(etcd.MembersList(config.clusterSize, config.namespace), ",")),
		fmt.Sprintf("--initial-cluster-token=%s", config.token),
		fmt.Sprintf("--initial-cluster-state=%s", config.initialState),
		fmt.Sprintf("--advertise-client-urls=https://%s.etcd.%s.svc.cluster.local:%d,https://%s:%d", config.podName, config.namespace, etcd.ClientPort, config.podIP, etcd.ClientPort),
		fmt.Sprintf("--listen-client-urls=https://%s:%d,https://127.0.0.1:%d", config.podIP, etcd.ClientPort, etcd.ClientPort),
		fmt.Sprintf("--listen-metrics-urls=http://%s:%d,http://127.0.0.1:%d", config.podIP, etcd.MetricsPort, etcd.MetricsPort),
		fmt.Sprintf("--listen-peer-urls=http://%s:%d", config.podIP, etcd.PeersPort),
		fmt.Sprintf("--initial-advertise-peer-urls=http://%s.etcd.%s.svc.cluster.local:%d", config.podName, config.namespace, etcd.PeersPort),
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
