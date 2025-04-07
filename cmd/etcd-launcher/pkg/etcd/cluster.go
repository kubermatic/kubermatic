/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package etcd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultClusterSize   = 3
	etcdCommandPath      = "/usr/local/bin/etcd"
	initialStateExisting = "existing"
	initialStateNew      = "new"
	envPeerTLSMode       = "PEER_TLS_MODE"
	peerTLSModeStrict    = "strict"

	timeoutListMembers    = time.Second * 5
	timeoutAddMember      = time.Second * 15
	timeoutRemoveMember   = time.Second * 30
	timeoutUpdatePeerURLs = time.Second * 10
)

type Cluster struct {
	Cluster        string // given as a CLI flag
	CaCertFile     string
	ClientCertFile string
	ClientKeyFile  string

	PodName               string
	PodIP                 string
	EtcdctlAPIVersion     string
	DataDir               string
	Token                 string
	EnableCorruptionCheck bool

	clusterClient ctrlruntimeclient.Client
	namespace     string // filled in later during init()

	initialState   string
	initialMembers []string
	usePeerTLSOnly bool
	clusterSize    int
}

func (e *Cluster) Init(ctx context.Context) (*kubermaticv1.Cluster, error) {
	if e.Cluster == "" {
		return nil, errors.New("--cluster is not set")
	}

	if e.EtcdctlAPIVersion != "2" && e.EtcdctlAPIVersion != "3" {
		return nil, errors.New("--api-version is either 2 or 3")
	}

	var err error

	// here we find the cluster state
	e.clusterClient, err = inClusterClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster client: %w", err)
	}

	cluster := &kubermaticv1.Cluster{}
	key := types.NamespacedName{Name: e.Cluster}
	if err := e.clusterClient.Get(ctx, key, cluster); err != nil {
		return nil, err
	}

	e.namespace = cluster.Status.NamespaceName

	return cluster, nil
}

func (e *Cluster) KubermaticCluster(ctx context.Context) (*kubermaticv1.Cluster, error) {
	cluster := &kubermaticv1.Cluster{}
	key := types.NamespacedName{Name: e.Cluster}
	if err := e.clusterClient.Get(ctx, key, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

func (e *Cluster) SetInitialState(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
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

func (e *Cluster) Exists() bool {
	return e.initialState == initialStateExisting
}

func (e *Cluster) SetInitialMembers(ctx context.Context, log *zap.SugaredLogger) {
	e.initialMembers = initialMemberList(ctx, log, e.clusterClient, e.clusterSize, e.namespace, e.usePeerTLSOnly)
}

func (e *Cluster) SetClusterSize(ctx context.Context) error {
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

func (e *Cluster) LogInitialState(log *zap.SugaredLogger) {
	log.Infow("initializing etcd",
		"initial-state", e.initialState,
		"initial-cluster", strings.Join(e.initialMembers, ","),
		"peer-tls-only", e.usePeerTLSOnly,
	)
}

func (e *Cluster) DeleteUnwantedDeadMembers(ctx context.Context, log *zap.SugaredLogger) (bool, error) {
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

func (e *Cluster) JoinCluster(ctx context.Context, log *zap.SugaredLogger) error {
	log.Info("pod is not a cluster member, trying to join..")

	// remove possibly stale member data dir..
	log.Info("removing possibly stale data dir")
	if err := os.RemoveAll(e.DataDir); err != nil {
		return fmt.Errorf("removing possible stale data dir: %w", err)
	}

	// join the cluster
	client, err := e.GetEtcdClient(ctx, log)
	if err != nil {
		return fmt.Errorf("can't find cluster client: %w", err)
	}

	// construct peer URLs for this new node

	peerURLs := []string{}

	if !e.usePeerTLSOnly {
		peerURLs = append(peerURLs, fmt.Sprintf("http://%s.etcd.%s.svc.cluster.local:2380", e.PodName, e.namespace))
	}

	peerURLs = append(peerURLs, fmt.Sprintf("https://%s.etcd.%s.svc.cluster.local:2381", e.PodName, e.namespace))

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

func (e *Cluster) RemoveStaleMember(ctx context.Context, log *zap.SugaredLogger, memberID uint64) error {
	client, err := e.GetEtcdClient(ctx, log)
	if err != nil {
		return fmt.Errorf("can't find cluster client: %w", err)
	}

	log.Warnw("No data dir, removing stale membership to rejoin cluster as new member")

	_, err = client.MemberRemove(ctx, memberID)
	if err != nil {
		closeClient(client, log)
		return fmt.Errorf("failed to remove own member information from cluster before rejoining: %w", err)
	}

	closeClient(client, log)

	return nil
}

func (e *Cluster) UpdatePeerURLs(ctx context.Context, log *zap.SugaredLogger) error {
	members, err := e.listMembers(ctx, log)
	if err != nil {
		return err
	}

	client, err := e.GetEtcdClient(ctx, log)
	if err != nil {
		return err
	}
	defer closeClient(client, log)

	for _, member := range members {
		peerURL, err := url.Parse(member.PeerURLs[0])
		if err != nil {
			return err
		}

		if member.Name == e.PodName {
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

func (e *Cluster) GetMemberByName(ctx context.Context, log *zap.SugaredLogger, memberName string) (*etcdserverpb.Member, error) {
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
		if member.Name == memberName || url.Hostname() == fmt.Sprintf("%s.etcd.%s.svc.cluster.local", e.PodName, e.namespace) {
			return member, nil
		}
	}

	return nil, nil
}

func (e *Cluster) IsClusterHealthy(ctx context.Context, log *zap.SugaredLogger) (bool, error) {
	return e.isHealthyWithEndpoints(ctx, log, clientEndpoints(e.clusterSize, e.namespace))
}

func initialMemberList(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, n int, namespace string, useTLSPeer bool) []string {
	members := []string{}
	for i := range n {
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

			if _, ok := pod.Annotations[resources.EtcdTLSEnabledAnnotation]; ok {
				members = append(
					members,
					fmt.Sprintf("etcd-%d=https://etcd-%d.etcd.%s.svc.cluster.local:2381", i, i, namespace),
				)
			}
		}
	}

	return members
}

func peerHostsList(n int, namespace string) []string {
	hosts := []string{}
	for i := range n {
		hosts = append(hosts, fmt.Sprintf("etcd-%d.etcd.%s.svc.cluster.local", i, namespace))
	}
	return hosts
}

func clientEndpoints(n int, namespace string) []string {
	endpoints := []string{}

	for i := range n {
		endpoints = append(endpoints, fmt.Sprintf("https://etcd-%d.etcd.%s.svc.cluster.local:2379", i, namespace))
	}

	return endpoints
}

func (e *Cluster) endpoint() string {
	return "https://127.0.0.1:2379"
}

func (e *Cluster) GetEtcdClient(ctx context.Context, log *zap.SugaredLogger) (*client.Client, error) {
	endpoints := clientEndpoints(e.clusterSize, e.namespace)
	return e.getClientWithEndpoints(ctx, log, endpoints)
}

// GetEtcdEndpointClients returns a slice of client configs with each config pointing to exactly one of the automatically discovered etcd endpoints.
func (e *Cluster) GetEtcdEndpointConfigs(ctx context.Context) ([]client.Config, error) {
	configs := []client.Config{}
	endpoints := clientEndpoints(e.clusterSize, e.namespace)

	tlsConfig, err := getTLSConfig(e)
	if err != nil {
		return nil, fmt.Errorf("failed to set up TLS client config: %w", err)
	}

	for _, endpoint := range endpoints {
		configs = append(configs, getConfig([]string{endpoint}, tlsConfig))
	}

	return configs, nil
}

func (e *Cluster) getLocalClient(ctx context.Context, log *zap.SugaredLogger) (*client.Client, error) {
	return e.getClientWithEndpoints(ctx, log, []string{e.endpoint()})
}

func (e *Cluster) getClientWithEndpoints(ctx context.Context, log *zap.SugaredLogger, eps []string) (*client.Client, error) {
	tlsConfig, err := getTLSConfig(e)
	if err != nil {
		return nil, fmt.Errorf("failed to set up TLS client config: %w", err)
	}

	var etcdClient *client.Client

	if err := wait.PollImmediateLog(ctx, log.With("endpoints", strings.Join(eps, ",")), 5*time.Second, 60*time.Second, func(ctx context.Context) (error, error) {
		cli, err := client.New(client.Config{
			Endpoints:   eps,
			DialTimeout: 2 * time.Second,
			TLS:         tlsConfig,
		})

		if err == nil && cli != nil {
			etcdClient = cli
			return nil, nil
		}

		return err, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to establish client connection: %w", err)
	}

	return etcdClient, nil
}

func getConfig(endpoints []string, tlsConfig *tls.Config) client.Config {
	return client.Config{
		Endpoints:   endpoints,
		DialTimeout: 2 * time.Second,
		TLS:         tlsConfig,
	}
}

func getTLSConfig(e *Cluster) (*tls.Config, error) {
	tlsInfo := transport.TLSInfo{
		CertFile:       e.ClientCertFile,
		KeyFile:        e.ClientKeyFile,
		TrustedCAFile:  e.CaCertFile,
		ClientCertAuth: true,
	}

	return tlsInfo.ClientConfig()
}

func (e *Cluster) listMembers(ctx context.Context, log *zap.SugaredLogger) ([]*etcdserverpb.Member, error) {
	client, err := e.getClientWithEndpoints(ctx, log, clientEndpoints(e.clusterSize, e.namespace))
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

func (e *Cluster) getUnwantedMembers(ctx context.Context, log *zap.SugaredLogger) ([]*etcdserverpb.Member, error) {
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
		for i := range len(member.PeerURLs) {
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

func (e *Cluster) isHealthyWithEndpoints(ctx context.Context, log *zap.SugaredLogger, endpoints []string) (bool, error) {
	client, err := e.getClientWithEndpoints(ctx, log, endpoints)
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

func (e *Cluster) isLeader(ctx context.Context, log *zap.SugaredLogger) (bool, error) {
	localClient, err := e.getLocalClient(ctx, log)
	if err != nil {
		return false, err
	}
	defer closeClient(localClient, log)

	for range 10 {
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

func (e *Cluster) removeDeadMembers(ctx context.Context, log *zap.SugaredLogger, unwantedMembers []*etcdserverpb.Member) error {
	client, err := e.GetEtcdClient(ctx, log)
	if err != nil {
		return fmt.Errorf("can't find cluster client: %w", err)
	}
	defer closeClient(client, log)

	for _, member := range unwantedMembers {
		log.Infow("checking cluster member for removal", "member-name", member.Name)

		if member.Name == e.PodName {
			continue
		}

		if err = wait.Poll(ctx, 1*time.Second, 15*time.Second, func(ctx context.Context) (error, error) {
			// attempt to update member in case a client URL has recently been added
			if m, err := e.GetMemberByName(ctx, log, member.Name); err != nil {
				return err, nil
			} else if m != nil {
				member = m
			}

			if len(member.ClientURLs) == 0 {
				return fmt.Errorf("no client URLs are found"), nil
			}

			// we use the cluster FQDN endpoint url here. Using the IP endpoint will
			// fail because the certificates don't include Pod IP addresses.
			healthy, err := e.isHealthyWithEndpoints(ctx, log, member.ClientURLs[len(member.ClientURLs)-1:])
			if err != nil {
				return fmt.Errorf("failed to check health: %w", err), nil
			}

			if !healthy {
				return fmt.Errorf("endpoints are not healthy"), nil
			}

			return nil, nil
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

func (e *Cluster) restoreDatadirFromBackupIfNeeded(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
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

	rawBackupFile, err := DecompressSnapshot(downloadedSnapshotFile)
	if err != nil {
		return fmt.Errorf("failed to decompress snapshot file %s: %w", objectName, err)
	}

	if err := os.RemoveAll(e.DataDir); err != nil {
		return fmt.Errorf("error deleting data directory before restore (%s): %w", e.DataDir, err)
	}

	sp := snapshot.NewV3(log.Desugar())

	return sp.Restore(snapshot.RestoreConfig{
		SnapshotPath:        rawBackupFile,
		Name:                e.PodName,
		OutputDataDir:       e.DataDir,
		OutputWALDir:        filepath.Join(e.DataDir, "member", "wal"),
		PeerURLs:            []string{fmt.Sprintf("https://%s.etcd.%s.svc.cluster.local:2381", e.PodName, e.namespace)},
		InitialCluster:      strings.Join(initialMemberList(ctx, log, e.clusterClient, e.clusterSize, e.namespace, e.usePeerTLSOnly), ","),
		InitialClusterToken: e.Token,
		SkipHashCheck:       false,
	})
}
