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
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/clusterexposer/controller"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	kubeconfigInner      = "kubeconfig-inner"
	kubeconfigInnerUsage = ""

	kubeconfigOuter      = "kubeconfig-outer"
	kubeconfigOuterUsage = ""

	buildIDFlag      = "build-id"
	buildIDFlagUsage = ""

	rootCmdUse   = "cluster-exposer"
	rootCmdShort = "cluster-exposer is a tool that helps exposing the kubermatic user clusters running in isolated environments"
	rootCmdLong  = `
cluster-exposer is a tool that helps exposing the kubermatic user clusters running in isolated environments.
TODO...
`
)

var (
	requiredFlags = sets.NewString(kubeconfigInner, kubeconfigOuter, buildIDFlag)

	kubeconfigInnerFile = ""
	kubeconfigOuterFile = ""
	buildID             = ""
	debug               = false

	rootCmd = &cobra.Command{
		Use:     rootCmdUse,
		Short:   rootCmdShort,
		Long:    rootCmdLong,
		PreRunE: checkRequired,
		RunE:    run,
	}
)

func main() {
	Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfigInnerFile, kubeconfigInner, "", kubeconfigInnerUsage)
	rootCmd.PersistentFlags().StringVar(&kubeconfigOuterFile, kubeconfigOuter, "", kubeconfigOuterUsage)
	rootCmd.PersistentFlags().StringVar(&buildID, buildIDFlag, "", buildIDFlagUsage)
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "")
}

func checkRequired(cmd *cobra.Command, _ []string) error {
	requiredError := false
	flagName := ""

	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		if isRequired(flag.Name) && !flag.Changed {
			requiredError = true
			flagName = flag.Name
		}
	})

	if requiredError {
		return errors.New("Required flag " + flagName + " has not been set")
	}

	return nil
}

func isRequired(flagName string) bool {
	return requiredFlags.Has(flagName)
}

func run(cmd *cobra.Command, _ []string) error {
	rawLog := log.New(debug, log.FormatJSON)
	log := rawLog.Sugar()
	outerCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigOuterFile)
	if err != nil {
		return fmt.Errorf("unable to set up client config for outer cluster %v", err)
	}

	// Get a config to talk to the apiserver
	innerCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigInnerFile)
	if err != nil {
		return fmt.Errorf("unable to set up client config for user cluster %v", err)
	}

	log.Info("Setting up outer manager")
	outerManager, err := manager.New(outerCfg, manager.Options{
		MetricsBindAddress: "0",
		Port:               0,
	})
	if err != nil {
		return fmt.Errorf("failed to set up outer manager: %v", err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	log.Info("Setting up inner manager")
	mgr, err := manager.New(innerCfg, manager.Options{
		MetricsBindAddress: "127.0.0.1:2047",
		Port:               0,
	})
	if err != nil {
		return fmt.Errorf("unable to set up inner manager %v", err)
	}
	if err := mgr.Add(outerManager); err != nil {
		return fmt.Errorf("failed to add outer manager: %v", err)
	}

	// Setup all Controllers
	log.Info("Setting up controller")
	if err := controller.Add(log, outerManager, mgr, buildID); err != nil {
		return fmt.Errorf("failed to register controller: %v", err)
	}
	log.Info("Registered controller")
	log.Info("Starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		return fmt.Errorf("unable to run the manager %v", err)
	}
	log.Info("Finished")
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
