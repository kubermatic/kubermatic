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
	"k8c.io/kubermatic/v2/pkg/validation"

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
// This tests the admission webhook standalone. In real-life scenarios, the defaulting
// webhook will ensure default values (duh) on the Cluster and is called by the
// kube-apiserver *before* the admission webhook is called. So for example this function
// ensures that an empty nodeport range fails, but in reality, this never happens
// because of the mutating webhook.
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
		cluster     kubermaticv1.Cluster
		oldCluster  *kubermaticv1.Cluster
		datacenter  *kubermaticv1.Datacenter
		wantAllowed bool
	}{
		{
			name: "Delete cluster success",
			op:   admissionv1.Delete,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			name: "Create cluster with Tunneling expose strategy succeeds when the FeatureGate is enabled",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        "NodePort",
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
					Version: "v3.28",
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Unsupported CNIPlugin version",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			name: "Supported nftables proxy mode with Canal",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.NFTablesProxyMode,
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
			name: "Supported nftables proxy mode with Cilium",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.NFTablesProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
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
			name: "Supported iptables proxy mode",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPTablesProxyMode,
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
			name: "Supported CNI for dual-stack",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyDualStack,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd01::/48"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd02::/108"}},
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyDualStack,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16", "fd01::/48"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20", "fd02::/108"}},
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
			name: "Reject EnableUserSSHKey agent update",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        "NodePort",
				EnableUserSSHKey:      ptr.To(true),
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
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        "NodePort",
				EnableUserSSHKey:      ptr.To(false),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain: "cluster.local",
					ProxyMode: resources.IPVSProxyMode,
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32768",
					},
				},
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Accept a cluster create request with externalCloudProvider enabled",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			name: "Accept enabling the externalCloudProvider feature",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        "NodePort",
				ExternalCloudProvider: false,
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
			}.BuildPtr(),
			wantAllowed: true,
		},
		{
			name: "Reject disabling the externalCloudProvider feature",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        "NodePort",
				ExternalCloudProvider: false,
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
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Reject updating the pods CIDR",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
						NodePortRange: "30000-32768",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
				},
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.193.0.0/20"}},
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
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Reject updating the nodeport range",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
						NodePortRange: "30000-32769",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
						NodePortRange: "30000-32768",
					},
				},
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Reject empty nodeport range",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			name: "Reject empty nodeport range update",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
						NodePortRange: "", // this will be defaulted to the
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Reject CNIPlugin version update (more than one minor version)",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.26",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Reject CNIPlugin version update (major version change)",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v9.99",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v2.21",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Reject CNIPlugin version update (invalid semver)",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "<invalid>",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Allow CNIPlugin version update (one minor version)",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.29",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: true,
		},
		{
			name: "Allow CNIPlugin version update (downgrade one minor version)",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.29",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: true,
		},
		{
			name: "Allow unsafe CNIPlugin version update with explicit label",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey:   project1.Name,
					validation.UnsafeCNIUpgradeLabel: "true",
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.29",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v2.0",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: true,
		},
		{
			name: "Reject CNIPlugin type change",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCilium,
					Version: "1.16.6",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Allow unsafe CNIPlugin type change with explicit label",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey:     project1.Name,
					validation.UnsafeCNIMigrationLabel: "true",
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCilium,
					Version: "1.16.6",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: true,
		},
		{
			name: "Reject remove CNIPlugin settings",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Allow add CNIPlugin settings",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
			}.BuildPtr(),
			wantAllowed: true,
		},
		{
			name: "Require project label",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:                  "foo",
				Namespace:             "kubermatic",
				ExposeStrategy:        kubermaticv1.ExposeStrategyNodePort.String(),
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.192.0.0/20"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "cluster.local",
					ProxyMode:                resources.IPVSProxyMode,
					NodeLocalDNSCacheEnabled: ptr.To(true),
				},
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			wantAllowed: false,
		},
		{
			name: "Project label is immutable",
			op:   admissionv1.Update,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project2.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project1.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
			}.BuildPtr(),
			wantAllowed: false,
		},
		{
			name: "Reject unsupported Kubernetes version",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project2.Name,
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
		{
			name: "Reject invalid rfc1035 name",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "4foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project2.Name,
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
		{
			name: "Reject unsupported Kubernetes version update",
			op:   admissionv1.Create,
			cluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project2.Name,
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
				Version: semver.NewSemverOrDie("1.99.99"),
			}.Build(),
			oldCluster: rawClusterGen{
				Name:      "foo",
				Namespace: "kubermatic",
				Labels: map[string]string{
					kubermaticv1.ProjectIDLabelKey: project2.Name,
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
				CNIPlugin: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.28",
				},
				ComponentSettings: kubermaticv1.ComponentSettings{
					Apiserver: kubermaticv1.APIServerSettings{
						NodePortRange: "30000-32000",
					},
				},
				Version: defaulting.DefaultKubernetesVersioning.Default,
			}.BuildPtr(),
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSeed := seed.DeepCopy()

			if tt.datacenter != nil {
				testSeed.Spec.Datacenters = map[string]kubermaticv1.Datacenter{
					datacenterName: *tt.datacenter,
				}
			}

			seedClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(testSeed, &project1, &project2).
				Build()

			seedGetter := test.NewSeedGetter(testSeed)
			configGetter := test.NewConfigGetter(&config)

			clusterValidator := validator{
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
				_, err = clusterValidator.ValidateCreate(ctx, &tt.cluster)
			case admissionv1.Update:
				_, err = clusterValidator.ValidateUpdate(ctx, tt.oldCluster, &tt.cluster)
			case admissionv1.Delete:
				_, err = clusterValidator.ValidateDelete(ctx, &tt.cluster)
			}

			allowed := err == nil

			if allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t: %v", allowed, tt.wantAllowed, err)
			}
		})
	}
}

type rawClusterGen struct {
	Name                                string
	Namespace                           string
	CloudProviderName                   string
	Labels                              map[string]string
	ExposeStrategy                      string
	EnableUserSSHKey                    *bool
	ExternalCloudProvider               bool
	NetworkConfig                       kubermaticv1.ClusterNetworkingConfig
	ComponentSettings                   kubermaticv1.ComponentSettings
	CNIPlugin                           *kubermaticv1.CNIPluginSettings
	UsePodSecurityPolicyAdmissionPlugin bool
	AdmissionPlugins                    []string
	Version                             *semver.Semver
}

func (r rawClusterGen) BuildPtr() *kubermaticv1.Cluster {
	c := r.Build()
	return &c
}

func (r rawClusterGen) Build() kubermaticv1.Cluster {
	version := r.Version
	if version == nil {
		version = defaulting.DefaultKubernetesVersioning.Default
	}

	c := kubermaticv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Namespace,
			Labels:    r.Labels,
		},
		Spec: kubermaticv1.ClusterSpec{
			HumanReadableName: "a test cluster",
			Version:           *version,
			ContainerRuntime:  "containerd",
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenterName,
				ProviderName:   string(kubermaticv1.HetznerCloudProvider),
				Hetzner: &kubermaticv1.HetznerCloudSpec{
					Token:   "thisis.reallyreallyfake",
					Network: "this-is-required-for-ccm",
				},
			},
			Features: map[string]bool{
				"externalCloudProvider": r.ExternalCloudProvider,
			},
			ExposeStrategy:                      kubermaticv1.ExposeStrategy(r.ExposeStrategy),
			EnableUserSSHKeyAgent:               r.EnableUserSSHKey,
			ClusterNetwork:                      r.NetworkConfig,
			ComponentsOverride:                  r.ComponentSettings,
			CNIPlugin:                           r.CNIPlugin,
			UsePodSecurityPolicyAdmissionPlugin: r.UsePodSecurityPolicyAdmissionPlugin,
			AdmissionPlugins:                    r.AdmissionPlugins,
		},
	}

	if r.CloudProviderName != "" {
		c.Spec.Cloud.ProviderName = r.CloudProviderName
		c.Spec.Cloud.Hetzner = nil

		switch r.CloudProviderName {
		case string(kubermaticv1.VSphereCloudProvider):
			c.Spec.Cloud.VSphere = &kubermaticv1.VSphereCloudSpec{
				Username: "fake",
				Password: "fake",
			}
		}
	}

	return c
}

func (r rawClusterGen) Do() []byte {
	c := r.Build()
	s := json.NewSerializer(json.DefaultMetaFactory, testScheme, testScheme, true)
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&c, buff)
	return buff.Bytes()
}

func TestValidateEventRateLimitEnforcement(t *testing.T) {
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

	project := kubermaticv1.Project{
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

	baseCluster := func() rawClusterGen {
		return rawClusterGen{
			Name:      "foo",
			Namespace: "kubermatic",
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: project.Name,
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
		}
	}

	tests := []struct {
		name                       string
		op                         admissionv1.Operation
		config                     *kubermaticv1.KubermaticConfiguration
		useEventRateLimitPlugin    bool
		oldUseEventRateLimitPlugin *bool
		userConfig                 *kubermaticv1.EventRateLimitConfig
		wantAllowed                bool
	}{
		{
			name: "Create cluster without enforcement - no plugin enabled",
			op:   admissionv1.Create,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{},
			},
			useEventRateLimitPlugin: false,
			wantAllowed:             true,
		},
		{
			name: "Create cluster with enforcement - plugin not enabled",
			op:   admissionv1.Create,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
							},
						},
					},
				},
			},
			useEventRateLimitPlugin: false,
			wantAllowed:             false,
		},
		{
			name: "Create cluster with enforcement - plugin enabled",
			op:   admissionv1.Create,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
							},
						},
					},
				},
			},
			useEventRateLimitPlugin: true,
			userConfig: &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   50,
					Burst: 100,
				},
			},
			wantAllowed: true,
		},
		{
			name: "Update cluster with enforcement - disable plugin",
			op:   admissionv1.Update,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
							},
						},
					},
				},
			},
			useEventRateLimitPlugin:    false,
			oldUseEventRateLimitPlugin: ptr.To(true),
			wantAllowed:                false,
		},
		{
			name: "Update cluster with enforcement - keep plugin enabled",
			op:   admissionv1.Update,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
							},
						},
					},
				},
			},
			useEventRateLimitPlugin:    true,
			oldUseEventRateLimitPlugin: ptr.To(true),
			userConfig: &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   50,
					Burst: 100,
				},
			},
			wantAllowed: true,
		},
		{
			name: "Update cluster without enforcement - disable plugin",
			op:   admissionv1.Update,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{},
			},
			useEventRateLimitPlugin:    false,
			oldUseEventRateLimitPlugin: ptr.To(true),
			wantAllowed:                true,
		},
		{
			name: "Create cluster with enforced=false - plugin not enabled",
			op:   admissionv1.Create,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(false),
							},
						},
					},
				},
			},
			useEventRateLimitPlugin: false,
			wantAllowed:             true,
		},
		{
			name: "Create with enforced config - user specifies different config",
			op:   admissionv1.Create,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
								DefaultConfig: &kubermaticv1.EventRateLimitConfig{
									Server: &kubermaticv1.EventRateLimitConfigItem{
										QPS:   50,
										Burst: 100,
									},
								},
							},
						},
					},
				},
			},
			useEventRateLimitPlugin: true,
			userConfig: &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   25,
					Burst: 50,
				},
			},
			wantAllowed: false,
		},
		{
			name: "Create with enforced config - user specifies same config",
			op:   admissionv1.Create,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
								DefaultConfig: &kubermaticv1.EventRateLimitConfig{
									Server: &kubermaticv1.EventRateLimitConfigItem{
										QPS:   50,
										Burst: 100,
									},
								},
							},
						},
					},
				},
			},
			useEventRateLimitPlugin: true,
			userConfig: &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   50,
					Burst: 100,
				},
			},
			wantAllowed: true,
		},
		{
			name: "Create with enforced config - no user config",
			op:   admissionv1.Create,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enforced: ptr.To(true),
								DefaultConfig: &kubermaticv1.EventRateLimitConfig{
									Server: &kubermaticv1.EventRateLimitConfigItem{
										QPS:   50,
										Burst: 100,
									},
								},
							},
						},
					},
				},
			},
			useEventRateLimitPlugin: true,
			userConfig: &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   50,
					Burst: 100,
				},
			},
			wantAllowed: true,
		},
		{
			name: "Create without enforced config - user can override",
			op:   admissionv1.Create,
			config: &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubermatic",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
						AdmissionPlugins: &kubermaticv1.AdmissionPluginsConfiguration{
							EventRateLimit: &kubermaticv1.EventRateLimitPluginConfiguration{
								Enabled: ptr.To(true),
								DefaultConfig: &kubermaticv1.EventRateLimitConfig{
									Server: &kubermaticv1.EventRateLimitConfigItem{
										QPS:   50,
										Burst: 100,
									},
								},
							},
						},
					},
				},
			},
			useEventRateLimitPlugin: true,
			userConfig: &kubermaticv1.EventRateLimitConfig{
				Server: &kubermaticv1.EventRateLimitConfigItem{
					QPS:   25,
					Burst: 50,
				},
			},
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testSeed := seed.DeepCopy()

			seedClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(testSeed, &project).
				Build()

			seedGetter := test.NewSeedGetter(testSeed)
			configGetter := test.NewConfigGetter(tt.config)

			clusterValidator := validator{
				client:                    seedClient,
				seedGetter:                seedGetter,
				configGetter:              configGetter,
				disableProviderValidation: true,
			}

			gen := baseCluster()
			cluster := gen.Build()
			cluster.Spec.UseEventRateLimitAdmissionPlugin = tt.useEventRateLimitPlugin
			cluster.Spec.EventRateLimitConfig = tt.userConfig

			ctx := context.Background()
			var err error

			switch tt.op {
			case admissionv1.Create:
				_, err = clusterValidator.ValidateCreate(ctx, &cluster)
			case admissionv1.Update:
				oldGen := baseCluster()
				oldCluster := oldGen.Build()
				if tt.oldUseEventRateLimitPlugin != nil {
					oldCluster.Spec.UseEventRateLimitAdmissionPlugin = *tt.oldUseEventRateLimitPlugin
				}
				_, err = clusterValidator.ValidateUpdate(ctx, &oldCluster, &cluster)
			}

			allowed := err == nil

			if allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t: %v", allowed, tt.wantAllowed, err)
			}
		})
	}
}
