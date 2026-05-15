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

package common

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/install/stack"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDirectoryMode os.FileMode = 0o755
	testFileMode      os.FileMode = 0o644
)

func TestEnsureGatewayAPICRDsDoesNotReplaceExistingCRD(t *testing.T) {
	ctx := context.Background()
	crdName := "gateways.gateway.networking.k8s.io"

	chartsDir := t.TempDir()
	crdDir := filepath.Join(chartsDir, EnvoyGatewayControllerChartName, "crd")
	if err := os.MkdirAll(crdDir, testDirectoryMode); err != nil {
		t.Fatalf("failed to create CRD directory: %v", err)
	}

	bundledCRD := []byte(`
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: gateways.gateway.networking.k8s.io
spec:
  group: rewritten.example.com
  names:
    kind: Gateway
    plural: gateways
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
`)
	if err := os.WriteFile(filepath.Join(crdDir, "gateway.yaml"), bundledCRD, testFileMode); err != nil {
		t.Fatalf("failed to write test CRD: %v", err)
	}

	existing := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: crdName},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: "gateway.networking.k8s.io",
			Names: apiextensionsv1.CustomResourceDefinitionNames{
				Kind:   "Gateway",
				Plural: "gateways",
			},
			Scope: apiextensionsv1.NamespaceScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
				{
					Name:    "v1",
					Served:  true,
					Storage: true,
					Schema: &apiextensionsv1.CustomResourceValidation{
						OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{Type: "object"},
					},
				},
			},
		},
		Status: apiextensionsv1.CustomResourceDefinitionStatus{
			Conditions: []apiextensionsv1.CustomResourceDefinitionCondition{
				{
					Type:   apiextensionsv1.Established,
					Status: apiextensionsv1.ConditionTrue,
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	client := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

	if err := EnsureGatewayAPICRDs(ctx, logrus.NewEntry(logrus.New()), client, stack.DeployOptions{ChartsDirectory: chartsDir}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var fetched apiextensionsv1.CustomResourceDefinition
	if err := client.Get(ctx, types.NamespacedName{Name: crdName}, &fetched); err != nil {
		t.Fatalf("failed to get CRD: %v", err)
	}

	if fetched.Spec.Group != "gateway.networking.k8s.io" {
		t.Fatalf("expected existing CRD to remain untouched, got group %q", fetched.Spec.Group)
	}
}

func TestEnsureGatewayAPICRDsIgnoresSkippedControllerChart(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	client := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).Build()
	chartsDir := t.TempDir()

	err := EnsureGatewayAPICRDs(ctx, logrus.NewEntry(logrus.New()), client, stack.DeployOptions{
		ChartsDirectory: chartsDir,
		SkipCharts:      []string{EnvoyGatewayControllerChartName},
	})
	if err == nil {
		t.Fatal("expected bundled Gateway API CRD loading to fail")
	}

	for _, want := range []string{EnvoyGatewayControllerChartName, "must be present even when the controller deployment is skipped"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

func TestEnsureGatewayAPICRDsRejectsMissingOrInvalidBundle(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	testCases := []struct {
		name  string
		setup func(t *testing.T, chartsDir string)
	}{
		{
			name: "missing CRD directory",
		},
		{
			name: "invalid CRD manifest",
			setup: func(t *testing.T, chartsDir string) {
				crdDir := filepath.Join(chartsDir, EnvoyGatewayControllerChartName, "crd")
				if err := os.MkdirAll(crdDir, testDirectoryMode); err != nil {
					t.Fatalf("failed to create CRD directory: %v", err)
				}
				if err := os.WriteFile(filepath.Join(crdDir, "gateway.yaml"), []byte("not: [valid"), testFileMode); err != nil {
					t.Fatalf("failed to create invalid CRD file: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := ctrlruntimefakeclient.NewClientBuilder().WithScheme(scheme).Build()
			chartsDir := t.TempDir()
			if tc.setup != nil {
				tc.setup(t, chartsDir)
			}

			err := EnsureGatewayAPICRDs(ctx, logrus.NewEntry(logrus.New()), client, stack.DeployOptions{ChartsDirectory: chartsDir})
			if err == nil {
				t.Fatal("expected bundled Gateway API CRD loading to fail")
			}

			for _, want := range []string{EnvoyGatewayControllerChartName, "must be present even when the controller deployment is skipped"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected error to contain %q, got %v", want, err)
				}
			}
		})
	}
}
