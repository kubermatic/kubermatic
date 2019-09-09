package presets_test

import (
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/presets"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestGetPreset(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name     string
		userInfo provider.UserInfo
		manager  *presets.Manager
		expected *kubermaticv1.Preset
	}{
		{
			name:     "test 1: get Preset for the specific email group",
			userInfo: provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								Fake: kubermaticv1.Fake{
									Credentials: []kubermaticv1.FakePresetCredentials{
										{Name: "test", Token: "aaaaa"},
									},
								},
							},
						},
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "com",
								Fake: kubermaticv1.Fake{
									Credentials: []kubermaticv1.FakePresetCredentials{
										{Name: "test", Token: "bbbbb"},
									},
								},
							},
						},
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Fake: kubermaticv1.Fake{
									Credentials: []kubermaticv1.FakePresetCredentials{
										{Name: "test", Token: "abc"},
										{Name: "pluto", Token: "def"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			expected: &kubermaticv1.Preset{
				Spec: kubermaticv1.PresetSpec{
					RequiredEmailDomain: "example.com",
					Fake: kubermaticv1.Fake{
						Credentials: []kubermaticv1.FakePresetCredentials{
							{Name: "test", Token: "abc"},
							{Name: "pluto", Token: "def"},
						},
					},
				},
			},
		},
		{
			name:     "test 1: get Preset for the all users",
			userInfo: provider.UserInfo{Email: "test@test.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								Fake: kubermaticv1.Fake{
									Credentials: []kubermaticv1.FakePresetCredentials{
										{Name: "test", Token: "aaaaa"},
									},
								},
							},
						},
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "acme.com",
								Fake: kubermaticv1.Fake{
									Credentials: []kubermaticv1.FakePresetCredentials{
										{Name: "test", Token: "bbbbb"},
									},
								},
							},
						},
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Fake: kubermaticv1.Fake{
									Credentials: []kubermaticv1.FakePresetCredentials{
										{Name: "test", Token: "abc"},
										{Name: "pluto", Token: "def"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			expected: &kubermaticv1.Preset{
				Spec: kubermaticv1.PresetSpec{
					Fake: kubermaticv1.Fake{
						Credentials: []kubermaticv1.FakePresetCredentials{
							{Name: "test", Token: "aaaaa"},
						},
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			preset := tc.manager.GetPreset(tc.userInfo)
			if !equality.Semantic.DeepEqual(preset, tc.expected) {
				t.Fatalf("expected: %v, got %v", tc.expected, preset)
			}
		})
	}
}

func TestCredentialEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name              string
		credentialName    string
		expectedError     string
		cloudSpec         kubermaticv1.CloudSpec
		expectedCloudSpec *kubermaticv1.CloudSpec
		dc                *kubermaticv1.Datacenter
		manager           *presets.Manager
		userInfo          provider.UserInfo
	}{
		{
			name:           "test 1: set credentials for Fake provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "com",
								Fake: kubermaticv1.Fake{
									Credentials: []kubermaticv1.FakePresetCredentials{
										{Name: "test", Token: "abcd"},
									},
								},
							},
						},
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Fake: kubermaticv1.Fake{
									Credentials: []kubermaticv1.FakePresetCredentials{
										{Name: "test", Token: "abc"},
										{Name: "pluto", Token: "def"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Fake: &kubermaticv1.FakeCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Fake: &kubermaticv1.FakeCloudSpec{Token: "abc"}},
		},
		{
			name:           "test 2: set credentials for GCP provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								GCP: kubermaticv1.GCP{
									Credentials: []kubermaticv1.GCPPresetCredentials{
										{Name: "test", ServiceAccount: "test_service_accouont"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{GCP: &kubermaticv1.GCPCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{GCP: &kubermaticv1.GCPCloudSpec{ServiceAccount: "test_service_accouont"}},
		},
		{
			name:           "test 3: set credentials for AWS provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								AWS: kubermaticv1.AWS{
									Credentials: []kubermaticv1.AWSPresetCredentials{
										{Name: "test", SecretAccessKey: "secret", AccessKeyID: "key"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{AWS: &kubermaticv1.AWSCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{AWS: &kubermaticv1.AWSCloudSpec{AccessKeyID: "key", SecretAccessKey: "secret"}},
		},
		{
			name:           "test 4: set credentials for Hetzner provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Hetzner: kubermaticv1.Hetzner{
									Credentials: []kubermaticv1.HetznerPresetCredentials{
										{Name: "test", Token: "secret"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Hetzner: &kubermaticv1.HetznerCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Hetzner: &kubermaticv1.HetznerCloudSpec{Token: "secret"}},
		},
		{
			name:           "test 5: set credentials for Packet provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Packet: kubermaticv1.Packet{
									Credentials: []kubermaticv1.PacketPresetCredentials{
										{Name: "test", APIKey: "secret", ProjectID: "project"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Packet: &kubermaticv1.PacketCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Packet: &kubermaticv1.PacketCloudSpec{APIKey: "secret", ProjectID: "project", BillingCycle: "hourly"}},
		},
		{
			name:           "test 6: set credentials for DigitalOcean provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example",
								Digitalocean: kubermaticv1.Digitalocean{
									Credentials: []kubermaticv1.DigitaloceanPresetCredentials{
										{Name: "test", Token: "abcdefg"},
									},
								},
							},
						},
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Digitalocean: kubermaticv1.Digitalocean{
									Credentials: []kubermaticv1.DigitaloceanPresetCredentials{
										{Name: "test", Token: "abcd"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{Token: "abcd"}},
		},
		{
			name:           "test 7: set credentials for OpenStack provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Openstack: kubermaticv1.Openstack{
									Credentials: []kubermaticv1.OpenstackPresetCredentials{
										{Name: "test", Tenant: "a", Domain: "b", Password: "c", Username: "d"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			dc:                &kubermaticv1.Datacenter{Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{EnforceFloatingIP: false}}},
			cloudSpec:         kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{Tenant: "a", Domain: "b", Password: "c", Username: "d"}},
		},
		{
			name:           "test 8: set credentials for Vsphere provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								VSphere: kubermaticv1.VSphere{
									Credentials: []kubermaticv1.VSpherePresetCredentials{
										{Name: "test", Username: "bob", Password: "secret"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{VSphere: &kubermaticv1.VSphereCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{VSphere: &kubermaticv1.VSphereCloudSpec{Password: "secret", Username: "bob"}},
		},
		{
			name:           "test 9: set credentials for Azure provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Azure: kubermaticv1.Azure{
									Credentials: []kubermaticv1.AzurePresetCredentials{
										{Name: "test", SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d"}},
		},
		{
			name:           "test 10: no credentials for Azure provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:     kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{}},
			expectedError: "can not find any credential for Azure provider",
		},
		{
			name:           "test 11: cloud provider spec is empty",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Azure: kubermaticv1.Azure{
									Credentials: []kubermaticv1.AzurePresetCredentials{
										{Name: "test", SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:     kubermaticv1.CloudSpec{},
			expectedError: "can not find provider to set credentials",
		},
		{
			name:           "test 12: set credentials for Kubevirt provider",
			credentialName: "test",
			userInfo:       provider.UserInfo{Email: "test@example.com"},
			manager: func() *presets.Manager {
				manager := presets.NewWithPresets(&kubermaticv1.PresetList{
					Items: []kubermaticv1.Preset{
						{
							Spec: kubermaticv1.PresetSpec{
								RequiredEmailDomain: "example.com",
								Kubevirt: kubermaticv1.Kubevirt{
									Credentials: []kubermaticv1.KubevirtPresetCredentials{
										{Name: "test", Kubeconfig: "test"},
									},
								},
							},
						},
					},
				})
				return manager
			}(),
			cloudSpec:         kubermaticv1.CloudSpec{Kubevirt: &kubermaticv1.KubevirtCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Kubevirt: &kubermaticv1.KubevirtCloudSpec{Kubeconfig: "test"}},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			cloudResult, err := tc.manager.SetCloudCredentials(tc.userInfo, tc.credentialName, tc.cloudSpec, tc.dc)

			if len(tc.expectedError) > 0 {
				if err == nil {
					t.Fatalf("expected error")
				}
				if err.Error() != tc.expectedError {
					t.Fatalf("expected: %s, got %v", tc.expectedError, err)
				}

			} else if !equality.Semantic.DeepEqual(cloudResult, tc.expectedCloudSpec) {
				t.Fatalf("expected: %v, got %v", tc.expectedCloudSpec, cloudResult)
			}
		})
	}
}
