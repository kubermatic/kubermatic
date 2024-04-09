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
	"testing"

	"k8c.io/kubermatic/v2/pkg/addon"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func installAddon(t *testing.T, client ctrlruntimeclient.Client, provider kubermaticv1.ProviderType, addonName string, allAddons map[string]*addon.Addon) *kubermaticv1.Cluster {
	for _, required := range RequiredAddons[addonName] {
		t.Logf("Installing required %s addon…", required)
		installAddon(t, client, provider, required, allAddons)
	}

	cluster := loadCluster(t, provider)
	setupClusterForAddon(t, cluster, addonName)

	data := getTemplateData(t, cluster)

	addonObj := allAddons[addonName]
	if addonObj == nil {
		t.Fatalf("No such addon: %s", addonName)
	}

	manifests, err := allAddons[addonName].Render("", data)
	if err != nil {
		t.Fatalf("Failed to render addon %s: %v", addonName, err)
	}

	setupEnvForAddon(t, client, provider, addonName, allAddons)
	applyManifests(t, client, manifests)

	return cluster
}

func setupEnvForAddon(t *testing.T, client ctrlruntimeclient.Client, provider kubermaticv1.ProviderType, addonName string, allAddons map[string]*addon.Addon) {
	// NOP
}

func setupClusterForAddon(t *testing.T, cluster *kubermaticv1.Cluster, addonName string) {
	// NOP
}

// setupEnvForUpgrade simulates the migration steps the KKP installer or addon controller
// might perform between KKP releases. Re-implementing those steps in a cheap way here is
// easier than turning this test case into a full-blown e2e test.
func setupEnvForUpgrade(t *testing.T, client ctrlruntimeclient.Client, provider kubermaticv1.ProviderType, cluster *kubermaticv1.Cluster, addonName string) {
	ctx := context.Background()

	for _, required := range RequiredAddons[addonName] {
		setupEnvForUpgrade(t, client, provider, cluster, required)
	}

	// failed to update object: CSIDriver.storage.k8s.io "csi.vsphere.vmware.com" is invalid: spec.podInfoOnMount: Invalid value: false: field is immutable
	// https://github.com/kubermatic/kubermatic/pull/12936
	if addonName == "csi" && provider == kubermaticv1.VSphereCloudProvider {
		driver := &storagev1.CSIDriver{}
		driver.Name = "csi.vsphere.vmware.com"
		if err := client.Delete(ctx, driver); err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("Failed to delete VSphere CSI Driver: %v", err)
		}
	}

	// ClusterRoleBinding for Azure's CSI was changed.
	// https://github.com/kubermatic/kubermatic/pull/13250
	if addonName == "csi" && provider == kubermaticv1.AzureCloudProvider {
		driver := &rbacv1.ClusterRoleBinding{}
		driver.Name = "csi-azuredisk-node-secret-binding"
		if err := client.Delete(ctx, driver); err != nil && !apierrors.IsNotFound(err) {
			t.Fatalf("Failed to delete Azure CSI ClusterRoleBinding: %v", err)
		}
	}
}
