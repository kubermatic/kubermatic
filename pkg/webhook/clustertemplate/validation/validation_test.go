/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"bytes"
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/ptr"
)

var (
	testScheme     = fake.NewScheme()
	datacenterName = "foo"
)

// TestHandle tests the admission handler, but with the cloud provider validation
// disabled (i.e. we do not check if the hetzner token is valid, which would
// be done by issuing a HTTP call).
//
// ***************** IMPORTANT ***************
//
// This tests the admission webhook for ClusterTemplates standalone.
//

func TestHandle(t *testing.T) {
	seed := kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				datacenterName: {
					Spec: kubermaticv1.DatacenterSpec{
						Hetzner: &kubermaticv1.DatacenterSpecHetzner{},
					},
				},
			},
		},
	}

	config := kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{},
	}

	project1 := kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "abcd1234",
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: "my project",
		},
		Status: kubermaticv1.ProjectStatus{
			Phase: kubermaticv1.ProjectActive,
		},
	}

	project2 := kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "wxyz0987",
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: "my other project",
		},
		Status: kubermaticv1.ProjectStatus{
			Phase: kubermaticv1.ProjectActive,
		},
	}

	tests := []struct {
		name        string
		op          admissionv1.Operation
		features    features.FeatureGate
		template    kubermaticv1.ClusterTemplate
		wantAllowed bool
	}{
		{
			name: "Delete cluster success",
			op:   admissionv1.Delete,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        "Tunneling",
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
			}.Build(),
			wantAllowed: true,
		},
		{
			name: "Create cluster with Tunneling expose strategy succeeds",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        "Tunneling",
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
			}.Build(),
			wantAllowed: true,
		},
		{
			name: "Create cluster with invalid provider name",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        "Tunneling",
				ExternalCloudProvider: true,
				CloudProviderName:     string(kubermaticv1.DigitaloceanCloudProvider),
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Create cluster expose strategy different from Tunneling should succeed",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        "NodePort",
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
			}.Build(),
			wantAllowed: true,
		},
		{
			name: "Unknown expose strategy",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        "ciao",
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Unsupported CNIPlugin type",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "calium",
					Version: "v3.19",
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Unsupported CNIPlugin version",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "canal",
					Version: "v3.5",
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Supported CNIPlugin",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "canal",
					Version: "v3.28",
				},
			}.Build(),
			wantAllowed: true,
		},
		{
			name: "Supported CNIPlugin none",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "none",
					Version: "",
				},
			}.Build(),
			wantAllowed: true,
		},
		{
			name: "Reject unsupported ebpf proxy mode (wrong CNI)",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.EBPFProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
					KonnectivityEnabled:      ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "canal",
					Version: "v3.28",
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Reject unsupported ebpf proxy mode (Konnectivity not enabled)",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.EBPFProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
					KonnectivityEnabled:      ptr.To(false),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "cilium",
					Version: "1.16.6",
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Supported ebpf proxy mode",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.EBPFProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
					KonnectivityEnabled:      ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "cilium",
					Version: "1.16.6",
				},
			}.Build(),
			wantAllowed: true,
		},
		{
			name: "Supported CNI for dual-stack",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyDualStack,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd01::/48"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd02::/120"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
					KonnectivityEnabled:      ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "canal",
					Version: "v3.28",
				},
			}.Build(),
			wantAllowed: true,
		},
		{
			name: "Unsupported CNI for dual-stack",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyDualStack,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd01::/48"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd02::/120"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
					KonnectivityEnabled:      ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    "canal",
					Version: "v3.21",
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Accept a cluster template create request with externalCloudProvider enabled",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        "NodePort",
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
			}.Build(),
			wantAllowed: true,
		},
		{
			name: "Reject empty nodeport range",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.192.0.0/20"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "",
					},
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Reject malformed nodeport range",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.192.0.0/20"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "-",
					},
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Reject unsupported Kubernetes version",
			op:   admissionv1.Create,
			template: rawTemplateGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project2.Name,
					"scope":                        "project",
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.192.0.0/20"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
				Version: semver.NewSemverOrDie("1.0.0"),
			}.Build(),
			wantAllowed: false,
		},
	}

	seedClient := fake.
		NewClientBuilder().
		WithScheme(testScheme).
		WithObjects(&seed, &project1, &project2).
		Build()

	seedGetter := test.NewSeedGetter(&seed)
	configGetter := test.NewConfigGetter(&config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateValidator := validator{
				features:                  tt.features,
				client:                    seedClient,
				seedGetter:                seedGetter,
				configGetter:              configGetter,
				disableProviderValidation: true,
			}

			ctx := context.Background()
			var err error

			switch tt.op {
			case admissionv1.Create:
				_, err = templateValidator.ValidateCreate(ctx, &tt.template)
			case admissionv1.Update:
				_, err = templateValidator.ValidateUpdate(ctx, nil, &tt.template)
			case admissionv1.Delete:
				_, err = templateValidator.ValidateDelete(ctx, &tt.template)
			}

			allowed := err == nil

			if allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t: %v", allowed, tt.wantAllowed, err)
			}
		})
	}
}

type rawTemplateGen struct {
	Name                  string
	Namespace             string
	Labels                map[string]string
	ExposeStrategy        string
	EnableUserSSHKey      *bool
	ExternalCloudProvider bool
	CloudProviderName     string
	NetworkConfig         kubermaticv1.ClusterNetworkingConfig
	ComponentSettings     kubermaticv1.ComponentSettings
	CNIPlugin             *kubermaticv1.CNIPluginSettings
	Version               *semver.Semver
}

func (r rawTemplateGen) BuildPtr() *kubermaticv1.ClusterTemplate {
	c := r.Build()
	return &c
}

func (r rawTemplateGen) Build() kubermaticv1.ClusterTemplate {
	version := r.Version
	if version == nil {
		version = defaulting.DefaultKubernetesVersioning.Default
	}

	c := kubermaticv1.ClusterTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "ClusterTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
			Labels:    r.Labels,
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: "a-test-cluster",
			Version:           *version,
			ContainerRuntime:  "containerd",
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenterName,
				ProviderName:   string(kubermaticv1.HetznerCloudProvider),
				Hetzner: &kubermaticv1.HetznerCloudSpec{
					Token:   "thisis.reallyreallyfake",
					Network: "this-is-required-for-external-ccm-to-work",
				},
			},
			Features: map[string]bool{
				"externalCloudProvider": r.ExternalCloudProvider,
			},
			ExposeStrategy:        kubermaticv1.ExposeStrategy(r.ExposeStrategy),
			EnableUserSSHKeyAgent: r.EnableUserSSHKey,
			ClusterNetwork:        r.NetworkConfig,
			ComponentsOverride:    r.ComponentSettings,
			CNIPlugin:             r.CNIPlugin,
		},
	}

	if r.CloudProviderName != "" {
		c.Spec.Cloud.ProviderName = r.CloudProviderName
	}

	return c
}

func (r rawTemplateGen) Do() []byte {
	c := r.Build()
	s := json.NewSerializer(json.DefaultMetaFactory, testScheme, testScheme, true)
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&c, buff)
	return buff.Bytes()
}
