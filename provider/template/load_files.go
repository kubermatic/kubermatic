package resources

import (
	"encoding/json"
	"github.com/ghodss/yaml"
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	k8stemplate "github.com/kubermatic/api/template/kubernetes"
)

// LoadNodeClassFile parses and returns the given nodeclass template
func LoadNodeClassFile(filename, name string, c *api.Cluster, nSpec *api.NodeSpec, dc provider.DatacenterMeta, keys []extensions.UserSSHKey) (*v1alpha1.NodeClass, error) {
	t, err := k8stemplate.ParseFile(filename)
	if err != nil {
		return nil, err
	}

	cfg := c.GetKubeconfig()
	cfg.AuthInfos[0].AuthInfo.Token = c.Address.KubeletToken
	jcfg, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}

	ycfg, err := yaml.JSONToYAML(jcfg)
	if err != nil {
		return nil, err
	}

	data := struct {
		Cluster    *api.Cluster
		NodeSpec   *api.NodeSpec
		Datacenter provider.DatacenterMeta
		Name       string
		Kubeconfig string
		Keys       []extensions.UserSSHKey
	}{
		Cluster:    c,
		NodeSpec:   nSpec,
		Datacenter: dc,
		Name:       name,
		Kubeconfig: string(ycfg),
		Keys:       keys,
	}

	var nc v1alpha1.NodeClass
	return &nc, t.Execute(data, &nc)
}
