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
	"errors"
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

	t.Run("create matching service", func(t *testing.T) {
		require.True(t, predicate.Create(event.CreateEvent{Object: oidcIssuerLoadBalancerService()}))
	})

	t.Run("create service without ingress", func(t *testing.T) {
		svc := oidcIssuerLoadBalancerService()
		svc.Status.LoadBalancer.Ingress = nil

		require.False(t, predicate.Create(event.CreateEvent{Object: svc}))
	})

	t.Run("create service without selector", func(t *testing.T) {
		svc := oidcIssuerLoadBalancerService()
		svc.Spec.Selector = nil

		require.False(t, predicate.Create(event.CreateEvent{Object: svc}))
	})

	t.Run("create non LoadBalancer service", func(t *testing.T) {
		svc := oidcIssuerLoadBalancerService()
		svc.Spec.Type = corev1.ServiceTypeClusterIP

		require.False(t, predicate.Create(event.CreateEvent{Object: svc}))
	})

	t.Run("update when ingress changes", func(t *testing.T) {
		oldSvc := oidcIssuerLoadBalancerService()
		newSvc := oldSvc.DeepCopy()
		newSvc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{IP: "10.10.45.1"}}

		require.True(t, predicate.Update(event.UpdateEvent{ObjectOld: oldSvc, ObjectNew: newSvc}))
	})

	t.Run("update when ports change", func(t *testing.T) {
		oldSvc := oidcIssuerLoadBalancerService()
		newSvc := oldSvc.DeepCopy()
		newSvc.Spec.Ports = []corev1.ServicePort{{Port: 443}}

		require.True(t, predicate.Update(event.UpdateEvent{ObjectOld: oldSvc, ObjectNew: newSvc}))
	})

	t.Run("update when service stops matching", func(t *testing.T) {
		oldSvc := oidcIssuerLoadBalancerService()
		newSvc := oldSvc.DeepCopy()
		newSvc.Spec.Type = corev1.ServiceTypeClusterIP

		require.True(t, predicate.Update(event.UpdateEvent{ObjectOld: oldSvc, ObjectNew: newSvc}))
	})

	t.Run("ignore unrelated update", func(t *testing.T) {
		oldSvc := oidcIssuerLoadBalancerService()
		newSvc := oldSvc.DeepCopy()
		newSvc.Labels = map[string]string{"changed": "true"}

		require.False(t, predicate.Update(event.UpdateEvent{ObjectOld: oldSvc, ObjectNew: newSvc}))
	})

	t.Run("delete matching service", func(t *testing.T) {
		require.True(t, predicate.Delete(event.DeleteEvent{Object: oidcIssuerLoadBalancerService()}))
	})

	t.Run("ignore generic event", func(t *testing.T) {
		require.False(t, predicate.Generic(event.GenericEvent{Object: oidcIssuerLoadBalancerService()}))
	})
}

func TestEnqueueClustersForOIDCIssuerLoadBalancerService(t *testing.T) {
	ctx := context.Background()

	objects := []ctrlruntimeclient.Object{
		clusterWithNetworkPolicy("legacy-oidc", "cluster-legacy-oidc", "dc-a", func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.OIDC.IssuerURL = "https://issuer.example.com" //nolint:staticcheck
		}),
		clusterWithNetworkPolicy("cluster-auth", "cluster-auth", "dc-a", func(cluster *kubermaticv1.Cluster) {
			cluster.Spec.AuthenticationConfiguration = &kubermaticv1.AuthenticationConfiguration{
				SecretName: "auth-config",
				SecretKey:  "config.yaml",
			}
		}),
		clusterWithNetworkPolicy("dc-auth", "cluster-dc-auth", "dc-auth", nil),
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
	}

	r := &Reconciler{
		Client: fake.NewClientBuilder().WithObjects(objects...).Build(),
		seedGetter: func() (*kubermaticv1.Seed, error) {
			return &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						"dc-auth": {
							Spec: kubermaticv1.DatacenterSpec{
								AuthenticationConfiguration: &kubermaticv1.AuthenticationConfiguration{
									SecretName: "dc-auth-config",
									SecretKey:  "config.yaml",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	requests := r.enqueueClustersForOIDCIssuerLoadBalancerService(ctx, oidcIssuerLoadBalancerService())

	require.Equal(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "cluster-auth"}},
		{NamespacedName: types.NamespacedName{Name: "dc-auth"}},
		{NamespacedName: types.NamespacedName{Name: "legacy-oidc"}},
	}, requests)
}

func TestEnqueueClustersForOIDCIssuerLoadBalancerServiceWithDefaultIssuer(t *testing.T) {
	ctx := context.Background()

	cluster := clusterWithNetworkPolicy("default-issuer", "cluster-default-issuer", "dc-a", nil)
	r := &Reconciler{
		Client:        fake.NewClientBuilder().WithObjects(cluster).Build(),
		oidcIssuerURL: "https://issuer.example.com",
		features: Features{
			KubernetesOIDCAuthentication: true,
		},
	}

	requests := r.enqueueClustersForOIDCIssuerLoadBalancerService(ctx, oidcIssuerLoadBalancerService())

	require.Equal(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "default-issuer"}},
	}, requests)
}

func TestEnqueueClustersForOIDCIssuerLoadBalancerServiceWithUnavailableSeed(t *testing.T) {
	ctx := context.Background()

	cluster := clusterWithNetworkPolicy("candidate", "cluster-candidate", "dc-a", nil)
	r := &Reconciler{
		Client: fake.NewClientBuilder().WithObjects(cluster).Build(),
		seedGetter: func() (*kubermaticv1.Seed, error) {
			return nil, errors.New("seed unavailable")
		},
	}

	requests := r.enqueueClustersForOIDCIssuerLoadBalancerService(ctx, oidcIssuerLoadBalancerService())

	require.Equal(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "candidate"}},
	}, requests)
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
