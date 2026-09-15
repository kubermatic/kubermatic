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
	"time"

	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestEnqueueKubeVirtClustersForAcceleratorQuotaResourceQuota(t *testing.T) {
	deletingCluster := kubeVirtCluster("deleting", "project-a")
	deletionTimestamp := metav1.NewTime(time.Now())
	deletingCluster.DeletionTimestamp = &deletionTimestamp
	deletingCluster.Finalizers = []string{"test-finalizer"}
	otherWorkerCluster := kubeVirtCluster("other-worker", "project-a")
	otherWorkerCluster.Labels[kubermaticv1.WorkerNameLabelKey] = "other-worker"

	r := &Reconciler{Client: fake.NewClientBuilder().WithObjects(
		kubeVirtCluster("cluster-b", "project-a"),
		kubeVirtCluster("cluster-a", "project-a"),
		kubeVirtCluster("other-project", "project-b"),
		awsCluster("aws", "project-a"),
		deletingCluster,
		otherWorkerCluster,
	).Build()}

	got := r.enqueueKubeVirtClustersForAcceleratorQuotaResourceQuota(
		context.Background(),
		projectResourceQuota("quota-a", "project-a", resources.AcceleratorAccountingEnabledAnnotationValue),
	)

	require.Equal(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: "cluster-a"}},
		{NamespacedName: types.NamespacedName{Name: "cluster-b"}},
	}, got)
}

func TestResourceQuotaAcceleratorAccountingActivationPredicate(t *testing.T) {
	pred := resourceQuotaAcceleratorAccountingActivationPredicate()
	inactive := projectResourceQuota("quota-a", "project-a", "")
	active := inactive.DeepCopy()
	active.Annotations = map[string]string{
		resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
	}
	deletingActive := active.DeepCopy()
	now := metav1.Now()
	deletingActive.DeletionTimestamp = &now
	deletingActive.Finalizers = []string{"test.kubermatic.io/cleanup"}

	require.False(t, pred.Create(event.CreateEvent{Object: inactive}))
	require.True(t, pred.Create(event.CreateEvent{Object: active}))
	require.False(t, pred.Create(event.CreateEvent{Object: deletingActive}))
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: inactive, ObjectNew: active}))
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: active, ObjectNew: active.DeepCopy()}))

	changedSubjectNameLabel := active.DeepCopy()
	changedSubjectNameLabel.Labels[kubermaticv1.ResourceQuotaSubjectNameLabelKey] = "project-b"
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: changedSubjectNameLabel, ObjectNew: active}))

	changedSubjectKindLabel := active.DeepCopy()
	changedSubjectKindLabel.Labels[kubermaticv1.ResourceQuotaSubjectKindLabelKey] = "other"
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: changedSubjectKindLabel, ObjectNew: active}))

	changedSubjectName := active.DeepCopy()
	changedSubjectName.Spec.Subject.Name = "project-b"
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: changedSubjectName, ObjectNew: active}))

	changedSubjectKind := active.DeepCopy()
	changedSubjectKind.Spec.Subject.Kind = "other"
	require.True(t, pred.Update(event.UpdateEvent{ObjectOld: changedSubjectKind, ObjectNew: active}))
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: active, ObjectNew: deletingActive}))

	unrelatedMetadataChange := active.DeepCopy()
	unrelatedMetadataChange.Annotations["example.com/unrelated"] = "value"
	require.False(t, pred.Update(event.UpdateEvent{ObjectOld: active, ObjectNew: unrelatedMetadataChange}))

	require.False(t, pred.Delete(event.DeleteEvent{Object: active}))
	require.False(t, pred.Delete(event.DeleteEvent{Object: deletingActive}))
	require.False(t, pred.Delete(event.DeleteEvent{Object: inactive}))
}

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
		seedGetter    func() (*kubermaticv1.Seed, error)
		oidcIssuerURL string
		features      Features
		expected      []reconcile.Request
	}{
		{
			name: "enqueue OIDC candidates",
			objects: []ctrlruntimeclient.Object{
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
			},
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
			expected: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "cluster-auth"}},
				{NamespacedName: types.NamespacedName{Name: "dc-auth"}},
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
		{
			name:    "enqueue network policy clusters when seed is unavailable",
			objects: []ctrlruntimeclient.Object{clusterWithNetworkPolicy("candidate", "cluster-candidate", "dc-a", nil)},
			seedGetter: func() (*kubermaticv1.Seed, error) {
				return nil, errors.New("seed unavailable")
			},
			expected: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "candidate"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Client:        fake.NewClientBuilder().WithObjects(tt.objects...).Build(),
				seedGetter:    tt.seedGetter,
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
