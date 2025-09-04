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

package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/cmd/etcd-launcher/pkg/etcd"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	"k8s.io/apimachinery/pkg/util/wait"
)

type runOptions struct {
	options

	podName string
	podIP   string
	token   string
	dataDir string

	enableCorruptionCheck bool

	quotaBackendBytes int64
}

func RunCommand(logger *zap.SugaredLogger) *cobra.Command {
	opt := runOptions{}

	cmd := &cobra.Command{
		Use:          "run",
		Short:        "Launch an etcd cluster for a specific KKP user cluster",
		RunE:         RunFunc(logger, &opt),
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.CopyInto(&opt.options)

			if opt.podName == "" {
				return fmt.Errorf("--pod-name cannot be empty")
			}

			if opt.podIP == "" {
				return fmt.Errorf("--pod-ip cannot be empty")
			}

			if opt.token == "" {
				return fmt.Errorf("--token cannot be empty")
			}

			opt.dataDir = fmt.Sprintf("/var/run/etcd/pod_%s/", opt.podName)

			return nil
		},
	}

	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if err := c.Usage(); err != nil {
			return err
		}

		// ensure we exit with code 1 later on
		return err
	})

	cmd.PersistentFlags().StringVar(&opt.podName, "pod-name", "", "name of this etcd pod")
	cmd.PersistentFlags().StringVar(&opt.podIP, "pod-ip", "", "IP address of this etcd pod")
	cmd.PersistentFlags().StringVar(&opt.token, "token", "", "etcd database token")
	cmd.PersistentFlags().BoolVar(&opt.enableCorruptionCheck, "enable-corruption-check", false, "enable experimental corruption check")
	cmd.PersistentFlags().Int64Var(&opt.quotaBackendBytes, "quota-backend-gb", 0, "maximum backend size of etcd in gb (0 means use etcd default)")

	return cmd
}

func RunFunc(log *zap.SugaredLogger, opt *runOptions) cobraFuncE {
	return handleErrors(log, func(cmd *cobra.Command, args []string) error {
		var err error

		e := &etcd.Cluster{
			Cluster:           opt.cluster,
			EtcdctlAPIVersion: opt.etcdctlAPIVersion,

			CaCertFile:     opt.etcdCAFile,
			ClientCertFile: opt.etcdCertFile,
			ClientKeyFile:  opt.etcdKeyFile,

			PodName:               opt.podName,
			PodIP:                 opt.podIP,
			DataDir:               opt.dataDir,
			Token:                 opt.token,
			EnableCorruptionCheck: opt.enableCorruptionCheck,
			QuotaBackendGB:        opt.quotaBackendBytes,
		}

		ctx := cmd.Context()

		reconciling.Configure(log)

		// init the current cluster object; we only care about the namespace name
		// (which is practically immutable) and so it's sufficient to fetch the
		// cluster now, once.
		kkpCluster, err := e.Init(ctx)
		if err != nil {
			log.Panicw("failed to initialise cluster state", zap.Error(err))
		}

		if err := e.SetClusterSize(ctx); err != nil {
			log.Panicw("failed to set cluster size", zap.Error(err))
		}

		if err := e.SetInitialState(ctx, log, kkpCluster); err != nil {
			log.Panicw("failed to set initialState", zap.Error(err))
		}

		e.SetInitialMembers(ctx, log)
		e.LogInitialState(log)

		if e.Exists() {
			// if the cluster already exists, try to connect and update peer URLs that might be out of sync.
			// etcd might fail to start if peer URLs in the etcd member state and the flags passed to it are different.
			// make sure that peer URLs in the cluster member data is updated / in sync with the etcd node's configuration.
			if err := e.UpdatePeerURLs(ctx, log); err != nil {
				log.Warnw("failed to update peerURL, etcd node might fail to start ...", zap.Error(err))
			}

			thisMember, err := e.GetMemberByName(ctx, log, e.PodName)

			switch {
			case err != nil:
				log.Warnw("failed to check cluster membership", zap.Error(err))
			case thisMember != nil:
				log.Infof("%v is a member", thisMember.GetPeerURLs())

				if _, err := os.Stat(filepath.Join(e.DataDir, "member")); errors.Is(err, fs.ErrNotExist) {
					if err := e.RemoveStaleMember(ctx, log, thisMember.ID); err != nil {
						log.Panicw("failed to remove stale membership to rejoin cluster as new member", zap.Error(err))
					}

					if err := e.JoinCluster(ctx, log); err != nil {
						log.Panicw("failed to join cluster as fresh member", zap.Error(err))
					}
				}
			default:
				// if no membership information was found but we were able to list from an etcd cluster, we can attempt to join
				if err := e.JoinCluster(ctx, log); err != nil {
					log.Panicw("failed to join cluster as fresh member", zap.Error(err))
				}
			}
		}

		// setup and start etcd command
		etcdCmd, err := e.StartEtcdCmd(ctx, log)
		if err != nil {
			log.Panicw("failed to start etcd cmd", zap.Error(err))
		}

		if err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 60*time.Second, false, func(ctx context.Context) (bool, error) {
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
				} else if _, err := e.DeleteUnwantedDeadMembers(ctx, log); err != nil {
					log.Warnw("failed to remove dead members", zap.Error(err))
				}
			}, 30*time.Second)
		}()

		if err = etcdCmd.Wait(); err != nil {
			log.Panic(err)
		}

		return nil
	})
}
