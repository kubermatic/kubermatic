package validation

import (
	"errors"
	"fmt"
	"testing"

	v1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

var (
	dc = provider.DatacenterMeta{
		Spec: provider.DatacenterSpec{
			Openstack: &provider.OpenstackSpec{
				// Used for a test case
				EnforceFloatingIP: true,
			},
		},
	}
)

func TestValidateCloudSpec(t *testing.T) {
	tests := []struct {
		name string
		spec v1.CloudSpec
		err  error
	}{
		{
			name: "valid openstack spec",
			err:  nil,
			spec: v1.CloudSpec{
				DatacenterName: "some-datacenter",
				Openstack: &v1.OpenstackCloudSpec{
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
			spec: v1.CloudSpec{
				DatacenterName: "some-datacenter",
				Openstack: &v1.OpenstackCloudSpec{
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
			spec: v1.CloudSpec{
				DatacenterName: "",
				Openstack: &v1.OpenstackCloudSpec{
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
			spec: v1.CloudSpec{
				DatacenterName: "some-datacenter",
				Openstack: &v1.OpenstackCloudSpec{
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
