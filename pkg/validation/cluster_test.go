/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

func TestValidateUpdateWindow(t *testing.T) {
	tests := []struct {
		name         string
		updateWindow kubermaticv1.UpdateWindow
		err          error
	}{
		{
			name: "valid update window",
			updateWindow: kubermaticv1.UpdateWindow{
				Start:  "04:00",
				Length: "1h",
			},
			err: nil,
		},
		{
			name: "invalid start date",
			updateWindow: kubermaticv1.UpdateWindow{
				Start:  "invalid",
				Length: "1h",
			},
			err: errors.New("error parsing update window: unable to parse start: invalid time of day \"invalid\": expected integer"),
		},
		{
			name: "invalid length",
			updateWindow: kubermaticv1.UpdateWindow{
				Start:  "Thu 04:00",
				Length: "1",
			},
			err: errors.New("error parsing update window: unable to parse duration: time: missing unit in duration 1"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateUpdateWindow(&test.updateWindow)
			if fmt.Sprint(err) != fmt.Sprint(test.err) {
				t.Errorf("Extected err to be %v, got %v", test.err, err)
			}
		})
	}
}
