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
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
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

func TestValidateClusterSpec(t *testing.T) {
	tests := []struct {
		name  string
		spec  *kubermaticv1.ClusterSpec
		valid bool
	}{
		{
			name:  "empty spec should not cause a panic",
			valid: false,
			spec:  &kubermaticv1.ClusterSpec{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateClusterSpec(test.spec, dc, features.FeatureGate{}, version.New([]*version.Version{{
				Version: semverlib.MustParse("1.2.3"),
			}}, nil, nil), nil, nil).ToAggregate()

			if (err == nil) != test.valid {
				t.Errorf("Extected err to be %v, got %v", test.valid, err)
			}
		})
	}
}

func TestValidateContainerRuntime(t *testing.T) {
	tests := []struct {
		name  string
		spec  *kubermaticv1.ClusterSpec
		valid bool
	}{
		{
			name: "valid container runtime",
			spec: &kubermaticv1.ClusterSpec{
				ContainerRuntime: "containerd",
				Version:          *semver.NewSemverOrDie("1.29.0"),
			},
			valid: true,
		},
		{
			name: "use empty container runtime",
			spec: &kubermaticv1.ClusterSpec{
				ContainerRuntime: "",
			},
			valid: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateContainerRuntime(test.spec)

			if (err == nil) != test.valid {
				t.Errorf("Extected validation result to be %v, got error %v", test.valid, err)
			}
		})
	}
}

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
				ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Project:  "some-project",
					Username: "some-user",
					Password: "some-password",
					Domain:   "some-domain",
					// Required due to the above defined DC
					FloatingIPPool: "some-network",
				},
			},
		},
		{
			name:  "valid openstack spec - only projectID specified",
			valid: true,
			spec: kubermaticv1.CloudSpec{
				DatacenterName: "some-datacenter",
				ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					ProjectID: "some-project",
					Username:  "some-user",
					Password:  "some-password",
					Domain:    "some-domain",
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
				ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Project:  "some-project",
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
				ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Project:        "some-project",
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
				ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
				Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{
					Token: "a-token",
				},
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Project:        "some-project",
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
				ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Project:        "some-project",
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
				ProviderName:   "closedstack", // *giggle*
				Openstack: &kubermaticv1.OpenstackCloudSpec{
					Project:        "some-project",
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
			err := ValidateCloudSpec(test.spec, dc, kubermaticv1.IPFamilyIPv4, nil, true).ToAggregate()

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
			err: errors.New("cannot parse \"invalid\" as \"15\""),
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
				LeaseDurationSeconds: ptr.To[int32](10),
				RenewDeadlineSeconds: ptr.To[int32](5),
				RetryPeriodSeconds:   ptr.To[int32](10),
			},
			wantErr: false,
		},
		{
			name: "invalid leader election settings",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				LeaseDurationSeconds: ptr.To[int32](5),
				RenewDeadlineSeconds: ptr.To[int32](10),
				RetryPeriodSeconds:   ptr.To[int32](10),
			},
			wantErr: true,
		},
		{
			name: "lease duration only",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				LeaseDurationSeconds: ptr.To[int32](10),
			},
			wantErr: true,
		},
		{
			name: "renew duration only",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				RenewDeadlineSeconds: ptr.To[int32](10),
			},
			wantErr: true,
		},
		{
			name: "retry period only",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				RetryPeriodSeconds: ptr.To[int32](10),
			},
			wantErr: false,
		},
		{
			name: "negative value",
			leaderElectionSettings: kubermaticv1.LeaderElectionSettings{
				LeaseDurationSeconds: ptr.To[int32](10),
				RenewDeadlineSeconds: ptr.To[int32](-5),
				RetryPeriodSeconds:   ptr.To[int32](10),
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
		dc            *kubermaticv1.Datacenter
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
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: false,
		},
		{
			name: "missing pods CIDR",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: true,
		},
		{
			name: "missing services CIDR",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: true,
		},
		{
			name: "valid dual-stack network config",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: false,
		},
		{
			name: "invalid dual-stack network config (IPv6 as primary address)",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"fd00::/104", "10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"fd03::/120", "10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: true,
		},
		{
			name: "invalid dual-stack network config (missing IPv6 services CIDR)",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: true,
		},
		{
			name: "valid ip family - IPv4",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyIPv4,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: false,
		},
		{
			name: "valid ip family - dual stack",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyDualStack,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: false,
		},
		{
			name: "invalid ip family",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyDualStack,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: true,
		},
		{
			name: "valid node CIDR mask sizes",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				NodeCIDRMaskSizeIPv4:     ptr.To[int32](26),
				NodeCIDRMaskSizeIPv6:     ptr.To[int32](112),
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: false,
		},
		{
			name: "invalid node CIDR mask size - IPv4",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				NodeCIDRMaskSizeIPv4:     ptr.To[int32](12),
				NodeCIDRMaskSizeIPv6:     ptr.To[int32](112),
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: true,
		},
		{
			name: "invalid node CIDR mask size - IPv6",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				NodeCIDRMaskSizeIPv4:     ptr.To[int32](24),
				NodeCIDRMaskSizeIPv6:     ptr.To[int32](64),
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: true,
		},
		{
			name: "missing DNS domain",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			wantErr: true,
		},
		{
			name: "missing proxy mode",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
				DNSDomain:                "cluster.local",
				NodeLocalDNSCacheEnabled: ptr.To(true),
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
			name: "invalid DNS domain",
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
		{
			name: "valid dual-stack datacenter config",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyDualStack,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					AWS: &kubermaticv1.DatacenterSpecAWS{},
				},
			},
			wantErr: false,
		},
		{
			name: "valid dual-stack datacenter config (openstack)",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyDualStack,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{
						IPv6Enabled: ptr.To(true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid dual-stack datacenter config (IPv6 not enabled for openstack datacenter)",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyDualStack,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{
						IPv6Enabled: ptr.To(false),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid dual-stack datacenter config (vsphere)",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyDualStack,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					VSphere: &kubermaticv1.DatacenterSpecVSphere{
						IPv6Enabled: ptr.To(true),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid dual-stack datacenter config (IPv6 not enabled for vsphere datacenter)",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyDualStack,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					VSphere: &kubermaticv1.DatacenterSpecVSphere{
						IPv6Enabled: ptr.To(false),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid dual-stack datacenter config (not known ipv6 cloud provider)",
			networkConfig: kubermaticv1.ClusterNetworkingConfig{
				IPFamily:                 kubermaticv1.IPFamilyDualStack,
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd00::/104"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd03::/120"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                "ipvs",
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Digitalocean: &kubermaticv1.DatacenterSpecDigitalocean{},
				},
			},
			wantErr: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := ValidateClusterNetworkConfig(&test.networkConfig, test.dc, nil, field.NewPath("spec", "networkConfig"))
			if test.wantErr == (len(errs) == 0) {
				t.Errorf("Want error: %t, but got: \"%v\"", test.wantErr, errs)
			}
		})
	}
}

func TestValidateCNIUpdate(t *testing.T) {
	tests := []struct {
		name    string
		old     *kubermaticv1.CNIPluginSettings
		new     *kubermaticv1.CNIPluginSettings
		wantErr bool
	}{
		{
			name: "allow minor version upgrade",
			old: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.11.0",
			},
			new: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.12.0",
			},
			wantErr: false,
		},
		{
			name: "allow minor version downgrade",
			old: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.12.0",
			},
			new: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.11.0",
			},
			wantErr: false,
		},
		{
			name: "allow patch version upgrade",
			old: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.13.0",
			},
			new: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.13.3",
			},
			wantErr: false,
		},
		{
			name: "allow patch version downgrade",
			old: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.13.3",
			},
			new: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.13.0",
			},
			wantErr: false,
		},
		{
			name: "invalid version upgrade",
			old: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.11.0",
			},
			new: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.13.0",
			},
			wantErr: true,
		},
		{
			name: "invalid version downgrade",
			old: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.14.0",
			},
			new: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCilium,
				Version: "1.12.1",
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := validateCNIUpdate(test.new, test.old, nil, *semver.NewSemverOrDie("v2.22"))
			if test.wantErr == (errs == nil) {
				t.Errorf("Want error: %t, but got: \"%v\"", test.wantErr, errs)
			}
		})
	}
}

func TestValidateGCPCloudSpec(t *testing.T) {
	testCases := []struct {
		name              string
		spec              *kubermaticv1.GCPCloudSpec
		dc                *kubermaticv1.Datacenter
		ipFamily          kubermaticv1.IPFamily
		gcpSubnetworkResp apiv1.GCPSubnetwork
		expectedError     error
	}{
		{
			name: "valid ipv4 gcp spec",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west3",
					},
				},
			},
			ipFamily: kubermaticv1.IPFamilyIPv4,
		},
		{
			name: "invalid gcp spec: service account cannot be empty",
			spec: &kubermaticv1.GCPCloudSpec{
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west3",
					},
				},
			},
			ipFamily:      kubermaticv1.IPFamilyIPv4,
			expectedError: errors.New("\"serviceAccount\" cannot be empty"),
		},
		{
			name: "invalid gcp spec: NodePortsAllowedIPRange",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "invalid",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west3",
					},
				},
			},
			ipFamily:      kubermaticv1.IPFamilyIPv4,
			expectedError: &net.ParseError{Type: "CIDR address", Text: "invalid"},
		},
		{
			name: "invalid gcp spec: NodePortsAllowedIPRanges",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"invalid",
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west3",
					},
				},
			},
			ipFamily:      kubermaticv1.IPFamilyIPv4,
			expectedError: fmt.Errorf("unable to parse CIDR \"invalid\": %w", &net.ParseError{Type: "CIDR address", Text: "invalid"}),
		},
		{
			name: "invalid dual-stack gcp spec: empty network",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west3",
					},
				},
			},
			ipFamily:      kubermaticv1.IPFamilyDualStack,
			expectedError: errors.New("network and subnetwork should be defined for GCP dual-stack (IPv4 + IPv6) cluster"),
		},
		{
			name: "invalid dual-stack gcp spec: empty subnetwork",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
				Network: "global/networks/dualstack",
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west3",
					},
				},
			},
			ipFamily:      kubermaticv1.IPFamilyDualStack,
			expectedError: errors.New("network and subnetwork should be defined for GCP dual-stack (IPv4 + IPv6) cluster"),
		},
		{
			name: "invalid dual-stack gcp spec: invalid subnetwork path",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
				Network:    "global/networks/dualstack",
				Subnetwork: "invalid",
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west3",
					},
				},
			},
			ipFamily:      kubermaticv1.IPFamilyDualStack,
			expectedError: errors.New("invalid GCP subnetwork path"),
		},
		{
			name: "invalid dual-stack gcp spec: wrong region",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
				Network:    "global/networks/dualstack",
				Subnetwork: "projects/kubermatic-dev/regions/europe-west2/subnetworks/dualstack-europe-west2",
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west3",
					},
				},
			},
			ipFamily:      kubermaticv1.IPFamilyDualStack,
			expectedError: errors.New("GCP subnetwork should belong to same cluster region"),
		},
		{
			name: "valid gcp dual-stack spec",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
				Network:    "global/networks/dualstack",
				Subnetwork: "projects/kubermatic-dev/regions/europe-west2/subnetworks/dualstack-europe-west2",
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west2",
					},
				},
			},
			ipFamily: kubermaticv1.IPFamilyDualStack,
			gcpSubnetworkResp: apiv1.GCPSubnetwork{
				IPFamily: kubermaticv1.IPFamilyDualStack,
			},
		},
		{
			name: "invalid gcp dual-stack spec: wrong network stack type",
			spec: &kubermaticv1.GCPCloudSpec{
				ServiceAccount:          "service-account",
				NodePortsAllowedIPRange: "0.0.0.0/0",
				NodePortsAllowedIPRanges: &kubermaticv1.NetworkRanges{
					CIDRBlocks: []string{
						"0.0.0.0/0",
						"::/0",
					},
				},
				Network:    "global/networks/default",
				Subnetwork: "projects/kubermatic-dev/regions/europe-west2/subnetworks/default",
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					GCP: &kubermaticv1.DatacenterSpecGCP{
						Region: "europe-west2",
					},
				},
			},
			ipFamily: kubermaticv1.IPFamilyDualStack,
			gcpSubnetworkResp: apiv1.GCPSubnetwork{
				IPFamily: kubermaticv1.IPFamilyIPv4,
			},
			expectedError: errors.New("GCP subnetwork should belong to same cluster network stack type"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGCPCloudSpec(tc.spec, tc.dc, tc.ipFamily, func(ctx context.Context, sa, region, subnetworkName string) (apiv1.GCPSubnetwork, error) {
				return tc.gcpSubnetworkResp, nil
			})
			assert.Equal(t, tc.expectedError, err)
		})
	}
}

func TestValidateEncryptionConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		clusterSpec *kubermaticv1.ClusterSpec
		expectErr   field.ErrorList
	}{
		{
			name: "small key",
			clusterSpec: &kubermaticv1.ClusterSpec{
				Features: map[string]bool{
					kubermaticv1.ClusterFeatureEncryptionAtRest: true,
				},
				EncryptionConfiguration: &kubermaticv1.EncryptionConfiguration{
					Enabled: true,
					Secretbox: &kubermaticv1.SecretboxEncryptionConfiguration{
						Keys: []kubermaticv1.SecretboxKey{
							{
								Name:  "small-key",
								Value: "cmLcMbw6gdxPHQ==",
							},
						},
					},
				},
			},
			expectErr: field.ErrorList{
				&field.Error{
					Type:     "FieldValueInvalid",
					Field:    "spec.encryptionConfiguration.secretbox.keys[0]",
					BadValue: "small-key",
					Detail:   "key length should be 32 it is 10",
				},
			},
		},
		{
			name: "bad base64",
			clusterSpec: &kubermaticv1.ClusterSpec{
				Features: map[string]bool{
					kubermaticv1.ClusterFeatureEncryptionAtRest: true,
				},
				EncryptionConfiguration: &kubermaticv1.EncryptionConfiguration{
					Enabled: true,
					Secretbox: &kubermaticv1.SecretboxEncryptionConfiguration{
						Keys: []kubermaticv1.SecretboxKey{
							{
								Name:  "small-key",
								Value: "cmLcMbw6gdxPH$==",
							},
						},
					},
				},
			},
			expectErr: field.ErrorList{
				&field.Error{
					Type:     "FieldValueInvalid",
					Field:    "spec.encryptionConfiguration.secretbox.keys[0]",
					BadValue: "small-key",
					Detail:   "illegal base64 data at input byte 13",
				},
			},
		},
		{
			name: "good key",
			clusterSpec: &kubermaticv1.ClusterSpec{
				Features: map[string]bool{
					kubermaticv1.ClusterFeatureEncryptionAtRest: true,
				},
				EncryptionConfiguration: &kubermaticv1.EncryptionConfiguration{
					Enabled: true,
					Secretbox: &kubermaticv1.SecretboxEncryptionConfiguration{
						Keys: []kubermaticv1.SecretboxKey{
							{
								Name:  "good-key",
								Value: "RGolflgAc+eBbm1lys87pTNQZVf0i67rlpPZGtTkVjQ=",
							},
						},
					},
				},
			},
			expectErr: field.ErrorList{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateEncryptionConfiguration(test.clusterSpec, field.NewPath("spec", "encryptionConfiguration"))
			assert.Equal(t, test.expectErr, err)
		})
	}
}

func TestValidateVersion(t *testing.T) {
	tests := []struct {
		name           string
		spec           *kubermaticv1.ClusterSpec
		versionManager *version.Manager
		currentVersion *semver.Semver
		valid          bool
	}{
		{
			name:  "empty spec should not cause a panic",
			valid: false,
			spec:  &kubermaticv1.ClusterSpec{},
		},
		{
			name:  "version supported",
			valid: true,
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.2.3"),
			},
			versionManager: version.New([]*version.Version{{
				Version: semverlib.MustParse("1.2.3"),
			}}, nil, nil),
		},
		{
			name:  "version not supported",
			valid: false,
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.2.4"),
			},
			versionManager: version.New([]*version.Version{{
				Version: semverlib.MustParse("1.2.3"),
			}}, nil, nil),
		},
		{
			name:  "version with automatic update via Automatic not supported",
			valid: false,
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.2.2"),
			},
			versionManager: version.New(
				[]*version.Version{
					{
						Version: semverlib.MustParse("1.2.2"),
					},
					{
						Version: semverlib.MustParse("1.2.3"),
					},
				},
				[]*version.Update{{
					From:      "1.2.2",
					To:        "1.2.3",
					Automatic: true,
				}}, nil),
		},
		{
			name:  "version with automatic update via AutomaticNodeUpdate not supported",
			valid: false,
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.2.2"),
			},
			versionManager: version.New(
				[]*version.Version{
					{
						Version: semverlib.MustParse("1.2.2"),
					},
					{
						Version: semverlib.MustParse("1.2.3"),
					},
				},
				[]*version.Update{{
					From:                "1.2.2",
					To:                  "1.2.3",
					AutomaticNodeUpdate: true,
				}}, nil),
		},
		{
			name:           "version update supported",
			valid:          true,
			currentVersion: semver.NewSemverOrDie("1.2.3"),
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.2.4"),
			},
			versionManager: version.New(
				[]*version.Version{
					{
						Version: semverlib.MustParse("1.2.3"),
					},
					{
						Version: semverlib.MustParse("1.2.4"),
					},
				},
				[]*version.Update{{
					From: "1.2.*",
					To:   "1.2.*",
				}}, nil),
		},
		{
			name:           "version update not supported",
			valid:          false,
			currentVersion: semver.NewSemverOrDie("1.2.2"),
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.2.4"),
			},
			versionManager: version.New(
				[]*version.Version{
					{
						Version: semverlib.MustParse("1.2.2"),
					},
					{
						Version: semverlib.MustParse("1.2.3"),
					},
					{
						Version: semverlib.MustParse("1.2.4"),
					},
				},
				[]*version.Update{{
					From: "1.2.*",
					To:   "1.2.3",
				}}, nil),
		},
		{
			name:  "cloud provider incompatibility version supported",
			valid: true,
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.2.0"),
				Cloud: kubermaticv1.CloudSpec{
					ProviderName: string(kubermaticv1.OpenstackCloudProvider),
				},
			},
			versionManager: version.New(
				[]*version.Version{
					{
						Version: semverlib.MustParse("1.2.0"),
					},
					{
						Version: semverlib.MustParse("1.3.0"),
					},
				}, nil, []*version.ProviderIncompatibility{
					{
						Provider:  kubermaticv1.OpenstackCloudProvider,
						Condition: kubermaticv1.InTreeCloudProviderCondition,
						Operation: kubermaticv1.CreateOperation,
						Version:   ">= 1.3.0",
					},
				},
			),
		},
		{
			name:  "cloud provider incompatibility version not supported",
			valid: false,
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.3.0"),
				Cloud: kubermaticv1.CloudSpec{
					ProviderName: string(kubermaticv1.OpenstackCloudProvider),
					Openstack:    &kubermaticv1.OpenstackCloudSpec{},
				},
			},
			versionManager: version.New(
				[]*version.Version{
					{
						Version: semverlib.MustParse("1.2.0"),
					},
					{
						Version: semverlib.MustParse("1.3.0"),
					},
				}, nil,
				[]*version.ProviderIncompatibility{
					{
						Provider:  kubermaticv1.OpenstackCloudProvider,
						Condition: kubermaticv1.InTreeCloudProviderCondition,
						Operation: kubermaticv1.CreateOperation,
						Version:   ">= 1.3.0",
					},
				},
			),
		},
		{
			name:           "cloud provider incompatibility version upgrade not supported",
			valid:          false,
			currentVersion: semver.NewSemverOrDie("1.2.0"),
			spec: &kubermaticv1.ClusterSpec{
				Version: semver.Semver("1.3.0"),
				Cloud: kubermaticv1.CloudSpec{
					ProviderName: string(kubermaticv1.OpenstackCloudProvider),
					Openstack:    &kubermaticv1.OpenstackCloudSpec{},
				},
			},
			versionManager: version.New(
				[]*version.Version{
					{
						Version: semverlib.MustParse("1.2.0"),
					},
					{
						Version: semverlib.MustParse("1.3.0"),
					},
				},
				[]*version.Update{
					{
						From: "1.2.*",
						To:   "1.2.*",
					},
					{
						From: "1.2.*",
						To:   "1.3.*",
					},
				},
				[]*version.ProviderIncompatibility{
					{
						Provider:  kubermaticv1.OpenstackCloudProvider,
						Condition: kubermaticv1.InTreeCloudProviderCondition,
						Operation: kubermaticv1.UpdateOperation,
						Version:   ">= 1.3.0",
					},
				},
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateVersion(test.spec, test.versionManager, test.currentVersion, nil)

			if (err == nil) != test.valid {
				t.Errorf("Extected err == nil to be %v, got err: %v", test.valid, err)
			}
		})
	}
}

func TestValidateCoreDNSReplicas(t *testing.T) {
	tests := []struct {
		name  string
		spec  *kubermaticv1.ClusterSpec
		valid bool
	}{
		{
			name:  "two identical CoreDNS replica counts",
			valid: true,
			spec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					CoreDNSReplicas: ptr.To[int32](2),
				},
				ComponentsOverride: kubermaticv1.ComponentSettings{
					CoreDNS: &kubermaticv1.DeploymentSettings{
						Replicas: ptr.To[int32](2),
					},
				},
			},
		},
		{
			name:  "two different CoreDNS replica counts",
			valid: false,
			spec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					CoreDNSReplicas: ptr.To[int32](1),
				},
				ComponentsOverride: kubermaticv1.ComponentSettings{
					CoreDNS: &kubermaticv1.DeploymentSettings{
						Replicas: ptr.To[int32](2),
					},
				},
			},
		},
		{
			name:  "only using the old mechanism to set CoreDNS replicas",
			valid: true,
			spec: &kubermaticv1.ClusterSpec{
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					CoreDNSReplicas: ptr.To[int32](1),
				},
			},
		},
		{
			name:  "using the new mechanism to set CoreDNS replicas",
			valid: true,
			spec: &kubermaticv1.ClusterSpec{
				ComponentsOverride: kubermaticv1.ComponentSettings{
					CoreDNS: &kubermaticv1.DeploymentSettings{
						Replicas: ptr.To[int32](2),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateCoreDNSReplicas(test.spec, &field.Path{})

			if (err == nil) != test.valid {
				t.Errorf("Extected err to be %v, got %v", test.valid, err)
			}
		})
	}
}

func TestValidateEventRateLimitConfig(t *testing.T) {
	tests := []struct {
		name    string
		spec    *kubermaticv1.ClusterSpec
		wantErr bool
	}{
		{
			name: "plugin disabled, no config - valid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: false,
			},
			wantErr: false,
		},
		{
			name: "plugin enabled via flag, no limits configured - invalid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
			},
			wantErr: true,
		},
		{
			name: "plugin enabled via flag, empty config - invalid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig:             &kubermaticv1.EventRateLimitConfig{},
			},
			wantErr: true,
		},
		{
			name: "plugin enabled via AdmissionPlugins list, no limits - invalid",
			spec: &kubermaticv1.ClusterSpec{
				AdmissionPlugins: []string{"EventRateLimit"},
			},
			wantErr: true,
		},
		{
			name: "plugin enabled, valid server config",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Server: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   100,
						Burst: 200,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "plugin enabled, valid namespace config",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Namespace: &kubermaticv1.EventRateLimitConfigItem{
						QPS:       50,
						Burst:     100,
						CacheSize: 1000,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "plugin enabled, valid config with all limit types",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Server: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   100,
						Burst: 200,
					},
					Namespace: &kubermaticv1.EventRateLimitConfigItem{
						QPS:       50,
						Burst:     100,
						CacheSize: 1000,
					},
					User: &kubermaticv1.EventRateLimitConfigItem{
						QPS:       10,
						Burst:     20,
						CacheSize: 500,
					},
					SourceAndObject: &kubermaticv1.EventRateLimitConfigItem{
						QPS:       5,
						Burst:     10,
						CacheSize: 100,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "QPS zero - invalid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Server: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   0,
						Burst: 100,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "QPS negative - invalid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Server: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   -5,
						Burst: 100,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Burst zero - invalid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Namespace: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   100,
						Burst: 0,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Burst negative - invalid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Namespace: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   100,
						Burst: -10,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "CacheSize negative - invalid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					User: &kubermaticv1.EventRateLimitConfigItem{
						QPS:       100,
						Burst:     200,
						CacheSize: -1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "CacheSize zero - valid",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Server: &kubermaticv1.EventRateLimitConfigItem{
						QPS:       100,
						Burst:     200,
						CacheSize: 0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "minimum valid values (QPS=1, Burst=1)",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: true,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Server: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   1,
						Burst: 1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "plugin disabled but config provided with invalid values - still validates config",
			spec: &kubermaticv1.ClusterSpec{
				UseEventRateLimitAdmissionPlugin: false,
				EventRateLimitConfig: &kubermaticv1.EventRateLimitConfig{
					Server: &kubermaticv1.EventRateLimitConfigItem{
						QPS:   -1,
						Burst: -1,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := ValidateEventRateLimitConfig(test.spec, field.NewPath("spec", "eventRateLimitConfig"))
			if test.wantErr == (len(errs) == 0) {
				t.Errorf("Want error: %t, but got: %v", test.wantErr, errs)
			}
		})
	}
}
