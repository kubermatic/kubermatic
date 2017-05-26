package cmd

import (
	goflag "flag"
	"log"

	"github.com/golang/glog"
	"github.com/kubermatic/api/pkg/bare-metal-provider/api"
	"github.com/kubermatic/api/pkg/bare-metal-provider/extensions"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var cfgFile, address, kubeconfig, authUser, authPass string

var viperWhiteList = []string{
	"v",
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "bare-metal-provider",
	Short: "A cloud provider provider for bare-metal servers",
	Long: `A api which acts as a custom cloud provider for bare-metal servers.
Every bare-metal server will register at the provider and will then be available to clients.
Every client will be able to assign free servers to a kubernetes cluster.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		config, err := clientcmd.BuildConfigFromFlags("", viper.GetString("kubeconfig"))
		if err != nil {
			log.Fatal(err)
		}

		client, err := kubernetes.NewForConfig(config)
		if err != nil {
			log.Fatal(err)
		}

		wrappedClientset, err := extensions.WrapClientsetWithExtensions(config)
		if err != nil {
			log.Fatal(err)
		}

		err = extensions.EnsureThirdPartyResourcesExist(client)
		if err != nil {
			log.Fatal(err)
		}

		s := api.New(
			viper.GetString("address"),
			wrappedClientset,
			viper.GetString("auth-user"),
			viper.GetString("auth-password"),
		)
		if err := s.Run(); err != nil {
			glog.Fatal(err)
		}
	},
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	err := goflag.CommandLine.Parse([]string{})
	if err != nil {
		log.Fatal(err)
	}
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.bare-metal-provider.yaml)")
	RootCmd.PersistentFlags().StringVar(&address, "address", "", "listen address (default is 127.0.0.1:8080)")
	RootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Kubeconfig to use to store nodes")
	RootCmd.PersistentFlags().StringVar(&authUser, "auth-user", "", "Enables basic auth on all endpoints & uses this username")
	RootCmd.PersistentFlags().StringVar(&authPass, "auth-password", "", "Enables basic auth on all endpoints & uses this password")
	err = viper.BindPFlags(RootCmd.PersistentFlags())
	if err != nil {
		log.Fatalf("Unable to bind Command Line flags: %s\n", err)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("bare-metal-provider")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath("./")
	viper.AddConfigPath("/etc/kubermatic")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		glog.Info("Using config file:", viper.ConfigFileUsed())
	}

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
		if err != nil {
			// ignore
		}
		flag.Changed = true
	}
}
