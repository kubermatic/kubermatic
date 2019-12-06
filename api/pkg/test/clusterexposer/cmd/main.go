package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/test/clusterexposer/controller"

	"k8s.io/client-go/tools/clientcmd"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

const (
	kubeconfigInner      = "kubeconfig-inner"
	kubeconfigInnerUsage = ""

	kubeconfigOuter      = "kubeconfig-outer"
	kubeconfigOuterUsage = ""

	namespace      = "namespace"
	namespaceUsage = ""

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
	requiredFlags = map[string]struct{}{
		kubeconfigInner: {},
		kubeconfigOuter: {},
		namespace:       {},
		buildIDFlag:     {},
	}

	kubeconfigInnerFile = ""
	kubeconfigOuterFile = ""
	namespaceName       = ""
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
	rootCmd.PersistentFlags().StringVar(&namespaceName, namespace, "", namespaceUsage)
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
	_, exists := requiredFlags[flagName]
	return exists
}

func run(cmd *cobra.Command, _ []string) error {
	rawLog := log.New(debug, log.Format(string(log.FormatJSON)))
	log := rawLog.Sugar()
	outerCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigOuterFile)
	if err != nil {
		return fmt.Errorf("unable to set up client config for outer cluster %v", err)
	}

	outerClient, err := runtimeclient.New(outerCfg, runtimeclient.Options{})
	if err != nil {
		return fmt.Errorf("unable to set up client for outer cluster %v", err)
	}

	// Get a config to talk to the apiserver
	log.Debug("setting up client for manager")
	innerCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigInnerFile)
	if err != nil {
		return fmt.Errorf("unable to set up client config for user cluster %v", err)

	}

	// Create a new Cmd to provide shared dependencies and start components
	log.Debug("Setting up manager")
	mgr, err := manager.New(innerCfg, manager.Options{
		MetricsBindAddress: "0",
		Port:               0,
	})
	if err != nil {
		return fmt.Errorf("unable to set up overall controller manager %v", err)
	}

	// Setup all Controllers
	log.Debug("Setting up controller")
	if err := controller.Add(context.Background(), outerClient, log, mgr, namespaceName, &kubeconfigInnerFile, cmd, buildID); err != nil {
		log.Fatalw("Failed to register controller", zap.Error(err))
	}
	log.Info("Registered controller")
	// Start the Cmd
	log.Info("Watching")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		return fmt.Errorf("unable to run the manager %v", err)
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
