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

package applicationcatalogmanager

import (
	"context"
	"testing"

	"go.uber.org/zap"

	catalogv1alpha1 "k8c.io/application-catalog-manager/pkg/apis/applicationcatalog/v1alpha1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileDefaultApplicationCatalog(t *testing.T) {
	testScheme := runtime.NewScheme()
	if err := catalogv1alpha1.AddToScheme(testScheme); err != nil {
		t.Fatalf("failed to add catalogv1alpha1 to scheme: %v", err)
	}
	if err := kubermaticv1.AddToScheme(testScheme); err != nil {
		t.Fatalf("failed to add kubermaticv1 to scheme: %v", err)
	}

	testCases := []struct {
		name            string
		existingCatalog *catalogv1alpha1.ApplicationCatalog
		config          *kubermaticv1.KubermaticConfiguration
		validate        func(t *testing.T, catalog *catalogv1alpha1.ApplicationCatalog)
		expectError     bool
	}{
		{
			name:            "no annotation when CatalogManager.Applications is empty",
			existingCatalog: nil,
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Applications: kubermaticv1.ApplicationDefinitionsConfiguration{
						CatalogManager: kubermaticv1.CatalogManagerConfiguration{
							Apps: []string{},
						},
					},
				},
			},
			validate: func(t *testing.T, catalog *catalogv1alpha1.ApplicationCatalog) {
				if catalog.Annotations != nil {
					if _, exists := catalog.Annotations[IncludeAnnotation]; exists {
						t.Error("Expected no defaultcatalog.k8c.io/include annotation when Applications is empty")
					}
				}
			},
			expectError: false,
		},
		{
			name:            "no annotation when CatalogManager.Applications is nil",
			existingCatalog: nil,
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Applications: kubermaticv1.ApplicationDefinitionsConfiguration{
						CatalogManager: kubermaticv1.CatalogManagerConfiguration{},
					},
				},
			},
			validate: func(t *testing.T, catalog *catalogv1alpha1.ApplicationCatalog) {
				if catalog.Annotations != nil {
					if _, exists := catalog.Annotations[IncludeAnnotation]; exists {
						t.Error("Expected no defaultcatalog.k8c.io/include annotation when Applications is nil")
					}
				}
			},
			expectError: false,
		},
		{
			name:            "annotation is added when CatalogManager.Applications has values",
			existingCatalog: nil,
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Applications: kubermaticv1.ApplicationDefinitionsConfiguration{
						CatalogManager: kubermaticv1.CatalogManagerConfiguration{
							Apps: []string{"ingress-nginx", "cert-manager", "argo-cd"},
						},
					},
				},
			},
			validate: func(t *testing.T, catalog *catalogv1alpha1.ApplicationCatalog) {
				if catalog.Annotations == nil {
					t.Error("Expected annotations to be initialized")
					return
				}
				annotation, exists := catalog.Annotations[IncludeAnnotation]
				if !exists {
					t.Error("Expected defaultcatalog.k8c.io/include annotation to exist")
					return
				}
				expected := "ingress-nginx,cert-manager,argo-cd"
				if annotation != expected {
					t.Errorf("Expected annotation value %q, got %q", expected, annotation)
				}
			},
			expectError: false,
		},
		{
			name: "annotation is updated when CatalogManager.Applications changes",
			existingCatalog: &catalogv1alpha1.ApplicationCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: DefaultApplicationCatalogName,
					Annotations: map[string]string{
						IncludeAnnotation: "old-app-1,old-app-2",
					},
				},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Applications: kubermaticv1.ApplicationDefinitionsConfiguration{
						CatalogManager: kubermaticv1.CatalogManagerConfiguration{
							Apps: []string{"new-app-1", "new-app-2"},
						},
					},
				},
			},
			validate: func(t *testing.T, catalog *catalogv1alpha1.ApplicationCatalog) {
				if catalog.Annotations == nil {
					t.Error("Expected annotations to be initialized")
					return
				}
				annotation, exists := catalog.Annotations[IncludeAnnotation]
				if !exists {
					t.Error("Expected defaultcatalog.k8c.io/include annotation to exist")
					return
				}
				expected := "new-app-1,new-app-2"
				if annotation != expected {
					t.Errorf("Expected annotation value %q, got %q", expected, annotation)
				}
			},
			expectError: false,
		},
		{
			name: "annotation is removed when CatalogManager.Applications becomes empty",
			existingCatalog: &catalogv1alpha1.ApplicationCatalog{
				ObjectMeta: metav1.ObjectMeta{
					Name: DefaultApplicationCatalogName,
					Annotations: map[string]string{
						IncludeAnnotation: "old-app-1,old-app-2",
					},
				},
			},
			config: &kubermaticv1.KubermaticConfiguration{
				Spec: kubermaticv1.KubermaticConfigurationSpec{
					Applications: kubermaticv1.ApplicationDefinitionsConfiguration{
						CatalogManager: kubermaticv1.CatalogManagerConfiguration{
							Apps: []string{},
						},
					},
				},
			},
			validate: func(t *testing.T, catalog *catalogv1alpha1.ApplicationCatalog) {
				if catalog.Annotations != nil {
					if _, exists := catalog.Annotations[IncludeAnnotation]; exists {
						t.Error("Expected defaultcatalog.k8c.io/include annotation to be removed when Applications is empty")
					}
				}
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger := zap.NewNop().Sugar()

			var client ctrlruntimeclient.Client
			if tc.existingCatalog != nil {
				client = ctrlruntimefakeclient.NewClientBuilder().WithScheme(testScheme).WithObjects(tc.existingCatalog).Build()
			} else {
				client = ctrlruntimefakeclient.NewClientBuilder().WithScheme(testScheme).Build()
			}

			err := ReconcileDefaultApplicationCatalog(ctx, tc.config, client, logger)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			catalog := &catalogv1alpha1.ApplicationCatalog{}
			err = client.Get(ctx, types.NamespacedName{Name: DefaultApplicationCatalogName}, catalog)
			if err != nil {
				t.Fatalf("Failed to get ApplicationCatalog: %v", err)
			}

			tc.validate(t, catalog)
		})
	}
}
