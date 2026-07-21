//go:build e2e

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

package pipeline

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"testing"
	"time"

	"github.com/onsi/gomega"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	// fileDefaultValuesHashAnnotation mirrors the exported SDK constant
	// appskubermaticv1.ApplicationFileDefaultValuesHashAnnotation
	// (sdk/apis/apps.kubermatic/v1/types.go). The operator stamps it with the
	// SHA1 of the file-embedded defaultValuesBlock on every reconcile.
	fileDefaultValuesHashAnnotation = "apps.kubermatic.k8c.io/file-default-values-hash"

	// systemAppDefName is a system ApplicationDefinition the operator/master reconciler
	// manages from an embedded YAML file (pkg/applicationdefinitions/system-applications).
	// cluster-autoscaler is used rather than cilium because perturbing its defaultValuesBlock
	// does not affect CNI on the shared cluster.
	systemAppDefName = "cluster-autoscaler"

	// adminEditMarker is appended to defaultValuesBlock to simulate an admin customization.
	// It is valid YAML so the validating webhook accepts the update.
	adminEditMarker = "\n# pipeline-e2e-admin-edit\n"

	reconcileWait = 2 * time.Minute
)

// TestApplicationDefinitionDefaultValues guards PR #15691. The operator/master reconciler
// for system ApplicationDefinitions (pkg/applicationdefinitions/application_catalog.go)
// stamps a SHA1 hash of the file-embedded defaultValuesBlock into the
// apps.kubermatic.k8c.io/file-default-values-hash annotation, and uses it to distinguish
// admin edits from stale values so upstream changes propagate on upgrade while admin
// customizations are preserved.
//
// Before the fix the reconciler wrote no hash annotation and simply preserved any non-empty
// defaultValuesBlock. Both assessments below assert on the annotation, so they fail if the
// fix is reverted.
func TestApplicationDefinitionDefaultValues(t *testing.T) {
	client := getClient(t)

	feature := features.New("system ApplicationDefinition defaultValuesBlock hash tracking").
		Assess("operator stamps the file-default-values-hash annotation in steady state", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)

			// on a warm KKP the operator has already reconciled the system app def, so the
			// hash annotation is present and matches the SHA1 of the current (file) value.
			g.Eventually(func(g gomega.Gomega) {
				appDef := &appskubermaticv1.ApplicationDefinition{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: systemAppDefName}, appDef)).To(gomega.Succeed())

				g.Expect(appDef.Annotations).To(gomega.HaveKeyWithValue(
					fileDefaultValuesHashAnnotation, sha1Hex(appDef.Spec.DefaultValuesBlock)),
					"operator did not stamp the file-default-values-hash annotation (PR #15691 regression)")
			}).WithContext(ctx).WithTimeout(reconcileWait).WithPolling(interval).Should(gomega.Succeed())

			return ctx
		}).
		Assess("admin edit to defaultValuesBlock is preserved and hash annotation stays stamped", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			g := gomega.NewWithT(t)

			original := &appskubermaticv1.ApplicationDefinition{}
			g.Expect(client.Get(ctx, types.NamespacedName{Name: systemAppDefName}, original)).To(gomega.Succeed())

			// append a marker so the cluster value diverges from the file value. the
			// annotation from the previous reconcile records the file hash, so the drift
			// is detected as an admin edit and the marker must survive reconcile.
			edited := original.DeepCopy()
			edited.Spec.DefaultValuesBlock = original.Spec.DefaultValuesBlock + adminEditMarker
			g.Expect(client.Update(ctx, edited)).To(gomega.Succeed(), "failed to apply admin edit")

			// the operator keeps the admin value (hash drifted from the recorded file hash)
			// and re-stamps the hash annotation. it never overwrites the marker with the file value.
			g.Consistently(func(g gomega.Gomega) {
				appDef := &appskubermaticv1.ApplicationDefinition{}
				g.Expect(client.Get(ctx, types.NamespacedName{Name: systemAppDefName}, appDef)).To(gomega.Succeed())

				g.Expect(appDef.Spec.DefaultValuesBlock).To(gomega.ContainSubstring(adminEditMarker),
					"operator overwrote the admin defaultValuesBlock edit (PR #15691 regression)")
				g.Expect(appDef.Annotations).To(gomega.HaveKey(fileDefaultValuesHashAnnotation),
					"operator dropped the file-default-values-hash annotation (PR #15691 regression)")
			}).WithContext(ctx).WithTimeout(reconcileWait).WithPolling(interval).Should(gomega.Succeed())

			return ctx
		}).
		Teardown(func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			// remove the admin marker so the shared app def is back to the file value for the
			// next feature/run. the operator reconciles the annotation back on its own.
			appDef := &appskubermaticv1.ApplicationDefinition{}
			if err := client.Get(ctx, types.NamespacedName{Name: systemAppDefName}, appDef); err != nil {
				return ctx
			}

			restored := stripMarker(appDef.Spec.DefaultValuesBlock)
			if restored != appDef.Spec.DefaultValuesBlock {
				appDef.Spec.DefaultValuesBlock = restored
				_ = client.Update(ctx, appDef)
			}
			return ctx
		}).
		Feature()

	testEnv.Test(t, feature)
}

// sha1Hex mirrors the helper in pkg/applicationdefinitions/application_catalog.go used to
// compute the file-default-values-hash annotation value.
func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func stripMarker(s string) string {
	if len(s) >= len(adminEditMarker) && s[len(s)-len(adminEditMarker):] == adminEditMarker {
		return s[:len(s)-len(adminEditMarker)]
	}
	return s
}
