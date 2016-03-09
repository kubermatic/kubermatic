package kubernetes

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kubermatic/api/provider"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

// Providers creates KubernetesProviders for each context in the kubeconfig
func Providers(
	kubeconfig string,
	dcs map[string]provider.DatacenterMeta,
	cps provider.CloudRegistry,
	secretsPath string,
) (provider.KubernetesRegistry, error) {
	kps := map[string]provider.KubernetesProvider{
		"fake-1": NewKubernetesFakeProvider("fake-1", cps),
		"fake-2": NewKubernetesFakeProvider("fake-2", cps),
	}

	clientcmdConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, err
	}

	cfgs := map[string]client.Config{}
	for ctx := range clientcmdConfig.Contexts {
		clientconfig := clientcmd.NewNonInteractiveClientConfig(
			*clientcmdConfig,
			ctx,
			&clientcmd.ConfigOverrides{},
		)

		cfg, err := clientconfig.ClientConfig()
		if err != nil {
			return nil, err
		}

		glog.Infof("Add kubernetes provider %q at %s", ctx, cfg.Host)

		kps[ctx] = NewKubernetesProvider(
			cfg,
			cps,
		)
		cfgs[ctx] = *cfg
	}

	// load secrets
	secrets, err := LoadSecrets(secretsPath)
	if err != nil {
		return nil, fmt.Errorf("cannot load secrets file %q: %v", secretsPath, err)
	}

	// create SeedProvider
	kps["seed"] = NewSeedProvider(dcs, cps, cfgs, secrets)

	return kps, nil
}
