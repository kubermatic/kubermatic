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
	testAppDefName       = "test-app"
	testDefaultValues    = "key: value\n"
	testUpdatedValues    = "key: updated-value\n"
	testAdminValues      = "key: admin-override\n"
	testValuesBlockEmpty = "{}"
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

func TestFreshInstall(t *testing.T) {
	// clusterAppDef has empty DefaultValuesBlock, no annotation
	fileAppDef := makeAppDef(testAppDefName, testDefaultValues, nil)
	clusterAppDef := makeAppDef(testAppDefName, "", nil)

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != testDefaultValues {
		t.Errorf("expected DefaultValuesBlock %q, got %q", testDefaultValues, result.Spec.DefaultValuesBlock)
	}

	expectedHash := sha1Hex(testDefaultValues)
	if got := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]; got != expectedHash {
		t.Errorf("expected hash annotation %q, got %q", expectedHash, got)
	}
}

func TestFirstUpgradeNoAnnotation(t *testing.T) {
	// clusterAppDef has stale non-empty DefaultValuesBlock, no annotation.
	// file value should overwrite the stale value.
	fileAppDef := makeAppDef(testAppDefName, testUpdatedValues, nil)
	clusterAppDef := makeAppDef(testAppDefName, testDefaultValues, nil)

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != testUpdatedValues {
		t.Errorf("expected DefaultValuesBlock %q, got %q", testUpdatedValues, result.Spec.DefaultValuesBlock)
	}

	expectedHash := sha1Hex(testUpdatedValues)
	if got := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]; got != expectedHash {
		t.Errorf("expected hash annotation %q, got %q", expectedHash, got)
	}
}

func TestReconcileNoAdminChange(t *testing.T) {
	// clusterAppDef has DefaultValuesBlock matching stored hash annotation.
	// file has a new value. expect new file value applied, annotation updated.
	oldHash := sha1Hex(testDefaultValues)
	fileAppDef := makeAppDef(testAppDefName, testUpdatedValues, nil)
	clusterAppDef := makeAppDef(testAppDefName, testDefaultValues, map[string]string{
		appskubermaticv1.ApplicationDefaultValuesHashAnnotation: oldHash,
	})

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != testUpdatedValues {
		t.Errorf("expected DefaultValuesBlock %q, got %q", testUpdatedValues, result.Spec.DefaultValuesBlock)
	}

	expectedHash := sha1Hex(testUpdatedValues)
	if got := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]; got != expectedHash {
		t.Errorf("expected hash annotation %q, got %q", expectedHash, got)
	}
}

func TestAdminCustomizationPreserved(t *testing.T) {
	// clusterAppDef has modified DefaultValuesBlock (hash differs from stored annotation).
	// expect admin value preserved, annotation set to file hash.
	oldHash := sha1Hex(testDefaultValues)
	fileAppDef := makeAppDef(testAppDefName, testUpdatedValues, nil)
	clusterAppDef := makeAppDef(testAppDefName, testAdminValues, map[string]string{
		appskubermaticv1.ApplicationDefaultValuesHashAnnotation: oldHash,
	})

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != testAdminValues {
		t.Errorf("expected admin DefaultValuesBlock %q to be preserved, got %q", testAdminValues, result.Spec.DefaultValuesBlock)
	}

	// annotation should store the file hash, not the admin hash
	expectedHash := sha1Hex(testUpdatedValues)
	if got := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]; got != expectedHash {
		t.Errorf("expected hash annotation %q (file hash), got %q", expectedHash, got)
	}
}

func TestAdminClearsField(t *testing.T) {
	// clusterAppDef has empty DefaultValuesBlock, annotation exists.
	// expect file value applied, annotation updated.
	oldHash := sha1Hex(testDefaultValues)
	fileAppDef := makeAppDef(testAppDefName, testUpdatedValues, nil)
	clusterAppDef := makeAppDef(testAppDefName, "", map[string]string{
		appskubermaticv1.ApplicationDefaultValuesHashAnnotation: oldHash,
	})

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != testUpdatedValues {
		t.Errorf("expected DefaultValuesBlock %q, got %q", testUpdatedValues, result.Spec.DefaultValuesBlock)
	}

	expectedHash := sha1Hex(testUpdatedValues)
	if got := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]; got != expectedHash {
		t.Errorf("expected hash annotation %q, got %q", expectedHash, got)
	}
}

func TestAdminRevertsToSystemDefault(t *testing.T) {
	// clusterAppDef has DefaultValuesBlock matching file value exactly,
	// stored hash from previous file version.
	// expect file value applied (same outcome either way).
	oldHash := sha1Hex("old-value: something\n")
	fileAppDef := makeAppDef(testAppDefName, testUpdatedValues, nil)
	clusterAppDef := makeAppDef(testAppDefName, testUpdatedValues, map[string]string{
		appskubermaticv1.ApplicationDefaultValuesHashAnnotation: oldHash,
	})

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != testUpdatedValues {
		t.Errorf("expected DefaultValuesBlock %q, got %q", testUpdatedValues, result.Spec.DefaultValuesBlock)
	}

	expectedHash := sha1Hex(testUpdatedValues)
	if got := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]; got != expectedHash {
		t.Errorf("expected hash annotation %q, got %q", expectedHash, got)
	}
}

func TestFileBecomesEmpty(t *testing.T) {
	// previous file had values, new file has empty DefaultValuesBlock.
	// no admin override. expect empty value applied.
	oldHash := sha1Hex(testDefaultValues)
	fileAppDef := makeAppDef(testAppDefName, "", nil)
	clusterAppDef := makeAppDef(testAppDefName, testDefaultValues, map[string]string{
		appskubermaticv1.ApplicationDefaultValuesHashAnnotation: oldHash,
	})

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != "" {
		t.Errorf("expected empty DefaultValuesBlock, got %q", result.Spec.DefaultValuesBlock)
	}

	// fileHash for empty string
	expectedHash := sha1Hex("")
	if got := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]; got != expectedHash {
		t.Errorf("expected hash annotation %q, got %q", expectedHash, got)
	}
}

func TestFileBecomesEmptyWithAdminOverride(t *testing.T) {
	// previous file had values, new file has empty DefaultValuesBlock.
	// admin has custom value. expect admin value preserved.
	oldHash := sha1Hex(testDefaultValues)
	fileAppDef := makeAppDef(testAppDefName, "", nil)
	clusterAppDef := makeAppDef(testAppDefName, testAdminValues, map[string]string{
		appskubermaticv1.ApplicationDefaultValuesHashAnnotation: oldHash,
	})

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != testAdminValues {
		t.Errorf("expected admin DefaultValuesBlock %q to be preserved, got %q", testAdminValues, result.Spec.DefaultValuesBlock)
	}

	// annotation should store the file hash (empty), not the admin hash
	expectedHash := sha1Hex("")
	if got := result.Annotations[appskubermaticv1.ApplicationDefaultValuesHashAnnotation]; got != expectedHash {
		t.Errorf("expected hash annotation %q (file hash for empty), got %q", expectedHash, got)
	}
}

func TestClusterValuesBlockEmptyJSONNoAnnotation(t *testing.T) {
	// clusterAppDef has "{}" DefaultValuesBlock, no annotation.
	// should be treated as empty, so file value applies.
	fileAppDef := makeAppDef(testAppDefName, testDefaultValues, nil)
	clusterAppDef := makeAppDef(testAppDefName, testValuesBlockEmpty, nil)

	result, err := reconcile(fileAppDef, clusterAppDef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Spec.DefaultValuesBlock != testDefaultValues {
		t.Errorf("expected DefaultValuesBlock %q, got %q", testDefaultValues, result.Spec.DefaultValuesBlock)
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
