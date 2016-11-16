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
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/helm/cmd/helm/downloader"
	"k8s.io/helm/cmd/helm/helmpath"
)

const dependencyUpDesc = `
Update the on-disk dependencies to mirror the requirements.yaml file.

This command verifies that the required charts, as expressed in 'requirements.yaml',
are present in 'charts/' and are at an acceptable version.

On successful update, this will generate a lock file that can be used to
rebuild the requirements to an exact version.
`

// dependencyUpdateCmd describes a 'helm dependency update'
type dependencyUpdateCmd struct {
	out       io.Writer
	chartpath string
	helmhome  helmpath.Home
	verify    bool
	keyring   string
}

// newDependencyUpdateCmd creates a new dependency update command.
func newDependencyUpdateCmd(out io.Writer) *cobra.Command {
	duc := &dependencyUpdateCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:     "update [flags] CHART",
		Aliases: []string{"up"},
		Short:   "update charts/ based on the contents of requirements.yaml",
		Long:    dependencyUpDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			cp := "."
			if len(args) > 0 {
				cp = args[0]
			}

			var err error
			duc.chartpath, err = filepath.Abs(cp)
			if err != nil {
				return err
			}

			duc.helmhome = helmpath.Home(homePath())

			return duc.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&duc.verify, "verify", false, "verify the packages against signatures")
	f.StringVar(&duc.keyring, "keyring", defaultKeyring(), "keyring containing public keys")

	return cmd
}

// run runs the full dependency update process.
func (d *dependencyUpdateCmd) run() error {
	man := &downloader.Manager{
		Out:       d.out,
		ChartPath: d.chartpath,
		HelmHome:  d.helmhome,
		Keyring:   d.keyring,
	}
	if d.verify {
		man.Verify = downloader.VerifyIfPossible
	}
	return man.Update()
}
