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
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-test/deep"
	jsonpatch "gomodules.xyz/jsonpatch/v2"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	testScheme = runtime.NewScheme()
)

func init() {
	_ = kubermaticv1.AddToScheme(testScheme)
}

func TestHandle(t *testing.T) {
	oneGB := resource.MustParse("1G")
	tests := []struct {
		name              string
		req               webhook.AdmissionRequest
		componentSettings kubermaticv1.ComponentSettings
		wantAllowed       bool
		wantPatches       []jsonpatch.JsonPatchOperation
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
							Name:      "foo",
							CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}},
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.19",
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
						}.Do(),
					},
				},
			},
			componentSettings: kubermaticv1.ComponentSettings{
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
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/nodePortRange", "30000-32768"),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/replicas", float64(2)),
				jsonpatch.NewOperation("add", "/spec/componentsOverride/apiserver/resources", map[string]interface{}{"requests": map[string]interface{}{"memory": "500M"}}),
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
			},
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
							Name:      "foo",
							CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}},
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.19",
							},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								IPVS:                     &kubermaticv1.IPVSConfiguration{StrictArp: pointer.BoolPtr(true)},
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{},
		},
		{
			name: "Default CNI plugin annotation added",
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
							Name:                  "foo",
							CloudSpec:             kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}},
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain:                "example.local",
								ProxyMode:                resources.IPTablesProxyMode,
								IPVS:                     &kubermaticv1.IPVSConfiguration{StrictArp: pointer.BoolPtr(true)},
								NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("add", "/spec/cniPlugin", map[string]interface{}{
					"type":    "canal",
					"version": "v3.19",
				}),
			},
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
							Name:      "foo",
							CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}},
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.19",
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{"10.240.16.0/20"}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{"172.25.0.0/16"}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", "ipvs"),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "cluster.local"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", true),
			},
		},
		{
			name: "Default network configuration for KubeVirt cloud provider",
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
							Name:      "foo",
							CloudSpec: kubermaticv1.CloudSpec{Kubevirt: &kubermaticv1.KubevirtCloudSpec{}},
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.19",
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{"10.241.0.0/20"}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{"172.26.0.0/16"}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", "ipvs"),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "cluster.local"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", true),
			},
		},
		{
			name: "Default network configuration with IPVS Settings",
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
							Name:      "foo",
							CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}},
							CNIPluginSpec: &kubermaticv1.CNIPluginSettings{
								Type:    kubermaticv1.CNIPluginTypeCanal,
								Version: "v3.19",
							},
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								IPVS: &kubermaticv1.IPVSConfiguration{},
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/services/cidrBlocks", []interface{}{"10.240.16.0/20"}),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/pods/cidrBlocks", []interface{}{"172.25.0.0/16"}),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/proxyMode", "ipvs"),
				jsonpatch.NewOperation("replace", "/spec/clusterNetwork/dnsDomain", "cluster.local"),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/nodeLocalDNSCacheEnabled", true),
				jsonpatch.NewOperation("add", "/spec/clusterNetwork/ipvs/strictArp", true),
			},
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
						Raw: rawClusterGen{Name: "foo", CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}}, ExternalCloudProvider: true}.Do(),
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
						Raw: rawClusterGen{Name: "foo", CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}}, ExternalCloudProvider: true}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}}, ExternalCloudProvider: false}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{
				jsonpatch.NewOperation("add", "/metadata/annotations", map[string]interface{}{"ccm-migration.k8c.io/migration-needed": "", "csi-migration.k8c.io/migration-needed": ""}),
				jsonpatch.NewOperation("add", "/spec/cloud/openstack/useOctavia", true),
			},
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
						Raw: rawClusterGen{Name: "foo", CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}}, ExternalCloudProvider: true}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", CloudSpec: kubermaticv1.CloudSpec{Openstack: &kubermaticv1.OpenstackCloudSpec{}}, ExternalCloudProvider: true}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{},
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
						Raw: rawClusterGen{Name: "foo", CloudSpec: kubermaticv1.CloudSpec{Hetzner: &kubermaticv1.HetznerCloudSpec{}}, ExternalCloudProvider: true}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{Name: "foo", CloudSpec: kubermaticv1.CloudSpec{Hetzner: &kubermaticv1.HetznerCloudSpec{}}, ExternalCloudProvider: false}.Do(),
					},
				},
			},
			wantAllowed: true,
			wantPatches: []jsonpatch.JsonPatchOperation{},
		},
	}
	for _, tt := range tests {
		t.Logf("Executing test: %s", tt.name)
		d, err := admission.NewDecoder(testScheme)
		if err != nil {
			t.Fatalf("error occurred while creating decoder: %v", err)
		}
		t.Run(tt.name, func(t *testing.T) {
			handler := AdmissionHandler{
				log:                      logr.Discard(),
				decoder:                  d,
				defaultComponentSettings: tt.componentSettings,
			}
			res := handler.Handle(context.TODO(), tt.req)
			if res.Allowed != tt.wantAllowed {
				t.Logf("Response: %v", res)
				t.Fatalf("Allowed %t, but wanted %t", res.Allowed, tt.wantAllowed)
			}

			t.Logf("Received patches: %+v", res.Patches)
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
	CloudSpec             kubermaticv1.CloudSpec
	CNIPluginSpec         *kubermaticv1.CNIPluginSettings
	ExternalCloudProvider bool
	NetworkConfig         kubermaticv1.ClusterNetworkingConfig
}

func (r rawClusterGen) Do() []byte {
	c := kubermaticv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kubermatic.k8s.io/v1",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Features: map[string]bool{
				"externalCloudProvider": r.ExternalCloudProvider,
			},
			Cloud:          r.CloudSpec,
			ClusterNetwork: r.NetworkConfig,
			CNIPlugin:      r.CNIPluginSpec,
		},
	}
	s := json.NewSerializerWithOptions(json.DefaultMetaFactory, testScheme, testScheme, json.SerializerOptions{Pretty: true})
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&c, buff)
	return buff.Bytes()
}
