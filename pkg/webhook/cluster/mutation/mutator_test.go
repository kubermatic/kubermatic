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

package mutation

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-test/deep"
	jsonpatch "gomodules.xyz/jsonpatch/v2"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/api/v3/pkg/semver"
	"k8c.io/kubermatic/v3/pkg/cni"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v3/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	config      = kubermaticv1.KubermaticConfiguration{}
	datacenters = []ctrlruntimeclient.Object{
		&kubermaticv1.Datacenter{
			ObjectMeta: metav1.ObjectMeta{
				Name: "openstack-dc",
			},
			Spec: kubermaticv1.DatacenterSpec{
				Provider: kubermaticv1.DatacenterProviderSpec{
					OpenStack: &kubermaticv1.DatacenterSpecOpenStack{},
				},
			},
		},

		&kubermaticv1.Datacenter{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kubevirt-dc",
			},
			Spec: kubermaticv1.DatacenterSpec{
				Provider: kubermaticv1.DatacenterProviderSpec{
					KubeVirt: &kubermaticv1.DatacenterSpecKubeVirt{},
				},
			},
		},

		&kubermaticv1.Datacenter{
			ObjectMeta: metav1.ObjectMeta{
				Name: "hetzner-dc",
			},
			Spec: kubermaticv1.DatacenterSpec{
				Provider: kubermaticv1.DatacenterProviderSpec{
					Hetzner: &kubermaticv1.DatacenterSpecHetzner{},
				},
			},
		},
	}

	defaultingTemplateName = "my-default-template"

	// defaultPatches are the patches that occur in every mutation because of the
	// inherit defaulting done for the KubermaticConfiguration and Seed. They are
	// collected here for brevity sake.
	defaultPatches = []jsonpatch.JsonPatchOperation{
		jsonpatch.NewOperation("replace", "/spec/exposeStrategy", string(defaulting.DefaultExposeStrategy)),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/etcd/clusterSize", float64(defaulting.DefaultEtcdClusterSize)),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/etcd/diskSize", defaulting.DefaultEtcdVolumeSize),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/replicas", float64(defaulting.DefaultKubernetesApiserverReplicas)),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/nodePortRange", resources.DefaultNodePortRange),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/controllerManager/replicas", float64(defaulting.DefaultSeedControllerManagerReplicas)),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/replicas", float64(defaulting.DefaultKubernetesSchedulerReplicas)),
		jsonpatch.NewOperation("add", "/spec/enableOperatingSystemManager", true),
		jsonpatch.NewOperation("add", "/spec/kubernetesDashboard", map[string]interface{}{"enabled": true}),
	}

	defaultNetworkingPatchesWithoutProxyMode = []jsonpatch.JsonPatchOperation{
		jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipFamily", string(kubermaticv1.IPFamilyIPv4)),
		jsonpatch.NewOperation("add", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{resources.DefaultClusterServicesCIDRIPv4}),
		jsonpatch.NewOperation("add", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{resources.DefaultClusterPodsCIDRIPv4}),
		jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv4", float64(resources.DefaultNodeCIDRMaskSizeIPv4)),
		jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "cluster.local"),
		jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", resources.DefaultNodeLocalDNSCacheEnabled),
	}
	defaultNetworkingPatches = append(
		defaultNetworkingPatchesWithoutProxyMode,
		jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", "ipvs"),
		jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipvs", map[string]interface{}{"strictArp": true}),
	)
	defaultNetworkingPatchesIptablesProxyMode = append(
		defaultNetworkingPatchesWithoutProxyMode,
		jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", "iptables"),
	)
)

func TestMutator(t *testing.T) {
	oneGB := resource.MustParse("1G")
	tests := []struct {
		name                   string
		oldCluster             *kubermaticv1.Cluster
		newCluster             *kubermaticv1.Cluster
		defaultClusterTemplate *kubermaticv1.ClusterTemplate
		wantAllowed            bool
		wantPatches            []jsonpatch.JsonPatchOperation
	}{
		{
			name: "Create cluster sets default component settings",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyIPv4,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPTablesProxyMode,
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
			}.Do(),
			defaultClusterTemplate: &kubermaticv1.ClusterTemplate{
				Spec: kubermaticv1.ClusterSpec{
					CNIPlugin: &kubermaticv1.CNIPluginSettings{
						Type: kubermaticv1.CNIPluginTypeCilium,
					},
					ComponentsOverride: kubermaticv1.ComponentSettings{
						Apiserver: kubermaticv1.APIServerSettings{
							DeploymentSettings: kubermaticv1.DeploymentSettings{
								Replicas: pointer.Int32(2),
								Resources: &corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										"memory": resource.MustParse("500M"),
									},
								},
								Tolerations: []corev1.Toleration{
									{
										Key:      "test-no-schedule",
										Operator: corev1.TolerationOpExists,
										Effect:   corev1.TaintEffectPreferNoSchedule,
									},
								},
							},
							EndpointReconcilingDisabled: pointer.Bool(true),
							NodePortRange:               "30000-32768",
						},
						ControllerManager: kubermaticv1.ControllerSettings{
							DeploymentSettings: kubermaticv1.DeploymentSettings{
								Replicas: pointer.Int32(2),
								Resources: &corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										"memory": resource.MustParse("500M"),
									},
								},
								Tolerations: []corev1.Toleration{
									{
										Key:      "test-no-schedule",
										Operator: corev1.TolerationOpExists,
										Effect:   corev1.TaintEffectPreferNoSchedule,
									},
								},
							},
							LeaderElectionSettings: kubermaticv1.LeaderElectionSettings{
								LeaseDurationSeconds: pointer.Int32(10),
								RenewDeadlineSeconds: pointer.Int32(5),
								RetryPeriodSeconds:   pointer.Int32(2),
							},
						},
						Scheduler: kubermaticv1.ControllerSettings{
							DeploymentSettings: kubermaticv1.DeploymentSettings{
								Replicas: pointer.Int32(2),
								Resources: &corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										"memory": resource.MustParse("500M"),
									},
								},
								Tolerations: []corev1.Toleration{
									{
										Key:      "test-no-schedule",
										Operator: corev1.TolerationOpExists,
										Effect:   corev1.TaintEffectPreferNoSchedule,
									},
								},
							},
							LeaderElectionSettings: kubermaticv1.LeaderElectionSettings{
								LeaseDurationSeconds: pointer.Int32(10),
								RenewDeadlineSeconds: pointer.Int32(5),
								RetryPeriodSeconds:   pointer.Int32(2),
							},
						},
						Etcd: kubermaticv1.EtcdStatefulSetSettings{
							ClusterSize:  pointer.Int32(7),
							StorageClass: "fast-storage",
							DiskSize:     &oneGB,
							Resources: &corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									"memory": resource.MustParse("500M"),
								},
							},
						},
						Prometheus: &kubermaticv1.StatefulSetSettings{
							Resources: &corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									"memory": resource.MustParse("500M"),
								},
							},
						},
					},
				},
			},

			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/nodePortRange", "30000-32768"),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/replicas", float64(2)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/resources", map[string]interface{}{"requests": map[string]interface{}{"memory": "500M"}}),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/tolerations", []interface{}{map[string]interface{}{"effect": "PreferNoSchedule", "key": "test-no-schedule", "operator": "Exists"}}),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/endpointReconcilingDisabled", true),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/controllerManager/replicas", float64(2)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/controllerManager/resources", map[string]interface{}{"requests": map[string]interface{}{"memory": "500M"}}),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/controllerManager/tolerations", []interface{}{map[string]interface{}{"effect": "PreferNoSchedule", "key": "test-no-schedule", "operator": "Exists"}}),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/controllerManager/leaderElection/leaseDurationSeconds", float64(10)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/controllerManager/leaderElection/renewDeadlineSeconds", float64(5)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/controllerManager/leaderElection/retryPeriodSeconds", float64(2)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/tolerations", []interface{}{map[string]interface{}{"effect": "PreferNoSchedule", "key": "test-no-schedule", "operator": "Exists"}}),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/leaderElection/renewDeadlineSeconds", float64(5)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/leaderElection/retryPeriodSeconds", float64(2)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/leaderElection/leaseDurationSeconds", float64(10)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/replicas", float64(2)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/resources", float64(5)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/resources", map[string]interface{}{"requests": map[string]interface{}{"memory": "500M"}}),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/etcd/clusterSize", float64(7)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/etcd/storageClass", "fast-storage"),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/etcd/diskSize", "1G"),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/etcd/resources", map[string]interface{}{"requests": map[string]interface{}{"memory": "500M"}}),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/prometheus", map[string]interface{}{"resources": map[string]interface{}{"requests": map[string]interface{}{"memory": "500M"}}}),
				jsonpatch.NewOperation("add", "/spec/features/apiserverNetworkPolicy", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
				jsonpatch.NewOperation("add", "/spec/enableOperatingSystemManager", true),
				jsonpatch.NewOperation("add", "/spec/kubernetesDashboard", map[string]interface{}{"enabled": true}),
				jsonpatch.NewOperation("replace", "/spec/exposeStrategy", string(defaulting.DefaultExposeStrategy)),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderOpenStack)),
			},
		},
		{
			name: "Create cluster sets default cni settings",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
			}.Do(),
			defaultClusterTemplate: &kubermaticv1.ClusterTemplate{
				Spec: kubermaticv1.ClusterSpec{
					CNIPlugin: &kubermaticv1.CNIPluginSettings{
						Type: kubermaticv1.CNIPluginTypeCilium,
					},
					ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
						IPFamily:                 kubermaticv1.IPFamilyIPv4,
						Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
						Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
						DNSDomain:                "example.local",
						ProxyMode:                resources.EBPFProxyMode,
						NodeLocalDNSCacheEnabled: pointer.Bool(true),
					},
				},
			},

			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/cniPlugin", map[string]interface{}{
					"type":    "cilium",
					"version": cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCilium),
				}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipFamily", string(kubermaticv1.IPFamilyIPv4)),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{"10.240.32.0/20"}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{"10.241.0.0/16"}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv4", float64(24)),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "example.local"),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", resources.EBPFProxyMode),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", true),
				jsonpatch.NewOperation("add", "/spec/features/apiserverNetworkPolicy", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderOpenStack)),
			),
		},
		{
			name: "Create cluster success",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyIPv4,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPVSProxyMode,
					IPVS:                     &kubermaticv1.IPVSConfiguration{StrictArp: pointer.Bool(true)},
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
			),
		},
		{
			name: "Default features",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyIPv4,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPTablesProxyMode,
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/features/apiserverNetworkPolicy", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
			),
		},
		{
			name: "Default the cloud provider name",
			oldCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "hetzner-dc",
					Hetzner:        &kubermaticv1.HetznerCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "hetzner-dc",
					Hetzner:        &kubermaticv1.HetznerCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatchesIptablesProxyMode...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderHetzner)),
			),
		},
		{
			name: "Fix bad cloud provider name",
			oldCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderHetzner,
					DatacenterName: "hetzner-dc",
					Hetzner:        &kubermaticv1.HetznerCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "hetzner-dc",
					Hetzner:        &kubermaticv1.HetznerCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatchesIptablesProxyMode...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderHetzner)),
			),
		},
		{
			name: "Default CNI plugin configuration added",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyIPv4,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPVSProxyMode,
					IPVS:                     &kubermaticv1.IPVSConfiguration{StrictArp: pointer.Bool(true)},
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/cniPlugin", map[string]interface{}{
					"type":    string(kubermaticv1.CNIPluginTypeCanal),
					"version": cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
				}),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
			),
		},
		{
			name: "CNI plugin version added if not set on existing cluster",
			oldCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPTablesProxyMode,
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyIPv4,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPTablesProxyMode,
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/cniPlugin", map[string]interface{}{
					"type":    string(kubermaticv1.CNIPluginTypeCanal),
					"version": cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
				}),
			),
		},
		{
			name: "Unsupported CNI plugin version bump on k8s version upgrade",
			oldCluster: rawClusterGen{
				Name:    "foo",
				Version: *semver.NewSemverOrDie("1.23"),
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPTablesProxyMode,
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: cni.CanalCNILastUnspecifiedVersion,
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name:    "foo",
				Version: *semver.NewSemverOrDie("1.24"),
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyIPv4,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPTablesProxyMode,
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: cni.CanalCNILastUnspecifiedVersion,
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("replace", "/spec/cniPlugin/version", cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal)),
			),
		},
		{
			name: "CNI plugin version bump to v3.22 on k8s version upgrade to 1.23",
			oldCluster: rawClusterGen{
				Name:    "foo",
				Version: *semver.NewSemverOrDie("1.22"),
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPTablesProxyMode,
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.21",
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name:    "foo",
				Version: *semver.NewSemverOrDie("1.23"),
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily:                 kubermaticv1.IPFamilyIPv4,
					Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
					Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
					NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
					DNSDomain:                "example.local",
					ProxyMode:                resources.IPTablesProxyMode,
					NodeLocalDNSCacheEnabled: pointer.Bool(true),
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.21",
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("replace", "/spec/cniPlugin/version", "v3.22"),
			),
		},
		{
			name: "Default network configuration for any cloud provider except KubeVirt",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatches...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderOpenStack)),
				jsonpatch.NewOperation("add", "/spec/features/externalCloudProvider", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
			),
		},
		{
			name: "Default configuration for KubeVirt cloud provider",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "kubevirt-dc",
					KubeVirt:       &kubermaticv1.KubeVirtCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipFamily", string(kubermaticv1.IPFamilyIPv4)),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{"10.241.0.0/20"}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{"172.26.0.0/16"}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", "ipvs"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipvs", map[string]interface{}{"strictArp": true}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "cluster.local"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", true),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv4", float64(24)),
				jsonpatch.NewOperation("add", "/spec/features/externalCloudProvider", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderKubeVirt)),
			),
		},
		{
			name: "Default network configuration with non-default IPVS Settings",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					ProxyMode: resources.IPVSProxyMode,
					IPVS: &kubermaticv1.IPVSConfiguration{
						StrictArp: pointer.Bool(false),
					},
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatchesWithoutProxyMode...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderOpenStack)),
				jsonpatch.NewOperation("add", "/spec/features/externalCloudProvider", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
			),
		},
		{
			name: "Default network configuration with iptables proxy mode",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					ProxyMode: resources.IPTablesProxyMode,
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatchesWithoutProxyMode...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderOpenStack)),
				jsonpatch.NewOperation("add", "/spec/features/externalCloudProvider", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
			),
		},
		{
			name: "Default dual-stack network configuration",
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.19",
				},
				NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
					IPFamily: kubermaticv1.IPFamilyDualStack,
				},
				Features: map[string]bool{
					kubermaticv1.ApiserverNetworkPolicy:    true,
					kubermaticv1.KubeSystemNetworkPolicies: true,
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{resources.DefaultClusterServicesCIDRIPv4, resources.DefaultClusterServicesCIDRIPv6}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{resources.DefaultClusterPodsCIDRIPv4, resources.DefaultClusterPodsCIDRIPv6}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv4", float64(resources.DefaultNodeCIDRMaskSizeIPv4)),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv6", float64(resources.DefaultNodeCIDRMaskSizeIPv6)),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "cluster.local"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", resources.DefaultNodeLocalDNSCacheEnabled),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", resources.IPVSProxyMode),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipvs", map[string]interface{}{"strictArp": true}),
				jsonpatch.NewOperation("add", "/spec/features/externalCloudProvider", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.CloudProviderOpenStack)),
			),
		},
		{
			name: "Update OpenStack cluster to enable the CCM/CSI migration",
			oldCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: false,
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.20",
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.20",
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatches...),
				jsonpatch.NewOperation("add", "/metadata/annotations", map[string]interface{}{"ccm-migration.k8c.io/migration-needed": "", "csi-migration.k8c.io/migration-needed": ""}),
				jsonpatch.NewOperation("add", "/spec/cloud/openstack/useOctavia", true),
				jsonpatch.NewOperation("add", "/spec/features/ccmClusterName", true),
			),
		},
		{
			name: "Update OpenStack cluster with enabled CCM/CSI migration",
			oldCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.20",
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderOpenStack,
					DatacenterName: "openstack-dc",
					OpenStack:      &kubermaticv1.OpenStackCloudSpec{},
				},
				ExternalCloudProvider: true,
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.20",
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(defaultPatches, defaultNetworkingPatches...),
		},
		{
			name: "Update non-OpenStack cluster to enable CCM/CSI migration",
			oldCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderHetzner,
					DatacenterName: "hetzner-dc",
					Hetzner:        &kubermaticv1.HetznerCloudSpec{},
				},
				ExternalCloudProvider: false,
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.20",
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderHetzner,
					DatacenterName: "hetzner-dc",
					Hetzner:        &kubermaticv1.HetznerCloudSpec{},
				},
				ExternalCloudProvider: true,
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeCanal,
					Version: "v3.20",
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(defaultPatches, defaultNetworkingPatchesIptablesProxyMode...),
		},
		{
			name: "Update cluster with CNI none",
			oldCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderHetzner,
					DatacenterName: "hetzner-dc",
					Hetzner:        &kubermaticv1.HetznerCloudSpec{},
				},
				ExternalCloudProvider: false,
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeNone,
					Version: "",
				},
			}.Do(),
			newCluster: rawClusterGen{
				Name: "foo",
				CloudSpec: kubermaticv1.CloudSpec{
					ProviderName:   kubermaticv1.CloudProviderHetzner,
					DatacenterName: "hetzner-dc",
					Hetzner:        &kubermaticv1.HetznerCloudSpec{},
				},
				ExternalCloudProvider: true,
				CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
					Type:    kubermaticv1.CNIPluginTypeNone,
					Version: "",
				},
			}.Do(),
			wantAllowed: true,
			wantPatches: append(defaultPatches, defaultNetworkingPatchesIptablesProxyMode...),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfig := config.DeepCopy()

			builder := fake.NewClientBuilder()
			if tt.defaultClusterTemplate != nil {
				testConfig.Spec.UserCluster.DefaultTemplate = defaultingTemplateName
				tt.defaultClusterTemplate.Name = defaultingTemplateName

				builder.WithObjects(tt.defaultClusterTemplate)
			}

			dummySeedClient := builder.WithObjects(datacenters...).Build()

			// this getter, as do all KubermaticConfigurationGetters, performs defaulting on the config
			configGetter, err := kubernetes.StaticKubermaticConfigurationGetterFactory(testConfig)
			if err != nil {
				t.Fatalf("Failed to create KubermaticConfigurationGetter: %v", err)
			}

			datacenterGetter, err := kubernetes.DatacenterGetterFactory(dummySeedClient)
			if err != nil {
				t.Fatalf("Failed to create DatacenterGetter: %v", err)
			}

			mutator := NewMutator(dummySeedClient, configGetter, datacenterGetter, nil)
			mutator.disableProviderMutation = true

			// marshal this before running the mutator, as it might be mutating the same memory
			original, err := json.Marshal(tt.newCluster)
			if err != nil {
				t.Fatalf("Failed to encode new cluster as JSON: %v", err)
			}

			mutatedCluster, mutateErr := mutator.Mutate(context.Background(), tt.oldCluster, tt.newCluster)
			if tt.wantAllowed && mutateErr != nil {
				t.Fatalf("Request should have succeeded, but failed: %v", mutateErr)
			}
			if !tt.wantAllowed && mutateErr == nil {
				t.Fatalf("Request should have failed, but succeeded")
			}

			mutated, err := json.Marshal(mutatedCluster)
			if err != nil {
				t.Fatalf("Failed to encode mutated cluster as JSON: %v", err)
			}

			patches, err := jsonpatch.CreatePatch(original, mutated)
			if err != nil {
				t.Fatalf("Failed to create patches: %v", err)
			}

			a := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range patches {
				a[p.Path] = p
			}
			w := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range tt.wantPatches {
				w[p.Path] = p
			}
			if diff := deep.Equal(a, w); len(diff) > 0 {
				t.Errorf("Diff found between wanted and actual patches: %+v", diff)
			}
		})
	}
}

type rawClusterGen struct {
	Name                  string
	Version               semver.Semver
	CloudSpec             kubermaticv1.CloudSpec
	CNIPluginSpec         *kubermaticv1.CNIPluginSettings
	ExternalCloudProvider bool
	NetworkConfig         kubermaticv1.ClusterNetworkingConfig
	Features              map[string]bool
}

func (r rawClusterGen) Do() *kubermaticv1.Cluster {
	c := kubermaticv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Version:        r.Version,
			Cloud:          r.CloudSpec,
			ClusterNetwork: r.NetworkConfig,
			CNIPlugin:      r.CNIPluginSpec,
		},
		Status: kubermaticv1.ClusterStatus{
			Versions: kubermaticv1.ClusterVersionsStatus{
				ControlPlane: r.Version,
			},
		},
	}

	// Only set this when enabled, a `false` value in r.ExternalCloudProvider does not
	// mean we should _disable_ the CCM explicitly, just that we do not set the feature
	// at all.
	if r.ExternalCloudProvider {
		if r.Features == nil {
			r.Features = map[string]bool{}
		}

		r.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] = true
	}

	for k, v := range r.Features {
		if c.Spec.Features == nil {
			c.Spec.Features = map[string]bool{}
		}

		c.Spec.Features[k] = v
	}

	return &c
}
