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

package apiserver

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
)

func TestApiserverDeploymentCreatorRollingConfig(t *testing.T) {
	tests := []struct {
		name              string
		componentSettings kubermaticv1.ComponentSettings
		want              *appsv1.RollingUpdateDeployment
	}{
		{
			name: "No replicas defaulting to one.",
			componentSettings: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{
					DeploymentSettings: kubermaticv1.DeploymentSettings{},
				},
			},
			want: nil,
		},
		{
			name: "One replica.",
			componentSettings: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{
					DeploymentSettings: kubermaticv1.DeploymentSettings{
						Replicas: resources.Int32(1),
					},
				},
			},
			want: nil,
		},
		{
			name: "Two replicas.",
			componentSettings: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{
					DeploymentSettings: kubermaticv1.DeploymentSettings{
						Replicas: resources.Int32(2),
					},
				},
			},
			want: &appsv1.RollingUpdateDeployment{
				MaxSurge:       resources.IntOrString(intstr.FromInt(0)),
				MaxUnavailable: resources.IntOrString(intstr.FromInt(1)),
			},
		},
		{
			name: "Three replicas.",
			componentSettings: kubermaticv1.ComponentSettings{
				Apiserver: kubermaticv1.APIServerSettings{
					DeploymentSettings: kubermaticv1.DeploymentSettings{
						Replicas: resources.Int32(3),
					},
				},
			},
			want: &appsv1.RollingUpdateDeployment{
				MaxSurge:       resources.IntOrString(intstr.FromInt(0)),
				MaxUnavailable: resources.IntOrString(intstr.FromInt(1)),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, dcFunc := DeploymentCreator(makeTemplateData(tt.componentSettings), false)()
			if got, err := dcFunc(&appsv1.Deployment{}); !reflect.DeepEqual(got, tt.want) {
				if err != nil {
					t.Fatalf("Unexpected error occurred while updating deployment: %v", err)
				}
				if !reflect.DeepEqual(tt.want, got.Spec.Strategy.RollingUpdate) {
					t.Errorf("Expected %v but got %v", tt.want, got.Spec.Strategy.RollingUpdate)
				}
			}
		})
	}
}

func makeTemplateData(cs kubermaticv1.ComponentSettings) *resources.TemplateData {
	return resources.NewTemplateData(
		context.TODO(),
		fakectrlruntimeclient.NewFakeClient(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "apiserver-tls",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "tokens",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "openvpn-client-certificates",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubelet-client-certificates",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ca",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "service-account-key",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "apiserver-etcd-client-certificate",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "apiserver-proxy-client-certificate",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "front-proxy-ca",
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubeletdnatcontroller-kubeconfig",
				},
			},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cloud-config",
				},
			},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "audit-config",
				},
			},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "dns-resolver",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "0.0.0.0",
				},
			},
		),
		&kubermaticv1.Cluster{
			Spec: kubermaticv1.ClusterSpec{
				ComponentsOverride: cs,
				ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
					DNSDomain: "domain.com",
					Services: kubermaticv1.NetworkRanges{
						CIDRBlocks: []string{"4.4.4.4/16"},
					},
				},
				Version: *semver.NewSemverOrDie("1.17.1"),
			},
		},
		&kubermaticv1.Datacenter{},
		&kubermaticv1.Seed{},
		"",
		"",
		"",
		resource.Quantity{},
		"",
		"",
		false,
		false,
		"",
		"",
		"",
		"",
		false,
		"",
		"",
		"",
		false,
	)
}
