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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/version"
)

const versionDesc = `
Show the client and server versions for Helm and tiller.

This will print a representation of the client and server versions of Helm and
Tiller. The output will look something like this:

Client: &version.Version{SemVer:"v2.0.0-beta.1", GitCommit:"ff52399e51bb880526e9cd0ed8386f6433b74da1", GitTreeState:"dirty"}
Server: &version.Version{SemVer:"v2.0.0-beta.1", GitCommit:"b0c113dfb9f612a9add796549da66c0d294508a3", GitTreeState:"clean"}

- SemVer is the semantic version of the release.
- GitCommit is the SHA for the commit that this version was built from.
- GitTreeState is "clean" if there are no local code changes when this binary was
  built, and "dirty" if the binary was built from locally modified code.

To print just the client version, use '--client'. To print just the server version,
use '--server'.
`

type versionCmd struct {
	out        io.Writer
	client     helm.Interface
	showClient bool
	showServer bool
}

func newVersionCmd(c helm.Interface, out io.Writer) *cobra.Command {
	version := &versionCmd{
		client: c,
		out:    out,
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "print the client/server version information",
		Long:  versionDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If neither is explicitly set, show both.
			if !version.showClient && !version.showServer {
				version.showClient, version.showServer = true, true
			}
			if version.showServer {
				// We do this manually instead of in PreRun because we only
				// need a tunnel if server version is requested.
				setupConnection(cmd, args)
			}
			version.client = ensureHelmClient(version.client)
			return version.run()
		},
	}
	f := cmd.Flags()
	f.BoolVarP(&version.showClient, "client", "c", false, "if set, show the client version")
	f.BoolVarP(&version.showServer, "server", "s", false, "if set, show the server version")

	return cmd
}

func (v *versionCmd) run() error {

	if v.showClient {
		cv := version.GetVersionProto()
		fmt.Fprintf(v.out, "Client: %#v\n", cv)
	}

	if !v.showServer {
		return nil
	}

	resp, err := v.client.GetVersion()
	if err != nil {
		if grpc.Code(err) == codes.Unimplemented {
			return errors.New("server is too old to know its version")
		}
		if flagDebug {
			fmt.Fprintln(os.Stderr, err)
		}
		return errors.New("cannot connect to Tiller")
	}
	fmt.Fprintf(v.out, "Server: %#v\n", resp.Version)
	return nil
}
