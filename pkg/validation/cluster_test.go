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
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
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
		name  string
		spec  kubermaticv1.CloudSpec
		valid bool
	}{
		{
			name:  "valid openstack spec",
			valid: true,
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
			name:  "valid openstack spec - only tenantID specified",
			valid: true,
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
			name:  "invalid openstack spec - no datacenter specified",
			valid: false,
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
			name:  "invalid openstack spec - no floating ip pool defined but required by dc",
			valid: false,
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
		{
			name:  "specifies multiple cloud providers",
			valid: false,
			spec: kubermaticv1.CloudSpec{
				DatacenterName: "some-datacenter",
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					Token: "a-token",
				},
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Tenant:         "some-tenant",
					Username:       "some-user",
					Password:       "some-password",
					Domain:         "some-domain",
					FloatingIPPool: "",
				},
			},
		},
		{
			name:  "valid provider name",
			valid: true,
			spec: kubermaticv1.CloudSpec{
				DatacenterName: "some-datacenter",
				ProviderName:   "openstack",
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Tenant:         "some-tenant",
					Username:       "some-user",
					Password:       "some-password",
					Domain:         "some-domain",
					FloatingIPPool: "some-network",
				},
			},
		},
		{
			name:  "invalid provider name",
			valid: false,
			spec: kubermaticv1.CloudSpec{
				DatacenterName: "some-datacenter",
				ProviderName:   "closedstack",
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Tenant:         "some-tenant",
					Username:       "some-user",
					Password:       "some-password",
					Domain:         "some-domain",
					FloatingIPPool: "some-network",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateCloudSpec(test.spec, dc, nil).ToAggregate()

			if (err == nil) != test.valid {
				t.Errorf("Extected err to be %v, got %v", test.valid, err)
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
			errs := ValidateLeaderElectionSettings(&test.leaderElectionSettings, field.NewPath("spec"))

			if test.wantErr == (len(errs) == 0) {
				t.Errorf("Want error: %t, but got: \"%v\"", test.wantErr, errs)
			}
		})
	}
}

func TestValidateClusterNetworkingConfig(t *testing.T) {
	tests := []struct {
		name          string
		networkConfig kubermaticv1.ClusterNetworkingConfig
		wantErr       bool
	}{
		{
			name:          "empty network config",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{},
			wantErr:       true,
		},
		{
			name: "valid network config",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
			},
			wantErr: false,
		},
		{
			name: "missing pods CIDR",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
			},
			wantErr: true,
		},
		{
			name: "missing services CIDR",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
			},
			wantErr: true,
		},
		{
			name: "missing DNS domain",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
			},
			wantErr: true,
		},
		{
			name: "missing proxy mode",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
			},
			wantErr: true,
		},
		{
			name: "invalid pod cidr",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"192.127.0.0:20"}},
			},
			wantErr: true,
		},
		{
			name: "invalid service cidr",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Services: kubermaticv1.NetworkRanges{CIDRBlocks: []string{"192.127/20"}},
			},
			wantErr: true,
		},
		{
			name: "invalid service cidr",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				DNSDomain: "cluster.bla",
			},
			wantErr: true,
		},
		{
			name: "invalid proxy mode",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				ProxyMode: "none",
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := ValidateClusterNetworkConfig(&test.networkConfig, nil, field.NewPath("spec", "networkConfig"))

			if test.wantErr == (len(errs) == 0) {
				t.Errorf("Want error: %t, but got: \"%v\"", test.wantErr, errs)
			}
		})
	}
}
