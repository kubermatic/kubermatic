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
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/addon"
	addonutils "k8c.io/kubermatic/v2/pkg/addon"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	currentAddonsPath  string
	previousAddonsPath string
)

func init() {
	flag.StringVar(&currentAddonsPath, "current", "../../../addons", "path to all addons for the current KKP version")
	flag.StringVar(&previousAddonsPath, "previous", "", "path to all addons for the current KKP version")

	ctrlruntimelog.SetLogger(zapr.NewLogger(zap.NewNop()))
}

func TestAddonsCanBeUpgraded(t *testing.T) {
	currentAddons, err := addonutils.LoadAddonsFromDirectory(currentAddonsPath)
	if err != nil {
		t.Fatalf("Failed to load current addons from %s: %v", currentAddonsPath, err)
	}

	if previousAddonsPath == "" {
		testAddonsCanBeApplied(t, currentAddons)
	} else {
		previousAddons, err := addonutils.LoadAddonsFromDirectory(previousAddonsPath)
		if err != nil {
			t.Fatalf("Failed to load previous addons from %s: %v", previousAddonsPath, err)
		}

		testAddonsCanBeUpgraded(t, previousAddons, currentAddons)
	}
}

func testAddonsCanBeApplied(t *testing.T, addons map[string]*addon.Addon) {
	for _, addonName := range sets.List(sets.KeySet(addons)) {
		providersToTest, exists := AddonProviderMatrix[addonName]
		if !exists {
			t.Fatalf("No providers configured for addon %s, please update pkg/test/addon/matrix.go.", addonName)
		}

		for _, provider := range providersToTest {
			t.Run(fmt.Sprintf("%s@%s", addonName, provider), func(t *testing.T) {
				testAddonCanBeApplied(t, addonName, provider, addons)
			})
		}
	}
}

func testAddonCanBeApplied(t *testing.T, addonName string, provider kubermaticv1.ProviderType, allAddons map[string]*addon.Addon) {
	_, client := createTestEnv(t)
	ctx := context.Background()

	installAddon(ctx, t, client, provider, addonName, allAddons)
}

func testAddonsCanBeUpgraded(t *testing.T, previousAddons, currentAddons map[string]*addon.Addon) {
	previousAddonNames := sets.KeySet(previousAddons)

	for _, addonName := range sets.List(previousAddonNames) {
		providersToTest, exists := AddonProviderMatrix[addonName]
		if !exists {
			t.Fatalf("No providers configured for addon %s, please update pkg/test/addon/matrix.go.", addonName)
		}

		for _, provider := range providersToTest {
			t.Run(fmt.Sprintf("%s@%s", addonName, provider), func(t *testing.T) {
				testAddonCanBeUpgraded(t, addonName, provider, previousAddons, currentAddons)
			})
		}
	}

	// for all newly added addons, still test if they apply
	newAddonNames := sets.KeySet(currentAddons).Difference(previousAddonNames)

	for _, addonName := range sets.List(newAddonNames) {
		providersToTest, exists := AddonProviderMatrix[addonName]
		if !exists {
			t.Fatalf("No providers configured for addon %s, please update pkg/test/addon/matrix.go.", addonName)
		}

		for _, provider := range providersToTest {
			t.Run(fmt.Sprintf("%s@%s", addonName, provider), func(t *testing.T) {
				testAddonCanBeApplied(t, addonName, provider, currentAddons)
			})
		}
	}
}

func testAddonCanBeUpgraded(t *testing.T, addonName string, provider kubermaticv1.ProviderType, previousAddons, currentAddons map[string]*addon.Addon) {
	_, client := createTestEnv(t)
	ctx := context.Background()

	t.Log("Applying previous manifests…")
	installAddon(ctx, t, client, provider, addonName, previousAddons)

	if _, ok := currentAddons[addonName]; ok {
		t.Log("Applying current manifests…")
		installAddon(ctx, t, client, provider, addonName, currentAddons)
	} else {
		t.Log("Addon was deleted, no upgrade possible.")
	}
}

func applyManifests(t *testing.T, client ctrlruntimeclient.Client, manifests []runtime.RawExtension) {
	success := true
	for _, manifest := range manifests {
		if err := applyManifest(t, client, manifest); err != nil {
			t.Errorf("Failed to apply: %v", err)
			success = false
		}
	}

	if !success {
		t.FailNow()
	}
}

func applyManifest(t *testing.T, client ctrlruntimeclient.Client, manifest runtime.RawExtension) error {
	kubeObj := &unstructured.Unstructured{}
	if _, _, err := unstructured.UnstructuredJSONScheme.Decode(manifest.Raw, nil, kubeObj); err != nil {
		return fmt.Errorf("invalid object in rendered YAML: %w", err)
	}

	ctx := context.Background()

	existing := kubeObj.DeepCopy()
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(existing), existing); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get %s %s: %w", kubeObj.GetKind(), kubeObj.GetName(), err)
		}

		// t.Logf("Creating %s %s…", kubeObj.GetKind(), kubeObj.GetName())
		if err := client.Create(ctx, kubeObj); err != nil {
			return fmt.Errorf("failed to create object: %w", err)
		}

		if kubeObj.GetKind() == "CustomResourceDefinition" {
			waitForEstablishedCRD(t, client, kubeObj.GetName())
		}
	} else {
		kubeObj.SetResourceVersion(existing.GetResourceVersion())

		// t.Logf("Updating %s %s…", kubeObj.GetKind(), kubeObj.GetName())
		if err := client.Update(ctx, kubeObj); err != nil {
			return fmt.Errorf("failed to update object: %w", err)
		}
	}

	return nil
}

func waitForEstablishedCRD(t *testing.T, client ctrlruntimeclient.Client, crdName string) {
	key := types.NamespacedName{Name: crdName}

	err := wait.Poll(context.Background(), 100*time.Millisecond, 5*time.Second, func(ctx context.Context) (transient error, terminal error) {
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := client.Get(ctx, key, crd); err != nil {
			return err, nil
		}

		for _, cond := range crd.Status.Conditions {
			if cond.Type == apiextensionsv1.Established {
				if cond.Status == apiextensionsv1.ConditionTrue {
					return nil, nil
				}

				return fmt.Errorf("%s condition is %s", apiextensionsv1.Established, cond.Status), nil
			}
		}

		return fmt.Errorf("CRD has no %s condition", apiextensionsv1.Established), nil
	})
	if err != nil {
		t.Fatalf("Failed to wait for CRD: %v", err)
	}
}

func createTestEnv(t *testing.T) (*rest.Config, ctrlruntimeclient.Client) {
	testEnv := &envtest.Environment{}

	// disable webhooks; it's impossible for the target services to ever become ready
	apiserver := testEnv.ControlPlane.GetAPIServer()
	args := apiserver.Configure()
	args.Append("disable-admission-plugins", "ValidatingAdmissionWebhook", "MutatingAdmissionWebhook")

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("Failed to start envTest: %v", err)
	}
	t.Cleanup(func() {
		if err := testEnv.Stop(); err != nil {
			t.Fatalf("Failed to stop envTest: %v", err)
		}
	})

	scheme := runtime.NewScheme()
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to setup scheme: %v", err)
	}
	if err := rbacv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to setup scheme: %v", err)
	}
	if err := storagev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to setup scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to setup scheme: %v", err)
	}
	if err := policyv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to setup scheme: %v", err)
	}

	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		t.Fatalf("Failed to create kube client: %v", err)
	}

	return cfg, client
}

func getTemplateData(t *testing.T, cluster *kubermaticv1.Cluster) *addonutils.TemplateData {
	data, err := addonutils.NewTemplateData(
		cluster,
		resources.Credentials{},
		"<kubeconfig>",
		"1.2.3.4", // DNS cluster IP
		"5.6.7.8", // DNS resolver IP
		nil,       // IPAM allocations
		nil,       // extra vars
	)
	if err != nil {
		t.Fatalf("Failed to create template data: %v", err)
	}

	return data
}

func loadCluster(t *testing.T, provider kubermaticv1.ProviderType) *kubermaticv1.Cluster {
	clusterFile := fmt.Sprintf("data/cluster-%s.yaml", provider)

	content, err := os.ReadFile(clusterFile)
	if err != nil {
		t.Fatalf("Failed to read cluster file: %v", err)
	}

	raw := &unstructured.Unstructured{}
	if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(content), 1024).Decode(raw); err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	cluster := &kubermaticv1.Cluster{}
	if err = yaml.UnmarshalStrict(content, cluster); err != nil {
		t.Fatalf("File is not a valid Cluster: %v", err)
	}

	// always auto-assume the current stable Kubernetes version, nobody will want
	// to keep bumping those YAML files...
	defaultVersion := *defaulting.DefaultKubernetesVersioning.Default

	cluster.Spec.Version = defaultVersion
	cluster.Status.Versions = kubermaticv1.ClusterVersionsStatus{
		ControlPlane:      defaultVersion,
		Apiserver:         defaultVersion,
		ControllerManager: defaultVersion,
		Scheduler:         defaultVersion,
	}

	return cluster
}
