/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package applicationdefinitions

import (
	"testing"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testAppDefName    = "test-app"
	testDefaultValues = "key: value\n"
	testUpdatedValues = "key: updated-value\n"
	testAdminValues   = "key: admin-override\n"
)

func makeAppDef(name, defaultValuesBlock string, annotations map[string]string) *appskubermaticv1.ApplicationDefinition {
	return &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
		Spec: appskubermaticv1.ApplicationDefinitionSpec{
			DefaultValuesBlock: defaultValuesBlock,
			Versions: []appskubermaticv1.ApplicationVersion{
				{
					Version: "v1.0.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								URL:          "https://example.com/chart",
								ChartName:    "test-chart",
								ChartVersion: "1.0.0",
							},
						},
					},
				},
			},
		},
	}
}

func reconcile(fileAppDef, clusterAppDef *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error) {
	factory := systemApplicationDefinitionReconcilerFactory(fileAppDef, nil, false)
	_, reconciler := factory()
	return reconciler(clusterAppDef)
}

func TestDefaultValuesBlockReconciliation(t *testing.T) {
	tests := []struct {
		name               string
		fileValues         string
		clusterValues      string
		clusterAnnotations map[string]string
		wantValues         string
		wantHashOf         string // the value whose sha1 should appear in the annotation
	}{
		{
			name:          "fresh install, empty cluster",
			fileValues:    testDefaultValues,
			clusterValues: "",
			wantValues:    testDefaultValues,
			wantHashOf:    testDefaultValues,
		},
		{
			name:          "fresh install, cluster has empty JSON",
			fileValues:    testDefaultValues,
			clusterValues: "{}",
			wantValues:    testDefaultValues,
			wantHashOf:    testDefaultValues,
		},
		{
			name:          "upgrade, no admin edit (cluster matches file)",
			fileValues:    testDefaultValues,
			clusterValues: testDefaultValues,
			wantValues:    testDefaultValues,
			wantHashOf:    testDefaultValues,
		},
		{
			name:          "upgrade, admin edited (cluster differs from file, no annotation)",
			fileValues:    testUpdatedValues,
			clusterValues: testAdminValues,
			wantValues:    testAdminValues,
			wantHashOf:    testUpdatedValues,
		},
		{
			name:          "steady state, no admin edit (file changed, propagate upstream)",
			fileValues:    testUpdatedValues,
			clusterValues: testDefaultValues,
			clusterAnnotations: map[string]string{
				appskubermaticv1.ApplicationDefaultValuesHashAnnotation: sha1Hex(testDefaultValues),
			},
			wantValues: testUpdatedValues,
			wantHashOf: testUpdatedValues,
		},
		{
			name:          "steady state, admin edited",
			fileValues:    testUpdatedValues,
			clusterValues: testAdminValues,
			clusterAnnotations: map[string]string{
				appskubermaticv1.ApplicationDefaultValuesHashAnnotation: sha1Hex(testDefaultValues),
			},
			wantValues: testAdminValues,
			wantHashOf: testUpdatedValues,
		},
		{
			name:          "admin clears field (empty), annotation exists",
			fileValues:    testUpdatedValues,
			clusterValues: "",
			clusterAnnotations: map[string]string{
				appskubermaticv1.ApplicationDefaultValuesHashAnnotation: sha1Hex(testDefaultValues),
			},
			wantValues: testUpdatedValues,
			wantHashOf: testUpdatedValues,
		},
		{
			name:          "admin sets to empty JSON, annotation exists",
			fileValues:    testUpdatedValues,
			clusterValues: "{}",
			clusterAnnotations: map[string]string{
				appskubermaticv1.ApplicationDefaultValuesHashAnnotation: sha1Hex(testDefaultValues),
			},
			wantValues: testUpdatedValues,
			wantHashOf: testUpdatedValues,
		},
		{
			name:          "file becomes empty, no admin edit",
			fileValues:    "",
			clusterValues: testDefaultValues,
			clusterAnnotations: map[string]string{
				appskubermaticv1.ApplicationDefaultValuesHashAnnotation: sha1Hex(testDefaultValues),
			},
			wantValues: "",
			wantHashOf: "",
		},
		{
			name:          "file becomes empty, admin edited",
			fileValues:    "",
			clusterValues: testAdminValues,
			clusterAnnotations: map[string]string{
				appskubermaticv1.ApplicationDefaultValuesHashAnnotation: sha1Hex(testDefaultValues),
			},
			wantValues: testAdminValues,
			wantHashOf: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileAppDef := makeAppDef(testAppDefName, tt.fileValues, nil)
			clusterAppDef := makeAppDef(testAppDefName, tt.clusterValues, tt.clusterAnnotations)

			result, err := reconcile(fileAppDef, clusterAppDef)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Spec.DefaultValuesBlock != tt.wantValues {
				t.Errorf("DefaultValuesBlock: got %q, want %q", result.Spec.DefaultValuesBlock, tt.wantValues)
			}

			expectedHash := sha1Hex(tt.wantHashOf)
			gotHash := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]
			if gotHash != expectedHash {
				t.Errorf("hash annotation: got %q, want %q", gotHash, expectedHash)
			}
		})
	}
}

// TestSha1Hex verifies the helper produces correct hashes.
func TestSha1Hex(t *testing.T) {
	// known SHA1 of empty string
	if got := sha1Hex(""); got != "da39a3ee5e6b4b0d3255bfef95601890afd80709" {
		t.Errorf("sha1Hex(\"\") = %q, want known hash", got)
	}

	// SHA1 is deterministic
	a := sha1Hex("test")
	b := sha1Hex("test")
	if a != b {
		t.Errorf("sha1Hex not deterministic: %q != %q", a, b)
	}

	// different inputs produce different hashes
	c := sha1Hex("other")
	if a == c {
		t.Errorf("sha1Hex(\"test\") == sha1Hex(\"other\")")
	}
}

// TestReconcilePreservesExistingAnnotations ensures the reconciler does not
// remove pre-existing annotations on the clusterAppDef.
func TestReconcilePreservesExistingAnnotations(t *testing.T) {
	fileAppDef := makeAppDef(testAppDefName, testDefaultValues, nil)
	clusterAppDef := makeAppDef(testAppDefName, "", map[string]string{
		"some-other-annotation": "should-survive",
	})

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := result.Annotations["some-other-annotation"]; got != "should-survive" {
		t.Errorf("expected pre-existing annotation preserved, got %q", got)
	}
}

// TestReconcileWithKubermaticConfig verifies the reconciler correctly applies
// KubermaticConfiguration to the application definition when mirror is false.
func TestReconcileWithKubermaticConfig(t *testing.T) {
	fileAppDef := makeAppDef(testAppDefName, testDefaultValues, nil)
	config := &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
				SystemApplications: kubermaticv1.SystemApplicationsConfiguration{
					HelmRegistryConfigFile: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "test-secret",
						},
						Key: "config.json",
					},
				},
			},
		},
	}
	clusterAppDef := makeAppDef(testAppDefName, "", nil)

	factory := systemApplicationDefinitionReconcilerFactory(fileAppDef, config, false)
	_, reconciler := factory()
	result, err := reconciler(clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// verify credentials were applied
	for _, v := range result.Spec.Versions {
		if v.Template.Source.Helm == nil {
			t.Fatal("expected Helm source to be set")
		}
		if v.Template.Source.Helm.Credentials == nil {
			t.Fatal("expected Helm credentials to be set from config")
		}
	}
}
