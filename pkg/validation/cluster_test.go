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
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8s.io/utils/pointer"
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
			err: errors.New("invalid time of day"),
		},
		{
			name: "invalid length",
			updateWindow: kubermaticv1.UpdateWindow{
				Start:  "Thu 04:00",
				Length: "1",
			},
			err: errors.New("missing unit in duration"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateUpdateWindow(&test.updateWindow)
			if (err != nil) != (test.err != nil) {
				t.Errorf("Extected err to be %v, got %v", test.err, err)
			}

			// loosely validate the returned error message
			if test.err != nil && !strings.Contains(err.Error(), test.err.Error()) {
				t.Errorf("Extected err to contain \"%v\", but got \"%v\"", test.err, err)
			}
		})
	}
}

func TestValidateLeaderElectionSettings(t *testing.T) {
	tests := []struct {
		name                   string
		leaderElectionSettings kubermaticv1.LeaderElectionSettings
		wantErr                bool
	}{
		{
			name:                   "empty leader election settings",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{},
			wantErr:                false,
		},
		{
			name: "valid leader election settings",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				LeaseDurationSeconds: pointer.Int32Ptr(int32(10)),
				RenewDeadlineSeconds: pointer.Int32Ptr(int32(5)),
				RetryPeriodSeconds:   pointer.Int32Ptr(int32(10)),
			},
			wantErr: false,
		},
		{
			name: "invalid leader election settings",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				LeaseDurationSeconds: pointer.Int32Ptr(int32(5)),
				RenewDeadlineSeconds: pointer.Int32Ptr(int32(10)),
				RetryPeriodSeconds:   pointer.Int32Ptr(int32(10)),
			},
			wantErr: true,
		},
		{
			name: "lease duration only",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				LeaseDurationSeconds: pointer.Int32Ptr(int32(10)),
			},
			wantErr: true,
		},
		{
			name: "renew duration only",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				RenewDeadlineSeconds: pointer.Int32Ptr(int32(10)),
			},
			wantErr: true,
		},
		{
			name: "retry period only",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				RetryPeriodSeconds: pointer.Int32Ptr(int32(10)),
			},
			wantErr: false,
		},
		{
			name: "negative value",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				LeaseDurationSeconds: pointer.Int32Ptr(int32(10)),
				RenewDeadlineSeconds: pointer.Int32Ptr(int32(-5)),
				RetryPeriodSeconds:   pointer.Int32Ptr(int32(10)),
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("Validating: %+v", test.leaderElectionSettings)
			err := ValidateLeaderElectionSettings(test.leaderElectionSettings)

			if test.wantErr == (err == nil) {
				t.Errorf("Want error: %t, but got: \"%v\"", test.wantErr, err)
			}
		})
	}
}
