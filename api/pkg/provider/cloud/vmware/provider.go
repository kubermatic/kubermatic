package vmware

import (
	"github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"

	"k8s.io/client-go/tools/clientcmd/api/v1"
)

type vmware struct {
	
}

func (v *vmware) InitializeCloudProvider(*kubermaticv1.CloudSpec, string) (*kubermaticv1.CloudSpec, error) {
	panic("implement me")
}

func (v *vmware) ValidateCloudSpec(*kubermaticv1.CloudSpec) error {
	panic("implement me")
}

func (v *vmware) CleanUpCloudProvider(*kubermaticv1.CloudSpec) error {
	panic("implement me")
}

func (v *vmware) ValidateNodeSpec(*kubermaticv1.CloudSpec, *apiv1.NodeSpec) error {
	panic("implement me")
}

func (v *vmware) CreateNodeClass(*v1.Cluster, *apiv1.NodeSpec, []*kubermaticv1.UserSSHKey, *apiv1.MasterVersion) (*v1alpha1.NodeClass, error) {
	panic("implement me")
}

func (v *vmware) NodeClassName(*apiv1.NodeSpec) string {
	panic("implement me")
}
