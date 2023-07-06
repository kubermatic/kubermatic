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
)

type defragOptions struct {
	options
}

func DefragCommand(log *zap.SugaredLogger) *cobra.Command {
	opt := defragOptions{}

	cmd := &cobra.Command{
		Use:          "defrag",
		Short:        "Run defragmentation on all etcds in a etcd cluster in sequence",
		RunE:         DefragFunc(log, &opt),
		SilenceUsage: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			opts.CopyInto(&opt.options)

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

	return cmd
}

func DefragFunc(log *zap.SugaredLogger, opt *defragOptions) cobraFuncE {
	return handleErrors(log, func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		log := log.With("cluster", opt.cluster)

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

		for _, endpoint := range client.Endpoints() {
			_, err := client.Defragment(ctx, endpoint)
			if err != nil {
				return fmt.Errorf("failed to defragment %s: %w", endpoint, err)
			}

			log.Infow("defragmented etcd member", "endpoint", endpoint)

			time.Sleep(5 * time.Second)
		}

		log.Info("finished defragmentation on all members")

		return nil
	})
}
