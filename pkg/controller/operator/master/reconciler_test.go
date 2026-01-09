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

package master

import (
	"context"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestCleanupGatewayAPIResources(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		validateFunc    func(t *testing.T, client ctrlruntimeclient.Client)
		expectError     bool
	}{
		{
			name: "deletes existing Gateway and HTTPRoute",
			existingObjects: []ctrlruntimeclient.Object{
				&gatewayapiv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "kubermatic",
					},
				},
				&gatewayapiv1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "kubermatic",
					},
				},
			},
			validateFunc: func(t *testing.T, client ctrlruntimeclient.Client) {
				gw := &gatewayapiv1.Gateway{}
				err := client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, gw)
				if !apierrors.IsNotFound(err) {
					t.Errorf("Expected Gateway to be deleted, got: %v", err)
				}

				hr := &gatewayapiv1.HTTPRoute{}
				err = client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, hr)
				if !apierrors.IsNotFound(err) {
					t.Errorf("Expected HTTPRoute to be deleted, got: %v", err)
				}
			},
			expectError: false,
		},
		{
			name: "only deletes Gateway if HTTPRoute does not exist",
			existingObjects: []ctrlruntimeclient.Object{
				&gatewayapiv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "kubermatic",
					},
				},
			},
			validateFunc: func(t *testing.T, client ctrlruntimeclient.Client) {
				gw := &gatewayapiv1.Gateway{}
				err := client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, gw)
				if !apierrors.IsNotFound(err) {
					t.Errorf("Expected Gateway to be deleted, got: %v", err)
				}
			},
			expectError: false,
		},
		{
			name: "only deletes HTTPRoute if Gateway does not exist",
			existingObjects: []ctrlruntimeclient.Object{
				&gatewayapiv1.HTTPRoute{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "kubermatic",
					},
				},
			},
			validateFunc: func(t *testing.T, client ctrlruntimeclient.Client) {
				hr := &gatewayapiv1.HTTPRoute{}
				err := client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, hr)
				if !apierrors.IsNotFound(err) {
					t.Errorf("Expected HTTPRoute to be deleted, got: %v", err)
				}
			},
			expectError: false,
		},
		{
			name:            "does not fail when Gateway and HTTPRoute do not exist",
			existingObjects: []ctrlruntimeclient.Object{},
			validateFunc:    nil,
			expectError:     false,
		},
		{
			name: "only deletes resources named 'kubermatic' in the configured namespace",
			existingObjects: []ctrlruntimeclient.Object{
				&gatewayapiv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "kubermatic",
					},
				},
				&gatewayapiv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-gateway",
						Namespace: "kubermatic",
					},
				},
				&gatewayapiv1.Gateway{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "other-namespace",
					},
				},
			},
			validateFunc: func(t *testing.T, client ctrlruntimeclient.Client) {
				// kubermatic/kubermatic Gateway should be deleted
				gw := &gatewayapiv1.Gateway{}
				err := client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, gw)
				if !apierrors.IsNotFound(err) {
					t.Errorf("Expected 'kubermatic/kubermatic' Gateway to be deleted, got: %v", err)
				}

				// other-gateway Gateway should still exist
				gw = &gatewayapiv1.Gateway{}
				err = client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "other-gateway"}, gw)
				if err != nil {
					t.Errorf("Expected 'other-gateway' Gateway to still exist, got: %v", err)
				}

				// other-namespace Gateway should still exist
				gw = &gatewayapiv1.Gateway{}
				err = client.Get(context.Background(),
					types.NamespacedName{Namespace: "other-namespace", Name: "kubermatic"}, gw)
				if err != nil {
					t.Errorf("Expected 'other-namespace/kubermatic' Gateway to still exist, got: %v", err)
				}
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			cfg := &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
						Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
							Enable: false,
						},
					},
				},
			}

			allObjects := append([]ctrlruntimeclient.Object{cfg}, tc.existingObjects...)
			client := fake.
				NewClientBuilder().
				WithObjects(allObjects...).
				Build()

			versions := kubermaticversion.Versions{
				KubermaticContainerTag: "latest",
				UIContainerTag:         "latest",
			}

			reconciler := &Reconciler{
				Client:   client,
				log:      zap.NewNop().Sugar(),
				recorder: record.NewFakeRecorder(100),
				scheme:   fake.NewScheme(),
				versions: versions,
			}

			err := reconciler.cleanupGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar())

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tc.validateFunc != nil {
				tc.validateFunc(t, client)
			}
		})
	}
}

func TestCleanupIngress(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		validateFunc    func(t *testing.T, client ctrlruntimeclient.Client)
		expectError     bool
	}{
		{
			name: "deletes existing Ingress",
			existingObjects: []ctrlruntimeclient.Object{
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "kubermatic",
					},
				},
			},
			validateFunc: func(t *testing.T, client ctrlruntimeclient.Client) {
				ingress := &networkingv1.Ingress{}
				err := client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, ingress)
				if !apierrors.IsNotFound(err) {
					t.Errorf("Expected Ingress to be deleted, got: %v", err)
				}
			},
			expectError: false,
		},
		{
			name:            "does not fail when Ingress does not exist",
			existingObjects: []ctrlruntimeclient.Object{},
			validateFunc:    nil,
			expectError:     false,
		},
		{
			name: "only deletes Ingress named 'kubermatic' in the configured namespace",
			existingObjects: []ctrlruntimeclient.Object{
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "kubermatic",
					},
				},
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-ingress",
						Namespace: "kubermatic",
					},
				},
				&networkingv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "kubermatic",
						Namespace: "other-namespace",
					},
				},
			},
			validateFunc: func(t *testing.T, client ctrlruntimeclient.Client) {
				// kubermatic/kubermatic Ingress should be deleted
				ingress := &networkingv1.Ingress{}
				err := client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, ingress)
				if !apierrors.IsNotFound(err) {
					t.Errorf("Expected 'kubermatic/kubermatic' Ingress to be deleted, got: %v", err)
				}

				// other-ingress Ingress should still exist
				ingress = &networkingv1.Ingress{}
				err = client.Get(context.Background(),
					types.NamespacedName{Namespace: "kubermatic", Name: "other-ingress"}, ingress)
				if err != nil {
					t.Errorf("Expected 'other-ingress' Ingress to still exist, got: %v", err)
				}

				// other-namespace Ingress should still exist
				ingress = &networkingv1.Ingress{}
				err = client.Get(context.Background(),
					types.NamespacedName{Namespace: "other-namespace", Name: "kubermatic"}, ingress)
				if err != nil {
					t.Errorf("Expected 'other-namespace/kubermatic' Ingress to still exist, got: %v", err)
				}
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			cfg := &kubermaticv1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Ingress: kubermaticv1.KubermaticIngressConfiguration{
						Domain: "example.com",
						Gateway: &kubermaticv1.KubermaticGatewayConfiguration{
							Enable: true,
						},
					},
				},
			}

			allObjects := append([]ctrlruntimeclient.Object{cfg}, tc.existingObjects...)
			client := fake.
				NewClientBuilder().
				WithObjects(allObjects...).
				Build()

			versions := kubermaticversion.Versions{
				KubermaticContainerTag: "latest",
				UIContainerTag:         "latest",
			}

			reconciler := &Reconciler{
				Client:   client,
				log:      zap.NewNop().Sugar(),
				recorder: record.NewFakeRecorder(100),
				scheme:   fake.NewScheme(),
				versions: versions,
			}

			err := reconciler.cleanupIngress(ctx, cfg, zap.NewNop().Sugar())

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tc.validateFunc != nil {
				tc.validateFunc(t, client)
			}
		})
	}
}

// TestCleanupGatewayAPIDoesNotDeleteIngress verifies that cleanupGatewayAPIResources
// does not delete the Ingress resource, as it only handles Gateway and HTTPRoute cleanup.
func TestCleanupGatewayAPIDoesNotDeleteIngress(t *testing.T) {
	ctx := context.Background()

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{},
	}

	existingObjects := []ctrlruntimeclient.Object{
		cfg,
		&gatewayapiv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubermatic",
				Namespace: "kubermatic",
			},
		},
		&gatewayapiv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubermatic",
				Namespace: "kubermatic",
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubermatic",
				Namespace: "kubermatic",
			},
		},
	}

	client := fake.
		NewClientBuilder().
		WithObjects(existingObjects...).
		Build()

	versions := kubermaticversion.Versions{
		KubermaticContainerTag: "latest",
		UIContainerTag:         "latest",
	}

	reconciler := &Reconciler{
		Client:   client,
		log:      zap.NewNop().Sugar(),
		recorder: record.NewFakeRecorder(100),
		scheme:   fake.NewScheme(),
		versions: versions,
	}

	// Run cleanupGatewayAPIResources - should only delete Gateway/HTTPRoute
	err := reconciler.cleanupGatewayAPIResources(ctx, cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("cleanupGatewayAPIResources failed: %v", err)
	}

	// Verify Gateway is deleted
	gw := &gatewayapiv1.Gateway{}
	err = client.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, gw)
	if !apierrors.IsNotFound(err) {
		t.Errorf("Expected Gateway to be deleted, got: %v", err)
	}

	// Verify HTTPRoute is deleted
	hr := &gatewayapiv1.HTTPRoute{}
	err = client.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, hr)
	if !apierrors.IsNotFound(err) {
		t.Errorf("Expected HTTPRoute to be deleted, got: %v", err)
	}

	// Verify Ingress still exists (cleanupGatewayAPIResources doesn't touch it)
	ingress := &networkingv1.Ingress{}
	err = client.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, ingress)
	if err != nil {
		t.Errorf("Expected Ingress to still exist, got: %v", err)
	}
}

// TestCleanupIngressDoesNotDeleteGatewayAPIResources verifies that cleanupIngress
// does not delete Gateway or HTTPRoute resources, as it only handles Ingress cleanup.
func TestCleanupIngressDoesNotDeleteGatewayAPIResources(t *testing.T) {
	ctx := context.Background()

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{},
	}

	existingObjects := []ctrlruntimeclient.Object{
		cfg,
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubermatic",
				Namespace: "kubermatic",
			},
		},
		&gatewayapiv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubermatic",
				Namespace: "kubermatic",
			},
		},
		&gatewayapiv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubermatic",
				Namespace: "kubermatic",
			},
		},
	}

	client := fake.
		NewClientBuilder().
		WithObjects(existingObjects...).
		Build()

	versions := kubermaticversion.Versions{
		KubermaticContainerTag: "latest",
		UIContainerTag:         "latest",
	}

	reconciler := &Reconciler{
		Client:   client,
		log:      zap.NewNop().Sugar(),
		recorder: record.NewFakeRecorder(100),
		scheme:   fake.NewScheme(),
		versions: versions,
	}

	// Run cleanupIngress - should only delete Ingress
	err := reconciler.cleanupIngress(ctx, cfg, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("cleanupIngress failed: %v", err)
	}

	// Verify Ingress is deleted
	ingress := &networkingv1.Ingress{}
	err = client.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, ingress)
	if !apierrors.IsNotFound(err) {
		t.Errorf("Expected Ingress to be deleted, got: %v", err)
	}

	// Verify Gateway still exists (cleanupIngress doesn't touch it)
	gw := &gatewayapiv1.Gateway{}
	err = client.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, gw)
	if err != nil {
		t.Errorf("Expected Gateway to still exist, got: %v", err)
	}

	// Verify HTTPRoute still exists (cleanupIngress doesn't touch it)
	hr := &gatewayapiv1.HTTPRoute{}
	err = client.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "kubermatic"}, hr)
	if err != nil {
		t.Errorf("Expected HTTPRoute to still exist, got: %v", err)
	}
}
