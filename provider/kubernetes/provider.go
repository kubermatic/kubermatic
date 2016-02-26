package kubernetes

import (
	"github.com/golang/glog"
	"github.com/kubermatic/api/provider"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

// Providers creates KubernetesProviders for each context in the kubeconfig
func Providers(
	kubeconfig string,
	cps provider.CloudRegistry,
	metas map[string]DatacenterMeta,
) (provider.KubernetesRegistry, error) {
	kps := map[string]provider.KubernetesProvider{
		"fake-1": NewKubernetesFakeProvider("fake-1", cps),
		"fake-2": NewKubernetesFakeProvider("fake-2", cps),
	}

	clientcmdConfig, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, err
	}

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

		meta := DatacenterMeta{
			Location: "Unknown",
			Country:  "Unknown",
			Provider: "Unknown",
		}

		if m, found := metas[ctx]; found {
			meta = m
		}

		glog.Infof("Add kubernetes provider %q at %s, meta=%+v", ctx, cfg.Host, meta)

		kps[ctx] = NewKubernetesProvider(
			cfg,
			cps,
			meta.Location,
			meta.Country,
			meta.Provider,
		)
	}

	return kps, nil
}
