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
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/kubermatic/api/pkg/nanny"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var endpoint, nodeUID, authUser, authPass string
var reloadInterval uint
var viperWhiteList = []string{
	"v",
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "kubelet-nanny",
	Short: "Manages the kubelet process on a bare metal setup for kubermatic",
	Long:  `Manages the kubelet process on a bare metal setup for kubermatic`,

	Run: func(cmd *cobra.Command, args []string) {
		if _, err := url.Parse(endpoint); err != nil || endpoint == "" {
			log.Fatalf("provider-url is not a valid url.\n\n")
		}

		UID, err := nanny.LoadNodeID(nanny.DefaultNodeIDFile)
		if err != nil {
			if nodeUID == "" {
				log.Fatalf("nodeID is empty while default nodeID is not readable.\n\n")
			}

			log.Printf("Falling back to given nodeID %q", nodeUID)
			UID = nodeUID
		}

		err = nanny.AddAsLocalHostname(UID)
		if err != nil {
			log.Fatal(err)
		}

		err = nanny.WriteNodeName(UID)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Using the provider api on %q", endpoint)
		p := nanny.NewProvider(http.DefaultClient, endpoint, authUser, authPass)
		n, err := nanny.NewNodeFromSystemData(UID)
		if err != nil {
			log.Fatalf("Failed to read system information to create the node instance: %v", err)
		}

		d, err := time.ParseDuration(fmt.Sprintf("%ds", reloadInterval))
		if err != nil {
			log.Fatalf("Unable to create duration: %v", err)

		}

		k, err := nanny.NewSystemdKubelet()
		if err != nil {
			log.Fatalf("Unable to open systemd dbus socker: %v", err)
		}

		nanny.StartCheckLoop(k, p, n, d)
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
	err := goflag.CommandLine.Parse([]string{})
	if err != nil {
		log.Fatal(err)
	}
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&endpoint, "provider-url", "", "URL of the provider api")
	RootCmd.PersistentFlags().StringVar(&nodeUID, "node-uid", "", "Unique name of the node")
	RootCmd.PersistentFlags().UintVar(&reloadInterval, "reload-interval", 10, "Interval to reload the node state in seconds")
	RootCmd.PersistentFlags().StringVar(&authUser, "provider-auth-user", "", "Sets username for basic auth in case the provider uses basic auth")
	RootCmd.PersistentFlags().StringVar(&authPass, "provider-auth-password", "", "Sets password for basic auth in case the provider uses basic auth")

	err = viper.BindPFlags(RootCmd.PersistentFlags())
	if err != nil {
		log.Fatalf("Unable to bind Command Line flags: %s\n", err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv()
	setFlagsUsingViper()
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
		_ = err
		flag.Changed = true
	}
}
