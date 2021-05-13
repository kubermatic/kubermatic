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

	logrtesting "github.com/go-logr/logr/testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
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
	tests := []struct {
		name        string
		req         webhook.AdmissionRequest
		wantAllowed bool
		features    features.FeatureGate
	}{
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
					Name: "cluster",
				},
			},
			wantAllowed: true,
		},
		{
			name: "Create cluster with Tunneling expose strategy succeeds when the FeatureGate is enabled",
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
							Name:           "foo",
							Namespace:      "kubermatic",
							ExposeStrategy: "Tunneling",
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
			features:    features.FeatureGate{features.TunnelingExposeStrategy: true},
		},
		{
			name: "Create cluster with Tunneling expose strategy fails when the FeatureGate is not enabled",
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
							Name:           "foo",
							Namespace:      "kubermatic",
							ExposeStrategy: "Tunneling",
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: false,
			features:    features.FeatureGate{features.TunnelingExposeStrategy: false},
		},
		{
			name: "Create cluster expose strategy different from Tunneling should succeed",
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
							Name:           "foo",
							Namespace:      "kubermatic",
							ExposeStrategy: "NodePort",
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Unknown expose strategy",
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
							Name:           "foo",
							Namespace:      "kubermatic",
							ExposeStrategy: "ciao",
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Reject EnableUserSSHKey agent update",
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
							Name:             "foo",
							Namespace:        "kubermatic",
							ExposeStrategy:   "NodePort",
							EnableUserSSHKey: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name:             "foo",
							Namespace:        "kubermatic",
							ExposeStrategy:   "NodePort",
							EnableUserSSHKey: false,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Accept a cluster create request with externalCloudProvider disabled",
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
							Namespace:             "kubermatic",
							ExposeStrategy:        "NodePort",
							ExternalCloudProvider: false,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Accept a cluster create request with externalCloudProvider enabled",
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
							Namespace:             "kubermatic",
							ExposeStrategy:        "NodePort",
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Accept enabling the externalCloudProvider feature",
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
							Name:                  "foo",
							Namespace:             "kubermatic",
							ExposeStrategy:        "NodePort",
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name:                  "foo",
							Namespace:             "kubermatic",
							ExposeStrategy:        "NodePort",
							ExternalCloudProvider: false,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: true,
		},
		{
			name: "Reject disabling the externalCloudProvider feature",
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
							Name:                  "foo",
							Namespace:             "kubermatic",
							ExposeStrategy:        "NodePort",
							ExternalCloudProvider: false,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name:                  "foo",
							Namespace:             "kubermatic",
							ExposeStrategy:        "NodePort",
							ExternalCloudProvider: true,
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: false,
		},
		{
			name: "Reject updating the pods CIDR",
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
							Name:      "foo",
							Namespace: "kubermatic",
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.192.0.0/20"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
					OldObject: runtime.RawExtension{
						Raw: rawClusterGen{
							Name:      "foo",
							Namespace: "kubermatic",
							NetworkConfig: kubermaticv1.ClusterNetworkingConfig{
								Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.193.0.0/20"}},
								Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.32.0/20"}},
								DNSDomain: "cluster.local",
								ProxyMode: resources.IPVSProxyMode,
							},
						}.Do(),
					},
				},
			},
			wantAllowed: false,
		},
	}
	for _, tt := range tests {
		d, err := admission.NewDecoder(testScheme)
		if err != nil {
			t.Fatalf("error occurred while creating decoder: %v", err)
		}
		handler := AdmissionHandler{
			log:      &logrtesting.NullLogger{},
			decoder:  d,
			features: tt.features,
		}
		t.Run(tt.name, func(t *testing.T) {
			if res := handler.Handle(context.TODO(), tt.req); res.Allowed != tt.wantAllowed {
				t.Errorf("Allowed %t, but wanted %t", res.Allowed, tt.wantAllowed)
				t.Logf("Response: %v", res)
			}
		})
	}
}

type rawClusterGen struct {
	Name                  string
	Namespace             string
	ExposeStrategy        string
	EnableUserSSHKey      bool
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
			Name:      r.Name,
			Namespace: r.Namespace,
		},
		Spec: kubermaticv1.ClusterSpec{
			Features: map[string]bool{
				"externalCloudProvider": r.ExternalCloudProvider,
			},
			ExposeStrategy:        kubermaticv1.ExposeStrategy(r.ExposeStrategy),
			EnableUserSSHKeyAgent: r.EnableUserSSHKey,
			ClusterNetwork:        r.NetworkConfig,
		},
	}
	s := json.NewSerializer(json.DefaultMetaFactory, testScheme, testScheme, true)
	buff := bytes.NewBuffer([]byte{})
	_ = s.Encode(&c, buff)
	return buff.Bytes()
}
