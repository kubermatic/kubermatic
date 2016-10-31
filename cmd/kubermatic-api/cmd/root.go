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
	"context"
	goflag "flag"
	"fmt"
	"log"
	"net/http"
	"os"

	ghandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api/handler"
	"github.com/kubermatic/api/provider"
	"github.com/kubermatic/api/provider/cloud"
	"github.com/kubermatic/api/provider/kubernetes"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfgFile, kubeConfig, dcFile, secretsFile, jwtKey, address string
var dev, auth bool

var viperWhiteList = []string{
	"v",
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "api",
	Short: "Kubermatic API Server",
	Long:  `API server for Kubermatic, providing access to seed and client resources within the clusters`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		// load list of datacenters

		dcs := map[string]provider.DatacenterMeta{}
		if dcFile != "" {
			var err error
			dcs, err = provider.DatacentersMeta(dcFile)
			if err != nil {
				log.Fatal(fmt.Printf("failed to load datacenter yaml %q: %v", dcFile, err))
			}
		}

		// create CloudProviders
		cps := cloud.Providers(dcs)

		// create KubernetesProvider for each context in the kubeconfig

		kps, err := kubernetes.Providers(kubeConfig, dcs, cps, secretsFile, dev)
		if err != nil {
			log.Fatal(err)
		}

		// start server
		ctx := context.Background()
		r := handler.NewRouting(ctx, dcs, kps, cps, auth, jwtKey)
		mux := mux.NewRouter()
		r.Register(mux)
		log.Println(fmt.Sprintf("Listening on %s", address))
		log.Fatal(http.ListenAndServe(address, ghandlers.CombinedLoggingHandler(os.Stdout, mux)))
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
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kubermatic-api.yaml)")
	RootCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Create dev-mode clusters only processed by dev-mode cluster controller")
	RootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", ".kubeconfig", "The kubeconfig file path with one context per Kubernetes provider")
	RootCmd.PersistentFlags().BoolVar(&auth, "auth", true, "Activate authentication with JSON Web Tokens")
	RootCmd.PersistentFlags().StringVar(&dcFile, "datacenters", "datacenters.yaml", "The datacenters.yaml file path")
	RootCmd.PersistentFlags().StringVar(&secretsFile, "secrets", "secrets.yaml", "The secrets.yaml file path")
	RootCmd.PersistentFlags().StringVar(&jwtKey, "jwt-key", "", "The JSON Web Token validation key, encoded in base64")
	RootCmd.PersistentFlags().StringVar(&address, "address", ":8080", "The address to listen on")
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

	viper.SetConfigName(".kubermatic-api") // name of config file (without extension)
	viper.AddConfigPath("$HOME")           // adding home directory as first search path
	viper.AddConfigPath(".")               // adding current directory as second search path
	viper.AutomaticEnv()                   // read in environment variables that match

	setFlagsUsingViper()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func setFlagsUsingViper() {
	fmt.Println("setFlagsUsingViper")
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
