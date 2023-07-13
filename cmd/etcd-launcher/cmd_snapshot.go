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
	"strings"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/cmd/etcd-launcher/pkg/etcd"
)

type snapshotOptions struct {
	options

	file string
}

func SnapshotCommand(log *zap.SugaredLogger) *cobra.Command {
	opt := snapshotOptions{}

	cmd := &cobra.Command{
		Use:          "snapshot",
		Short:        "Create etcd database snapshot and save it to file",
		RunE:         SnapshotFunc(log, &opt),
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

	cmd.PersistentFlags().StringVar(&opt.file, "file", "/backup/snapshot.db", "file to save database snapshot to")

	return cmd
}

func SnapshotFunc(log *zap.SugaredLogger, opt *snapshotOptions) cobraFuncE {
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

		configs, err := e.GetEtcdEndpointConfigs(ctx)
		if err != nil {
			return fmt.Errorf("failed to get etcd cluster clients: %w", err)
		}

		// snapshots can only be taken from a single endpoint, but it's possible
		// that one of the endpoints is down right now (e.g. because an update
		// or autoscaling-caused rescheduling event is happening). So we loop
		// over all endpoints and try to take a snapshot.
		for _, config := range configs {
			if len(config.Endpoints) != 1 {
				log.Warnw("unexpected number of endpoints, skipping this configuration", "endpoints", strings.Join(config.Endpoints, ","))
				continue
			}

			snapv3 := snapshot.NewV3(log.Desugar())
			err := snapv3.Save(ctx, config, opt.file)

			// if the snapshot was successful, we have what we want and do not
			// need to loop over the remaining endpoints. We can exit the program
			// successfully then.
			if err == nil {
				log.Infow("saved snapshot from endpoint", "endpoint", strings.Join(config.Endpoints, ","), "file", opt.file)
				return nil
			}

			// log an error if we were not able to take a snapshot, before the loop
			// moves on to the next one.
			log.Errorw("failed to save snapshot from endpoint, trying next endpoint", "endpoint", strings.Join(config.Endpoints, ","), zap.Error(err))
		}

		// we failed to take any snapshot, so we need to exit the program with an error.
		return fmt.Errorf("exhausted all endpoints, no snapshot was successful")
	})
}
