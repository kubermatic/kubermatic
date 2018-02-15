package resources

import (
	"encoding/json"

	"github.com/ghodss/yaml"
	nodesetv1alpha1 "github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8stemplate "github.com/kubermatic/kubermatic/api/pkg/template/kubernetes"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
)

// LoadNodeClassFile parses and returns the given nodeclass template
func LoadNodeClassFile(filename, name string, c *kubermaticv1.Cluster, nSpec *apiv1.NodeSpec, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey, version *apiv1.MasterVersion) (*nodesetv1alpha1.NodeClass, error) {
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

	var nc nodesetv1alpha1.NodeClass
	return &nc, t.Execute(data, &nc)
}

// LoadMachineFile parses and returns the given machine manifest
func LoadMachineFile(filename string, c *kubermaticv1.Cluster, node *apiv2.Node, dc provider.DatacenterMeta, keys []*kubermaticv1.UserSSHKey, version *apiv1.MasterVersion) (*machinev1alpha1.Machine, error) {
	t, err := k8stemplate.ParseFile(filename)
	if err != nil {
		return nil, err
	}

	data := struct {
		Cluster    *kubermaticv1.Cluster
		Node       *apiv2.Node
		Datacenter provider.DatacenterMeta
		Name       string
		Keys       []*kubermaticv1.UserSSHKey
		Version    *apiv1.MasterVersion
	}{
		Cluster:    c,
		Node:       node,
		Datacenter: dc,
		Keys:       keys,
		Version:    version,
	}

	var machine machinev1alpha1.Machine
	return &machine, t.Execute(data, &machine)
}
