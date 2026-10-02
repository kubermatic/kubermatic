//go:build addon_integration

/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package addon

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/addon/migrations"

	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func installAddon(ctx context.Context, t *testing.T, client ctrlruntimeclient.Client, provider kubermaticv1.ProviderType, addonName string, variant string, allAddons map[string]*addon.Addon) *kubermaticv1.Cluster {
	for _, required := range RequiredAddons[addonName] {
		t.Logf("Installing required %s addon…", required)
		installAddon(ctx, t, client, provider, required, "", allAddons)
	}

	cluster := loadCluster(t, provider)
	setupClusterForAddon(t, cluster, addonName, variant)

	data := getTemplateData(t, cluster)

	addonObj := allAddons[addonName]
	if addonObj == nil {
		t.Fatalf("No such addon: %s", addonName)
	}

	manifests, err := allAddons[addonName].Render("", data)
	if err != nil {
		t.Fatalf("Failed to render addon %s: %v", addonName, err)
	}

	migration := migrations.RelevantMigrations(cluster, addonName)
	log := zap.NewNop().Sugar()

	if err := migration.PreApply(ctx, log, cluster, nil, client); err != nil {
		t.Fatalf("Failed to run preApply migrations: %v", err)
	}

	applyManifests(t, client, manifests)

	if err := migration.PostApply(ctx, log, cluster, nil, client); err != nil {
		t.Fatalf("Failed to run preApply migrations: %v", err)
	}

	return cluster
}

// addonVariants returns the cluster variants an addon needs to be tested with.
// Most addons have a single, unnamed variant that uses the cluster fixture as-is.
// Canal ships one manifest per CNI version, of which only the one matching the
// cluster's CNI version is rendered, so it is tested once per supported version.
func addonVariants(addonName string) []string {
	if addonName == kubermaticv1.CNIPluginTypeCanal.String() {
		versions, err := cni.GetSupportedCNIPluginVersions(kubermaticv1.CNIPluginTypeCanal)
		if err == nil && versions.Len() > 0 {
			return sets.List(versions)
		}
	}

	return []string{""}
}

func testName(addonName string, variant string, provider kubermaticv1.ProviderType) string {
	if variant == "" {
		return fmt.Sprintf("%s@%s", addonName, provider)
	}

	return fmt.Sprintf("%s-%s@%s", addonName, variant, provider)
}

func setupClusterForAddon(t *testing.T, cluster *kubermaticv1.Cluster, addonName string, variant string) {
	if addonName == kubermaticv1.CNIPluginTypeCanal.String() && variant != "" {
		cluster.Spec.CNIPlugin = &kubermaticv1.CNIPluginSettings{
			Type:    kubermaticv1.CNIPluginTypeCanal,
			Version: variant,
		}
	}
}
