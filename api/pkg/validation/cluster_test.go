package validation

import (
	"errors"
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

var (
	dc = &kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			Openstack: &kubermaticv1.DatacenterSpecOpenstack{
				// Used for a test case
				EnforceFloatingIP: true,
			},
		},
	}
)

func TestValidateCloudSpec(t *testing.T) {
	tests := []struct {
		name string
		spec kubermaticv1.CloudSpec
		err  error
	}{
		{
			name: "valid openstack spec",
			err:  nil,
			spec: kubermaticv1.CloudSpec{
				DatacenterName: "some-datacenter",
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Tenant:   "some-tenant",
					Username: "some-user",
					Password: "some-password",
					Domain:   "some-domain",
					// Required due to the above defined DC
					FloatingIPPool: "some-network",
				},
			},
		},
		{
			name: "valid openstack spec - only tenantID specified",
			err:  nil,
			spec: kubermaticv1.CloudSpec{
				DatacenterName: "some-datacenter",
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					TenantID: "some-tenant",
					Username: "some-user",
					Password: "some-password",
					Domain:   "some-domain",
					// Required due to the above defined DC
					FloatingIPPool: "some-network",
				},
			},
		},
		{
			name: "invalid openstack spec - no datacenter specified",
			err:  errors.New("no node datacenter specified"),
			spec: kubermaticv1.CloudSpec{
				DatacenterName: "",
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Tenant:   "some-tenant",
					Username: "some-user",
					Password: "some-password",
					Domain:   "some-domain",
					// Required due to the above defined DC
					FloatingIPPool: "some-network",
				},
			},
		},
		{
			name: "invalid openstack spec - no floating ip pool defined but required by dc",
			err:  errors.New("no floating ip pool specified"),
			spec: kubermaticv1.CloudSpec{
				DatacenterName: "some-datacenter",
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Tenant:         "some-tenant",
					Username:       "some-user",
					Password:       "some-password",
					Domain:         "some-domain",
					FloatingIPPool: "",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateCloudSpec(test.spec, dc)
			if fmt.Sprint(err) != fmt.Sprint(test.err) {
				t.Errorf("Extected err to be %v, got %v", test.err, err)
			}
		})
	}
}
