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

package appcatalogmanager

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	catalogv1alpha1 "k8c.io/application-catalog-manager/pkg/apis/applicationcatalog/v1alpha1"
	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	applicationcatalogmanager "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/application-catalog"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

func TestApplicationCatalogManagerDeployment(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1)).Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	err = ensureCleanState(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("Failed to ensure clean state: %v", err)
	}

	err = enableFeatureGateInConfig(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("Failed to enable feature gate: %v", err)
	}

	namespace := "kubermatic"

	err = waitForDeploymentReady(ctx, seedClient, namespace, ApplicationCatalogManagerDeploymentName)
	if err != nil {
		t.Fatalf("application-catalog-manager deployment not ready: %v", err)
	}

	logger.Info("Verifying application-catalog-webhook deployment...")
	err = waitForDeploymentReady(ctx, seedClient, namespace, ApplicationCatalogWebhookDeploymentName)
	if err != nil {
		t.Fatalf("application-catalog-webhook deployment not ready: %v", err)
	}

	logger.Info("Application Catalog Manager deployment is ready")

	logger.Info("Verifying ApplicationCatalog CRD exists...")
	err = verifyApplicationCatalogCRDExists(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("ApplicationCatalog CRD verification failed: %v", err)
	}

	logger.Info("Verifying webhook service...")
	err = verifyWebhookService(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("webhook service verification failed: %v", err)
	}
	logger.Info("Webhook service verified")

	sa := &corev1.ServiceAccount{}
	err = seedClient.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      ApplicationCatalogServiceAccountName,
	}, sa)
	if err != nil {
		t.Fatalf("ServiceAccount not found: %v", err)
	}
	logger.Info("ServiceAccount verified")

	cfg, err := getKubermaticConfiguration(ctx, seedClient, namespace, getKubermaticConfigurationName())
	if err != nil {
		t.Fatalf("Failed to get KubermaticConfiguration: %v", err)
	}

	crbName := applicationcatalogmanager.CatalogManagerClusterRoleName(cfg)
	crb := &rbacv1.ClusterRoleBinding{}
	err = seedClient.Get(ctx, types.NamespacedName{Name: crbName}, crb)
	if err != nil {
		t.Fatalf("ClusterRoleBinding not found: %v", err)
	}

	if crb.RoleRef.Name != crbName {
		t.Errorf("ClusterRoleBinding references wrong ClusterRole: %s", crb.RoleRef.Name)
	}

	if len(crb.Subjects) != 1 {
		t.Fatalf("Expected 1 subject in ClusterRoleBinding, got %d", len(crb.Subjects))
	}

	if crb.Subjects[0].Name != ApplicationCatalogServiceAccountName {
		t.Errorf("ClusterRoleBinding subject has wrong name: %s", crb.Subjects[0].Name)
	}

	if crb.Subjects[0].Namespace != namespace {
		t.Errorf("ClusterRoleBinding subject has wrong namespace: %s", crb.Subjects[0].Namespace)
	}

	logger.Info("Verifying ClusterRole...")
	err = wait.PollUntilContextTimeout(ctx, DefaultInterval, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		if err := verifyClusterRole(ctx, seedClient, applicationcatalogmanager.CatalogManagerClusterRoleName(cfg)); err != nil {
			logger.Infow("ClusterRole verification failed, retrying", "error", err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("ClusterRole verification failed: %v", err)
	}

	err = wait.PollUntilContextTimeout(ctx, DefaultInterval, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		currentCatalog, err := getApplicationCatalog(ctx, logger, seedClient, DefaultApplicationCatalogName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		if currentCatalog.Spec.Helm == nil {
			return false, nil
		}

		if len(currentCatalog.Spec.Helm.Charts) == 0 {
			return false, nil
		}

		defaultCharts := getDefaultChartNames()
		if len(currentCatalog.Spec.Helm.Charts) != len(defaultCharts) {
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed waiting for default charts injection: %v", err)
	}

	logger.Info("All deployment verification tests passed!")
}

func TestMultiCatalogSupport(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(utils.DefaultLogOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	err = ensureCleanState(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("Failed to ensure clean state: %v", err)
	}

	err = enableFeatureGateInConfig(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("Failed to enable feature gate: %v", err)
	}

	catalogAName := "test-catalog-a"
	catalogBName := "test-catalog-b"
	chartAName := "catalog-a-app"
	chartBName := "catalog-b-app"

	logger.Info("Testing multi-catalog support")

	_ = deleteApplicationCatalog(ctx, seedClient, catalogAName)
	_ = deleteApplicationCatalog(ctx, seedClient, catalogBName)
	_ = seedClient.Delete(ctx, &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: chartAName},
	})
	_ = seedClient.Delete(ctx, &appskubermaticv1.ApplicationDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: chartBName},
	})

	logger.Infof("Creating catalog %q with chart %q", catalogAName, chartAName)
	chartA := catalogv1alpha1.ChartConfig{
		ChartName: chartAName,
		Metadata: &catalogv1alpha1.ChartMetadata{
			DisplayName: "Catalog A App",
			Description: "Application from catalog A",
		},
		ChartVersions: []catalogv1alpha1.ChartVersion{
			{ChartVersion: "1.0.0", AppVersion: "v1.0.0"},
		},
	}

	_, err = createApplicationCatalog(
		ctx,
		seedClient,
		catalogAName,
		WithCharts(chartA),
	)
	if err != nil {
		t.Fatalf("Failed to create ApplicationCatalog %q: %v", catalogAName, err)
	}
	t.Cleanup(func() {
		err := deleteApplicationCatalog(ctx, seedClient, catalogAName)
		if err != nil {
			logger.Warnf("Failed to delete ApplicationCatalog %q: %v", catalogAName, err)
		}
		err = seedClient.Delete(ctx, &appskubermaticv1.ApplicationDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: chartAName},
		})
		if err != nil {
			logger.Warnf("Failed to delete ApplicationDefinition %q: %v", chartAName, err)
		}
	})

	logger.Infof("Creating catalog %q with chart %q", catalogBName, chartBName)
	chartB := catalogv1alpha1.ChartConfig{
		ChartName: chartBName,
		Metadata: &catalogv1alpha1.ChartMetadata{
			DisplayName: "Catalog B App",
			Description: "Application from catalog B",
		},
		ChartVersions: []catalogv1alpha1.ChartVersion{
			{ChartVersion: "1.0.0", AppVersion: "v1.0.0"},
		},
	}

	_, err = createApplicationCatalog(
		ctx,
		seedClient,
		catalogBName,
		WithCharts(chartB),
	)
	if err != nil {
		t.Fatalf("Failed to create ApplicationCatalog %q: %v", catalogBName, err)
	}
	t.Cleanup(func() {
		err = deleteApplicationCatalog(ctx, seedClient, catalogBName)
		if err != nil {
			logger.Warnf("Failed to delete ApplicationCatalog %q: %v", catalogBName, err)
		}
		err = seedClient.Delete(ctx, &appskubermaticv1.ApplicationDefinition{
			ObjectMeta: metav1.ObjectMeta{Name: chartBName},
		})
		if err != nil {
			logger.Warnf("Failed to delete ApplicationDefinition %q: %v", chartBName, err)
		}
	})

	logger.Info("Waiting for ApplicationDefinitions from both catalogs...")

	appDefs, err := waitForApplicationDefinitions(ctx, seedClient, chartAName, chartBName)
	if err != nil {
		t.Fatalf("Failed waiting for ApplicationDefinitions: %v", err)
	}

	appDefA := appDefs[chartAName]
	if !verifyOwnershipLabels(appDefA, catalogAName) {
		t.Errorf("ApplicationDefinition %q should be owned by catalog %q", chartAName, catalogAName)
	}

	appDefB := appDefs[chartBName]
	if !verifyOwnershipLabels(appDefB, catalogBName) {
		t.Errorf("ApplicationDefinition %q should be owned by catalog %q", chartBName, catalogBName)
	}

	if appDefA.Labels[catalogv1alpha1.LabelApplicationCatalogName] != catalogAName {
		t.Errorf(
			"ApplicationDefinition %q has wrong catalog-name label: %q",
			chartAName, appDefA.Labels[catalogv1alpha1.LabelApplicationCatalogName],
		)
	}

	if appDefB.Labels[catalogv1alpha1.LabelApplicationCatalogName] != catalogBName {
		t.Errorf(
			"ApplicationDefinition %q has wrong catalog-name label: %q",
			chartBName, appDefB.Labels[catalogv1alpha1.LabelApplicationCatalogName],
		)
	}

	originalDescriptionB := appDefs[chartBName].Spec.Description

	logger.Infof("Updating catalog %q...", catalogAName)
	catalogA, err := getApplicationCatalog(ctx, logger, seedClient, catalogAName)
	if err != nil {
		t.Fatalf("Failed to get ApplicationCatalog: %v", err)
	}

	if catalogA.Spec.Helm == nil || len(catalogA.Spec.Helm.Charts) == 0 {
		t.Fatal("Catalog A has no charts")
	}

	catalogAUpdatedDesc := "Updated description for app A"
	catalogA.Spec.Helm.Charts[0].Metadata.Description = catalogAUpdatedDesc
	err = updateApplicationCatalog(ctx, seedClient, catalogA)
	if err != nil {
		t.Fatalf("Failed to update ApplicationCatalog: %v", err)
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		currentAppDefA := &appskubermaticv1.ApplicationDefinition{}
		err := seedClient.Get(ctx, types.NamespacedName{Name: chartAName}, currentAppDefA)
		if err != nil {
			logger.Infow("Failed to get ApplicationDefinition A", "error", err)
			return false, nil
		}

		if currentAppDefA.Spec.Description != catalogAUpdatedDesc {
			logger.Infow("ApplicationDefinition A description not updated yet", "current", currentAppDefA.Spec.Description)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed waiting for ApplicationDefinition update: %v", err)
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		currentAppDefB := &appskubermaticv1.ApplicationDefinition{}
		err := seedClient.Get(ctx, types.NamespacedName{Name: chartBName}, currentAppDefB)
		if err != nil {
			logger.Infow("Failed to get ApplicationDefinition B", "error", err)
			return false, nil
		}

		if currentAppDefB.Spec.Description != originalDescriptionB {
			return false, fmt.Errorf("AppDefB should not be updated when Catalog A is updated")
		}

		if !verifyOwnershipLabels(currentAppDefB, catalogBName) {
			return false, fmt.Errorf("ApplicationDefinition %q lost ownership from catalog %q", chartBName, catalogBName)
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed waiting for ApplicationDefinition update: %v", err)
	}

	logger.Info("Successfully verified both catalogs manage their own ApplicationDefinitions")
}

func TestWebhookPreservesCustomCharts(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(utils.DefaultLogOptions)
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))
	logger := rawLogger.Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	err = ensureCleanState(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("Failed to ensure clean state: %v", err)
	}

	err = enableFeatureGateInConfig(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("Failed to enable feature gate: %v", err)
	}

	err = waitForDeploymentReady(ctx, seedClient, "kubermatic", ApplicationCatalogManagerDeploymentName)
	if err != nil {
		t.Fatalf("application-catalog-manager deployment not ready: %v", err)
	}

	logger.Info("Verifying application-catalog-webhook deployment...")
	err = waitForDeploymentReady(ctx, seedClient, "kubermatic", ApplicationCatalogWebhookDeploymentName)
	if err != nil {
		t.Fatalf("application-catalog-webhook deployment not ready: %v", err)
	}

	defaultCatalog, err := getApplicationCatalog(ctx, logger, seedClient, "default-catalog")
	if err != nil {
		t.Fatalf("Failed to get default ApplicationCatalog: %v", err)
	}

	customChart := catalogv1alpha1.ChartConfig{
		ChartName: "custom-test-app",
		Metadata: &catalogv1alpha1.ChartMetadata{
			DisplayName: "Custom Test App",
			Description: "A custom test application",
		},
		ChartVersions: []catalogv1alpha1.ChartVersion{
			{
				ChartVersion: "1.0.0",
				AppVersion:   "v1.0.0",
			},
		},
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		defaultCatalog.Spec.Helm.Charts = append(defaultCatalog.Spec.Helm.Charts, customChart)
		err = updateApplicationCatalog(ctx, seedClient, defaultCatalog)
		if err != nil {
			logger.Infow("Failed to update ApplicationCatalog with custom chart", "error", err)
			return false, nil
		}

		logger.Infow("updated default ApplicationCatlaog and added a new custom chart")
		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed waiting for custom chart to be processed: %v", err)
	}

	defaultCharts := getDefaultChartNames()
	expectedCount := len(defaultCharts) + 1
	logger.Info("Waiting for webhook to process and merge charts...")

	err = wait.PollUntilContextTimeout(ctx, DefaultInterval, DefaultTimeout, true, func(ctx context.Context) (bool, error) {
		currentCatalog, err := getApplicationCatalog(ctx, logger, seedClient, defaultCatalog.Name)
		if err != nil {
			logger.Infow("Failed to get ApplicationCatalog", "error", err)
			return false, nil
		}

		if currentCatalog.Spec.Helm == nil {
			logger.Infow("Helm spec is nil in ApplicationCatalog", "name", defaultCatalog.Name)
			return false, nil
		}

		if len(currentCatalog.Spec.Helm.Charts) != expectedCount {
			logger.Infow(
				"Chart count mismatch in ApplicationCatalog",
				"expected", expectedCount,
				"got", len(currentCatalog.Spec.Helm.Charts),
			)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed waiting for charts merge: %v", err)
	}

	updatedCatalog, err := getApplicationCatalog(ctx, logger, seedClient, defaultCatalog.Name)
	if err != nil {
		t.Fatalf("Failed to get updated ApplicationCatalog: %v", err)
	}

	if updatedCatalog.Spec.Helm == nil {
		t.Fatalf("Helm spec is nil after webhook processing")
	}

	charts := updatedCatalog.Spec.Helm.Charts
	if len(charts) != expectedCount {
		t.Errorf("Expected %d total charts, got %d", expectedCount, len(charts))
	}

	customChartFound := false
	for _, chart := range charts {
		if chart.ChartName == "custom-test-app" {
			customChartFound = true
			if chart.Metadata == nil || chart.Metadata.DisplayName != "Custom Test App" {
				t.Error("Custom chart metadata was not preserved")
			}

			break
		}
	}

	if !customChartFound {
		t.Error("Custom chart was not preserved after webhook processing")
	}

	logger.Infof("Successfully verified custom chart preservation with %d total charts", len(charts))
}

func TestAppCatalogMigration(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t).WithOptions(zap.AddCallerSkip(1)).Sugar()

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("Failed to build client: %v", err)
	}

	namespace := "kubermatic"

	err = ensureCleanState(ctx, logger, seedClient)
	if err != nil {
		t.Fatalf("Failed to ensure clean state: %v", err)
	}

	cfg, err := getKubermaticConfiguration(ctx, seedClient, namespace, getKubermaticConfigurationName())
	if err != nil {
		t.Fatalf("Failed to get KubermaticConfiguration: %v", err)
	}

	cfg.Spec.Applications.DefaultApplicationCatalog.Enable = true
	err = updateKubermaticConfiguration(ctx, logger, seedClient, cfg)
	if err != nil {
		t.Fatalf("Failed to update KubermaticConfiguration: %v", err)
	}

	var oldApps []appskubermaticv1.ApplicationDefinition
	err = wait.PollUntilContextTimeout(ctx, DefaultInterval, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
		oldApps, err = listApplicationDefinitionsWithoutCatalogLabels(ctx, seedClient)
		if err != nil {
			logger.Infow("failed to list application definitions", "error", err)
			return false, nil
		}

		if len(oldApps) == 0 {
			logger.Info("No old-style applications found yet")
			return false, nil
		}

		chartCount, err := countApplicationDefinitionFiles()
		if err != nil {
			logger.Infow("failed to count application definition files", "error", err)
			return false, nil
		}

		if len(oldApps) != chartCount {
			logger.Infof("Waiting for all old-style applications to be created... Current: %d, Expected: %d", len(oldApps), chartCount)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed waiting for default app catalog (old method) to be enabled: %v", err)
	}

	logger.Info("Enabling ExternalApplicationCatalogManager feature gate...")
	config, err := getKubermaticConfiguration(ctx, seedClient, namespace, getKubermaticConfigurationName())
	if err != nil {
		t.Fatalf("Failed to get KubermaticConfiguration: %v", err)
	}

	err = EnableFeatureGateInConfig(ctx, seedClient, config)
	if err != nil {
		t.Fatalf("Failed to enable feature gate: %v", err)
	}

	logger.Info("Waiting for application-catalog-manager deployment...")
	err = waitForDeploymentReady(ctx, seedClient, namespace, ApplicationCatalogManagerDeploymentName)
	if err != nil {
		t.Fatalf("application-catalog-manager deployment not ready: %v", err)
	}

	logger.Info("Waiting for application-catalog-webhook deployment...")
	err = waitForDeploymentReady(ctx, seedClient, namespace, ApplicationCatalogWebhookDeploymentName)
	if err != nil {
		t.Fatalf("application-catalog-webhook deployment not ready: %v", err)
	}

	logger.Info("Waiting for default-catalog ApplicationCatalog CR...")
	err = wait.PollUntilContextTimeout(ctx, DefaultInterval, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err := getApplicationCatalog(ctx, logger, seedClient, DefaultApplicationCatalogName)
		if err != nil {
			logger.Infow("failed to get applicationcatalog", "err", err)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("default-catalog ApplicationCatalog not found: %v", err)
	}

	logger.Info("Waiting for default charts to be injected...")
	expectedNewApps := getDefaultAppNames()
	err = wait.PollUntilContextTimeout(ctx, DefaultInterval, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		catalog, err := getApplicationCatalog(ctx, logger, seedClient, DefaultApplicationCatalogName)
		if err != nil {
			return false, err
		}

		if catalog.Spec.Helm == nil {
			return false, nil
		}

		if len(catalog.Spec.Helm.Charts) != len(expectedNewApps) {
			logger.Infof("Waiting for charts: got %d, want %d", len(catalog.Spec.Helm.Charts), len(expectedNewApps))
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("default charts not injected: %v", err)
	}

	newApps := []appskubermaticv1.ApplicationDefinition{}
	logger.Info("Listing new-style applications (with catalog labels)...")
	err = wait.PollUntilContextTimeout(ctx, DefaultInterval, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		newApps, err = listApplicationDefinitionsWithCatalogLabels(ctx, seedClient, DefaultApplicationCatalogName)
		if err != nil {
			logger.Infow("failed to list new-style applications", "error", err)
			return false, nil
		}

		logger.Infof("Found %d new-style applications", len(newApps))
		if len(newApps) != len(expectedNewApps) {
			sort.Slice(expectedNewApps, func(i, j int) bool {
				return expectedNewApps[i] < expectedNewApps[j]
			})
			sort.Slice(newApps, func(i, j int) bool {
				return newApps[i].Name < newApps[j].Name
			})

			diff := cmp.Diff(expectedNewApps, getApplicationDefinitionNames(newApps))
			logger.Infof("\n\n`+` means newApps\n`-` means expected DefaultApps\n\n%s", diff)
			return false, nil
		}

		logger.Info("Verifying ownership labels on new-style applications...")
		errs := make([]error, 0)
		for _, app := range newApps {
			if !verifyOwnershipLabels(&app, DefaultApplicationCatalogName) {
				errs = append(errs, fmt.Errorf("application %q does not have correct ownership labels", app.Name))
			}
		}
		if len(errs) > 0 {
			return false, fmt.Errorf("ownership label verification failed: %+v", errs)
		}

		logger.Info("Comparing Specs between old and new ApplicationDefinitions...")
		if err := compareApplicationDefinitionSpecs(oldApps, newApps); err != nil {
			logger.Infow("Spec comparison failed", "error", err)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("default charts not injected: %v", err)
	}

	logger.Infof("Migration test completed successfully - verified %d applications", len(newApps))
}
