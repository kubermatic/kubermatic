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
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kubermatic/api/controller/cluster"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/util/wait"
)

var cfgFile, kubeConfig, masterResources, externalURL, dcFile string
var dev bool

var viperWhiteList = []string{
	"v",
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "api",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		if masterResources == "" {
			print("master-resources path is undefined\n\n")
			flag.Usage()
			os.Exit(1)
		}

		// load list of datacenters
		dcs := map[string]provider.DatacenterMeta{}
		if dcFile != "" {
			var err error
			dcs, err = provider.DatacentersMeta(dcFile)
			if err != nil {
				log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", dcFile, err))
			}
		}

		// create controller for each context
		clientcmdConfig, err := clientcmd.LoadFromFile(kubeConfig)
		if err != nil {
			log.Fatal(err)
		}

		for ctx := range clientcmdConfig.Contexts {
			// create kube client
			clientcmdConfig, err := clientcmd.LoadFromFile(kubeConfig)
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
			client, err := client.New(cfg)
			if err != nil {
				log.Fatal(err)
			}

			// start controller
			cps := cloud.Providers(dcs)
			ctrl, err := cluster.NewController(
				ctx, client, cps, masterResources, externalURL, dev,
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
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.api.yaml)")
	RootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	RootCmd.PersistentFlags().StringVar(&masterResources, "master-resources", "", "The master resources path (required).")
	RootCmd.PersistentFlags().StringVar(&externalURL, "external-url", "seed1.kubermatic.io", "The external url for the apiserver host and the the dc.")
	RootCmd.PersistentFlags().StringVar(&dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	RootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Create dev-mode clusters only processed by dev-mode cluster controller")
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
		var a = pflag.Lookup(config)
		viper.SetDefault(a.Name, a.DefValue)
		// If the flag is set, override viper value
		if a.Changed {
			viper.Set(a.Name, a.Value.String())
		}
		// Viper will give precedence first to calls to the Set command,
		// then to values from the config.yml
		err := a.Value.Set(viper.GetString(a.Name))
		if err != nil {
			// ignore
		}
		a.Changed = true
	}
}
