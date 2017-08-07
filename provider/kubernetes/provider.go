package kubernetes

import (
	"github.com/golang/glog"
	"github.com/kubermatic/api/provider"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Providers creates KubernetesProviders for each context in the kubeconfig
func Providers(
	kubeconfig string,
	dcs map[string]provider.DatacenterMeta,
	cps provider.CloudRegistry,
	workerName string,
) (provider.KubernetesRegistry, error) {
	kps := map[string]provider.KubernetesProvider{
		"fake-1": NewKubernetesFakeProvider("fake-1", cps, dcs),
		"fake-2": NewKubernetesFakeProvider("fake-2", cps, dcs),
	}

	clientcmdConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, err
	}

	cfgs := map[string]rest.Config{}
	for ctx := range clientcmdConfig.Contexts {
		clientconfig := clientcmd.NewNonInteractiveClientConfig(
			*clientcmdConfig,
			ctx,
			&clientcmd.ConfigOverrides{},
			nil,
		)

		var cfg *rest.Config
		cfg, err = clientconfig.ClientConfig()
		if err != nil {
			return nil, err
		}

		glog.Infof("Add kubernetes provider %q at %s", ctx, cfg.Host)

		kps[ctx] = NewKubernetesProvider(
			cfg,
			cps,
			workerName,
			dcs,
		)
		cfgs[ctx] = *cfg
	}

	return kps, nil
}
