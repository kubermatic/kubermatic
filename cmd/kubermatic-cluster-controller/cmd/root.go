// Copyright © 2016 Loodse GmbH <info@loodse.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	goflag "flag"
	"fmt"
	"log"
	"os"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/cluster"
	"github.com/kubermatic/api/controller/version"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/extensions/etcd"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/wait"
	kkubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var cfgFile, kubeConfig, masterResources, externalURL, dcFile, versions, updates, addonResources, workerName string
var viperWhiteList = []string{
	"v",
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "kubermatic-cluster-controller",
	Short: "Controller for Kubermatic",
	Long:  `Cluster controller... Needs better description`,

	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetString("master-resources") == "" {
			print("master-resources path is undefined\n\n")
			os.Exit(1)
		}

		// load list of datacenters
		dcs := map[string]provider.DatacenterMeta{}
		if path := viper.GetString("datacenters"); path != "" {
			var err error
			dcs, err = provider.DatacentersMeta(path)
			if err != nil {
				log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", path, err))
			}
		}

		// load versions
		versions := map[string]*api.MasterVersion{}
		if path := viper.GetString("versions"); path != "" {
			var err error
			versions, err = version.LoadVersions(path)
			if err != nil {
				log.Fatal(fmt.Printf("failed to load version yaml %q: %v", path, err))
			}
		}

		// load updates
		updates := []api.MasterUpdate{}
		if path := viper.GetString("updates"); path != "" {
			var err error
			updates, err = version.LoadUpdates(path)
			if err != nil {
				log.Fatal(fmt.Printf("failed to load updates yaml %q: %v", path, err))
			}
		}

		// create controller for each context
		clientcmdConfig, err := clientcmd.LoadFromFile(viper.GetString("kubeconfig"))
		if err != nil {
			log.Fatal(err)
		}

		for ctx := range clientcmdConfig.Contexts {
			// create kube client
			clientcmdConfig, err := clientcmd.LoadFromFile(viper.GetString("kubeconfig"))
			if err != nil {
				log.Fatal(err)
			}
			clientConfig := clientcmd.NewNonInteractiveClientConfig(
				*clientcmdConfig,
				ctx,
				&clientcmd.ConfigOverrides{},
				nil,
			)

			cfg, err := clientConfig.ClientConfig()
			if err != nil {
				log.Fatal(err)
			}
			client, err := kkubernetes.NewForConfig(cfg)
			if err != nil {
				log.Fatal(err)
			}
			tprClient, err := extensions.WrapClientsetWithExtensions(cfg)
			if err != nil {
				log.Fatal(err)
			}

			etcdClusterClient, err := etcd.WrapClientsetWithExtensions(cfg)
			if err != nil {
				log.Fatal(err)
			}

			// start controller
			cps := cloud.Providers(dcs)
			ctrl, err := cluster.NewController(
				ctx,
				client,
				tprClient,
				etcdClusterClient,
				cps,
				versions,
				updates,
				viper.GetString("master-resources"),
				viper.GetString("external-url"),
				viper.GetString("worker-name"),
				viper.GetString("addon-resources"),
			)
			if err != nil {
				log.Fatal(err)
			}
			go ctrl.Run(wait.NeverStop)
		}

		<-wait.NeverStop
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/kubermatic/kubermatic-cluster-controller.yaml)")
	RootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	RootCmd.PersistentFlags().StringVar(&masterResources, "master-resources", "", "The master resources path (Required).")
	RootCmd.PersistentFlags().StringVar(&externalURL, "external-url", "", "The external url for the apiserver host and the the dc.(Required)")
	RootCmd.PersistentFlags().StringVar(&dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	RootCmd.PersistentFlags().StringVar(&workerName, "worker-name", "", "Create clusters only processed by worker-name cluster controller")
	RootCmd.PersistentFlags().StringVar(&addonResources, "addon-resources", "/etc/kubermaitc/addons", "Path to addon helm charts")
	RootCmd.PersistentFlags().StringVar(&versions, "versions", "versions.yaml", "The versions.yaml file path")
	RootCmd.PersistentFlags().StringVar(&updates, "updates", "updates.yaml", "The updates.yaml file path")

	err := viper.BindPFlags(RootCmd.PersistentFlags())
	if err != nil {
		log.Fatalf("Unable to bind Command Line flags: %s\n", err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("kubermatic-cluster-controller") // name of config file (without extension)
	viper.AddConfigPath("$HOME")                         // adding home directory as first search path
	viper.AddConfigPath(".")                             // adding current directory as second search path
	viper.AddConfigPath("/etc/kubermatic")               // adding /etc/kubermatic as third search path
	viper.AutomaticEnv()                                 // read in environment variables that match

	setFlagsUsingViper()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func setFlagsUsingViper() {
	for _, config := range viperWhiteList {
		var flag = pflag.Lookup(config)
		viper.SetDefault(flag.Name, flag.DefValue)
		// If the flag is set, override viper value
		if flag.Changed {
			viper.Set(flag.Name, flag.Value.String())
		}
		// Viper will give precedence first to calls to the Set command,
		// then to values from the config.yml
		err := flag.Value.Set(viper.GetString(flag.Name))
		if err == nil {
			fmt.Printf("failed to set flag %s: %v", flag.Name, err)
		}

		flag.Changed = true
	}
}
