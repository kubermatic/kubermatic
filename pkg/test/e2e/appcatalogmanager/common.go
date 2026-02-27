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
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"go.uber.org/zap"

	catalogv1alpha1 "k8c.io/application-catalog-manager/pkg/apis/applicationcatalog/v1alpha1"
	catalogcharts "k8c.io/application-catalog-manager/pkg/charts"
	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	applicationcatalogmanager "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/application-catalog"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	kwait "k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var kubermaticConfigurationName string

func init() {
	utils.DefaultLogOptions.AddFlags(flag.CommandLine)
	flag.StringVar(&kubermaticConfigurationName, "kubermatic-configuration-name", "kubermatic", "Name of the KubermaticConfiguration")
}

func getKubermaticConfigurationName() string {
	if !flag.Parsed() {
		flag.Parse()
	}
	return kubermaticConfigurationName
}

const (
	DefaultInterval = 5 * time.Second
	DefaultTimeout  = 10 * time.Minute

	LabelManagedByApplicationCatalog = applicationcatalogmanager.LabelManagedByApplicationCatalog
	LabelApplicationCatalogName      = applicationcatalogmanager.LabelApplicationCatalogName

	IncludeAnnotation = "defaultcatalog.k8c.io/include"

	ApplicationCatalogManagerDeploymentName = applicationcatalogmanager.ApplicationCatalogManagerDeploymentName
	ApplicationCatalogServiceAccountName    = applicationcatalogmanager.ApplicationCatalogServiceAccountName
	ApplicationCatalogWebhookDeploymentName = applicationcatalogmanager.ApplicationCatalogWebhookDeploymentName
	ApplicationCatalogWebhookServiceName    = applicationcatalogmanager.ApplicationCatalogWebhookServiceName
	ApplicationCatalogCRDName               = applicationcatalogmanager.ApplicationCatalogCRDName
	DefaultApplicationCatalogName           = applicationcatalogmanager.DefaultApplicationCatalogName
)

func verifyApplicationCatalogCRDExists(ctx context.Context, logger *zap.SugaredLogger, seedClient ctrlruntimeclient.Client) error {
	err := wait.PollUntilContextTimeout(ctx, DefaultInterval, DefaultTimeout, true, func(ctx context.Context) (bool, error) {
		crdList := &apiextensionsv1.CustomResourceDefinitionList{}
		if err := seedClient.List(ctx, crdList); err != nil {
			return false, err
		}

		for _, crd := range crdList.Items {
			if crd.Name == ApplicationCatalogCRDName {
				logger.Infof("ApplicationCatalog CRD found: %s", crd.Name)
				return true, nil
			}
		}
		return false, nil
	})
	return err
}

func verifyWebhookService(ctx context.Context, logger *zap.SugaredLogger, seedClient ctrlruntimeclient.Client) error {
	svc := &corev1.Service{}
	return wait.PollUntilContextTimeout(ctx, DefaultInterval, DefaultTimeout, true, func(ctx context.Context) (bool, error) {
		err := seedClient.Get(ctx, types.NamespacedName{
			Namespace: "kubermatic",
			Name:      ApplicationCatalogWebhookServiceName,
		}, svc)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		var hasPort443 bool
		for _, port := range svc.Spec.Ports {
			if port.Port == 443 {
				hasPort443 = true
				break
			}
		}
		if !hasPort443 {
			logger.Warn("Webhook service does not accept traffic on port 443 yet")
			return false, nil
		}

		return true, nil
	})
}

func enableFeatureGateInConfig(ctx context.Context, logger *zap.SugaredLogger, seedClient ctrlruntimeclient.Client) error {
	logger.Info("Enabling ExternalApplicationCatalogManager feature gate...")

	cfg, err := getKubermaticConfiguration(ctx, seedClient, "kubermatic", getKubermaticConfigurationName())
	if err != nil {
		return fmt.Errorf("Failed to get KubermaticConfiguration: %w", err)
	}

	if cfg.Spec.FeatureGates != nil && cfg.Spec.FeatureGates[features.ExternalApplicationCatalogManager] {
		cfg.Spec.FeatureGates[features.ExternalApplicationCatalogManager] = false

		err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 3*time.Minute, true, func(ctx context.Context) (bool, error) {
			if err := seedClient.Update(ctx, cfg); err != nil {
				logger.Infow("failed to update the KubermaticConfiguration", "error", err)
				return false, nil
			}

			return true, nil
		})
		if err != nil {
			return fmt.Errorf("timed out waiting for application-catalog-manager deployment after feature gate enablement: %w", err)
		}
	}

	err = EnableFeatureGateInConfig(ctx, seedClient, cfg)
	if err != nil {
		return fmt.Errorf("failed to enable ExternalApplicationCatalogManager feature gate: %w", err)
	}

	logger.Info("Feature gate enabled, waiting for operator reconciliation...")
	return nil
}

func ensureCleanState(ctx context.Context, logger *zap.SugaredLogger, seedClient ctrlruntimeclient.Client) error {
	namespace := "kubermatic"

	logger.Info("Ensuring clean state for Application Catalog Manager tests...")

	cfg, err := getKubermaticConfiguration(ctx, seedClient, namespace, getKubermaticConfigurationName())
	if err != nil {
		return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}

	logger.Info("ExternalApplicationCatalogManager feature gate is enabled, disabling...")

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
		err := disableFeatureGateInConfig(ctx, seedClient, cfg)
		if err != nil {
			logger.Infow("failed to disable feature gate", "error", err)
			return false, nil
		}

		err = disableOldCatalogMethod(ctx, seedClient, cfg)
		if err != nil {
			logger.Infow("failed to disable old catalog method", "error", err)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("timed out waiting to disable ExternalApplicationCatalogManager feature gate: %w", err)
	}

	logger.Info("Feature gate disabled, waiting for deployments to be removed...")

	err = wait.PollUntilContextTimeout(ctx, DefaultInterval, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		err1 := WaitForDeploymentDeleted(ctx, seedClient, namespace, ApplicationCatalogManagerDeploymentName)
		err2 := WaitForDeploymentDeleted(ctx, seedClient, namespace, ApplicationCatalogWebhookDeploymentName)

		if err1 == nil && err2 == nil {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return fmt.Errorf("timed out waiting for application-catalog-manager deployments to be removed: %w", err)
	}

	logger.Info("Deleting ApplicationCatalog instances...")
	_, err = deleteAllApplicationCatalogs(ctx, seedClient)
	if err != nil {
		return fmt.Errorf("failed to delete ApplicationCatalog instances: %w", err)
	}

	err = waitForAllApplicationCatalogsDeleted(ctx, logger, seedClient)
	if err != nil {
		return fmt.Errorf("timed out waiting for ApplicationCatalog instances to be deleted: %w", err)
	}

	logger.Info("Clearing ApplicationDefinition filtering...")
	if cfg.Spec.Applications.CatalogManager.Apps != nil {
		err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(ctx context.Context) (bool, error) {
			cfg, err = getKubermaticConfiguration(ctx, seedClient, namespace, getKubermaticConfigurationName())
			if err != nil {
				return false, fmt.Errorf("failed to re-fetch KubermaticConfiguration: %w", err)
			}

			cfg.Spec.Applications.CatalogManager.Apps = nil
			if err := seedClient.Update(ctx, cfg); err != nil {
				logger.Infow("failed to update KubermaticConfiguration", "error", err)
				return false, nil
			}

			return true, nil
		})
		if err != nil {
			return fmt.Errorf("timed out waiting to clear ApplicationDefinition filtering: %w", err)
		}

		logger.Info("ApplicationDefinition filtering cleared")
	}

	logger.Info("Deleting non-system ApplicationDefinitions...")

	deletedNames, err := waitDeleteNonSystemApplicationDefinitions(ctx, logger, seedClient)
	if err != nil {
		return fmt.Errorf("failed to delete non-system ApplicationDefinitions: %w", err)
	}

	if err := waitForNonSystemApplicationDefinitionsDeleted(ctx, logger, seedClient); err != nil {
		return fmt.Errorf("timed out waiting for ApplicationDefinitions to be deleted: %w", err)
	}

	if len(deletedNames) > 0 {
		logger.Infof("Deleted %d non-system ApplicationDefinitions: %v", len(deletedNames), deletedNames)
	} else {
		logger.Info("No non-system ApplicationDefinitions to delete")
	}

	logger.Info("Clean state ensured")
	return nil
}

func getKubermaticConfiguration(ctx context.Context, client ctrlruntimeclient.Client, namespace, name string) (*kubermaticv1.KubermaticConfiguration, error) {
	var cfg kubermaticv1.KubermaticConfiguration
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &cfg); err != nil {
		return nil, fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
	}
	return &cfg, nil
}

// EnableFeatureGateInConfig enables the ExternalApplicationCatalogManager feature gate in KubermaticConfiguration.
func EnableFeatureGateInConfig(ctx context.Context, client ctrlruntimeclient.Client, cfg *kubermaticv1.KubermaticConfiguration) error {
	if cfg.Spec.FeatureGates == nil {
		cfg.Spec.FeatureGates = make(map[string]bool)
	}
	cfg.Spec.FeatureGates[features.ExternalApplicationCatalogManager] = true
	return client.Update(ctx, cfg)
}

// disableFeatureGateInConfig disables the ExternalApplicationCatalogManager feature gate in KubermaticConfiguration.
func disableFeatureGateInConfig(ctx context.Context, client ctrlruntimeclient.Client, cfg *kubermaticv1.KubermaticConfiguration) error {
	if cfg.Spec.FeatureGates == nil {
		return nil
	}
	delete(cfg.Spec.FeatureGates, features.ExternalApplicationCatalogManager)
	return client.Update(ctx, cfg)
}

func disableOldCatalogMethod(ctx context.Context, client ctrlruntimeclient.Client, cfg *kubermaticv1.KubermaticConfiguration) error {
	cfg.Spec.Applications.DefaultApplicationCatalog.Enable = false
	return client.Update(ctx, cfg)
}

// verifyClusterRole verifies that the ClusterRole for application-catalog-manager
// has the expected API group rules.
func verifyClusterRole(ctx context.Context, client ctrlruntimeclient.Client, name string) error {
	var cr rbacv1.ClusterRole
	if err := client.Get(ctx, types.NamespacedName{Name: name}, &cr); err != nil {
		return fmt.Errorf("failed to get ClusterRole: %w", err)
	}

	expectedGroups := map[string]bool{
		"applicationcatalog.k8c.io": true,
		"apps.kubermatic.k8c.io":    true,
	}

	for _, rule := range cr.Rules {
		for _, apiGroup := range rule.APIGroups {
			if expectedGroups[apiGroup] {
				delete(expectedGroups, apiGroup)
			}
		}
	}

	if len(expectedGroups) > 0 {
		var missing []string
		for group := range expectedGroups {
			missing = append(missing, group)
		}
		return fmt.Errorf("ClusterRole missing expected API groups: %v", missing)
	}

	return nil
}

func WaitForDeploymentDeleted(ctx context.Context, client ctrlruntimeclient.Client, namespace, name string) error {
	return kwait.PollImmediateLog(ctx, zap.NewNop().Sugar(), DefaultInterval, DefaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		var deployment appsv1.Deployment
		err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &deployment)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to get deployment: %w", err)
		}
		return fmt.Errorf("deployment %q still exists", name), nil
	})
}

func deleteAllApplicationCatalogs(ctx context.Context, client ctrlruntimeclient.Client) ([]string, error) {
	list := &catalogv1alpha1.ApplicationCatalogList{}
	if err := client.List(ctx, list); err != nil {
		if apierrors.IsNotFound(err) {
			return []string{}, nil
		}
		var noKindMatchErr *meta.NoKindMatchError
		if errors.As(err, &noKindMatchErr) {
			return []string{}, nil
		}

		return nil, fmt.Errorf("failed to list ApplicationCatalogs: %w", err)
	}

	var deletedNames []string
	for i := range list.Items {
		catalog := &list.Items[i]
		if err := client.Delete(ctx, catalog); err != nil {
			if !apierrors.IsNotFound(err) {
				return deletedNames, fmt.Errorf("failed to delete ApplicationCatalog %q: %w", catalog.Name, err)
			}
		}
		deletedNames = append(deletedNames, catalog.Name)
	}

	return deletedNames, nil
}

func waitForAllApplicationCatalogsDeleted(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	return kwait.PollImmediateLog(ctx, logger, DefaultInterval, DefaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		list := &catalogv1alpha1.ApplicationCatalogList{}
		if err := client.List(ctx, list); err != nil {
			return nil, fmt.Errorf("failed to list ApplicationCatalogs: %w", err)
		}

		if len(list.Items) > 0 {
			names := make([]string, len(list.Items))
			for i := range list.Items {
				names[i] = list.Items[i].Name
			}
			return fmt.Errorf("%d ApplicationCatalogs still exist: %v", len(list.Items), names), nil
		}

		return nil, nil
	})
}

func waitDeleteNonSystemApplicationDefinitions(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client) ([]string, error) {
	var deletedNames []string
	var err error
	err = kwait.PollImmediateLog(ctx, logger, DefaultInterval, 3*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		deletedNames, err = deleteNonSystemApplicationDefinitions(ctx, logger, client)
		if err != nil {
			return fmt.Errorf("failed to delete non-system ApplicationDefinitions: %w", err), nil
		}

		return nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("timed out waiting to clear ApplicationDefinition filtering: %w", err)
	}

	return deletedNames, nil
}

func waitForNonSystemApplicationDefinitionsDeleted(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client) error {
	return kwait.PollImmediateLog(ctx, logger, DefaultInterval, 1*time.Minute, func(ctx context.Context) (transient error, terminal error) {
		list := &appskubermaticv1.ApplicationDefinitionList{}
		if err := client.List(ctx, list); err != nil {
			return nil, fmt.Errorf("failed to list ApplicationDefinitions: %w", err)
		}

		exists := []string{}

		errs := make([]error, 0)
		for _, appDef := range list.Items {
			if !isSystemApplication(&appDef) {
				logger.Infof("Non-system ApplicationDefinition %q still exists, forcing reconciliation", appDef.Name)
				exists = append(exists, appDef.Name)

				oldAppDef := appDef.DeepCopy()
				if appDef.Annotations == nil {
					appDef.Annotations = make(map[string]string)
				}

				appDef.Annotations["kubermatic.io/force-reconciliation"] = uuid.New().String()

				err := client.Patch(ctx, &appDef, ctrlruntimeclient.MergeFrom(oldAppDef))
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to patch ApplicationDefinition %q: %w", appDef.Name, err))
				}
			}
		}
		if len(exists) > 0 {
			return fmt.Errorf("non-system ApplicationDefinitions still exist: %+v", exists), nil
		}
		if len(errs) > 0 {
			return kerrors.NewAggregate(errs), nil
		}

		return nil, nil
	})
}

func deleteApplicationCatalog(ctx context.Context, client ctrlruntimeclient.Client, name string) error {
	catalog := &catalogv1alpha1.ApplicationCatalog{}
	err := client.Get(ctx, types.NamespacedName{Name: name}, catalog)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get ApplicationCatalog: %w", err)
	}

	return client.Delete(ctx, catalog)
}

func isSystemApplication(appDef *appskubermaticv1.ApplicationDefinition) bool {
	if appDef.Labels == nil {
		return false
	}
	_, exists := appDef.Labels[appskubermaticv1.ApplicationManagedByLabel]
	return exists
}

func deleteNonSystemApplicationDefinitions(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client) ([]string, error) {
	list := &appskubermaticv1.ApplicationDefinitionList{}
	if err := client.List(ctx, list); err != nil {
		return nil, fmt.Errorf("failed to list ApplicationDefinitions: %w", err)
	}

	var deletedNames []string
	for i := range list.Items {
		appDef := &list.Items[i]
		if !isSystemApplication(appDef) {
			if err := client.Delete(ctx, appDef); err != nil {
				logger.Warnf("Failed to delete ApplicationDefinition", "error", err)
				if !apierrors.IsNotFound(err) {
					return deletedNames, fmt.Errorf("failed to delete ApplicationDefinition %q: %w", appDef.Name, err)
				}
			}
			deletedNames = append(deletedNames, appDef.Name)
		}
	}

	return deletedNames, nil
}

// CatalogOption is a functional option for configuring ApplicationCatalog.
type CatalogOption func(*catalogv1alpha1.ApplicationCatalog)

// createApplicationCatalog creates an ApplicationCatalog with the given options.
func createApplicationCatalog(ctx context.Context, client ctrlruntimeclient.Client, name string, opts ...CatalogOption) (*catalogv1alpha1.ApplicationCatalog, error) {
	catalog := &catalogv1alpha1.ApplicationCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	for _, opt := range opts {
		opt(catalog)
	}

	if catalog.Spec.Helm == nil {
		catalog.Spec.Helm = &catalogv1alpha1.HelmSpec{}
	}

	if err := client.Create(ctx, catalog); err != nil {
		return nil, fmt.Errorf("failed to create ApplicationCatalog: %w", err)
	}

	return catalog, nil
}

func WithIncludeDefaults() CatalogOption {
	return func(c *catalogv1alpha1.ApplicationCatalog) {
		if c.Spec.Helm == nil {
			c.Spec.Helm = &catalogv1alpha1.HelmSpec{}
		}
		c.Spec.Helm.IncludeDefaults = true
	}
}

func WithIncludeAnnotation(apps ...string) CatalogOption {
	return func(c *catalogv1alpha1.ApplicationCatalog) {
		if c.Annotations == nil {
			c.Annotations = make(map[string]string)
		}
		c.Annotations[IncludeAnnotation] = strings.Join(apps, ",")
	}
}

func WithCharts(charts ...catalogv1alpha1.ChartConfig) CatalogOption {
	return func(c *catalogv1alpha1.ApplicationCatalog) {
		if c.Spec.Helm == nil {
			c.Spec.Helm = &catalogv1alpha1.HelmSpec{}
		}
		c.Spec.Helm.Charts = charts
	}
}

func WithRepositoryURL(url string) CatalogOption {
	return func(c *catalogv1alpha1.ApplicationCatalog) {
		if c.Spec.Helm == nil {
			c.Spec.Helm = &catalogv1alpha1.HelmSpec{}
		}
		if c.Spec.Helm.RepositorySettings == nil {
			c.Spec.Helm.RepositorySettings = &catalogv1alpha1.RepositorySettings{}
		}
		c.Spec.Helm.RepositorySettings.BaseURL = url
	}
}

func getDefaultChartNames() []string {
	dc := catalogcharts.GetDefaultCharts()
	r := make([]string, len(dc))

	for i, chart := range dc {
		r[i] = chart.ChartName
	}

	return r
}

func getDefaultAppNames() []string {
	dc := catalogcharts.GetDefaultCharts()
	r := make([]string, len(dc))

	for i, chart := range dc {
		r[i] = chart.Metadata.AppName
	}

	return r
}

func countApplicationDefinitionFiles() (int, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return 0, fmt.Errorf("failed to get caller information")
	}

	baseDir := filepath.Join(path.Dir(filename), "../../../ee/default-application-catalog/applicationdefinitions")

	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory %q: %w", baseDir, err)
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			count++
		}
	}

	return count, nil
}

// getApplicationCatalog returns an ApplicationCatalog by name.
func getApplicationCatalog(
	ctx context.Context,
	logger *zap.SugaredLogger,
	client ctrlruntimeclient.Client,
	name string,
) (*catalogv1alpha1.ApplicationCatalog, error) {
	var catalog catalogv1alpha1.ApplicationCatalog
	err := wait.PollUntilContextTimeout(ctx, 2*time.Second, 1*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		if err := client.Get(ctx, types.NamespacedName{Name: name}, &catalog); err != nil {
			logger.Infow("failed to get ApplicationCatalog", "error", err)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get ApplicationCatalog %q: %w", name, err)
	}

	return &catalog, nil
}

func waitForDeploymentReady(ctx context.Context, client ctrlruntimeclient.Client, namespace, name string) error {
	return kwait.PollImmediateLog(ctx, zap.NewNop().Sugar(), DefaultInterval, DefaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		var deployment appsv1.Deployment
		err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &deployment)
		if err != nil {
			return fmt.Errorf("failed to get deployment: %w", err), nil
		}

		if deployment.Status.ReadyReplicas != *deployment.Spec.Replicas {
			return fmt.Errorf("deployment not ready: %d/%d replicas ready", deployment.Status.ReadyReplicas, *deployment.Spec.Replicas), nil
		}

		return nil, nil
	})
}

func waitForApplicationDefinition(ctx context.Context, client ctrlruntimeclient.Client, name string) (*appskubermaticv1.ApplicationDefinition, error) {
	var appDef appskubermaticv1.ApplicationDefinition

	err := kwait.PollImmediateLog(ctx, zap.NewNop().Sugar(), DefaultInterval, DefaultTimeout, func(ctx context.Context) (transient error, terminal error) {
		err := client.Get(ctx, types.NamespacedName{Name: name}, &appDef)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Errorf("ApplicationDefinition %q not found yet", name), nil
			}
			return nil, fmt.Errorf("failed to get ApplicationDefinition: %w", err)
		}
		return nil, nil
	})

	if err != nil {
		return nil, fmt.Errorf("timed out waiting for ApplicationDefinition %q: %w", name, err)
	}

	return &appDef, nil
}

// waitForApplicationDefinitions waits for multiple ApplicationDefinitions to be created.
func waitForApplicationDefinitions(ctx context.Context, client ctrlruntimeclient.Client, names ...string) (map[string]*appskubermaticv1.ApplicationDefinition, error) {
	result := make(map[string]*appskubermaticv1.ApplicationDefinition)

	for _, name := range names {
		appDef, err := waitForApplicationDefinition(ctx, client, name)
		if err != nil {
			return nil, err
		}
		result[name] = appDef
	}

	return result, nil
}

// verifyOwnershipLabels checks if an ApplicationDefinition has the correct ownership labels.
func verifyOwnershipLabels(appDef *appskubermaticv1.ApplicationDefinition, catalogName string) bool {
	if appDef.Labels == nil {
		return false
	}

	if appDef.Labels[catalogv1alpha1.LabelManagedByApplicationCatalog] != "true" {
		return false
	}

	if appDef.Labels[catalogv1alpha1.LabelApplicationCatalogName] != catalogName {
		return false
	}

	return true
}

func updateKubermaticConfiguration(ctx context.Context, logger *zap.SugaredLogger, client ctrlruntimeclient.Client, config *kubermaticv1.KubermaticConfiguration) error {
	return wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		err = client.Update(ctx, config)
		if err != nil {
			logger.Infow("failed to update KubermaticConfiguration", "error", err)
			return false, nil
		}

		return true, nil
	})
}

func updateApplicationCatalog(ctx context.Context, client ctrlruntimeclient.Client, catalog *catalogv1alpha1.ApplicationCatalog) error {
	return client.Update(ctx, catalog)
}

func listApplicationDefinitionsWithoutCatalogLabels(ctx context.Context, client ctrlruntimeclient.Client) ([]appskubermaticv1.ApplicationDefinition, error) {
	list := &appskubermaticv1.ApplicationDefinitionList{}
	if err := client.List(ctx, list); err != nil {
		return nil, fmt.Errorf("failed to list ApplicationDefinitions: %w", err)
	}

	var result []appskubermaticv1.ApplicationDefinition
	for _, appDef := range list.Items {
		if !hasCatalogOwnershipLabels(&appDef) && !isSystemApplication(&appDef) {
			result = append(result, appDef)
		}
	}

	return result, nil
}

// listApplicationDefinitionsWithCatalogLabels lists ApplicationDefinitions that have
// the applicationcatalog.k8c.io/managed-by: "true" label, optionally filtered by catalog name.
func listApplicationDefinitionsWithCatalogLabels(ctx context.Context, client ctrlruntimeclient.Client, catalogName string) ([]appskubermaticv1.ApplicationDefinition, error) {
	list := &appskubermaticv1.ApplicationDefinitionList{}
	if err := client.List(ctx, list); err != nil {
		return nil, fmt.Errorf("failed to list ApplicationDefinitions: %w", err)
	}

	var result []appskubermaticv1.ApplicationDefinition
	for _, appDef := range list.Items {
		if hasCatalogOwnershipLabels(&appDef) {
			if catalogName == "" || appDef.Labels[catalogv1alpha1.LabelApplicationCatalogName] == catalogName {
				result = append(result, appDef)
			}
		}
	}

	return result, nil
}

// hasCatalogOwnershipLabels checks if an ApplicationDefinition has the new-style
// catalog ownership labels (applicationcatalog.k8c.io/managed-by and catalog-name).
func hasCatalogOwnershipLabels(appDef *appskubermaticv1.ApplicationDefinition) bool {
	if appDef.Labels == nil {
		return false
	}

	if appDef.Labels[catalogv1alpha1.LabelManagedByApplicationCatalog] != "true" {
		return false
	}

	if appDef.Labels[catalogv1alpha1.LabelApplicationCatalogName] == "" {
		return false
	}

	return true
}

func getApplicationDefinitionNames(apps []appskubermaticv1.ApplicationDefinition) []string {
	names := make([]string, len(apps))
	for i, app := range apps {
		names[i] = app.Name
	}
	return names
}

// compareApplicationDefinitionSpecs performs a deep comparison of the Spec field
// between old and new ApplicationDefinitions. Returns a detailed error if Specs differ.
func compareApplicationDefinitionSpecs(oldApps, newApps []appskubermaticv1.ApplicationDefinition) error {
	sort.Slice(oldApps, func(i, j int) bool {
		return oldApps[i].Name < oldApps[j].Name
	})
	sort.Slice(newApps, func(i, j int) bool {
		return newApps[i].Name < newApps[j].Name
	})

	oldAppsMap := make(map[string]appskubermaticv1.ApplicationDefinition)
	for _, app := range oldApps {
		oldAppsMap[app.Name] = app
	}

	newAppsMap := make(map[string]appskubermaticv1.ApplicationDefinition)
	for _, app := range newApps {
		newAppsMap[app.Name] = app
	}

	hashLogo := cmp.FilterPath(func(p cmp.Path) bool {
		return p.Last().String() == ".Logo"
	}, cmp.Transformer("HashLogo", func(s string) string {
		if s == "" {
			return "(empty)"
		}
		return fmt.Sprintf("MD5:%x (len=%d)", md5.Sum([]byte(s)), len(s))
	}))
	optTreatNilAsFalse := cmp.Comparer(func(x, y *bool) bool {
		vx := x != nil && *x
		vy := y != nil && *y
		return vx == vy
	})
	opts := []cmp.Option{
		cmpopts.EquateEmpty(),
		hashLogo,
		optTreatNilAsFalse,
	}

	for name, oldApp := range oldAppsMap {
		newApp, exists := newAppsMap[name]
		if !exists {
			continue
		}

		oldSpec := normalizeSpec(oldApp.Spec)
		newSpec := normalizeSpec(newApp.Spec)

		if diff := cmp.Diff(oldSpec, newSpec, opts...); diff != "" {
			fmt.Printf("===> [%s]:\n%s\n\n", name, diff)
			return fmt.Errorf("spec mismatch for ApplicationDefinition %q:\n%s", name, diff)
		}
	}

	return nil
}

// normalizeSpec creates a normalized copy of ApplicationDefinitionSpec
// with all nested slices sorted for consistent comparison.
func normalizeSpec(spec appskubermaticv1.ApplicationDefinitionSpec) appskubermaticv1.ApplicationDefinitionSpec {
	normalized := spec.DeepCopy()

	if len(normalized.Versions) > 0 {
		sort.Slice(normalized.Versions, func(i, j int) bool {
			return normalized.Versions[i].Version < normalized.Versions[j].Version
		})
	}

	if normalized.Selector.Datacenters != nil {
		sort.Strings(normalized.Selector.Datacenters)
	}

	return *normalized
}
