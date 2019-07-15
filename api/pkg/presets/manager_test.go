package presets_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/presets"
)

func TestCredentialEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name              string
		credentialName    string
		expectedError     string
		cloudSpec         kubermaticv1.CloudSpec
		expectedCloudSpec *kubermaticv1.CloudSpec
		dc                *kubermaticv1.NodeLocation
		manager           *presets.Manager
	}{
		{
			name:           "test 1: set credentials for Fake provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().Fake.Credentials = []presets.FakeCredentials{
					{Name: "test", Token: "abc"},
					{Name: "pluto", Token: "def"},
				}
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Fake: &kubermaticv1.FakeCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Fake: &kubermaticv1.FakeCloudSpec{Token: "abc"}},
		},
		{
			name:           "test 2: set credentials for GCP provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().GCP.Credentials = []presets.GCPCredentials{
					{Name: "test", ServiceAccount: "test_service_accouont"},
				}
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{GCP: &kubermaticv1.GCPCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{GCP: &kubermaticv1.GCPCloudSpec{ServiceAccount: "test_service_accouont"}},
		},
		{
			name:           "test 3: set credentials for AWS provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().AWS.Credentials = []presets.AWSCredentials{
					{Name: "test", SecretAccessKey: "secret", AccessKeyID: "key"},
				}
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{AWS: &kubermaticv1.AWSCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{AWS: &kubermaticv1.AWSCloudSpec{AccessKeyID: "key", SecretAccessKey: "secret"}},
		},
		{
			name:           "test 4: set credentials for Hetzner provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().Hetzner.Credentials = []presets.HetznerCredentials{
					{Name: "test", Token: "secret"},
				}
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Hetzner: &kubermaticv1.HetznerCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Hetzner: &kubermaticv1.HetznerCloudSpec{Token: "secret"}},
		},
		{
			name:           "test 5: set credentials for Packet provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().Packet.Credentials = []presets.PacketCredentials{
					{Name: "test", APIKey: "secret", ProjectID: "project"},
				}
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Packet: &kubermaticv1.PacketCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Packet: &kubermaticv1.PacketCloudSpec{APIKey: "secret", ProjectID: "project", BillingCycle: "hourly"}},
		},
		{
			name:           "test 6: set credentials for DigitalOcean provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().Digitalocean.Credentials = []presets.DigitaloceanCredentials{
					{Name: "test", Token: "abcd"},
				}
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{Token: "abcd"}},
		},
		{
			name:           "test 7: set credentials for OpenStack provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().Openstack.Credentials = []presets.OpenstackCredentials{
					{Name: "test", Tenant: "a", Domain: "b", Password: "c", Username: "d"},
				}
				return manager
			}(),
			dc:                &kubermaticv1.NodeLocation{Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{EnforceFloatingIP: false}}},
			cloudSpec:         kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{Tenant: "a", Domain: "b", Password: "c", Username: "d"}},
		},
		{
			name:           "test 8: set credentials for Vsphere provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().VSphere.Credentials = []presets.VSphereCredentials{
					{Name: "test", Username: "bob", Password: "secret"},
				}
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{VSphere: &kubermaticv1.VSphereCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{VSphere: &kubermaticv1.VSphereCloudSpec{Password: "secret", Username: "bob"}},
		},
		{
			name:           "test 9: set credentials for Azure provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().Azure.Credentials = []presets.AzureCredentials{
					{Name: "test", SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d"},
				}
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d"}},
		},
		{
			name:           "test 10: no credentials for Azure provider",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				return manager
			}(),
			cloudSpec:     kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{}},
			expectedError: "can not find any credential for Azure provider",
		},
		{
			name:           "test 11: cloud provider spec is empty",
			credentialName: "test",
			manager: func() *presets.Manager {
				manager := presets.New()
				manager.GetPresets().Openstack.Credentials = []presets.OpenstackCredentials{
					{Name: "test", Tenant: "a", Domain: "b", Password: "c", Username: "d"},
				}
				return manager
			}(),
			cloudSpec:     kubermaticv1.CloudSpec{},
			expectedError: "can not find provider to set credentials",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			cloudResult, err := tc.manager.SetCloudCredentials(tc.credentialName, tc.cloudSpec, tc.dc)

			if len(tc.expectedError) > 0 {
				if err == nil {
					t.Fatalf("expected error")
				}
				if err.Error() != tc.expectedError {
					t.Fatalf("expected: %s, got %v", tc.expectedError, err)
				}

			} else {
				if !equality.Semantic.DeepEqual(cloudResult, tc.expectedCloudSpec) {
					t.Fatalf("expected: %v, got %v", tc.expectedCloudSpec, cloudResult)
				}
			}
		})
	}
}
