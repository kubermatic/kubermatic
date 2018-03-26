package resources

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8stemplate "github.com/kubermatic/kubermatic/api/pkg/template/kubernetes"
	machinev1alpha1 "github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
)

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
	_, err = t.Execute(data, &machine)
	return &machine, err
}
