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
	"go.etcd.io/etcd/v3/clientv3"
	"go.etcd.io/etcd/v3/clientv3/snapshot"
	"go.etcd.io/etcd/v3/etcdserver/api/v3rpc/rpctypes"
	"go.etcd.io/etcd/v3/etcdserver/etcdserverpb"
	"go.etcd.io/etcd/v3/pkg/transport"
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
	log.Infof("initial-cluster: %s", strings.Join(initialMemberList(e.clusterSize, e.namespace), ","))

	// setup and start etcd command
	etcdCmd, err := startEtcdCmd(e, log)
	if err != nil {
		log.Panicw("start etcd cmd", zap.Error(err))
	}

	if err = wait.Poll(1*time.Second, 30*time.Second, func() (bool, error) {
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

	// handle changes to peerURLs (https -> http)
	if err := e.updatePeerURLToHTTP(log); err != nil {
		log.Panicw("failed to update peerURL", zap.Error(err))
	}

	// reconcile dead members continuously. Initially we did this once as a step at the end of start up. We did that because scale up/down operations required a full restart of the ring with each node add/remove. However, this is no longer the case, so we need to separate the reconcile from the start up process and do it continuously.
	go func() {
		wait.Forever(func() {
			if _, err := deleteUnwantedDeadMembers(e, log); err != nil {
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
	containsUnwantedMembers, err := e.containsUnwantedMembers(log)
	if err != nil {
		log.Warnw("failed to list members ", zap.Error(err))
		return false, nil
	}
	// we only need to reconcile if we have members that we shouldn't have
	if !containsUnwantedMembers {
		log.Info("cluster members reconciled..")
		return true, nil
	}
	// to avoide race conditions, we will run only on the cluster leader
	leader, err := e.isLeader(log)
	if err != nil || !leader {
		log.Warnw("failed to remove member, error occurred or didn't get the current leader", zap.Error(err))
		return false, nil
	}
	if err := e.removeDeadMembers(log); err != nil {
		log.Warnw("failed to remove member", zap.Error(err))
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

	if _, err := client.MemberAdd(context.Background(), []string{fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.podName, e.namespace)}); err != nil {
		close(client, log)
		return errors.Wrap(err, "add itself as a member")
	}

	defer close(client, log)

	log.Info("joined etcd cluster successfully.")
	return nil
}

func (e *etcdCluster) updatePeerURLToHTTP(log *zap.SugaredLogger) error {
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
		if member.Name == e.podName && peerURL.Scheme == "https" {
			peerURL.Scheme = "http"
			_, err = client.MemberUpdate(context.Background(), member.ID, []string{peerURL.String()})
			return err
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
	endpoints := clientEndpoints(e.clusterSize, e.namespace)
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

func (e *etcdCluster) listMembers(log *zap.SugaredLogger) ([]*etcdserverpb.Member, error) {
	client, err := e.getClientWithEndpoints(clientEndpoints(e.clusterSize, e.namespace))
	if err != nil {
		return nil, fmt.Errorf("can't find cluster client: %v", err)
	}
	defer close(client, log)

	resp, err := client.MemberList(context.Background())
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
		// if the member is not started yet, its name would be empty, in that case, we match for peerURL host.
		if member.Name == name || url.Host == fmt.Sprintf("%s.etcd.%s.svc.cluster.local:2380", e.podName, e.namespace) {
			return member, nil
		}
	}
	return nil, nil
}

func (e *etcdCluster) containsUnwantedMembers(log *zap.SugaredLogger) (bool, error) {
	members, err := e.listMembers(log)
	if err != nil {
		return false, err
	}
	expectedMembers := peerURLsList(e.clusterSize, e.namespace)
	for _, member := range members {
		if len(member.GetPeerURLs()) != 1 {
			return true, nil
		}
		peerURL, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return false, err
		}
		if !contains(expectedMembers, peerURL.Host) {
			return true, nil
		}
	}
	return false, nil
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

func (e *etcdCluster) removeDeadMembers(log *zap.SugaredLogger) error {
	members, err := e.listMembers(log)
	if err != nil {
		return err
	}

	client, err := e.getClusterClient()
	if err != nil {
		return fmt.Errorf("can't find cluster client: %v", err)
	}
	defer close(client, log)

	for _, member := range members {
		if member.Name == e.podName {
			continue
		}
		if err = wait.Poll(1*time.Second, 15*time.Second, func() (bool, error) {
			// we use the cluster FQDN endpoint url here. Using the IP endpoint will
			// fail because the certificates don't include Pod IP addresses.
			return e.isHealthyWithEndpoints(member.ClientURLs[len(member.ClientURLs)-1:], log)
		}); err != nil {
			log.Infow("member is not responding, removing from cluster", "member-name", member.Name)
			_, err = client.MemberRemove(context.Background(), member.ID)
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
		PeerURLs:            []string{fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.podName, e.namespace)},
		InitialCluster:      strings.Join(initialMemberList(e.clusterSize, e.namespace), ","),
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
	} else {
		e.initialState = initialStateNew
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
