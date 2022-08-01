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
	"io"
	"io/fs"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	client "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	defaultClusterSize       = 3
	defaultEtcdctlAPIVersion = "3"
	etcdCommandPath          = "/usr/local/bin/etcd"
	initialStateExisting     = "existing"
	initialStateNew          = "new"
	envPeerTLSMode           = "PEER_TLS_MODE"
	peerTLSModeStrict        = "strict"

	timeoutListMembers    = time.Second * 5
	timeoutAddMember      = time.Second * 15
	timeoutRemoveMember   = time.Second * 30
	timeoutUpdatePeerURLs = time.Second * 10
)

type etcdCluster struct {
	cluster               string // given as a CLI flag
	namespace             string // filled in later during init()
	clusterSize           int
	clusterClient         ctrlruntimeclient.Client
	podName               string
	podIP                 string
	etcdctlAPIVersion     string
	dataDir               string
	token                 string
	enableCorruptionCheck bool
	initialState          string
	initialMembers        []string
	usePeerTLSOnly        bool
}

func main() {
	log := createLogger()
	ctx := signals.SetupSignalHandler()

	// normal workflow, we don't do migrations anymore
	e := &etcdCluster{}
	err := e.parseConfigFlags()
	if err != nil {
		log.Panicw("failed to get launcher configuration", zap.Error(err))
	}

	// here we find the cluster state
	e.clusterClient, err = inClusterClient(log)
	if err != nil {
		log.Panicw("failed to get in-cluster client", zap.Error(err))
	}

	// init the current cluster object; we only care about the namespace name
	// (which is practically immutable) and so it's sufficient to fetch the
	// cluster now, once.
	kkpCluster, err := e.init(ctx)
	if err != nil {
		log.Panicw("failed to get KKP cluster", zap.Error(err))
	}

	if err := e.SetClusterSize(ctx); err != nil {
		log.Panicw("failed to set cluster size", zap.Error(err))
	}

	if err := e.SetInitialState(ctx, log, kkpCluster); err != nil {
		log.Panicw("failed to set initialState", zap.Error(err))
	}

	e.initialMembers = initialMemberList(ctx, log, e.clusterClient, e.clusterSize, e.namespace, e.usePeerTLSOnly)

	log.Info("initializing etcd..")
	log.Infof("initial-state: %s", e.initialState)
	log.Infof("initial-cluster: %s", strings.Join(e.initialMembers, ","))
	if e.usePeerTLSOnly {
		log.Info("peer-tls-mode: strict")
	}

	// if the cluster already exists, try to connect and update peer URLs that might be out of sync.
	// etcd might fail to start if peer URLs in the etcd member state and the flags passed to it are different
	if e.initialState == initialStateExisting {
		// make sure that peer URLs in the cluster member data is
		// updated / in sync with the etcd node's configuration
		if err := e.UpdatePeerURLs(ctx, log); err != nil {
			log.Warnw("failed to update peerURL, etcd node might fail to start ...", zap.Error(err))
		}
	}

	thisMember, err := e.GetMemberByName(ctx, log, e.podName)

	switch {
	case err != nil:
		log.Warnw("failed to check cluster membership", zap.Error(err))
	case thisMember != nil:
		log.Infof("%v is a member", thisMember.GetPeerURLs())

		if _, err := os.Stat(filepath.Join(e.dataDir, "member")); errors.Is(err, fs.ErrNotExist) {
			client, err := e.GetClusterClient()
			if err != nil {
				log.Panicw("can't find cluster client: %v", zap.Error(err))
			}

			log.Warnw("No data dir, removing stale membership to rejoin cluster as new member")

			_, err = client.MemberRemove(ctx, thisMember.ID)
			if err != nil {
				closeClient(client, log)
				log.Panicw("remove itself due to data dir loss", zap.Error(err))
			}

			closeClient(client, log)
			if err := joinCluster(ctx, e, log); err != nil {
				log.Panicw("join cluster as fresh member", zap.Error(err))
			}
		}
	default:
		// if no membership information was found but we were able to list from an etcd cluster, we can attempt to join
		if err := joinCluster(ctx, e, log); err != nil {
			log.Panicw("failed to join cluster as fresh member", zap.Error(err))
		}
	}

	// setup and start etcd command
	etcdCmd, err := startEtcdCmd(e, log)
	if err != nil {
		log.Panicw("start etcd cmd", zap.Error(err))
	}

	if err = wait.Poll(1*time.Second, 60*time.Second, func() (bool, error) {
		return e.IsClusterHealthy(ctx, log)
	}); err != nil {
		log.Panicw("manager thread failed to connect to cluster", zap.Error(err))
	}

	// reconcile dead members continuously. Initially we did this once as a step at the end of start up. We did that because scale up/down operations required a full restart of the ring with each node add/remove. However, this is no longer the case, so we need to separate the reconcile from the start up process and do it continuously.
	go func() {
		wait.Forever(func() {
			// refresh the cluster size so the etcd-launcher is aware of scaling operations
			if err := e.SetClusterSize(ctx); err != nil {
				log.Warnw("failed to refresh cluster size", zap.Error(err))
			} else if _, err := deleteUnwantedDeadMembers(ctx, e, log); err != nil {
				log.Warnw("failed to remove dead members", zap.Error(err))
			}
		}, 30*time.Second)
	}()

	if err = etcdCmd.Wait(); err != nil {
		log.Panic(err)
	}
}

func createLogger() *zap.SugaredLogger {
	logOpts := kubermaticlog.NewDefaultOptions()
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	return rawLog.Sugar()
}

func inClusterClient(log *zap.SugaredLogger) (ctrlruntimeclient.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in cluster config: %w", err)
	}
	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster client: %w", err)
	}
	return client, nil
}

func startEtcdCmd(e *etcdCluster, log *zap.SugaredLogger) (*exec.Cmd, error) {
	if _, err := os.Stat(etcdCommandPath); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("find etcd executable: %w", err)
	}

	cmd := exec.Command(etcdCommandPath, etcdCmd(e)...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Infof("starting etcd command: %s %s", etcdCommandPath, strings.Join(etcdCmd(e), " "))
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start etcd: %w", err)
	}
	return cmd, nil
}

func deleteUnwantedDeadMembers(ctx context.Context, e *etcdCluster, log *zap.SugaredLogger) (bool, error) {
	unwantedMembers, err := e.getUnwantedMembers(ctx, log)
	if err != nil {
		log.Warnw("failed to get unwanted members", zap.Error(err))
		return false, nil
	}
	// we only need to reconcile if we have members that we shouldn't have
	if len(unwantedMembers) == 0 {
		log.Debug("no unwanted members present")
		return true, nil
	}

	// to avoide race conditions, we will run only on the cluster leader
	leader, err := e.isLeader(ctx, log)
	if err != nil {
		log.Warnw("failed to determine if member is cluster leader", zap.Error(err))
		return false, nil
	}

	if !leader {
		log.Info("current member is not leader, skipping dead member removal")
		return false, nil
	}

	if err := e.removeDeadMembers(ctx, log, unwantedMembers); err != nil {
		return false, err
	}

	return false, nil
}

func joinCluster(ctx context.Context, e *etcdCluster, log *zap.SugaredLogger) error {
	log.Info("pod is not a cluster member, trying to join..")
	// remove possibly stale member data dir..
	log.Info("removing possibly stale data dir")
	if err := os.RemoveAll(e.dataDir); err != nil {
		return fmt.Errorf("removing possible stale data dir: %w", err)
	}
	// join the cluster
	client, err := e.GetClusterClient()
	if err != nil {
		return fmt.Errorf("can't find cluster client: %w", err)
	}

	// construct peer URLs for this new node

	peerURLs := []string{}

	if !e.usePeerTLSOnly {
		peerURLs = append(peerURLs, fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.podName, e.namespace))
	}

	peerURLs = append(peerURLs, fmt.Sprintf("https://%s.etcd.%s.svc.cluster.local:2381", e.podName, e.namespace))

	ctx, cancelFunc := context.WithTimeout(ctx, timeoutAddMember)
	defer cancelFunc()

	if _, err := client.MemberAdd(ctx, peerURLs); err != nil {
		closeClient(client, log)
		return fmt.Errorf("add itself as a member: %w", err)
	}

	defer closeClient(client, log)

	log.Info("joined etcd cluster successfully.")
	return nil
}

func (e *etcdCluster) UpdatePeerURLs(ctx context.Context, log *zap.SugaredLogger) error {
	members, err := e.listMembers(ctx, log)
	if err != nil {
		return err
	}
	client, err := e.GetClusterClient()
	if err != nil {
		return err
	}

	defer closeClient(client, log)

	for _, member := range members {
		peerURL, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return err
		}

		if member.Name == e.podName {
			ctx, cancelFunc := context.WithTimeout(ctx, timeoutUpdatePeerURLs)
			defer cancelFunc()
			// if both plaintext and TLS peer URLs are supposed to be present
			// update the member to include both plaintext and TLS peer URLs
			if !e.usePeerTLSOnly && (len(member.PeerURLs) == 1 || peerURL.Scheme != "http") {
				plainPeerURL, err := url.Parse(fmt.Sprintf("http://%s", net.JoinHostPort(peerURL.Hostname(), "2380")))
				if err != nil {
					return err
				}

				tlsPeerURL, err := url.Parse(fmt.Sprintf("https://%s", net.JoinHostPort(peerURL.Hostname(), "2381")))
				if err != nil {
					return err
				}

				log.Infof("updating member %d to include plaintext and tls peer ports", member.ID)

				_, err = client.MemberUpdate(
					ctx,
					member.ID,
					[]string{plainPeerURL.String(), tlsPeerURL.String()},
				)
				return err
			}

			// if we're supposed to run with TLS peer endpoints only, two peer URLs are
			// not a valid configuration and should be replaced with TLS only
			if len(member.PeerURLs) == 2 && e.usePeerTLSOnly {
				tlsPeerURL, err := url.Parse(fmt.Sprintf("https://%s", net.JoinHostPort(peerURL.Hostname(), "2381")))
				if err != nil {
					return err
				}

				log.Infof("updating member %d to set tls peer port only", member.ID)

				_, err = client.MemberUpdate(
					ctx,
					member.ID,
					[]string{tlsPeerURL.String()},
				)
				return err
			}
		}
	}

	return nil
}

func initialMemberList(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, n int, namespace string, useTLSPeer bool) []string {
	members := []string{}
	for i := 0; i < n; i++ {
		var pod corev1.Pod
		if err := client.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("etcd-%d", i), Namespace: namespace}, &pod); err != nil {
			log.Warnw("failed to get Pod information for etcd, guessing peer URLs", zap.Error(err))
			if useTLSPeer {
				members = append(members, fmt.Sprintf("etcd-%d=https://etcd-%d.etcd.%s.svc.cluster.local:2381", i, i, namespace))
			} else {
				members = append(members, fmt.Sprintf("etcd-%d=http://etcd-%d.etcd.%s.svc.cluster.local:2380", i, i, namespace))
			}
		} else {
			// use information on the pod to determine if the plaintext and TLS peer ports are going to be open

			if !hasStrictTLS(&pod) {
				members = append(members, fmt.Sprintf("etcd-%d=http://etcd-%d.etcd.%s.svc.cluster.local:2380", i, i, namespace))
			}

			if _, ok := pod.ObjectMeta.Annotations[resources.EtcdTLSEnabledAnnotation]; ok {
				members = append(
					members,
					fmt.Sprintf("etcd-%d=https://etcd-%d.etcd.%s.svc.cluster.local:2381", i, i, namespace),
				)
			}
		}
	}

	return members
}

func hasStrictTLS(pod *corev1.Pod) bool {
	for _, env := range pod.Spec.Containers[0].Env {
		if env.Name == "PEER_TLS_MODE" && env.Value == "strict" {
			return true
		}
	}

	return false
}

func peerHostsList(n int, namespace string) []string {
	hosts := []string{}
	for i := 0; i < n; i++ {
		hosts = append(hosts, fmt.Sprintf("etcd-%d.etcd.%s.svc.cluster.local", i, namespace))
	}
	return hosts
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
	flag.StringVar(&e.cluster, "cluster", "", "name of the user cluster")
	flag.StringVar(&e.podName, "pod-name", "", "name of this etcd pod")
	flag.StringVar(&e.podIP, "pod-ip", "", "IP address of this etcd pod")
	flag.StringVar(&e.etcdctlAPIVersion, "api-version", defaultEtcdctlAPIVersion, "etcdctl API version")
	flag.StringVar(&e.token, "token", "", "etcd database token")
	flag.BoolVar(&e.enableCorruptionCheck, "enable-corruption-check", false, "enable etcd experimental corruption check")
	flag.Parse()

	if e.cluster == "" {
		return errors.New("-namespace is not set")
	}
	if e.podName == "" {
		return errors.New("-pod-name is not set")
	}

	if e.podIP == "" {
		return errors.New("-pod-ip is not set")
	}

	if e.etcdctlAPIVersion != "2" && e.etcdctlAPIVersion != "3" {
		return errors.New("-api-version is either 2 or 3")
	}

	if e.token == "" {
		return errors.New("-token is not set")
	}

	e.dataDir = fmt.Sprintf("/var/run/etcd/pod_%s/", e.podName)

	return nil
}

func etcdCmd(config *etcdCluster) []string {
	cmd := []string{
		fmt.Sprintf("--name=%s", config.podName),
		fmt.Sprintf("--data-dir=%s", config.dataDir),
		fmt.Sprintf("--initial-cluster=%s", strings.Join(config.initialMembers, ",")),
		fmt.Sprintf("--initial-cluster-token=%s", config.token),
		fmt.Sprintf("--initial-cluster-state=%s", config.initialState),
		fmt.Sprintf("--advertise-client-urls=https://%s.etcd.%s.svc.cluster.local:2379,https://%s:2379", config.podName, config.namespace, config.podIP),
		fmt.Sprintf("--listen-client-urls=https://%s:2379,https://127.0.0.1:2379", config.podIP),
		fmt.Sprintf("--listen-metrics-urls=http://%s:2378,http://127.0.0.1:2378", config.podIP),
		"--client-cert-auth",
		fmt.Sprintf("--trusted-ca-file=%s", resources.EtcdTrustedCAFile),
		fmt.Sprintf("--cert-file=%s", resources.EtcdCertFile),
		fmt.Sprintf("--key-file=%s", resources.EtcdKeyFile),
		fmt.Sprintf("--peer-cert-file=%s", resources.EtcdCertFile),
		fmt.Sprintf("--peer-key-file=%s", resources.EtcdKeyFile),
		fmt.Sprintf("--peer-trusted-ca-file=%s", resources.EtcdTrustedCAFile),
		"--auto-compaction-retention=8",
	}

	// set TLS only peer URLs
	if config.usePeerTLSOnly {
		cmd = append(cmd, []string{
			fmt.Sprintf("--listen-peer-urls=https://%s:2381", config.podIP),
			fmt.Sprintf("--initial-advertise-peer-urls=https://%s.etcd.%s.svc.cluster.local:2381", config.podName, config.namespace),
			"--peer-client-cert-auth",
		}...)
	} else {
		// 'mixed' mode clusters should start with both plaintext and HTTPS peer ports
		cmd = append(cmd, []string{
			fmt.Sprintf("--listen-peer-urls=http://%s:2380,https://%s:2381", config.podIP, config.podIP),
			fmt.Sprintf("--initial-advertise-peer-urls=http://%s.etcd.%s.svc.cluster.local:2380,https://%s.etcd.%s.svc.cluster.local:2381", config.podName, config.namespace, config.podName, config.namespace),
		}...)
	}

	if config.enableCorruptionCheck {
		cmd = append(cmd, []string{
			"--experimental-initial-corrupt-check=true",
			"--experimental-corrupt-check-time=240m",
		}...)
	}
	return cmd
}

func (e *etcdCluster) GetClusterClient() (*client.Client, error) {
	endpoints := clientEndpoints(e.clusterSize, e.namespace)
	return e.getClientWithEndpoints(endpoints)
}

func (e *etcdCluster) getLocalClient() (*client.Client, error) {
	return e.getClientWithEndpoints([]string{e.endpoint()})
}

func (e *etcdCluster) getClientWithEndpoints(eps []string) (*client.Client, error) {
	var err error
	tlsInfo := transport.TLSInfo{
		CertFile:       resources.EtcdClientCertFile,
		KeyFile:        resources.EtcdClientKeyFile,
		TrustedCAFile:  resources.EtcdTrustedCAFile,
		ClientCertAuth: true,
	}
	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client TLS config: %w", err)
	}
	for i := 0; i < 5; i++ {
		cli, err := client.New(client.Config{
			Endpoints:   eps,
			DialTimeout: 2 * time.Second,
			TLS:         tlsConfig,
		})
		if err == nil && cli != nil {
			return cli, nil
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("failed to establish client connection: %w", err)
}

func (e *etcdCluster) listMembers(ctx context.Context, log *zap.SugaredLogger) ([]*etcdserverpb.Member, error) {
	client, err := e.getClientWithEndpoints(clientEndpoints(e.clusterSize, e.namespace))
	if err != nil {
		return nil, fmt.Errorf("can't find cluster client: %w", err)
	}
	defer closeClient(client, log)

	ctx, cancelFunc := context.WithTimeout(ctx, timeoutListMembers)
	defer cancelFunc()

	resp, err := client.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Members, err
}

func (e *etcdCluster) GetMemberByName(ctx context.Context, log *zap.SugaredLogger, memberName string) (*etcdserverpb.Member, error) {
	members, err := e.listMembers(ctx, log)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		url, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return nil, err
		}
		// if the member is not started yet, its name would be empty, in that case, we match for peerURL hostname
		if member.Name == memberName || url.Hostname() == fmt.Sprintf("%s.etcd.%s.svc.cluster.local", e.podName, e.namespace) {
			return member, nil
		}
	}
	return nil, nil
}

func (e *etcdCluster) getUnwantedMembers(ctx context.Context, log *zap.SugaredLogger) ([]*etcdserverpb.Member, error) {
	unwantedMembers := []*etcdserverpb.Member{}

	members, err := e.listMembers(ctx, log)
	if err != nil {
		return []*etcdserverpb.Member{}, err
	}

	expectedMembers := peerHostsList(e.clusterSize, e.namespace)
	for _, member := range members {
		if len(member.GetPeerURLs()) != 1 && len(member.GetPeerURLs()) != 2 {
			unwantedMembers = append(unwantedMembers, member)
			continue
		}

		// check all found peer URLs for being valid
		for i := 0; i < len(member.PeerURLs); i++ {
			peerURL, err := url.Parse(member.PeerURLs[i])
			if err != nil {
				return []*etcdserverpb.Member{}, err
			}
			if !contains(expectedMembers, peerURL.Hostname()) {
				unwantedMembers = append(unwantedMembers, member)
			}
		}
	}

	return unwantedMembers, nil
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func (e *etcdCluster) IsClusterHealthy(ctx context.Context, log *zap.SugaredLogger) (bool, error) {
	return e.isHealthyWithEndpoints(ctx, log, clientEndpoints(e.clusterSize, e.namespace))
}

func (e *etcdCluster) isHealthyWithEndpoints(ctx context.Context, log *zap.SugaredLogger, endpoints []string) (bool, error) {
	client, err := e.getClientWithEndpoints(endpoints)
	if err != nil {
		return false, err
	}
	defer closeClient(client, log)
	// just get a key from etcd, this is how `etcdctl endpoint health` works!
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	_, err = client.Get(ctx, "healthy")
	defer cancel()
	if err != nil && !errors.Is(err, rpctypes.ErrPermissionDenied) {
		// silently swallow/drop transient errors
		return false, nil
	}
	return true, nil
}

func (e *etcdCluster) isLeader(ctx context.Context, log *zap.SugaredLogger) (bool, error) {
	localClient, err := e.getLocalClient()
	if err != nil {
		return false, err
	}
	defer closeClient(localClient, log)

	for i := 0; i < 10; i++ {
		resp, err := localClient.Status(ctx, e.endpoint())
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

func (e *etcdCluster) removeDeadMembers(ctx context.Context, log *zap.SugaredLogger, unwantedMembers []*etcdserverpb.Member) error {
	client, err := e.GetClusterClient()
	if err != nil {
		return fmt.Errorf("can't find cluster client: %w", err)
	}
	defer closeClient(client, log)

	for _, member := range unwantedMembers {
		log.Infow("checking cluster member for removal", "member-name", member.Name)

		if member.Name == e.podName {
			continue
		}

		if err = wait.Poll(1*time.Second, 15*time.Second, func() (bool, error) {
			// attempt to update member in case a client URL has recently been added
			if m, err := e.GetMemberByName(ctx, log, member.Name); err != nil {
				return false, err
			} else if m != nil {
				member = m
			}

			if len(member.ClientURLs) == 0 {
				return false, nil
			}

			// we use the cluster FQDN endpoint url here. Using the IP endpoint will
			// fail because the certificates don't include Pod IP addresses.
			return e.isHealthyWithEndpoints(ctx, log, member.ClientURLs[len(member.ClientURLs)-1:])
		}); err != nil {
			log.Infow("member is not responding, removing from cluster", "member-name", member.Name)
			ctx, cancelFunc := context.WithTimeout(ctx, timeoutRemoveMember)
			defer cancelFunc()
			_, err = client.MemberRemove(ctx, member.ID)
			return err
		}
	}
	return nil
}

func (e *etcdCluster) restoreDatadirFromBackupIfNeeded(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	restoreList := &kubermaticv1.EtcdRestoreList{}
	if err := seedClient.List(ctx, restoreList, &ctrlruntimeclient.ListOptions{Namespace: e.namespace}); err != nil {
		return fmt.Errorf("failed to list EtcdRestores: %w", err)
	}

	var activeRestore *kubermaticv1.EtcdRestore
	for _, restore := range restoreList.Items {
		if restore.Status.Phase == kubermaticv1.EtcdRestorePhaseStsRebuilding {
			if activeRestore != nil {
				return fmt.Errorf("found more than one restore in state %v, refusing to restore anything", kubermaticv1.EtcdRestorePhaseStsRebuilding)
			}
			activeRestore = restore.DeepCopy()
		}
	}
	if activeRestore == nil {
		// no active restores for this cluster
		return nil
	}

	log.Infow("restoring datadir from backup", "backup-name", activeRestore.Spec.BackupName)

	s3Client, bucketName, err := resources.GetEtcdRestoreS3Client(ctx, activeRestore, false, seedClient, cluster, nil)
	if err != nil {
		return fmt.Errorf("failed to get s3 client: %w", err)
	}

	objectName := fmt.Sprintf("%s-%s", cluster.GetName(), activeRestore.Spec.BackupName)
	downloadedSnapshotFile := fmt.Sprintf("/tmp/%s", objectName)

	if err := s3Client.FGetObject(ctx, bucketName, objectName, downloadedSnapshotFile, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("failed to download backup (%s/%s): %w", bucketName, objectName, err)
	}

	if err := os.RemoveAll(e.dataDir); err != nil {
		return fmt.Errorf("error deleting data directory before restore (%s): %w", e.dataDir, err)
	}

	sp := snapshot.NewV3(log.Desugar())

	return sp.Restore(snapshot.RestoreConfig{
		SnapshotPath:        downloadedSnapshotFile,
		Name:                e.podName,
		OutputDataDir:       e.dataDir,
		OutputWALDir:        filepath.Join(e.dataDir, "member", "wal"),
		PeerURLs:            []string{fmt.Sprintf("https://%s.etcd.%s.svc.cluster.local:2381", e.podName, e.namespace)},
		InitialCluster:      strings.Join(initialMemberList(ctx, log, e.clusterClient, e.clusterSize, e.namespace, e.usePeerTLSOnly), ","),
		InitialClusterToken: e.token,
		SkipHashCheck:       false,
	})
}

func closeClient(c io.Closer, log *zap.SugaredLogger) {
	err := c.Close()
	if err != nil {
		log.Warn(zap.Error(err))
	}
}

func (e *etcdCluster) init(ctx context.Context) (*kubermaticv1.Cluster, error) {
	cluster := &kubermaticv1.Cluster{}
	key := types.NamespacedName{Name: e.cluster}
	if err := e.clusterClient.Get(ctx, key, cluster); err != nil {
		return nil, err
	}

	e.namespace = cluster.Status.NamespaceName

	return cluster, nil
}

func (e *etcdCluster) KubermaticCluster(ctx context.Context) (*kubermaticv1.Cluster, error) {
	cluster := &kubermaticv1.Cluster{}
	key := types.NamespacedName{Name: e.cluster}
	if err := e.clusterClient.Get(ctx, key, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

func (e *etcdCluster) SetInitialState(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	// check if the etcd cluster is initialized successfully.
	if cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEtcdClusterInitialized, corev1.ConditionTrue) {
		e.initialState = initialStateExisting
		// if "strict" mode is enforced, set it up for existing clusters
		if os.Getenv(envPeerTLSMode) == peerTLSModeStrict {
			e.usePeerTLSOnly = true
		}
	} else {
		e.initialState = initialStateNew
		// new clusters can use "strict" TLS mode for etcd (TLS-only peering connections)
		e.usePeerTLSOnly = true

		if err := e.restoreDatadirFromBackupIfNeeded(ctx, log, e.clusterClient, cluster); err != nil {
			return fmt.Errorf("failed to restore datadir from backup: %w", err)
		}
	}

	return nil
}

func (e *etcdCluster) SetClusterSize(ctx context.Context) error {
	sts := &appsv1.StatefulSet{}

	if err := e.clusterClient.Get(ctx, types.NamespacedName{Name: "etcd", Namespace: e.namespace}, sts); err != nil {
		return fmt.Errorf("failed to get etcd sts: %w", err)
	}
	e.clusterSize = defaultClusterSize
	if sts.Spec.Replicas != nil {
		e.clusterSize = int(*sts.Spec.Replicas)
	}
	return nil
}
