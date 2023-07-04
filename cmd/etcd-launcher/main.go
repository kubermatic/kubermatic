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
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

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

type options struct {
	cluster           string
	etcdctlAPIVersion string
}

func (o *options) CopyInto(other *options) {
	other.cluster = o.cluster
	other.etcdctlAPIVersion = o.etcdctlAPIVersion
}

type cobraFuncE func(cmd *cobra.Command, args []string) error

var opts = &options{}

func main() {
	log := createLogger()
	ctx := signals.SetupSignalHandler()

	versions := kubermaticversion.NewDefaultVersions()

	rootCmd := &cobra.Command{
		Use:           "etcd-launcher",
		Short:         "Runs etcd clusters for KKP user cluster control planes",
		SilenceErrors: true,
		Version:       versions.Kubermatic,
	}

	// cobra does not make any distinction between "error that happened because of bad flags"
	// and "error that happens because of something going bad inside the RunE function", and
	// so would always show the Usage, no matter what error occurred. Tow ork around this, we
	// set SilenceUsages on all commands and manually print the error using the FlagErrorFunc.
	rootCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		if err := c.Usage(); err != nil {
			return err
		}

		// ensure we exit with code 1 later on
		return err
	})

	rootCmd.PersistentFlags().StringVar(&opts.cluster, "cluster", "", "KKP user cluster to run this etcd-launcher for")
	rootCmd.PersistentFlags().StringVar(&opts.etcdctlAPIVersion, "api-version", defaultEtcdctlAPIVersion, "etcdctl API version")

	addCommands(rootCmd, log, versions)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}

}

func createLogger() *zap.SugaredLogger {
	logOpts := kubermaticlog.NewDefaultOptions()
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	return rawLog.Sugar()
}

func addCommands(cmd *cobra.Command, logger *zap.SugaredLogger, versions kubermaticversion.Versions) {
	cmd.AddCommand(
		RunCommand(logger),
	)
}

func handleErrors(logger *zap.SugaredLogger, action cobraFuncE) cobraFuncE {
	return func(cmd *cobra.Command, args []string) error {
		err := action(cmd, args)
		if err != nil {
			logger.Errorf("Operation failed: %v.", err)
		}

		return err
	}
}
