/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilintstr "k8s.io/apimachinery/pkg/util/intstr"
)

var kindConfigContent = `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
 - role: control-plane
 - role: worker
   extraPortMappings:
   # nodeport-proxy for user-cluster control-planes
   - containerPort: 30652
     hostPort: 6443
   - containerPort: 32121
     hostPort: 8088
   # envoy gateway for kubermatic api and dashboard
   - containerPort: 31514
     hostPort: 80
   - containerPort: 32394
     hostPort: 443`

func ptr[T any](v T) *T {
	return &v
}

var kindKubermaticNamespace = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "kubermatic",
	},
}

var kindEnvoyGatewayNamespace = corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: "envoy-gateway-system",
	},
}

var kindStorageClass = storagev1.StorageClass{
	ObjectMeta: metav1.ObjectMeta{
		Name: "kubermatic-fast",
	},
	Provisioner:       "rancher.io/local-path",
	VolumeBindingMode: ptr(storagev1.VolumeBindingWaitForFirstConsumer),
}

var kindNodeportProxyService = corev1.Service{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "nodeport-proxy-kind",
		Namespace: "kubermatic",
	},
	Spec: corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name:     "sni-listener",
				Protocol: "TCP",
				Port:     6443,
				TargetPort: utilintstr.IntOrString{
					IntVal: 6443,
				},
				NodePort: 30652,
			},
			{
				Name:     "tunneling-listener",
				Protocol: "TCP",
				Port:     8088,
				TargetPort: utilintstr.IntOrString{
					IntVal: 8088,
				},
				NodePort: 32121,
			},
		},
		Selector: map[string]string{
			"app.kubernetes.io/name": "nodeport-proxy-envoy",
		},
		Type:                  "NodePort",
		ExternalTrafficPolicy: "Cluster",
		IPFamilies: []corev1.IPFamily{
			"IPv4",
		},
		IPFamilyPolicy:        ptr(corev1.IPFamilyPolicy("SingleStack")),
		InternalTrafficPolicy: ptr(corev1.ServiceInternalTrafficPolicyType("Cluster")),
	},
}

var kindLocalSeed = kubermaticv1.Seed{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "kubermatic",
		Namespace: "kubermatic",
	},
	Spec: kubermaticv1.SeedSpec{
		Country:  "DE",      // TODO: some clever heuristic or geolocation service?
		Location: "Hamburg", // TODO: some clever heuristic or geolocation service?
		Kubeconfig: corev1.ObjectReference{
			Namespace: "kubermatic",
			Name:      "kubeconfig-seed",
		},
		Datacenters: map[string]kubermaticv1.Datacenter{
			"kubevirt": {
				Country:  "DE",      // TODO: some clever heuristic or geolocation service?
				Location: "Hamburg", // TODO: some clever heuristic or geolocation service?
				Spec: kubermaticv1.DatacenterSpec{
					Kubevirt: &kubermaticv1.DatacenterSpecKubevirt{
						EnableDefaultNetworkPolicies: ptr(false),
						DNSPolicy:                    "ClusterFirst",
						Images: kubermaticv1.KubeVirtImageSources{
							HTTP: &kubermaticv1.KubeVirtHTTPSource{
								OperatingSystems: map[providerconfig.OperatingSystem]kubermaticv1.OSVersions{
									providerconfig.OperatingSystemFlatcar: map[string]string{
										"3374.2.2": "docker://quay.io/kubermatic-virt-disks/flatcar:3374.2.2",
									},
									providerconfig.OperatingSystemRHEL: map[string]string{
										"8": "docker://quay.io/kubermatic-virt-disks/rhel:8",
									},
									providerconfig.OperatingSystemRockyLinux: map[string]string{
										"8": "docker://quay.io/kubermatic-virt-disks/rockylinux:8",
									},
									providerconfig.OperatingSystemUbuntu: map[string]string{
										"20.04": "docker://quay.io/kubermatic-virt-disks/ubuntu:20.04",
										"22.04": "docker://quay.io/kubermatic-virt-disks/ubuntu:22.04",
									},
								},
							},
						},
					},
				},
			},
		},
		ExposeStrategy: "Tunneling",
	},
}

var kindLocalPreset = kubermaticv1.Preset{
	ObjectMeta: metav1.ObjectMeta{
		Name: "local",
	},
	Spec: kubermaticv1.PresetSpec{
		Kubevirt: &kubermaticv1.Kubevirt{
			ProviderPreset: kubermaticv1.ProviderPreset{
				Enabled: ptr(true),
			},
			Kubeconfig: "",
		},
	},
}

var kindKubeconfigSeedSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "kubeconfig-seed",
		Namespace: "kubermatic",
	},
	StringData: map[string]string{"kubeconfig": ""},
}
