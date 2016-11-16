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

package main // import "k8s.io/helm/cmd/helm"

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"

	"k8s.io/helm/pkg/kube"
)

const (
	localRepoIndexFilePath = "index.yaml"
	homeEnvVar             = "HELM_HOME"
	hostEnvVar             = "HELM_HOST"
	tillerNamespace        = "kube-system"
)

var (
	helmHome    string
	tillerHost  string
	kubeContext string
)

// flagDebug is a signal that the user wants additional output.
var flagDebug bool

var globalUsage = `The Kubernetes package manager

To begin working with Helm, run the 'helm init' command:

	$ helm init

This will install Tiller to your running Kubernetes cluster.
It will also set up any necessary local configuration.

Common actions from this point include:

- helm search:    search for charts
- helm fetch:     download a chart to your local directory to view
- helm install:   upload the chart to Kubernetes
- helm list:      list releases of charts

Environment:
  $HELM_HOME      set an alternative location for Helm files. By default, these are stored in ~/.helm
  $HELM_HOST      set an alternative Tiller host. The format is host:port
  $KUBECONFIG     set an alternate Kubernetes configuration file (default "~/.kube/config")
`

func newRootCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "helm",
		Short:        "The Helm package manager for Kubernetes.",
		Long:         globalUsage,
		SilenceUsage: true,
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			teardown()
		},
	}
	home := os.Getenv(homeEnvVar)
	if home == "" {
		home = "$HOME/.helm"
	}
	thost := os.Getenv(hostEnvVar)
	p := cmd.PersistentFlags()
	p.StringVar(&helmHome, "home", home, "location of your Helm config. Overrides $HELM_HOME")
	p.StringVar(&tillerHost, "host", thost, "address of tiller. Overrides $HELM_HOST")
	p.StringVar(&kubeContext, "kube-context", "", "name of the kubeconfig context to use")
	p.BoolVarP(&flagDebug, "debug", "", false, "enable verbose output")

	// Tell gRPC not to log to console.
	grpclog.SetLogger(log.New(ioutil.Discard, "", log.LstdFlags))

	rup := newRepoUpdateCmd(out)
	rup.Deprecated = "use 'helm repo update'\n"

	cmd.AddCommand(
		newCreateCmd(out),
		newDeleteCmd(nil, out),
		newDependencyCmd(out),
		newFetchCmd(out),
		newGetCmd(nil, out),
		newHomeCmd(out),
		newHistoryCmd(nil, out),
		newInitCmd(out),
		newInspectCmd(nil, out),
		newInstallCmd(nil, out),
		newLintCmd(out),
		newListCmd(nil, out),
		newPackageCmd(nil, out),
		newRepoCmd(out),
		newRollbackCmd(nil, out),
		newSearchCmd(out),
		newServeCmd(out),
		newStatusCmd(nil, out),
		newUpgradeCmd(nil, out),
		newVerifyCmd(out),
		newVersionCmd(nil, out),
		// Deprecated
		rup,
	)
	return cmd
}

func main() {
	cmd := newRootCmd(os.Stdout)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func setupConnection(c *cobra.Command, args []string) error {
	if tillerHost == "" {
		tunnel, err := newTillerPortForwarder(tillerNamespace, kubeContext)
		if err != nil {
			return err
		}

		tillerHost = fmt.Sprintf(":%d", tunnel.Local)
		if flagDebug {
			fmt.Printf("Created tunnel using local port: '%d'\n", tunnel.Local)
		}
	}

	// Set up the gRPC config.
	if flagDebug {
		fmt.Printf("SERVER: %q\n", tillerHost)
	}
	return nil
}

func teardown() {
	if tillerTunnel != nil {
		tillerTunnel.Close()
	}
}

func checkArgsLength(argsReceived int, requiredArgs ...string) error {
	expectedNum := len(requiredArgs)
	if argsReceived != expectedNum {
		arg := "arguments"
		if expectedNum == 1 {
			arg = "argument"
		}
		return fmt.Errorf("This command needs %v %s: %s", expectedNum, arg, strings.Join(requiredArgs, ", "))
	}
	return nil
}

// prettyError unwraps or rewrites certain errors to make them more user-friendly.
func prettyError(err error) error {
	if err == nil {
		return nil
	}
	// This is ridiculous. Why is 'grpc.rpcError' not exported? The least they
	// could do is throw an interface on the lib that would let us get back
	// the desc. Instead, we have to pass ALL errors through this.
	return errors.New(grpc.ErrorDesc(err))
}

func homePath() string {
	return os.ExpandEnv(helmHome)
}

// getKubeClient is a convenience method for creating kubernetes config and client
// for a given kubeconfig context
func getKubeClient(context string) (*restclient.Config, *unversioned.Client, error) {
	config, err := kube.GetConfig(context).ClientConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("could not get kubernetes config for context '%s': %s", context, err)
	}
	client, err := unversioned.New(config)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get kubernetes client: %s", err)
	}
	return config, client, nil
}
