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

package helmclient

import (
	"context"
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// secretStorageDriver is the name of the secret storage driver.
// Release information will be stored in Secrets in the namespace of the release.
// More information at https://helm.sh/docs/topics/advanced/#storage-backends
const secretStorageDriver = "secret"

// HelmSettings holds the Helm configuration for caching repositories.
type HelmSettings struct {
	// RepositoryConfig is the path to the repositories file.
	RepositoryConfig string

	// RepositoryCache is the path to the repository cache directory.
	RepositoryCache string
}

// NewSettings creates a new HelmSettings which store caching configuration to rootDir.
func NewSettings(rootDir string) HelmSettings {
	return HelmSettings{
		RepositoryConfig: filepath.Join(rootDir, "repositories.yaml"),
		RepositoryCache:  filepath.Join(rootDir, "repository"),
	}
}

// GetterProviders build the getter.Providers from the HelmSettings.
func (s HelmSettings) GetterProviders() getter.Providers {
	return getter.All(&cli.EnvSettings{
		RepositoryConfig: s.RepositoryConfig,
		RepositoryCache:  s.RepositoryCache,
	})
}

// AuthSettings holds the different kinds of credentials for Helm repository and registry.
type AuthSettings struct {
	// Username used for basic authentication
	Username string

	// Password used for basic authentication
	Password string

	// RegistryConfigFile is the path to registry config file. It's dockercfg
	// file that follows the same format rules as ~/.docker/config.json
	RegistryConfigFile string
}

// newRegistryClient returns a new registry client with authentication is RegistryConfigFile is defined.
func (a *AuthSettings) newRegistryClient() (*registry.Client, error) {
	if a.RegistryConfigFile == "" {
		return registry.NewClient()
	}
	return registry.NewClient(registry.ClientOptCredentialsFile(a.RegistryConfigFile))
}

// registryClientAndGetterOptions return registry.Client and authentication options for Getter.
func (a *AuthSettings) registryClientAndGetterOptions() (*registry.Client, []getter.Option, error) {
	regClient, err := a.newRegistryClient()
	if err != nil {
		return nil, nil, err
	}
	options := []getter.Option{getter.WithRegistryClient(regClient)}

	if a.Username != "" && a.Password != "" {
		options = append(options, getter.WithBasicAuth(a.Username, a.Password))
	}
	return regClient, options, nil
}

// DeployOpts holds the options for installing or upgrading a Helm chart.
type DeployOpts struct {
	// wait corresponds to the --wait flag on Helm cli.
	// if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a ready state before marking the release as successful. It will wait for as long as --timeout
	wait bool

	// timeout corresponds to the --timeout flag on Helm cli.
	// time to wait for any individual Kubernetes operation.
	timeout time.Duration

	// atomic corresponds to the --atomic flag on Helm cli.
	// if set, the installation process deletes the installation on failure; the upgrade process rolls back changes made in case of failed upgrade.
	atomic bool

	// enableDNS  corresponds to the --enable-dns flag on Helm cli.
	// enable DNS lookups when rendering templates.
	// if you enable this flag, you have to verify that helm template function 'getHostByName' is not being used in a chart to disclose any information you do not want to be passed to DNS servers.(c.f CVE-2023-25165)
	enableDNS bool
}

// NewDeployOpts creates a new DeployOpts. It raises an error if the inputs are not valid.
func NewDeployOpts(wait bool, timeout time.Duration, atomic bool, enableDNS bool) (*DeployOpts, error) {
	if atomic && !wait {
		return nil, fmt.Errorf("invalid values: if atomic=true then wait must also be true")
	}
	if wait && timeout == 0 {
		return nil, fmt.Errorf("invalid values: if wait = true then timeout must be greater than 0")
	}
	return &DeployOpts{
		wait:      wait,
		timeout:   timeout,
		atomic:    atomic,
		enableDNS: enableDNS,
	}, nil
}

// HelmClient is a client that allows interacting with Helm.
// If you want to use it in a concurrency context, you must create several clients with different HelmSettings. Otherwise
// writing repository.xml or download index file may fails as it will be written by several threads.
type HelmClient struct {
	ctx context.Context

	restClientGetter genericclioptions.RESTClientGetter

	settings HelmSettings

	// Provider represents any getter and the schemes that it supports.
	// For example, an HTTP provider may provide one getter that handles both 'http' and 'https' schemes
	getterProviders getter.Providers

	// Configuration injects the dependencies that all actions (eg install or upgrade) share.
	actionConfig *action.Configuration

	// Namespace where chart will be installed or updated.
	targetNamespace string

	logger *zap.SugaredLogger
}

func NewClient(ctx context.Context, restClientGetter genericclioptions.RESTClientGetter, settings HelmSettings, targetNamespace string, logger *zap.SugaredLogger) (*HelmClient, error) {
	// Even if namespace is set in the actionConfig.init() function, upgrade action take the namespace from RESTClientGetter.
	// If the namespaces are different, the release will be installed in the namespace set in the RESTClientGetter but the
	// release information will be stored in the targetNamespace which leads to a release which cannot be uninstalled with Helm.
	kcNamspace, _, err := restClientGetter.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return nil, fmt.Errorf("can not get namespace from RESTClientGetter: %w", err)
	}
	if kcNamspace != targetNamespace {
		return nil, fmt.Errorf("namespace set in RESTClientGetter should be the same as targetNamespace. RESTClientGetter namespace=%s, targetNamespace=%s", kcNamspace, targetNamespace)
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(restClientGetter, targetNamespace, secretStorageDriver, logger.Infof); err != nil {
		return nil, fmt.Errorf("can not initialize helm actionConfig: %w", err)
	}

	return &HelmClient{
		ctx:              ctx,
		restClientGetter: restClientGetter,
		settings:         settings,
		getterProviders:  settings.GetterProviders(),
		actionConfig:     actionConfig,
		targetNamespace:  targetNamespace,
		logger:           logger,
	}, nil
}

// DownloadChart from url into dest folder and return the chart location (eg /tmp/foo/apache-1.0.0.tgz)
// The dest folder must exist.
func (h HelmClient) DownloadChart(url string, chartName string, version string, dest string, auth AuthSettings) (string, error) {
	var repoName string
	var err error
	if strings.HasPrefix(url, "oci://") {
		repoName = url
	} else {
		repoName, err = h.ensureRepository(url, auth)
		if err != nil {
			return "", err
		}
	}

	regClient, options, err := auth.registryClientAndGetterOptions()
	if err != nil {
		return "", err
	}
	var out strings.Builder
	chartDownloader := downloader.ChartDownloader{
		Out:              &out,
		Verify:           downloader.VerifyNever,
		RepositoryConfig: h.settings.RepositoryConfig,
		RepositoryCache:  h.settings.RepositoryCache,
		Getters:          h.getterProviders,
		RegistryClient:   regClient,
		Options:          options,
	}

	// todo note: we may want to check the verificaton return by chartDownloader.DownloadTo. for the moment it's set to downloader.VerifyNever in struct init
	chartRef := repoName + "/" + chartName
	chartLoc, _, err := chartDownloader.DownloadTo(chartRef, version, dest)
	if err != nil {
		h.logger.Errorw("failed to download chart", "chart", chartRef, "version", version, "log", out.String())
		return "", err
	}

	h.logger.Debugw("successfully downloaded chart", "chart", chartRef, "version", version, "log", out.String())
	return chartLoc, nil
}

// InstallOrUpgrade installs the chart located at chartLoc into targetNamespace if it's not already installed.
// Otherwise it upgrades the chart.
// charLoc is the path to the chart archive (e.g. /tmp/foo/apache-1.0.0.tgz) or folder containing the chart (e.g. /tmp/mychart/apache).
func (h HelmClient) InstallOrUpgrade(chartLoc string, releaseName string, values map[string]interface{}, deployOpts DeployOpts, auth AuthSettings) (*release.Release, error) {
	// To know if we have to do an install or an upgrade, we have to fetch the last version of the helm release. We do
	// that with the following command:
	//              h.actionConfig.Releases.Last(releaseName)
	// Unfortunately, this command load all the versions and returns the last one. By default, the history is limited to
	// 256 versions. In the case of a big chart like Cilium, loading all history consume several hundred of RAM, leading
	// to OOM kill. cf https://github.com/kubermatic/kubermatic/issues/12078
	//
	// To fix that, we have limited the history. So when we upgrade a chart, Helm automatically cleans the history.
	// However, for that to happen, an upgrade must happen, which may not be possible if the history is already too
	// big. The Controller is OOM killed when it tries to figure out if it has to do an install or an upgrade. So we manually purge
	// the history.
	// This manual purged will not be need in KPP 2.23
	restConfig, err := h.restClientGetter.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get restConfig to create k8s client: %w", err)
	}
	k8sClient, err := ctrlruntimeclient.New(restConfig, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	if err := purgeOldReleases(h.ctx, h.logger, k8sClient, h.targetNamespace, releaseName); err != nil {
		return nil, fmt.Errorf("failed to purge history of helm release '%s': %w", releaseName, err)
	}

	if _, err := h.actionConfig.Releases.Last(releaseName); err != nil {
		return h.Install(chartLoc, releaseName, values, deployOpts, auth)
	}
	return h.Upgrade(chartLoc, releaseName, values, deployOpts, auth)
}

// Install the chart located at chartLoc into targetNamespace. If the chart was already installed, an error is returned.
// charLoc is the path to the chart archive (eg /tmp/foo/apache-1.0.0.tgz) or folder containing the chart (e.g. /tmp/mychart/apache).
func (h HelmClient) Install(chartLoc string, releaseName string, values map[string]interface{}, deployOpts DeployOpts, auth AuthSettings) (*release.Release, error) {
	chartToInstall, err := h.buildDependencies(chartLoc, auth)
	if err != nil {
		return nil, err
	}

	installClient := action.NewInstall(h.actionConfig)
	installClient.Namespace = h.targetNamespace
	installClient.ReleaseName = releaseName
	installClient.Wait = deployOpts.wait
	installClient.Timeout = deployOpts.timeout
	installClient.Atomic = deployOpts.atomic
	installClient.EnableDNS = deployOpts.enableDNS

	rel, err := installClient.RunWithContext(h.ctx, chartToInstall, values)
	if err != nil {
		// even if an error occurred release may be not null (e.g if timeout is reached)
		return rel, err
	}
	return rel, nil
}

// Upgrade the chart located at chartLoc into targetNamespace. If the chart is not already installed, an error is returned.
// charLoc is the path to the chart archive (e.g. /tmp/foo/apache-1.0.0.tgz) or folder containing the chart (e.g. /tmp/mychart/apache).
func (h HelmClient) Upgrade(chartLoc, releaseName string, values map[string]interface{}, deployOpts DeployOpts, auth AuthSettings) (*release.Release, error) {
	chartToUpgrade, err := h.buildDependencies(chartLoc, auth)
	if err != nil {
		return nil, err
	}

	upgradeClient := action.NewUpgrade(h.actionConfig)
	upgradeClient.Namespace = h.targetNamespace
	upgradeClient.Wait = deployOpts.wait
	upgradeClient.Timeout = deployOpts.timeout
	upgradeClient.Atomic = deployOpts.atomic
	upgradeClient.EnableDNS = deployOpts.enableDNS

	// restrict history to avoid OOM kill
	upgradeClient.MaxHistory = 1

	// Don't reuse values from the previous release.
	// By default, Helm will merge values with the ones of the last release. This behavior may be helpful to for CLI but
	// as CR is the source of truth, we don't want that.
	upgradeClient.ResetValues = true

	rel, err := upgradeClient.RunWithContext(h.ctx, releaseName, chartToUpgrade, values)
	if err != nil {
		// even if an error occurred release may be not null (e.g if timeout is reached)
		return rel, err
	}
	return rel, nil
}

// Uninstall the release in targetNamespace.
func (h HelmClient) Uninstall(releaseName string) (*release.UninstallReleaseResponse, error) {
	uninstallClient := action.NewUninstall(h.actionConfig)
	uninstallReleaseResponse, err := uninstallClient.Run(releaseName)

	// Don't raise an error is the released has already been uninstalled.
	if errors.Is(err, driver.ErrReleaseNotFound) {
		h.logger.Debug("helm release not found. nothing to do")
		return uninstallReleaseResponse, nil
	}
	return uninstallReleaseResponse, err
}

// buildDependencies adds missing repositories and then does a Helm dependency build (i.e. download the chart dependencies
// from repositories into "charts" folder).
func (h HelmClient) buildDependencies(chartLoc string, auth AuthSettings) (*chart.Chart, error) {
	fi, err := os.Stat(chartLoc)
	if err != nil {
		return nil, fmt.Errorf("can not find chart at `%s': %w", chartLoc, err)
	}

	chartToInstall, err := loader.Load(chartLoc)
	if err != nil {
		return nil, fmt.Errorf("can not load chart: %w", err)
	}

	var out strings.Builder

	// If we got the chart from the filesystem (i.e. cloned from a git repository), we have to build dependencies because
	// charts directory may not exist.
	//
	// note: if we got the chart from a remote helm repository, we don't have to build dependencies because the package
	// (i.e. the tgz) should already contain it.
	if fi.IsDir() {
		regClient, err := auth.newRegistryClient()
		if err != nil {
			return nil, fmt.Errorf("can not initialize registry client: %w", err)
		}
		man := &downloader.Manager{
			Out:              &out,
			ChartPath:        chartLoc,
			Getters:          h.getterProviders,
			RepositoryConfig: h.settings.RepositoryConfig,
			RepositoryCache:  h.settings.RepositoryCache,
			RegistryClient:   regClient,
			Debug:            true,
			Verify:           downloader.VerifyNever,
			SkipUpdate:       true,
		}

		// Helm does not download dependency if the repository is unknown (i.e. not present in repository.xml)
		// so we explicitly add to the repository file.
		var dependencies []*chart.Dependency
		if chartToInstall.Lock != nil {
			dependencies = chartToInstall.Lock.Dependencies
		} else {
			dependencies = chartToInstall.Metadata.Dependencies
		}

		for _, dep := range dependencies {
			// oci or file dependencies can not be added as a repository.
			if strings.HasPrefix(dep.Repository, "http://") || strings.HasPrefix(dep.Repository, "https://") {
				if _, err := h.ensureRepository(dep.Repository, auth); err != nil {
					return nil, fmt.Errorf("can not download index for repository: %w", err)
				}
			}
		}

		// Equivalent of helm dependency build.
		err = man.Build()
		if err != nil {
			h.logger.Errorw("can not build dependencies", "chart", chartLoc, "log", out.String())
			return nil, fmt.Errorf("can not build dependencies: %w", err)
		}

		// We have to reload the chart to load the downloaded dependencies.
		chartToInstall, err = loader.Load(chartLoc)
		if err != nil {
			return nil, fmt.Errorf("can not reload chart: %w", err)
		}
	}
	h.logger.Debugw("successfully built dependencies", "chart", chartLoc, "log", out.String())
	return chartToInstall, nil
}

// ensureRepository adds the repository url if it doesn't exist and downloads the latest index file.
// The repository is added with the name helm-manager-$(sha256 url).
func (h HelmClient) ensureRepository(url string, auth AuthSettings) (string, error) {
	repoFile, err := repo.LoadFile(h.settings.RepositoryConfig)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}

	repoName, err := computeRepoName(url)
	if err != nil {
		return "", fmt.Errorf("can not compute repository's name for '%s': %w", url, err)
	}
	desiredEntry := &repo.Entry{
		Name:     repoName,
		URL:      url,
		Username: auth.Username,
		Password: auth.Password,
	}

	// Ensure we have the last version of the index file.
	chartRepo, err := repo.NewChartRepository(desiredEntry, h.getterProviders)
	if err != nil {
		return "", err
	}
	// Constructor of the ChartRepository uses the default Helm cache. However, we have to use the cached defined in the
	// client settings.
	chartRepo.CachePath = h.settings.RepositoryCache

	if _, err := chartRepo.DownloadIndexFile(); err != nil {
		return "", fmt.Errorf("can not download index file: %w", err)
	}

	if !repoFile.Has(repoName) {
		repoFile.Add(desiredEntry)
		return repoName, repoFile.WriteFile(h.settings.RepositoryConfig, 0644)
	}
	return repoName, nil
}

// purgeOldReleases will purge the helm release history to keep only the 3 most recent releases.
// Helm releases are stored in secrets (because we use the secret driver) labeled with name=<the name of the release>, version=<the version of the release> and owner=helm.
func purgeOldReleases(ctx context.Context, logger *zap.SugaredLogger, k8sClient ctrlruntimeclient.Client, namespace string, releaseName string) error {
	// List secret storging helm releaseinformation but get only metadata to avoid OOM kill.
	var secrets metav1.PartialObjectMetadataList
	secrets.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("SecretList"))
	if err := k8sClient.List(ctx, &secrets, &ctrlruntimeclient.ListOptions{Namespace: namespace}, &ctrlruntimeclient.MatchingLabels{"name": releaseName, "owner": "helm"}); err != nil {
		return fmt.Errorf("failed to list secrets: %w", err)
	}

	// Sort releases in decreasing order.
	var sortError error
	sort.Slice(secrets.Items, func(i, j int) bool {
		secretI := secrets.Items[i]
		versionI, err := strconv.Atoi(secretI.Labels["version"])
		if err != nil { // should not be possible
			sortError = fmt.Errorf("failed to get version for secret %s/%s: %w", secretI.Namespace, secretI.Name, err)
		}
		secretJ := secrets.Items[j]
		versionJ, err := strconv.Atoi(secretJ.Labels["version"])
		if err != nil {
			sortError = fmt.Errorf("failed to get version for secret %s/%s: %w", secretJ.Namespace, secretJ.Name, err)
		}

		return versionI > versionJ
	})

	if sortError != nil {
		return fmt.Errorf("failed to sort secrets: %w", sortError)
	}

	// Keep the 3 most recent releases.
	// We keep 3 because if atomic is true and upgrade fails, we have the following history:
	//              version 1 // the previous installed version
	//              version 2 // the upgrade that fails.
	//              version 3 // the version generated by the rollback (atomic=true)
	for i := 3; i < len(secrets.Items); i++ {
		if err := k8sClient.Delete(ctx, &secrets.Items[i]); err != nil {
			return fmt.Errorf("failed to delete secret %s/%s (release version %s): %w", secrets.Items[i].Namespace, secrets.Items[i].Name, secrets.Items[i].Labels["version"], err)
		}
		logger.Debugw("helm release purged", "namespace", secrets.Items[i].Namespace, "secret-name", secrets.Items[i].Name, "name", secrets.Items[i].Labels["name"], "version", secrets.Items[i].Labels["version"])
	}
	return nil
}

// computeRepoName computes the name of the repository from url.
// the name is helm-manager-$(sha256 url). we use the same algorithm as https://github.com/helm/helm/blob/49819b4ef782e80b0c7f78c30bd76b51ebb56dc8/pkg/downloader/manager.go#L518
// because if you run the client using default Helm settings, the repository will appear as an unmanaged repository.
func computeRepoName(url string) (string, error) {
	in := strings.NewReader(url)
	hash := crypto.SHA256.New()
	if _, err := io.Copy(hash, in); err != nil {
		return "", err
	}
	return "helm-manager-" + hex.EncodeToString(hash.Sum(nil)), nil
}
