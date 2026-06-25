/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestOIDCIssuerLoadBalancerServicePredicate(t *testing.T) {
	predicate := oidcIssuerLoadBalancerServicePredicate()

	eventTests := []struct {
		name     string
		matches  func() bool
		expected bool
	}{
		{
			name: "ignore create event",
			matches: func() bool {
				return predicate.Create(event.CreateEvent{Object: oidcIssuerLoadBalancerService()})
			},
			expected: false,
		},
		{
			name: "delete matching service",
			matches: func() bool {
				return predicate.Delete(event.DeleteEvent{Object: oidcIssuerLoadBalancerService()})
			},
			expected: true,
		},
		{
			name: "ignore generic event",
			matches: func() bool {
				return predicate.Generic(event.GenericEvent{Object: oidcIssuerLoadBalancerService()})
			},
			expected: false,
		},
	}

	for _, tt := range eventTests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.matches())
		})
	}

	updateTests := []struct {
		name     string
		mutate   func(oldSvc, newSvc *corev1.Service)
		expected bool
	}{
		{
			name: "update when load balancer ingress is assigned",
			mutate: func(oldSvc, newSvc *corev1.Service) {
				oldSvc.Status.LoadBalancer.Ingress = nil
			},
			expected: true,
		},
		{
			name: "update when ingress changes",
			mutate: func(oldSvc, newSvc *corev1.Service) {
				newSvc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{IP: "10.10.45.1"}}
			},
			expected: true,
		},
		{
			name: "update when ports change",
			mutate: func(oldSvc, newSvc *corev1.Service) {
				newSvc.Spec.Ports = []corev1.ServicePort{{Port: 443}}
			},
			expected: true,
		},
		{
			name: "update when service stops matching",
			mutate: func(oldSvc, newSvc *corev1.Service) {
				newSvc.Spec.Type = corev1.ServiceTypeClusterIP
			},
			expected: true,
		},
		{
			name: "ignore update when neither service is a candidate",
			mutate: func(oldSvc, newSvc *corev1.Service) {
				oldSvc.Status.LoadBalancer.Ingress = nil
				newSvc.Status.LoadBalancer.Ingress = nil
				newSvc.Labels = map[string]string{"changed": "true"}
			},
			expected: false,
		},
		{
			name: "ignore unrelated update",
			mutate: func(oldSvc, newSvc *corev1.Service) {
				newSvc.Labels = map[string]string{"changed": "true"}
			},
			expected: false,
		},
	}

	for _, tt := range updateTests {
		t.Run(tt.name, func(t *testing.T) {
			oldSvc := oidcIssuerLoadBalancerService()
			newSvc := oldSvc.DeepCopy()
			tt.mutate(oldSvc, newSvc)

			require.Equal(t, tt.expected, predicate.Update(event.UpdateEvent{ObjectOld: oldSvc, ObjectNew: newSvc}))
		})
	}
}

func TestEnqueueClustersForOIDCIssuerLoadBalancerService(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		objects       []ctrlruntimeclient.Object
		oidcIssuerURL string
		features      Features
		expected      []reconcile.Request
	}{
		{
			name: "enqueue OIDC candidates",
			objects: []ctrlruntimeclient.Object{
				clusterWithNetworkPolicy("legacy-oidc", "cluster-legacy-oidc", "dc-a", func(cluster *kubermaticv1.Cluster) {
					cluster.Spec.OIDC.IssuerURL = "https://issuer.example.com"
				}),
				clusterWithNetworkPolicy("no-oidc", "cluster-no-oidc", "dc-a", nil),
				&kubermaticv1.Cluster{
					ObjectMeta: metav1.ObjectMeta{
						Name: "no-network-policy",
					},
					Spec: kubermaticv1.ClusterSpec{
						Features: map[string]bool{},
					},
					Status: kubermaticv1.ClusterStatus{
						NamespaceName: "cluster-no-network-policy",
					},
				},
			},
			expected: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "legacy-oidc"}},
			},
		},
		{
			name:          "enqueue cluster using default issuer",
			objects:       []ctrlruntimeclient.Object{clusterWithNetworkPolicy("default-issuer", "cluster-default-issuer", "dc-a", nil)},
			oidcIssuerURL: "https://issuer.example.com",
			features: Features{
				KubernetesOIDCAuthentication: true,
			},
			expected: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "default-issuer"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Client:        fake.NewClientBuilder().WithObjects(tt.objects...).Build(),
				oidcIssuerURL: tt.oidcIssuerURL,
				features:      tt.features,
			}

			requests := r.enqueueClustersForOIDCIssuerLoadBalancerService(ctx, oidcIssuerLoadBalancerService())

			require.Equal(t, tt.expected, requests)
		})
	}
}

func clusterWithNetworkPolicy(name, namespace, datacenter string, mutate func(*kubermaticv1.Cluster)) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenter,
			},
			Features: map[string]bool{
				kubermaticv1.ApiserverNetworkPolicy: true,
			},
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: namespace,
		},
	}

	if mutate != nil {
		mutate(cluster)
	}

	return cluster
}

func oidcIssuerLoadBalancerService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "issuer-gateway",
			Namespace: "issuer",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Selector: map[string]string{
				"app": "issuer",
			},
			Ports: []corev1.ServicePort{
				{Port: 80},
			},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{{IP: "10.10.45.0"}},
			},
		},
	}
}
