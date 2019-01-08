package validation_test

import (
	"errors"
	"github.com/kubermatic/kubermatic/api/pkg/validation"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// EqualError reports whether errors a and b are considered equal.
// They're equal if both are nil, or both are not nil and a.Error() == b.Error().
func EqualError(a, b error) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && a.Error() == b.Error())
}

func TestValidateCreateNodeSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		Name       string
		Cluster    *kubermaticv1.Cluster
		Spec       *apiv1.NodeSpec
		Datacenter *provider.DatacenterMeta
		Expected   error
	}{
		{
			"should pass validation when openstack provider not used",
			&kubermaticv1.Cluster{},
			&apiv1.NodeSpec{},
			&provider.DatacenterMeta{},
			nil,
		},
		{
			"should pass validation when floating ip not enforced and empty",
			&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							FloatingIPPool: "",
						},
					},
				},
			},
			&apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Openstack: &apiv1.OpenstackNodeSpec{UseFloatingIP: false},
				},
			},
			&provider.DatacenterMeta{
				Spec: provider.DatacenterSpec{
					Openstack: &provider.OpenstackSpec{EnforceFloatingIP: false},
				},
			},
			nil,
		},
		{
			"should fail validation when floating ip enforced and empty",
			&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							FloatingIPPool: "",
						},
					},
				},
			},
			&apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Openstack: &apiv1.OpenstackNodeSpec{UseFloatingIP: false},
				},
			},
			&provider.DatacenterMeta{
				Spec: provider.DatacenterSpec{
					Openstack: &provider.OpenstackSpec{EnforceFloatingIP: true},
				},
			},
			errors.New("no floating ip pool specified"),
		},
		{
			"should fail validation when floating ip not enforced, but required by user and empty",
			&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							FloatingIPPool: "",
						},
					},
				},
			},
			&apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Openstack: &apiv1.OpenstackNodeSpec{UseFloatingIP: true},
				},
			},
			&provider.DatacenterMeta{
				Spec: provider.DatacenterSpec{
					Openstack: &provider.OpenstackSpec{EnforceFloatingIP: false},
				},
			},
			errors.New("no floating ip pool specified"),
		},
		{
			"should pass validation when floating ip not enforced, but filled and required by user",
			&kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							FloatingIPPool: "ext-network",
						},
					},
				},
			},
			&apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Openstack: &apiv1.OpenstackNodeSpec{UseFloatingIP: true},
				},
			},
			&provider.DatacenterMeta{
				Spec: provider.DatacenterSpec{
					Openstack: &provider.OpenstackSpec{EnforceFloatingIP: false},
				},
			},
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			err := validation.ValidateCreateNodeSpec(c.Cluster, c.Spec, c.Datacenter)

			if !EqualError(err, c.Expected) {
				t.Fatalf("expected err to be '%v', but got '%v'", c.Expected, err)
			}
		})
	}
}
