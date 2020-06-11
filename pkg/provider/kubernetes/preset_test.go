package kubernetes_test

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetPreset(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name          string
		presetName    string
		userInfo      provider.UserInfo
		presets       []runtime.Object
		expected      *kubermaticv1.Preset
		expectedError string
	}{
		{
			name:       "test 1: get Preset for the specific email group and name",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presetName: "test-3",
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-1",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "aaaaa",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-2",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "test.com",
						Fake: &kubermaticv1.Fake{
							Token: "bbbbb",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-3",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Fake: &kubermaticv1.Fake{
							Token: "abc",
						},
					},
				},
			},
			expected: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-3",
				},
				Spec: kubermaticv1.PresetSpec{
					RequiredEmailDomain: "example.com",
					Fake: &kubermaticv1.Fake{
						Token: "abc",
					},
				},
			},
		},
		{
			name:       "test 1: get Preset for the rest of the users",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presetName: "test-1",
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-1",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "aaaaa",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-2",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "test.com",
						Fake: &kubermaticv1.Fake{
							Token: "bbbbb",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-3",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Fake: &kubermaticv1.Fake{
							Token: "abc",
						},
					},
				}},
			expected: &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-1",
				},
				Spec: kubermaticv1.PresetSpec{
					Fake: &kubermaticv1.Fake{
						Token: "aaaaa",
					},
				},
			},
		},
		{
			name:       "test 3: get Preset which doesn't belong to specific group",
			presetName: "test-2",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-1",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "aaaaa",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-2",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "acme.com",
						Fake: &kubermaticv1.Fake{
							Token: "bbbbb",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-3",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "test.com",
						Fake: &kubermaticv1.Fake{
							Token: "abc",
						},
					},
				},
			},
			expectedError: "missing preset 'test-2' for the user 'test@example.com'",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.presets...)
			provider, err := kubernetes.NewPresetsProvider(context.Background(), fakeClient, "", true)
			if err != nil {
				t.Fatal(err)
			}
			preset, err := provider.GetPreset(&tc.userInfo, tc.presetName)
			if len(tc.expectedError) > 0 {
				if err == nil {
					t.Fatalf("expected error")
				}
				if err.Error() != tc.expectedError {
					t.Fatalf("expected: %s, got %v", tc.expectedError, err)
				}

			} else if !equality.Semantic.DeepEqual(preset, tc.expected) {
				t.Fatalf("expected: %v, got %v", tc.expected, preset)
			}
		})
	}
}

func TestGetPresets(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name     string
		userInfo provider.UserInfo
		presets  []runtime.Object
		expected []kubermaticv1.Preset
	}{
		{
			name:     "test 1: get Presets for the specific email group and all users",
			userInfo: provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-1",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "aaaaa",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-2",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "com",
						Fake: &kubermaticv1.Fake{
							Token: "bbbbb",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-3",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Fake: &kubermaticv1.Fake{
							Token: "abc",
						},
					},
				},
			},
			expected: []kubermaticv1.Preset{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-1",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "aaaaa",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-3",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Fake: &kubermaticv1.Fake{
							Token: "abc",
						},
					},
				},
			},
		},
		{
			name:     "test 1: get Presets for the all users, not for the specific email group",
			userInfo: provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-1",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "aaaaa",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-2",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "bbbbb",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-3",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "test.com",
						Fake: &kubermaticv1.Fake{
							Token: "abc",
						},
					},
				},
			},
			expected: []kubermaticv1.Preset{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-1",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "aaaaa",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tes-2",
					},
					Spec: kubermaticv1.PresetSpec{
						Fake: &kubermaticv1.Fake{
							Token: "bbbbb",
						},
					},
				},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.presets...)
			provider, err := kubernetes.NewPresetsProvider(context.Background(), fakeClient, "", true)
			if err != nil {
				t.Fatal(err)
			}
			presets, err := provider.GetPresets(&tc.userInfo)
			if err != nil {
				t.Fatal(err)
			}
			if !equality.Semantic.DeepEqual(presets, tc.expected) {
				t.Fatalf("expected: %v, got %v", tc.expected, presets)
			}
		})
	}
}

func TestCredentialEndpoint(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name              string
		presetName        string
		userInfo          provider.UserInfo
		expectedError     string
		cloudSpec         kubermaticv1.CloudSpec
		expectedCloudSpec *kubermaticv1.CloudSpec
		dc                *kubermaticv1.Datacenter
		presets           []runtime.Object
	}{
		{
			name:       "test 1: set credentials for Fake provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "com",
						Fake: &kubermaticv1.Fake{
							Token: "abcd",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Fake: &kubermaticv1.Fake{
							Token: "abc",
						},
					},
				},
			},
			cloudSpec:         kubermaticv1.CloudSpec{Fake: &kubermaticv1.FakeCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Fake: &kubermaticv1.FakeCloudSpec{Token: "abc"}},
		},
		{
			name:       "test 2: set credentials for GCP provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						GCP: &kubermaticv1.GCP{
							ServiceAccount: "test_service_accouont",
						},
					},
				},
			},

			cloudSpec:         kubermaticv1.CloudSpec{GCP: &kubermaticv1.GCPCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{GCP: &kubermaticv1.GCPCloudSpec{ServiceAccount: "test_service_accouont"}},
		},
		{
			name:       "test 3: set credentials for AWS provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						AWS: &kubermaticv1.AWS{
							SecretAccessKey: "secret", AccessKeyID: "key",
						},
					},
				},
			},

			cloudSpec:         kubermaticv1.CloudSpec{AWS: &kubermaticv1.AWSCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{AWS: &kubermaticv1.AWSCloudSpec{AccessKeyID: "key", SecretAccessKey: "secret"}},
		},
		{
			name:       "test 4: set credentials for Hetzner provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Hetzner: &kubermaticv1.Hetzner{
							Token: "secret",
						},
					},
				},
			},
			cloudSpec:         kubermaticv1.CloudSpec{Hetzner: &kubermaticv1.HetznerCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Hetzner: &kubermaticv1.HetznerCloudSpec{Token: "secret"}},
		},
		{
			name:       "test 5: set credentials for Packet provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Packet: &kubermaticv1.Packet{
							APIKey: "secret", ProjectID: "project",
						},
					},
				},
			},
			cloudSpec:         kubermaticv1.CloudSpec{Packet: &kubermaticv1.PacketCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Packet: &kubermaticv1.PacketCloudSpec{APIKey: "secret", ProjectID: "project", BillingCycle: "hourly"}},
		},
		{
			name:       "test 6: set credentials for DigitalOcean provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "fake",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example",
						Digitalocean: &kubermaticv1.Digitalocean{
							Token: "abcdefg",
						},
					},
				},
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Digitalocean: &kubermaticv1.Digitalocean{
							Token: "abcd",
						},
					},
				},
			},
			cloudSpec:         kubermaticv1.CloudSpec{Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Digitalocean: &kubermaticv1.DigitaloceanCloudSpec{Token: "abcd"}},
		},
		{
			name:       "test 7: set credentials for OpenStack provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Openstack: &kubermaticv1.Openstack{
							Tenant: "a", Domain: "b", Password: "c", Username: "d",
						},
					},
				},
			},
			dc:                &kubermaticv1.Datacenter{Spec: kubermaticv1.DatacenterSpec{Openstack: &kubermaticv1.DatacenterSpecOpenstack{EnforceFloatingIP: false}}},
			cloudSpec:         kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{Tenant: "a", Domain: "b", Password: "c", Username: "d"}},
		},
		{
			name:       "test 8: set credentials for Vsphere provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						VSphere: &kubermaticv1.VSphere{
							Username: "bob", Password: "secret",
						},
					},
				},
			},
			cloudSpec:         kubermaticv1.CloudSpec{VSphere: &kubermaticv1.VSphereCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{VSphere: &kubermaticv1.VSphereCloudSpec{Password: "secret", Username: "bob"}},
		},
		{
			name:       "test 9: set credentials for Azure provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Azure: &kubermaticv1.Azure{
							SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d",
						},
					},
				},
			},
			cloudSpec:         kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d"}},
		},
		{
			name:       "test 10: no credentials for Azure provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
					},
				},
			},
			cloudSpec:     kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{}},
			expectedError: "the preset test doesn't contain credential for Azure provider",
		},
		{
			name:       "test 11: cloud provider spec is empty",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Azure: &kubermaticv1.Azure{
							SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d",
						},
					},
				},
			},
			cloudSpec:     kubermaticv1.CloudSpec{},
			expectedError: "can not find provider to set credentials",
		},
		{
			name:       "test 12: set credentials for Kubevirt provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Kubevirt: &kubermaticv1.Kubevirt{
							Kubeconfig: "test",
						},
					},
				},
			},
			cloudSpec:         kubermaticv1.CloudSpec{Kubevirt: &kubermaticv1.KubevirtCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Kubevirt: &kubermaticv1.KubevirtCloudSpec{Kubeconfig: "test"}},
		},
		{
			name:       "test 13: credential with wrong email domain returns error",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "test.com",
						Azure: &kubermaticv1.Azure{
							SubscriptionID: "a", ClientID: "b", ClientSecret: "c", TenantID: "d",
						},
					},
				},
			},

			cloudSpec:     kubermaticv1.CloudSpec{Azure: &kubermaticv1.AzureCloudSpec{}},
			expectedError: "missing preset 'test' for the user 'test@example.com'",
		},
		{
			name:       "test 14: set credentials for Alibaba provider",
			presetName: "test",
			userInfo:   provider.UserInfo{Email: "test@example.com"},
			presets: []runtime.Object{
				&kubermaticv1.Preset{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test",
					},
					Spec: kubermaticv1.PresetSpec{
						RequiredEmailDomain: "example.com",
						Alibaba: &kubermaticv1.Alibaba{
							AccessKeySecret: "secret", AccessKeyID: "key",
						},
					},
				},
			},

			cloudSpec:         kubermaticv1.CloudSpec{Alibaba: &kubermaticv1.AlibabaCloudSpec{}},
			expectedCloudSpec: &kubermaticv1.CloudSpec{Alibaba: &kubermaticv1.AlibabaCloudSpec{AccessKeyID: "key", AccessKeySecret: "secret"}},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.presets...)
			provider, err := kubernetes.NewPresetsProvider(context.Background(), fakeClient, "", true)
			if err != nil {
				t.Fatal(err)
			}
			cloudResult, err := provider.SetCloudCredentials(&tc.userInfo, tc.presetName, tc.cloudSpec, tc.dc)

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
