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

package main

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/nodeport-proxy/envoymanager"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconciliation(t *testing.T) {
	testCases := []struct {
		name                  string
		initialServices       []ctrlruntimeclient.Object
		expectedServices      corev1.ServiceList
		sniListenerPort       int
		tunnelingListenerPort int
	}{
		{
			name: "Service without annotation gets ignored",
			initialServices: []ctrlruntimeclient.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Ports: []corev1.ServicePort{{
							Port:     443,
							NodePort: 30443,
						}},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "lb-ns",
						Name:      "lb",
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
					},
				},
			},
			expectedServices: corev1.ServiceList{
				Items: []corev1.Service{
					{
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Port:     443,
								NodePort: 30443,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:       "lb-ns",
							Name:            "lb",
							ResourceVersion: "1",
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Name:       "healthz",
								Port:       8002,
								TargetPort: intstr.FromInt(8002),
								Protocol:   corev1.ProtocolTCP,
							}},
						},
					},
				},
			},
		},
		{
			name: "Service without clusterIP gets ignored",
			initialServices: []ctrlruntimeclient.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Port:     443,
							NodePort: 30443,
						}},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "lb-ns",
						Name:      "lb",
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
					},
				},
			},
			expectedServices: corev1.ServiceList{
				Items: []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{{
								Port:     443,
								NodePort: 30443,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:       "lb-ns",
							Name:            "lb",
							ResourceVersion: "1",
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Name:       "healthz",
								Port:       8002,
								TargetPort: intstr.FromInt(8002),
								Protocol:   corev1.ProtocolTCP,
							}},
						},
					},
				},
			},
		},
		{
			name: "Service without NodePort gets ignored",
			initialServices: []ctrlruntimeclient.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Ports: []corev1.ServicePort{{
							Port: 443,
						}},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "lb-ns",
						Name:      "lb",
					},
				},
			},
			expectedServices: corev1.ServiceList{
				Items: []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Port: 443,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:       "lb-ns",
							Name:            "lb",
							ResourceVersion: "1",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{{
								Name:       "healthz",
								Port:       8002,
								TargetPort: intstr.FromInt(8002),
								Protocol:   corev1.ProtocolTCP,
							}},
						},
					},
				},
			},
		},
		{
			name: "Reconciliation with existing port following old nameschema",
			initialServices: []ctrlruntimeclient.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   "cluster",
						Name:        "apiserver",
						Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Ports: []corev1.ServicePort{{
							Port:     443,
							NodePort: 30443,
						}},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "lb-ns",
						Name:      "lb",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:       "cluster-apiserver-443-30443",
							Port:       30443,
							TargetPort: intstr.FromInt(30443),
							NodePort:   31443,
						}},
					},
				},
			},
			expectedServices: corev1.ServiceList{
				Items: []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   "cluster",
							Name:        "apiserver",
							Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Port:     443,
								NodePort: 30443,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:       "lb-ns",
							Name:            "lb",
							ResourceVersion: "1",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "cluster-apiserver-443-30443",
									Port:       30443,
									TargetPort: intstr.FromInt(30443),
									NodePort:   31443,
									Protocol:   corev1.ProtocolTCP,
								},
								{
									Name:       "healthz",
									Port:       8002,
									TargetPort: intstr.FromInt(8002),
									Protocol:   corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Reconciliation with exiting port following new nameschema",
			initialServices: []ctrlruntimeclient.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   "cluster",
						Name:        "apiserver",
						Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Ports: []corev1.ServicePort{{
							Port:     443,
							NodePort: 30443,
						}},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "lb-ns",
						Name:      "lb",
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{
							Name:       "cluster-apiserver-30443",
							Port:       30443,
							TargetPort: intstr.FromInt(30443),
							NodePort:   31443,
						}},
					},
				},
			},
			expectedServices: corev1.ServiceList{
				Items: []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   "cluster",
							Name:        "apiserver",
							Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Port:     443,
								NodePort: 30443,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:       "lb-ns",
							Name:            "lb",
							ResourceVersion: "1",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "cluster-apiserver-30443",
									Port:       30443,
									TargetPort: intstr.FromInt(30443),
									NodePort:   31443,
									Protocol:   corev1.ProtocolTCP,
								},
								{
									Name:       "healthz",
									Port:       8002,
									TargetPort: intstr.FromInt(8002),
									Protocol:   corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Reconciliation without existing port uses new nameschema",
			initialServices: []ctrlruntimeclient.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   "cluster",
						Name:        "apiserver",
						Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Ports: []corev1.ServicePort{{
							Port:     443,
							NodePort: 30443,
						}},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "lb-ns",
						Name:      "lb",
					},
				},
			},
			expectedServices: corev1.ServiceList{
				Items: []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   "cluster",
							Name:        "apiserver",
							Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Port:     443,
								NodePort: 30443,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:       "lb-ns",
							Name:            "lb",
							ResourceVersion: "1",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "cluster-apiserver-30443",
									Port:       30443,
									TargetPort: intstr.FromInt(30443),
									Protocol:   corev1.ProtocolTCP,
								},
								{
									Name:       "healthz",
									Port:       8002,
									TargetPort: intstr.FromInt(8002),
									Protocol:   corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
		{
			name:            "Activated SNI listener",
			sniListenerPort: 6443,
			initialServices: []ctrlruntimeclient.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   "cluster",
						Name:        "apiserver",
						Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Ports: []corev1.ServicePort{{
							Port:     443,
							NodePort: 30443,
						}},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "lb-ns",
						Name:      "lb",
					},
				},
			},
			expectedServices: corev1.ServiceList{
				Items: []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   "cluster",
							Name:        "apiserver",
							Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Port:     443,
								NodePort: 30443,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:       "lb-ns",
							Name:            "lb",
							ResourceVersion: "1",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "cluster-apiserver-30443",
									Port:       30443,
									TargetPort: intstr.FromInt(30443),
									Protocol:   corev1.ProtocolTCP,
								},
								{
									Name:       "healthz",
									Port:       8002,
									TargetPort: intstr.FromInt(8002),
									Protocol:   corev1.ProtocolTCP,
								},
								{
									Name:       "sni-listener",
									Port:       6443,
									TargetPort: intstr.FromInt(6443),
									Protocol:   corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
		{
			name:                  "Activated HTTP/2 CONNECT listener",
			tunnelingListenerPort: 8443,
			initialServices: []ctrlruntimeclient.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:   "cluster",
						Name:        "apiserver",
						Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
					},
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Ports: []corev1.ServicePort{{
							Port:     443,
							NodePort: 30443,
						}},
					},
				},
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "lb-ns",
						Name:      "lb",
					},
				},
			},
			expectedServices: corev1.ServiceList{
				Items: []corev1.Service{
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:   "cluster",
							Name:        "apiserver",
							Annotations: map[string]string{"nodeport-proxy.k8s.io/expose": "true"},
						},
						Spec: corev1.ServiceSpec{
							ClusterIP: "1.2.3.4",
							Ports: []corev1.ServicePort{{
								Port:     443,
								NodePort: 30443,
							}},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Namespace:       "lb-ns",
							Name:            "lb",
							ResourceVersion: "1",
						},
						Spec: corev1.ServiceSpec{
							Ports: []corev1.ServicePort{
								{
									Name:       "cluster-apiserver-30443",
									Port:       30443,
									TargetPort: intstr.FromInt(30443),
									Protocol:   corev1.ProtocolTCP,
								},
								{
									Name:       "healthz",
									Port:       8002,
									TargetPort: intstr.FromInt(8002),
									Protocol:   corev1.ProtocolTCP,
								},
								{
									Name:       "tunneling-listener",
									Port:       8443,
									TargetPort: intstr.FromInt(8443),
									Protocol:   corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			client := fake.NewClientBuilder().WithObjects(tc.initialServices...).Build()
			updater := &LBUpdater{
				lbNamespace: "lb-ns",
				lbName:      "lb",
				client:      client,
				log:         zap.NewNop().Sugar(),
				opts: envoymanager.Options{
					ExposeAnnotationKey:        nodeportproxy.DefaultExposeAnnotationKey,
					EnvoySNIListenerPort:       tc.sniListenerPort,
					EnvoyTunnelingListenerPort: tc.tunnelingListenerPort,
				},
			}

			if _, err := updater.Reconcile(ctx, reconcile.Request{}); err != nil {
				t.Fatalf("error reconciling: %v", err)
			}

			resultingServices := &corev1.ServiceList{}
			if err := client.List(ctx, resultingServices); err != nil {
				t.Fatalf("failed to list services: %v", err)
			}

			for i := range resultingServices.Items {
				resultingServices.Items[i].ResourceVersion = ""
			}

			for i := range tc.expectedServices.Items {
				tc.expectedServices.Items[i].ResourceVersion = ""
			}

			if !diff.SemanticallyEqual(resultingServices.Items, tc.expectedServices.Items) {
				t.Fatalf("resulting services differ from expected services:\n%s", diff.ObjectDiff(tc.expectedServices.Items, resultingServices.Items))
			}
		})
	}
}

// These tests are here for good reasons. Do not change them and make sure
// they continue to pass.
//
// Desired behavior in a nutshell:
// * Treat the name of the Port on the LoadBalancer service as canonical source of truth
// * Copy over the NodePort from the LoadBalancer service
// * If there is no corresponding port on the LoadBalancer service use the new name schema
// * If there is no corresponding port on the LoadBalancer service reset the NodePort.
func TestFillNodePortsAndNames(t *testing.T) {
	testCases := []struct {
		name      string
		inputPort corev1.ServicePort
		lbPorts   []corev1.ServicePort
		expected  corev1.ServicePort
	}{
		{
			name: "Matching via old name schema",
			inputPort: corev1.ServicePort{
				Name:     "namespace-name",
				Port:     30443,
				NodePort: 443,
			},
			lbPorts: []corev1.ServicePort{{
				Name:     "namespace-name-443-30443",
				NodePort: 31443,
			}},
			expected: corev1.ServicePort{
				Name:     "namespace-name-443-30443",
				Port:     30443,
				NodePort: 31443,
			},
		},
		{
			name: "Matching via new name schema",
			inputPort: corev1.ServicePort{
				Name:     "namespace-name",
				Port:     30443,
				NodePort: 443,
			},
			lbPorts: []corev1.ServicePort{{
				Name:     "namespace-name-30443",
				NodePort: 31443,
			}},
			expected: corev1.ServicePort{
				Name:     "namespace-name-30443",
				Port:     30443,
				NodePort: 31443,
			},
		},
		{
			name: "Default to new name schema, reset NodePort",
			inputPort: corev1.ServicePort{
				Name:     "namespace-name",
				Port:     30443,
				NodePort: 443,
			},
			expected: corev1.ServicePort{
				Name: "namespace-name-30443",
				Port: 30443,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fillNodePortsAndNames([]corev1.ServicePort{tc.inputPort}, tc.lbPorts)
			if !apiequality.Semantic.DeepEqual(result[0], tc.expected) {
				t.Fatalf("result differs from expected, result:\n%v\nexpected:\n%v", result[0], tc.expected)
			}
		})
	}
}
