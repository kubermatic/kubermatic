/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package template

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"path"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/helmclient"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// HelmTemplate install upgrade or uninstall helm chart into cluster.
type HelmTemplate struct {
	Ctx context.Context

	// Kubeconfig of the user-cluster.
	Kubeconfig string

	// CacheDir is the directory path where helm caches will be download.
	CacheDir string

	Log *zap.SugaredLogger

	// Namespace where credential secrets are stored.
	SecretNamespace string

	// ClusterName of the user-cluster
	ClusterName string

	// SeedClient to seed cluster.
	SeedClient ctrlruntimeclient.Client
}

// InstallOrUpgrade the chart located at chartLoc with parameters (releaseName, values) defined applicationInstallation into cluster.
func (h HelmTemplate) InstallOrUpgrade(chartLoc string, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	helmCacheDir, err := util.CreateHelmTempDir(h.CacheDir)
	if err != nil {
		return util.NoStatusUpdate, err
	}
	defer util.CleanUpHelmTempDir(helmCacheDir, h.Log)

	var auth = helmclient.AuthSettings{}
	source := applicationInstallation.Status.ApplicationVersion.Template.Source.Helm
	if applicationInstallation.Status.ApplicationVersion.Template.DependencyCredentials != nil {
		auth, err = util.HelmAuthFromCredentials(h.Ctx, h.SeedClient, path.Join(helmCacheDir, "reg-creg"), h.SecretNamespace, source, applicationInstallation.Status.ApplicationVersion.Template.DependencyCredentials.HelmCredentials)
		if err != nil {
			return util.NoStatusUpdate, err
		}
	}

	deployOpts, err := getDeployOpts(appDefinition, applicationInstallation)
	if err != nil {
		return util.NoStatusUpdate, err
	}

	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: &h.Kubeconfig,
		Namespace:  &applicationInstallation.Spec.Namespace.Name,
	}

	helmClient, err := helmclient.NewClient(
		h.Ctx,
		restClientGetter,
		helmclient.NewSettings(helmCacheDir),
		applicationInstallation.Spec.Namespace.Name,
		h.Log)

	if err != nil {
		return util.NoStatusUpdate, err
	}

	values, err := applicationInstallation.Spec.GetParsedValues()
	if err != nil {
		return util.NoStatusUpdate, fmt.Errorf("failed to unmarshal values: %w", err)
	}

	renderedValues, err := h.templatePreDefinedValues(values)

	if err != nil {
		return util.NoStatusUpdate, fmt.Errorf("failed to render pre-defined values: %w", err)
	}

	helmRelease, err := helmClient.InstallOrUpgrade(chartLoc, getReleaseName(applicationInstallation), renderedValues, *deployOpts, auth)
	statusUpdater := util.NoStatusUpdate

	// In some case, even if an error occurred, the helmRelease is updated.
	if helmRelease != nil {
		statusUpdater = func(status *appskubermaticv1.ApplicationInstallationStatus) {
			status.HelmRelease = &appskubermaticv1.HelmRelease{
				Name:    helmRelease.Name,
				Version: helmRelease.Version,
				Info: &appskubermaticv1.HelmReleaseInfo{
					FirstDeployed: metav1.Time(helmRelease.Info.FirstDeployed),
					LastDeployed:  metav1.Time(helmRelease.Info.LastDeployed),
					Deleted:       metav1.Time(helmRelease.Info.Deleted),
					Description:   helmRelease.Info.Description,
					Status:        appskubermaticv1.HelmReleaseStatus(helmRelease.Info.Status),
					Notes:         helmRelease.Info.Notes,
				},
			}
		}
	}

	return statusUpdater, err
}

// Uninstall the chart from the user cluster.
func (h HelmTemplate) Uninstall(applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	helmCacheDir, err := util.CreateHelmTempDir(h.CacheDir)
	if err != nil {
		return util.NoStatusUpdate, err
	}
	defer util.CleanUpHelmTempDir(helmCacheDir, h.Log)

	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: &h.Kubeconfig,
		Namespace:  &applicationInstallation.Spec.Namespace.Name,
	}

	helmClient, err := helmclient.NewClient(
		h.Ctx,
		restClientGetter,
		helmclient.NewSettings(helmCacheDir),
		applicationInstallation.Spec.Namespace.Name,
		h.Log)

	if err != nil {
		return util.NoStatusUpdate, err
	}

	uninstallReleaseResponse, err := helmClient.Uninstall(getReleaseName(applicationInstallation))
	statusUpdater := util.NoStatusUpdate

	if uninstallReleaseResponse != nil {
		statusUpdater = func(status *appskubermaticv1.ApplicationInstallationStatus) {
			status.HelmRelease = &appskubermaticv1.HelmRelease{
				Name:    uninstallReleaseResponse.Release.Name,
				Version: uninstallReleaseResponse.Release.Version,
				Info: &appskubermaticv1.HelmReleaseInfo{
					FirstDeployed: metav1.Time(uninstallReleaseResponse.Release.Info.FirstDeployed),
					LastDeployed:  metav1.Time(uninstallReleaseResponse.Release.Info.LastDeployed),
					Deleted:       metav1.Time(uninstallReleaseResponse.Release.Info.Deleted),
					Description:   uninstallReleaseResponse.Release.Info.Description,
					Status:        appskubermaticv1.HelmReleaseStatus(uninstallReleaseResponse.Release.Info.Status),
					Notes:         uninstallReleaseResponse.Release.Info.Notes,
				},
			}
		}
	}

	return statusUpdater, err
}

// getReleaseName computes the release name from the applicationInstallation.
// The releaseName length must be less or equal to 53. So we first start to compute this release Name:
//
//	releaseName := applicationInstallation.Namespace + "-" + applicationInstallation.Name
//
// If the length is more 53 characters then we fall back to:
//
//	releaseName := applicationInstallation.Name[:43] + "-" + sha1Sum(applicationInstallation.Namespace )[:9]
func getReleaseName(applicationInstallation *appskubermaticv1.ApplicationInstallation) string {
	// tech note: in fact releaseName must respect more constraints to be valid cf https://github.com/helm/helm/blob/v3.9.0/pkg/chartutil/validate_name.go#L66
	namespacedName := applicationInstallation.Namespace + "-" + applicationInstallation.Name
	if len(namespacedName) > 53 {
		hash := sha1.New()
		hash.Write([]byte(applicationInstallation.Namespace))

		namespaceSha1 := hex.EncodeToString(hash.Sum(nil))
		appName := applicationInstallation.Name

		if len(appName) > 43 { // 43 = 53 - len( "-" + namespaceSha1[:9])
			appName = appName[:43]
		}
		return appName + "-" + namespaceSha1[:9]
	}
	return namespacedName
}

// getDeployOpts builds helmclient.DeployOpts from values provided by appInstall or fallback to the values of appDefinition or fallback to the default options.
// Default options are wait=false that implies timeout=0 and atomic=false.
func getDeployOpts(appDefinition *appskubermaticv1.ApplicationDefinition, appInstall *appskubermaticv1.ApplicationInstallation) (*helmclient.DeployOpts, error) {
	// Read options from applicationInstallation.
	if appInstall.Spec.DeployOptions != nil && appInstall.Spec.DeployOptions.Helm != nil {
		return helmclient.NewDeployOpts(appInstall.Spec.DeployOptions.Helm.Wait, appInstall.Spec.DeployOptions.Helm.Timeout.Duration, appInstall.Spec.DeployOptions.Helm.Atomic, appInstall.Spec.DeployOptions.Helm.EnableDNS)
	}

	// Fallback to options defined in ApplicationDefinition.
	if appDefinition.Spec.DefaultDeployOptions != nil && appDefinition.Spec.DefaultDeployOptions.Helm != nil {
		return helmclient.NewDeployOpts(appDefinition.Spec.DefaultDeployOptions.Helm.Wait, appDefinition.Spec.DefaultDeployOptions.Helm.Timeout.Duration, appDefinition.Spec.DefaultDeployOptions.Helm.Atomic, appDefinition.Spec.DefaultDeployOptions.Helm.EnableDNS)
	}

	// Fallback to default options.
	return helmclient.NewDeployOpts(false, 0, false, false)
}

// IsStuck aims to identify if a helm release is stuck. This targets an upstream issue in helm, which has not been resolved. For further details see:
// - https://github.com/helm/helm/issues/7476
// - https://github.com/helm/helm/issues/4558
func (h HelmTemplate) IsStuck(applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
	// if the release was successful, exit early
	if applicationInstallation.Status.Conditions[appskubermaticv1.Ready].Status == "True" {
		return false, nil
	}
	// currently we observe the stuck error exclusively with this message. If it does not exist, exit early
	if applicationInstallation.Status.Conditions[appskubermaticv1.Ready].Message != "another operation (install/upgrade/rollback) is in progress" {
		return false, nil
	}

	helmCacheDir, err := util.CreateHelmTempDir(h.CacheDir)
	if err != nil {
		return false, fmt.Errorf("failed to create helmCacheDir: %w", err)
	}

	defer util.CleanUpHelmTempDir(helmCacheDir, h.Log)
	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: &h.Kubeconfig,
		Namespace:  &applicationInstallation.Spec.Namespace.Name,
	}
	helmClient, err := helmclient.NewClient(
		h.Ctx,
		restClientGetter,
		helmclient.NewSettings(helmCacheDir),
		applicationInstallation.Spec.Namespace.Name,
		h.Log)
	if err != nil {
		return false, fmt.Errorf("failed to create helmClient: %w", err)
	}

	// retrieve metadata of the latest release
	releaseName := getReleaseName(applicationInstallation)
	metadata, err := helmClient.GetMetadata(releaseName)
	if err != nil {
		return false, fmt.Errorf("failed to retrieve metadata for checking if release %q is stuck: %w", releaseName, err)
	}

	// if the status of the release is not still pending, exit early
	if metadata.Status != "pending-upgrade" {
		return false, nil
	}

	return true, nil
}

// Rollback rolls an Application back to the previous release.
func (h HelmTemplate) Rollback(applicationInstallation *appskubermaticv1.ApplicationInstallation) error {
	helmCacheDir, err := util.CreateHelmTempDir(h.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to create helmCacheDir: %w", err)
	}

	defer util.CleanUpHelmTempDir(helmCacheDir, h.Log)
	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: &h.Kubeconfig,
		Namespace:  &applicationInstallation.Spec.Namespace.Name,
	}
	helmClient, err := helmclient.NewClient(
		h.Ctx,
		restClientGetter,
		helmclient.NewSettings(helmCacheDir),
		applicationInstallation.Spec.Namespace.Name,
		h.Log)
	if err != nil {
		return fmt.Errorf("failed to create helmClient: %w", err)
	}

	return helmClient.Rollback(getReleaseName(applicationInstallation))
}

func (h *HelmTemplate) templatePreDefinedValues(applicationValues map[string]any) (map[string]any, error) {
	templateData, err := GetTemplateData(h.Ctx, h.SeedClient, h.ClusterName)

	if err != nil {
		return nil, err
	}
	return RenderValueTemplate(applicationValues, templateData)
}
