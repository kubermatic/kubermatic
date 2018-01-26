package resources

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8stemplate "github.com/kubermatic/kubermatic/api/pkg/template/kubernetes"
)

// LoadNodeClassFile parses and returns the given nodeclass template
func LoadNodeClassFile(filename, name string, c *kubermaticv1.Cluster, nSpec *apiv1.NodeSpec, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey, version *apiv1.MasterVersion) (*v1alpha1.NodeClass, error) {
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
		Cluster    *kubermaticv1.Cluster
		NodeSpec   *apiv1.NodeSpec
		Datacenter provider.DatacenterMeta
		Name       string
		Kubeconfig string
		Keys       []*kubermaticv1.UserSSHKey
		Version    *apiv1.MasterVersion
	}{
		Cluster:    c,
		NodeSpec:   nSpec,
		Datacenter: dc,
		Name:       name,
		Kubeconfig: string(ycfg),
		Keys:       keys,
		Version:    version,
	}

	var nc v1alpha1.NodeClass
	return &nc, t.Execute(data, &nc)
}
