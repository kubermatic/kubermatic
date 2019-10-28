package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-test/deep"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	exposeAnnotationKey = defaultExposeAnnotationKey
}

func TestReconciliation(t *testing.T) {
	testCases := []struct {
		name             string
		initialServices  []runtime.Object
		expectedServices corev1.ServiceList
	}{
		{
			name: "Service without annotation gets ignored",
			initialServices: []runtime.Object{
				&corev1.Service{
					Spec: corev1.ServiceSpec{
						ClusterIP: "1.2.3.4",
						Ports: []corev1.ServicePort{{
							Port:     443,
							NodePort: 30443,
						}},
					},
				},
				&corev1.Service{
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
							Namespace: "lb-ns",
							Name:      "lb",
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
			initialServices: []runtime.Object{
				&corev1.Service{
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
							Namespace: "lb-ns",
							Name:      "lb",
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
			initialServices: []runtime.Object{
				&corev1.Service{
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
							Namespace: "lb-ns",
							Name:      "lb",
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
			initialServices: []runtime.Object{
				&corev1.Service{
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
							Namespace: "lb-ns",
							Name:      "lb",
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
			initialServices: []runtime.Object{
				&corev1.Service{
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
							Namespace: "lb-ns",
							Name:      "lb",
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
			initialServices: []runtime.Object{
				&corev1.Service{
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
							Namespace: "lb-ns",
							Name:      "lb",
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
	}

	var breakNow bool
	for _, tc := range testCases {
		if breakNow {
			break
		}
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClient(tc.initialServices...)
			updater := &LBUpdater{
				lbNamespace: "lb-ns",
				lbName:      "lb",
				client:      client,
			}

			if _, err := updater.Reconcile(reconcile.Request{}); err != nil {
				t.Fatalf("error reconciling: %v", err)
			}

			resultingServices := &corev1.ServiceList{}
			if err := client.List(context.Background(), resultingServices); err != nil {
				t.Fatalf("failed to list services: %v", err)
			}

			if diff := deep.Equal(*resultingServices, tc.expectedServices); diff != nil {
				expected, _ := json.Marshal(tc.expectedServices)
				actual, _ := json.Marshal(*resultingServices)
				breakNow = true
				t.Fatalf("resulting services differ from expected services, diff:\n%v, expected:\n%s\nactual:\n%s", diff, expected, actual)
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
// * If there is no corresponding port on the LoadBalancer service reset the NodePort
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
