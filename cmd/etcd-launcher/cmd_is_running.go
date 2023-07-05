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
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/cmd/etcd-launcher/pkg/etcd"
	"k8c.io/kubermatic/v2/pkg/util/wait"
)

type isRunningOptions struct {
	options

	testKey         string
	testValue       string
	intervalSeconds int
	timeoutSeconds  int
}

func IsRunningCommand(logger *zap.SugaredLogger) *cobra.Command {
	opt := isRunningOptions{}

	cmd := &cobra.Command{
		Use:          "is-running",
		Short:        "Check if the etcd cluster for a specific user cluster is available by writing to its KV store",
		RunE:         IsRunningFunc(logger, &opt),
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.CopyInto(&opt.options)

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&opt.testKey, "key", "kubermatic/quorum-check", "key to write into etcd for testing its availability")
	cmd.PersistentFlags().StringVar(&opt.testValue, "value", "something", "value to write into etcd for testing its availability")
	cmd.PersistentFlags().IntVar(&opt.intervalSeconds, "interval", 2, "interval in seconds between attempts to write to etcd")
	cmd.PersistentFlags().IntVar(&opt.intervalSeconds, "timeout", 50, "timeout in seconds before giving up writing to etcd")

	return cmd

}

func IsRunningFunc(log *zap.SugaredLogger, opt *isRunningOptions) cobraFuncE {
	return handleErrors(log, func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		log = log.With("cluster", opt.cluster)

		e := &etcd.Cluster{
			Cluster:           opt.cluster,
			EtcdctlAPIVersion: opt.etcdctlAPIVersion,

			CaCertFile:     opt.etcdCAFile,
			ClientCertFile: opt.etcdCertFile,
			ClientKeyFile:  opt.etcdKeyFile,
		}

		if _, err := e.Init(ctx); err != nil {
			return fmt.Errorf("failed to initialize etcd cluster configuration: %w", err)
		}

		if err := e.SetClusterSize(ctx); err != nil {
			return fmt.Errorf("failed to set expected cluster size: %w", err)
		}

		client, err := e.GetEtcdClient(ctx, log)
		if err != nil {
			return fmt.Errorf("failed to get etcd cluster client: %w", err)
		}

		// try to write to etcd and log transient errors.
		err = wait.PollImmediateLog(ctx, log, time.Duration(opt.intervalSeconds)*time.Second, time.Duration(opt.timeoutSeconds)*time.Second, func() (error, error) {
			_, err := client.Put(ctx, opt.testKey, opt.testKey)
			return err, nil
		})

		if err != nil {
			log.Error("failed to wait for etcd to become ready")
			return err
		}

		log.Info("etcd is ready")
		return nil
	})
}
