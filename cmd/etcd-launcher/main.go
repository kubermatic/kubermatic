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
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go"
	"github.com/pkg/errors"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	client "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
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
	envPeerTLSMode           = "PEER_TLS_MODE"
	peerTLSModeStrict        = "strict"

	timeoutListMembers    = time.Second * 5
	timeoutAddMember      = time.Second * 15
	timeoutRemoveMember   = time.Second * 30
	timeoutUpdatePeerURLs = time.Second * 10
)

type etcdCluster struct {
	namespace             string
	clusterSize           int
	podName               string
	podIP                 string
	etcdctlAPIVersion     string
	dataDir               string
	token                 string
	enableCorruptionCheck bool
	initialState          string
	usePeerTLSOnly        bool
}

func main() {
	log := createLogger()

	// normal workflow, we don't do migrations anymore
	e := &etcdCluster{}
	err := e.parseConfigFlags()
	if err != nil {
		log.Panicw("failed to get launcher configuration", zap.Error(err))
	}

	// here we find the cluster state
	clusterClient, err := inClusterClient(log)
	if err != nil {
		log.Panicw("failed to get in-cluster client", zap.Error(err))
	}

	if err := e.setClusterSize(clusterClient); err != nil {
		log.Panicw("failed to set cluster size", zap.Error(err))
	}

	if err := e.setInitialState(clusterClient, log); err != nil {
		log.Panicw("failed to set initialState", zap.Error(err))
	}

	log.Info("initializing etcd..")
	log.Infof("initial-state: %s", e.initialState)
	log.Infof("initial-cluster: %s", strings.Join(initialMemberList(e.clusterSize, e.namespace, e.usePeerTLSOnly), ","))
	if e.usePeerTLSOnly {
		log.Info("peer-tls-mode: strict")
	}

	// if the cluster already exists, try to connect and update peer URLs that might be out of sync.
	// etcd might fail to start if peer URLs in the etcd member state and the flags passed to it are different
	if e.initialState == initialStateExisting {
		// make sure that peer URLs in the cluster member data is
		// updated / in sync with the etcd node's configuration
		if err := e.updatePeerURLs(log); err != nil {
			log.Warnw("failed to update peerURL, etcd node might fail to start ...", zap.Error(err))
		}
	}

	// setup and start etcd command
	etcdCmd, err := startEtcdCmd(e, log)
	if err != nil {
		log.Panicw("start etcd cmd", zap.Error(err))
	}

	if err = wait.Poll(1*time.Second, 60*time.Second, func() (bool, error) {
		return e.isClusterHealthy(log)
	}); err != nil {
		log.Panicw("manager thread failed to connect to cluster", zap.Error(err))
	}

	thisMember, err := e.getMemberByName(e.podName, log)
	if err != nil {
		log.Panicw("failed to check cluster membership", zap.Error(err))
	}
	if thisMember != nil {
		log.Infof("%v is a member", thisMember.GetPeerURLs())

		if _, err := os.Stat(filepath.Join(e.dataDir, "member")); os.IsNotExist(err) {
			client, err := e.getClusterClient()
			if err != nil {
				log.Panicw("can't find cluster client: %v", zap.Error(err))
			}
			log.Warnw("No data dir, to ensure recovery removing and adding the member")
			_, err = client.MemberRemove(context.Background(), thisMember.ID)
			if err != nil {
				close(client, log)
				log.Panicw("remove itself due to data dir loss", zap.Error(err))
			}
			close(client, log)
			if err := joinCluster(e, log); err != nil {
				log.Panicw("join cluster as fresh member", zap.Error(err))
			}
		}
	} else if err := joinCluster(e, log); err != nil {
		log.Panicw("join cluster as fresh member", zap.Error(err))
	}

	// reconcile dead members continuously. Initially we did this once as a step at the end of start up. We did that because scale up/down operations required a full restart of the ring with each node add/remove. However, this is no longer the case, so we need to separate the reconcile from the start up process and do it continuously.
	go func() {
		wait.Forever(func() {
			// refresh the cluster size so the etcd-launcher is aware of scaling operations
			if err := e.setClusterSize(clusterClient); err != nil {
				log.Warnw("failed to refresh cluster size", zap.Error(err))
			} else if _, err := deleteUnwantedDeadMembers(e, log); err != nil {
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
		return nil, errors.Wrap(err, "failed to get in cluster config")
	}
	client, err := ctrlruntimeclient.New(config, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cluster client")
	}
	return client, nil
}

func getK8cCluster(client ctrlruntimeclient.Client, name string, log *zap.SugaredLogger) (*kubermaticv1.Cluster, error) {
	ret := kubermaticv1.Cluster{}
	if err := client.Get(context.Background(), types.NamespacedName{Name: name, Namespace: ""}, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

func startEtcdCmd(e *etcdCluster, log *zap.SugaredLogger) (*exec.Cmd, error) {
	if _, err := os.Stat(etcdCommandPath); os.IsNotExist(err) {
		return nil, errors.Wrap(err, "find etcd executable")
	}

	cmd := exec.Command(etcdCommandPath, etcdCmd(e)...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Infof("starting etcd command: %s %s", etcdCommandPath, strings.Join(etcdCmd(e), " "))
	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start etcd")
	}
	return cmd, nil
}

func deleteUnwantedDeadMembers(e *etcdCluster, log *zap.SugaredLogger) (bool, error) {
	unwantedMembers, err := e.getUnwantedMembers(log)
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
	leader, err := e.isLeader(log)
	if err != nil {
		log.Warnw("failed to determine if member is cluster leader", zap.Error(err))
		return false, nil
	}

	if !leader {
		log.Info("current member is not leader, skipping dead member removal")
		return false, nil
	}

	if err := e.removeDeadMembers(log, unwantedMembers); err != nil {
		return false, err
	}

	return false, nil
}

func joinCluster(e *etcdCluster, log *zap.SugaredLogger) error {
	log.Info("pod is not a cluster member, trying to join..")
	// remove possibly stale member data dir..
	log.Info("removing possibly stale data dir")
	if err := os.RemoveAll(e.dataDir); err != nil {
		return errors.Wrap(err, "removing possible stale data dir")
	}
	// join the cluster
	client, err := e.getClusterClient()
	if err != nil {
		return errors.Wrap(err, "can't find cluster client")
	}

	// construct peer URLs for this new node

	peerURLs := []string{}

	if !e.usePeerTLSOnly {
		peerURLs = append(peerURLs, fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.podName, e.namespace))
	}

	peerURLs = append(peerURLs, fmt.Sprintf("https://%s.etcd.%s.svc.cluster.local:2381", e.podName, e.namespace))

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeoutAddMember)
	defer cancelFunc()

	if _, err := client.MemberAdd(ctx, peerURLs); err != nil {
		close(client, log)
		return errors.Wrap(err, "add itself as a member")
	}

	defer close(client, log)

	log.Info("joined etcd cluster successfully.")
	return nil
}

func (e *etcdCluster) updatePeerURLs(log *zap.SugaredLogger) error {
	members, err := e.listMembers(log)
	if err != nil {
		return err
	}
	client, err := e.getClusterClient()
	if err != nil {
		return err
	}

	defer close(client, log)

	for _, member := range members {
		peerURL, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return err
		}

		if member.Name == e.podName {
			ctx, cancelFunc := context.WithTimeout(context.Background(), timeoutUpdatePeerURLs)
			defer cancelFunc()
			// if both plaintext and TLS peer URLs are supposed to be present
			// update the member to include both plaintext and TLS peer URLs
			if !e.usePeerTLSOnly && (len(member.PeerURLs) == 1 || peerURL.Scheme != "http") {
				plainPeerURL, err := url.Parse(fmt.Sprintf("http://%s:2380", peerURL.Hostname()))
				if err != nil {
					return err
				}

				tlsPeerURL, err := url.Parse(fmt.Sprintf("https://%s:2381", peerURL.Hostname()))
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
				tlsPeerURL, err := url.Parse(fmt.Sprintf("https://%s:2381", peerURL.Hostname()))
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

func initialMemberList(n int, namespace string, useTLSPeer bool) []string {
	format := "etcd-%d=http://etcd-%d.etcd.%s.svc.cluster.local:2380"

	if useTLSPeer {
		format = "etcd-%d=https://etcd-%d.etcd.%s.svc.cluster.local:2381"
	}

	members := []string{}
	for i := 0; i < n; i++ {
		members = append(members, fmt.Sprintf(format, i, i, namespace))
	}
	return members
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
	flag.StringVar(&e.namespace, "namespace", "", "namespace of the user cluster")
	flag.StringVar(&e.podName, "pod-name", "", "name of this etcd pod")
	flag.StringVar(&e.podIP, "pod-ip", "", "IP address of this etcd pod")
	flag.StringVar(&e.etcdctlAPIVersion, "api-version", defaultEtcdctlAPIVersion, "etcdctl API version")
	flag.StringVar(&e.token, "token", "", "etcd database token")
	flag.BoolVar(&e.enableCorruptionCheck, "enable-corruption-check", false, "enable etcd experimental corruption check")
	flag.Parse()

	if e.namespace == "" {
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
		fmt.Sprintf("--initial-cluster=%s", strings.Join(initialMemberList(config.clusterSize, config.namespace, config.usePeerTLSOnly), ",")),
		fmt.Sprintf("--initial-cluster-token=%s", config.token),
		fmt.Sprintf("--initial-cluster-state=%s", config.initialState),
		fmt.Sprintf("--advertise-client-urls=https://%s.etcd.%s.svc.cluster.local:2379,https://%s:2379", config.podName, config.namespace, config.podIP),
		fmt.Sprintf("--listen-client-urls=https://%s:2379,https://127.0.0.1:2379", config.podIP),
		fmt.Sprintf("--listen-metrics-urls=http://%s:2378,http://127.0.0.1:2378", config.podIP),
		"--client-cert-auth",
		fmt.Sprintf("--trusted-ca-file=%s", resources.EtcdTrustedCAFile),
		fmt.Sprintf("--cert-file=%s", resources.EtcdCertFile),
		fmt.Sprintf("--key-file=%s", resources.EtcdKetFile),
		fmt.Sprintf("--peer-cert-file=%s", resources.EtcdCertFile),
		fmt.Sprintf("--peer-key-file=%s", resources.EtcdKetFile),
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
			"--experimental-corrupt-check-time=10m",
		}...)
	}
	return cmd
}

func (e *etcdCluster) getClusterClient() (*client.Client, error) {
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
		return nil, fmt.Errorf("failed to generate client TLS config: %v", err)
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
	return nil, fmt.Errorf("failed to establish client connection: %v", err)

}

func (e *etcdCluster) listMembers(log *zap.SugaredLogger) ([]*etcdserverpb.Member, error) {
	client, err := e.getClientWithEndpoints(clientEndpoints(e.clusterSize, e.namespace))
	if err != nil {
		return nil, fmt.Errorf("can't find cluster client: %v", err)
	}
	defer close(client, log)

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeoutListMembers)
	defer cancelFunc()

	resp, err := client.MemberList(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Members, err
}

func (e *etcdCluster) getMemberByName(name string, log *zap.SugaredLogger) (*etcdserverpb.Member, error) {
	members, err := e.listMembers(log)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		url, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return nil, err
		}
		// if the member is not started yet, its name would be empty, in that case, we match for peerURL hostname
		if member.Name == name || url.Hostname() == fmt.Sprintf("%s.etcd.%s.svc.cluster.local", e.podName, e.namespace) {
			return member, nil
		}
	}
	return nil, nil
}

func (e *etcdCluster) getUnwantedMembers(log *zap.SugaredLogger) ([]*etcdserverpb.Member, error) {
	unwantedMembers := []*etcdserverpb.Member{}

	members, err := e.listMembers(log)
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

func (e *etcdCluster) isClusterHealthy(log *zap.SugaredLogger) (bool, error) {
	return e.isHealthyWithEndpoints(clientEndpoints(e.clusterSize, e.namespace), log)
}

func (e *etcdCluster) isHealthyWithEndpoints(endpoints []string, log *zap.SugaredLogger) (bool, error) {
	client, err := e.getClientWithEndpoints(endpoints)
	if err != nil {
		return false, err
	}
	defer close(client, log)
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

func (e *etcdCluster) isLeader(log *zap.SugaredLogger) (bool, error) {
	localClient, err := e.getLocalClient()
	if err != nil {
		return false, err
	}
	defer close(localClient, log)

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

func (e *etcdCluster) removeDeadMembers(log *zap.SugaredLogger, unwantedMembers []*etcdserverpb.Member) error {
	client, err := e.getClusterClient()
	if err != nil {
		return fmt.Errorf("can't find cluster client: %v", err)
	}
	defer close(client, log)

	for _, member := range unwantedMembers {
		log.Infow("checking cluster member for removal", "member-name", member.Name)

		if member.Name == e.podName {
			continue
		}
		if err = wait.Poll(1*time.Second, 15*time.Second, func() (bool, error) {
			if len(member.ClientURLs) == 0 {
				return false, nil
			}

			// we use the cluster FQDN endpoint url here. Using the IP endpoint will
			// fail because the certificates don't include Pod IP addresses.
			return e.isHealthyWithEndpoints(member.ClientURLs[len(member.ClientURLs)-1:], log)
		}); err != nil {
			log.Infow("member is not responding, removing from cluster", "member-name", member.Name)
			ctx, cancelFunc := context.WithTimeout(context.Background(), timeoutRemoveMember)
			defer cancelFunc()
			_, err = client.MemberRemove(ctx, member.ID)
			return err
		}
	}
	return nil
}

func (e *etcdCluster) restoreDatadirFromBackupIfNeeded(ctx context.Context, k8cCluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	restoreList := &kubermaticv1.EtcdRestoreList{}
	if err := client.List(ctx, restoreList, &ctrlruntimeclient.ListOptions{Namespace: k8cCluster.Status.NamespaceName}); err != nil {
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

	s3Client, bucketName, err := resources.GetEtcdRestoreS3Client(ctx, activeRestore, false, client, k8cCluster)
	if err != nil {
		return fmt.Errorf("failed to get s3 client: %w", err)
	}

	objectName := fmt.Sprintf("%s-%s", k8cCluster.GetName(), activeRestore.Spec.BackupName)
	downloadedSnapshotFile := fmt.Sprintf("/tmp/%s", objectName)

	if err := s3Client.FGetObject(bucketName, objectName, downloadedSnapshotFile, minio.GetObjectOptions{}); err != nil {
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
		InitialCluster:      strings.Join(initialMemberList(e.clusterSize, e.namespace, e.usePeerTLSOnly), ","),
		InitialClusterToken: e.token,
		SkipHashCheck:       false,
	})
}

func close(c io.Closer, log *zap.SugaredLogger) {
	err := c.Close()
	if err != nil {
		log.Warn(zap.Error(err))
	}
}

func (e *etcdCluster) setInitialState(clusterClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	k8cCluster, err := getK8cCluster(clusterClient, strings.ReplaceAll(e.namespace, "cluster-", ""), log)
	if err != nil {
		return fmt.Errorf("failed to get user cluster: %v", err)
	}

	// check if the etcd cluster is initialized successfully.
	if k8cCluster.Status.HasConditionValue(kubermaticv1.ClusterConditionEtcdClusterInitialized, corev1.ConditionTrue) {
		e.initialState = initialStateExisting
		// if "strict" mode is enforced, set it up for existing clusters
		if os.Getenv(envPeerTLSMode) == peerTLSModeStrict {
			e.usePeerTLSOnly = true
		}
	} else {
		e.initialState = initialStateNew
		// new clusters can use "strict" TLS mode for etcd (TLS-only peering connections)
		e.usePeerTLSOnly = true

		if err := e.restoreDatadirFromBackupIfNeeded(context.Background(), k8cCluster, clusterClient, log); err != nil {
			return fmt.Errorf("failed to restore datadir from backup: %v", err)
		}
	}
	return nil
}

func (e *etcdCluster) setClusterSize(clusterClient ctrlruntimeclient.Client) error {
	sts := &appsv1.StatefulSet{}

	if err := clusterClient.Get(context.Background(), types.NamespacedName{Name: "etcd", Namespace: e.namespace}, sts); err != nil {
		return fmt.Errorf("failed to get etcd sts: %v", err)
	}
	e.clusterSize = defaultClusterSize
	if sts.Spec.Replicas != nil {
		e.clusterSize = int(*sts.Spec.Replicas)
	}
	return nil
}
