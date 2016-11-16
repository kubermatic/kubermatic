/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/storage/driver"
)

const upgradeDesc = `
This command upgrades a release to a new version of a chart.

The upgrade arguments must be a release and chart. The chart
argument can be either: a chart reference('stable/mariadb'), a path to a chart directory,
a packaged chart, or a fully qualified URL. For chart references, the latest
version will be specified unless the '--version' flag is set.

To override values in a chart, use either the '--values' flag and pass in a file
or use the '--set' flag and pass configuration from the command line.
`

type upgradeCmd struct {
	release      string
	chart        string
	out          io.Writer
	client       helm.Interface
	dryRun       bool
	disableHooks bool
	valuesFile   string
	values       *values
	verify       bool
	keyring      string
	install      bool
	namespace    string
	version      string
}

func newUpgradeCmd(client helm.Interface, out io.Writer) *cobra.Command {

	upgrade := &upgradeCmd{
		out:    out,
		client: client,
		values: new(values),
	}

	cmd := &cobra.Command{
		Use:               "upgrade [RELEASE] [CHART]",
		Short:             "upgrade a release",
		Long:              upgradeDesc,
		PersistentPreRunE: setupConnection,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkArgsLength(len(args), "release name", "chart path"); err != nil {
				return err
			}

			upgrade.release = args[0]
			upgrade.chart = args[1]
			upgrade.client = ensureHelmClient(upgrade.client)

			return upgrade.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&upgrade.valuesFile, "values", "f", "", "path to a values YAML file")
	f.BoolVar(&upgrade.dryRun, "dry-run", false, "simulate an upgrade")
	f.Var(upgrade.values, "set", "set values on the command line. Separate values with commas: key1=val1,key2=val2")
	f.BoolVar(&upgrade.disableHooks, "disable-hooks", false, "disable pre/post upgrade hooks")
	f.BoolVar(&upgrade.verify, "verify", false, "verify the provenance of the chart before upgrading")
	f.StringVar(&upgrade.keyring, "keyring", defaultKeyring(), "path to the keyring that contains public singing keys")
	f.BoolVarP(&upgrade.install, "install", "i", false, "if a release by this name doesn't already exist, run an install")
	f.StringVar(&upgrade.namespace, "namespace", "default", "namespace to install the release into (only used if --install is set)")
	f.StringVar(&upgrade.version, "version", "", "specify the exact chart version to use. If this is not specified, the latest version is used")

	return cmd
}

func (u *upgradeCmd) run() error {
	chartPath, err := locateChartPath(u.chart, u.version, u.verify, u.keyring)
	if err != nil {
		return err
	}

	if u.install {
		// If a release does not exist, install it. If another error occurs during
		// the check, ignore the error and continue with the upgrade.
		//
		// The returned error is a grpc.rpcError that wraps the message from the original error.
		// So we're stuck doing string matching against the wrapped error, which is nested somewhere
		// inside of the grpc.rpcError message.
		_, err := u.client.ReleaseContent(u.release, helm.ContentReleaseVersion(1))
		if err != nil && strings.Contains(err.Error(), driver.ErrReleaseNotFound.Error()) {
			fmt.Fprintf(u.out, "Release %q does not exist. Installing it now.\n", u.release)
			ic := &installCmd{
				chartPath:    chartPath,
				client:       u.client,
				out:          u.out,
				name:         u.release,
				valuesFile:   u.valuesFile,
				dryRun:       u.dryRun,
				verify:       u.verify,
				disableHooks: u.disableHooks,
				keyring:      u.keyring,
				values:       u.values,
				namespace:    u.namespace,
			}
			return ic.run()
		}
	}

	rawVals, err := u.vals()
	if err != nil {
		return err
	}

	_, err = u.client.UpdateRelease(u.release, chartPath, helm.UpdateValueOverrides(rawVals), helm.UpgradeDryRun(u.dryRun), helm.UpgradeDisableHooks(u.disableHooks))
	if err != nil {
		return fmt.Errorf("UPGRADE FAILED: %v", prettyError(err))
	}

	success := u.release + " has been upgraded. Happy Helming!\n"
	fmt.Fprintf(u.out, success)

	// Print the status like status command does
	status, err := u.client.ReleaseStatus(u.release)
	if err != nil {
		return prettyError(err)
	}
	PrintStatus(u.out, status)

	return nil
}

func (u *upgradeCmd) vals() ([]byte, error) {
	var buffer bytes.Buffer

	// User specified a values file via -f/--values
	if u.valuesFile != "" {
		bytes, err := ioutil.ReadFile(u.valuesFile)
		if err != nil {
			return []byte{}, err
		}
		buffer.Write(bytes)
	}

	// User specified value pairs via --set
	// These override any values in the specified file
	if len(u.values.pairs) > 0 {
		bytes, err := u.values.yaml()
		if err != nil {
			return []byte{}, err
		}
		buffer.Write(bytes)
	}

	return buffer.Bytes(), nil
}
