package vsphere

import (
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/vmware/govmomi"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

const (
	tplPath = "/opt/template/nodes/vsphere.yaml"
)

type vsphere struct {
}

func (v *vsphere) getClient(cloud *kubermaticv1.CloudSpec) (*govmomi.Client, error) {}

func (v *vsphere) InitializeCloudProvider(spec *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return nil, nil
}

func (v *vsphere) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (v *vsphere) CleanUpCloudProvider(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (v *vsphere) ValidateNodeSpec(spec *kubermaticv1.CloudSpec, nSpec *apiv1.NodeSpec) error {
	return nil
}

func (v *vsphere) CreateNodeClass(cluster *v1.Cluster, nSpec *apiv1.NodeSpec, keys []*kubermaticv1.UserSSHKey, ver *apiv1.MasterVersion) (*v1alpha1.NodeClass, error) {
	return nil, nil
}

func (v *vsphere) NodeClassName(nSpec *apiv1.NodeSpec) string {
	return ""
}
