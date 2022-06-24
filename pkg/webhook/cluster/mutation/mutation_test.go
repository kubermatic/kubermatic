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

package mutation

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-test/deep"
	jsonpatch "gomodules.xyz/jsonpatch/v2"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/version/cni"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	testScheme = runtime.NewScheme()
	config     = kubermaticv1.KubermaticConfiguration{}
	seed       = kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-seed",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				"openstack-dc": {
					Spec: kubermaticv1.DatacenterSpec{
						Openstack: &kubermaticv1.DatacenterSpecOpenstack{},
					},
				},
				"hetzner-dc": {
					Spec: kubermaticv1.DatacenterSpec{
						Hetzner: &kubermaticv1.DatacenterSpecHetzner{},
					},
				},
				"kubevirt-dc": {
					Spec: kubermaticv1.DatacenterSpec{
						Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{},
					},
				},
			},
		},
	}
	defaultingTemplateName = "my-default-template"

	// defaultPatches are the patches that occur in every mutation because of the
	// inherit defaulting done for the KubermaticConfiguration and Seed. They are
	// collected here for brevity sake.
	defaultPatches = []jsonpatch.JsonPatchOperation{
		jsonpatch.NewOperation("replace", "/spec/exposeStrategy", string(defaults.DefaultExposeStrategy)),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/etcd/clusterSize", float64(kubermaticv1.DefaultEtcdClusterSize)),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/etcd/diskSize", defaults.DefaultEtcdVolumeSize),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/replicas", float64(defaults.DefaultAPIServerReplicas)),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/nodePortRange", defaults.DefaultNodePortRange),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/controllerManager/replicas", float64(defaults.DefaultControllerManagerReplicas)),
		jsonpatch.NewOperation("add", "/spec/componentsOverride/scheduler/replicas", float64(defaults.DefaultSchedulerReplicas)),
		jsonpatch.NewOperation("add", "/spec/kubernetesDashboard/enabled", true),
	}

	defaultNetworkingPatchesWithoutProxyMode = []jsonpatch.JsonPatchOperation{
		jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipFamily", string(kubermaticv1.IPFamilyIPv4)),
		jsonpatch.NewOperation("replace", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{resources.DefaultClusterServicesCIDRIPv4}),
		jsonpatch.NewOperation("replace", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{resources.DefaultClusterPodsCIDRIPv4}),
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

func init() {
	_ = kubermaticv1.AddToScheme(testScheme)
}

func TestHandle(t *testing.T) {
	oneGB := resource.MustParse("1G")
	tests := []struct {
		name                   string
		req                    webhook.AdmissionRequest
		defaultClusterTemplate *kubermaticv1.ClusterTemplate
		wantAllowed            bool
		wantPatches            []jsonpatch.JsonPatchOperation
	}{
		{
			name: "Create cluster sets default component settings",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
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
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
						}.Do(),
					},
				},
			},
			defaultClusterTemplate: &kubermaticv1.ClusterTemplate{
				Spec: kubermaticv1.ClusterSpec{
					CNIPlugin: &kubermaticv1.CNIPluginSettings{
						Type: kubermaticv1.CNIPluginTypeCilium,
					},
					ComponentsOverride: kubermaticv1.ComponentSettings{
						Apiserver: kubermaticv1.APIServerSettings{
							DeploymentSettings: kubermaticv1.DeploymentSettings{
								Replicas: pointer.Int32Ptr(2),
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
							EndpointReconcilingDisabled: pointer.BoolPtr(true),
							NodePortRange:               "30000-32768",
						},
						ControllerManager: kubermaticv1.ControllerSettings{
							DeploymentSettings: kubermaticv1.DeploymentSettings{
								Replicas: pointer.Int32Ptr(2),
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
								LeaseDurationSeconds: pointer.Int32Ptr(10),
								RenewDeadlineSeconds: pointer.Int32Ptr(5),
								RetryPeriodSeconds:   pointer.Int32Ptr(2),
							},
						},
						Scheduler: kubermaticv1.ControllerSettings{
							DeploymentSettings: kubermaticv1.DeploymentSettings{
								Replicas: pointer.Int32Ptr(2),
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
								LeaseDurationSeconds: pointer.Int32Ptr(10),
								RenewDeadlineSeconds: pointer.Int32Ptr(5),
								RetryPeriodSeconds:   pointer.Int32Ptr(2),
							},
						},
						Etcd: kubermaticv1.EtcdStatefulSetSettings{
							ClusterSize:  pointer.Int32Ptr(7),
							StorageClass: "fast-storage",
							DiskSize:     &oneGB,
							Resources: &corev1.ResourceRequirements{
								Requests: map[corev1.ResourceName]resource.Quantity{
									"memory": resource.MustParse("500M"),
								},
							},
						},
						Prometheus: kubermaticv1.StatefulSetSettings{
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
				jsonpatch.NewOperation("add", "/spec/componentsOverride/prometheus/resources", map[string]interface{}{"requests": map[string]interface{}{"memory": "500M"}}),
				jsonpatch.NewOperation("add", "/spec/features/apiserverNetworkPolicy", true),
				jsonpatch.NewOperation("add", "/spec/kubernetesDashboard/enabled", true),
				jsonpatch.NewOperation("replace", "/spec/exposeStrategy", string(defaults.DefaultExposeStrategy)),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.OpenstackCloudProvider)),
			},
		},
		{
			name: "Create cluster sets default cni settings",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
						}.Do(),
					},
				},
			},
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
						NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
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
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{"10.240.32.0/20"}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{"10.241.0.0/16"}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv4", float64(24)),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "example.local"),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", resources.EBPFProxyMode),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", true),
				jsonpatch.NewOperation("add", "/spec/features/apiserverNetworkPolicy", true),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.OpenstackCloudProvider)),
			),
		},
		{
			name: "Create cluster success",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
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
								IPVS:                     &kubermaticv1.IPVSConfiguration{StrictArp: pointer.BoolPtr(true)},
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
							Features: map[string]bool{
								kubermaticv1.ApiserverNetworkPolicy:    true,
								kubermaticv1.KubeSystemNetworkPolicies: true,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: defaultPatches,
		},
		{
			name: "Default features",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
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
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/features/apiserverNetworkPolicy", true),
			),
		},
		{
			name: "Default the cloud provider name",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
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
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
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
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatchesIptablesProxyMode...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.HetznerCloudProvider)),
			),
		},
		{
			name: "Fix bad cloud provider name",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "hetzner-dc",
								Hetzner:        &kubermaticv1.HetznerCloudSpec{},
							},
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.19",
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.HetznerCloudProvider),
								DatacenterName: "hetzner-dc",
								Hetzner:        &kubermaticv1.HetznerCloudSpec{},
							},
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.19",
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatchesIptablesProxyMode...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.HetznerCloudProvider)),
			),
		},
		{
			name: "Default CNI plugin configuration added",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								IPFamily:                 kubermaticv1.IPFamilyIPv4,
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPVSProxyMode,
								IPVS:                     &kubermaticv1.IPVSConfiguration{StrictArp: pointer.BoolPtr(true)},
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
							Features: map[string]bool{
								kubermaticv1.ApiserverNetworkPolicy:    true,
								kubermaticv1.KubeSystemNetworkPolicies: true,
							},
						}.Do(),
					},
				},
			},
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
			name: "CNI plugin version added if not set on existing cluster",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								IPFamily:                 kubermaticv1.IPFamilyIPv4,
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
							Features: map[string]bool{
								kubermaticv1.ApiserverNetworkPolicy:    true,
								kubermaticv1.KubeSystemNetworkPolicies: true,
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
							Features: map[string]bool{
								kubermaticv1.ApiserverNetworkPolicy:    true,
								kubermaticv1.KubeSystemNetworkPolicies: true,
							},
						}.Do(),
					},
				},
			},
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
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name:    "foo",
							Version: *semver.NewSemverOrDie("1.22"),
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								IPFamily:                 kubermaticv1.IPFamilyIPv4,
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
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
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name:    "foo",
							Version: *semver.NewSemverOrDie("1.21"),
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
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
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("replace", "/spec/cniPlugin/version", cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal)),
			),
		},
		{
			name: "CNI plugin version bump to v3.22 on k8s version upgrade to 1.23",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name:    "foo",
							Version: *semver.NewSemverOrDie("1.23"),
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								IPFamily:                 kubermaticv1.IPFamilyIPv4,
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								NodeCIDRMaskSizeIPv4:     pointer.Int32(24),
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
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
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name:    "foo",
							Version: *semver.NewSemverOrDie("1.22"),
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
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
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("replace", "/spec/cniPlugin/version", "v3.22"),
			),
		},
		{
			name: "Default network configuration for any cloud provider except KubeVirt",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
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
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatches...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.OpenstackCloudProvider)),
			),
		},
		{
			name: "Default configuration for KubeVirt cloud provider",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								DatacenterName: "kubevirt-dc",
								Kubevirt:       &kubermaticv1.KubevirtCloudSpec{},
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
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipFamily", string(kubermaticv1.IPFamilyIPv4)),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{"10.241.0.0/20"}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{"172.26.0.0/16"}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", "ipvs"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipvs", map[string]interface{}{"strictArp": true}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "cluster.local"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", true),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv4", float64(24)),
				jsonpatch.NewOperation("replace", "/spec/features/externalCloudProvider", true),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.KubevirtCloudProvider)),
			),
		},
		{
			name: "Default network configuration with non-default IPVS Settings",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.19",
							},
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								ProxyMode: resources.IPVSProxyMode,
								IPVS: &kubermaticv1.IPVSConfiguration{
									StrictArp: pointer.BoolPtr(false),
								},
							},
							Features: map[string]bool{
								kubermaticv1.ApiserverNetworkPolicy:    true,
								kubermaticv1.KubeSystemNetworkPolicies: true,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatchesWithoutProxyMode...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.OpenstackCloudProvider)),
			),
		},
		{
			name: "Default network configuration with iptables proxy mode",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
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
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatchesWithoutProxyMode...),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.OpenstackCloudProvider)),
			),
		},
		{
			name: "Default dual-stack network configuration",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Create,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
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
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				defaultPatches,
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{resources.DefaultClusterServicesCIDRIPv4, resources.DefaultClusterServicesCIDRIPv6}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{resources.DefaultClusterPodsCIDRIPv4, resources.DefaultClusterPodsCIDRIPv6}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv4", float64(resources.DefaultNodeCIDRMaskSizeIPv4)),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeCidrMaskSizeIPv6", float64(resources.DefaultNodeCIDRMaskSizeIPv6)),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "cluster.local"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", resources.DefaultNodeLocalDNSCacheEnabled),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", resources.IPVSProxyMode),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipvs", map[string]interface{}{"strictArp": true}),
				jsonpatch.NewOperation("replace", "/spec/cloud/providerName", string(kubermaticv1.OpenstackCloudProvider)),
			),
		},
		{
			name: "Delete cluster success",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Delete,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
						}.Do(),
					},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Update OpenStack cluster to enable the CCM/CSI migration",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.20",
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: false,
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.20",
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(
				append(defaultPatches, defaultNetworkingPatches...),
				jsonpatch.NewOperation("add", "/metadata/annotations", map[string]interface{}{"ccm-migration.k8c.io/migration-needed": "", "csi-migration.k8c.io/migration-needed": ""}),
				jsonpatch.NewOperation("add", "/spec/cloud/openstack/useOctavia", true),
			),
		},
		{
			name: "Update OpenStack cluster with enabled CCM/CSI migration",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.20",
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.OpenstackCloudProvider),
								DatacenterName: "openstack-dc",
								Openstack:      &kubermaticv1.OpenstackCloudSpec{},
							},
							ExternalCloudProvider: true,
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.20",
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(defaultPatches, defaultNetworkingPatches...),
		},
		{
			name: "Update non-OpenStack cluster to enable CCM/CSI migration",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.HetznerCloudProvider),
								DatacenterName: "hetzner-dc",
								Hetzner:        &kubermaticv1.HetznerCloudSpec{},
							},
							ExternalCloudProvider: true,
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.20",
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.HetznerCloudProvider),
								DatacenterName: "hetzner-dc",
								Hetzner:        &kubermaticv1.HetznerCloudSpec{},
							},
							ExternalCloudProvider: false,
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.20",
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(defaultPatches, defaultNetworkingPatchesIptablesProxyMode...),
		},
		{
			name: "Update cluster with CNI none",
			req: webhook.AdmissionRequest{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Operation: admissionv1.Update,
					RequestKind: &metav1.GroupVersionKind{
						Group:   kubermaticv1.GroupName,
						Version: kubermaticv1.GroupVersion,
						Kind:    "Cluster",
					},
					Name: "foo",
					Object: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.HetznerCloudProvider),
								DatacenterName: "hetzner-dc",
								Hetzner:        &kubermaticv1.HetznerCloudSpec{},
							},
							ExternalCloudProvider: true,
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeNone,
								Version: "",
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name: "foo",
							CloudSpec: kubermaticv1.CloudSpec{
								ProviderName:   string(kubermaticv1.HetznerCloudProvider),
								DatacenterName: "hetzner-dc",
								Hetzner:        &kubermaticv1.HetznerCloudSpec{},
							},
							ExternalCloudProvider: false,
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeNone,
								Version: "",
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: append(defaultPatches, defaultNetworkingPatchesIptablesProxyMode...),
		},
	}
	for _, tt := range tests {
		d, err := admission.NewDecoder(testScheme)
		if err != nil {
			t.Fatalf("error occurred while creating decoder: %v", err)
		}

		t.Run(tt.name, func(t *testing.T) {
			testSeed := seed.DeepCopy()

			builder := fake.NewClientBuilder()
			if tt.defaultClusterTemplate != nil {
				testSeed.Spec.DefaultClusterTemplate = defaultingTemplateName

				tt.defaultClusterTemplate.Labels = map[string]string{"scope": kubermaticv1.SeedTemplateScope}
				tt.defaultClusterTemplate.Name = defaultingTemplateName
				tt.defaultClusterTemplate.Namespace = testSeed.Namespace

				builder.WithObjects(tt.defaultClusterTemplate)
			}
			dummySeedClient := builder.Build()

			// this getter, as do all KubermaticConfigurationGetters, performs defaulting on the config
			configGetter, err := provider.StaticKubermaticConfigurationGetterFactory(&config)
			if err != nil {
				t.Fatalf("Failed to create KubermaticConfigurationGetter: %v", err)
			}

			handler := AdmissionHandler{
				log:                     logr.Discard(),
				decoder:                 d,
				seedGetter:              test.NewSeedGetter(testSeed),
				configGetter:            configGetter,
				client:                  dummySeedClient,
				disableProviderMutation: true,
			}
			res := handler.Handle(context.Background(), tt.req)
			if res.AdmissionResponse.Result != nil && res.AdmissionResponse.Result.Code == http.StatusInternalServerError {
				t.Fatalf("Request failed: %v", res.AdmissionResponse.Result.Message)
			}

			if res.Allowed != tt.wantAllowed {
				t.Logf("Response: %#v", res)
				t.Fatalf("Allowed %t, but wanted %t", res.Allowed, tt.wantAllowed)
			}

			a := map[string]jsonpatch.JsonPatchOperation{}
			for _, p := range res.Patches {
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

func (r rawClusterGen) Do() []byte {
	c := kubermaticv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8c.io/v1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: r.Version,
			Features: map[string]bool{
				"externalCloudProvider": r.ExternalCloudProvider,
			},
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

	for k, v := range r.Features {
		c.Spec.Features[k] = v
	}
	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, testScheme, testScheme, json.SerializerOptions{Pretty: true})
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&c, buff)
	return buff.Bytes()
}
