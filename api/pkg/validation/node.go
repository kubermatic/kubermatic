package validation

import (
	"errors"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

func ValidateCreateNodeSpec(c *kubermaticv1.Cluster, spec *apiv1.NodeSpec, dc *provider.DatacenterMeta) error {
	if c.Spec.Cloud.Openstack != nil {
		if (dc.Spec.Openstack.EnforceFloatingIP || spec.Cloud.Openstack.UseFloatingIP) && len(c.Spec.Cloud.Openstack.FloatingIPPool) == 0 {
			return errors.New("no floating ip pool specified")
		}
	}

	return nil
}
