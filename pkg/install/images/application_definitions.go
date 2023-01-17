package images

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers"
	"k8c.io/kubermatic/v2/pkg/cni/cilium"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

func GetImagesFromSystemApplicationDefinitions(logger logrus.FieldLogger, config *kubermaticv1.KubermaticConfiguration, helmClient helm.Client, helmTimeout time.Duration, registryPrefix string) ([]string, error) {
	var images []string

	var appDefReconcilers []reconciling.NamedApplicationDefinitionReconcilerFactory
	appDefReconcilers = append(appDefReconcilers,
		cilium.ApplicationDefinitionReconciler(config),
	)

	for _, createFunc := range appDefReconcilers {
		appName, creator := createFunc()
		appDef, err := creator(&appskubermaticv1.ApplicationDefinition{})
		if err != nil {
			return nil, err
		}
		appLog := logger.WithFields(logrus.Fields{"application-name": appName})
		appDefImages, err := getImagesFromApplicationDefinition(appLog, helmClient, appDef, helmTimeout, registryPrefix)
		if err != nil {
			return nil, err
		}
		images = append(images, appDefImages...)
	}

	return images, nil
}

func getImagesFromApplicationDefinition(logger logrus.FieldLogger, helmClient helm.Client, appDef *appskubermaticv1.ApplicationDefinition, helmTimeout time.Duration, registryPrefix string) ([]string, error) {
	if appDef.Spec.Method != appskubermaticv1.HelmTemplateMethod {
		// Only Helm ApplicationDefinitions are supported at the moment
		logger.Debugf("Skipping the ApplicationDefinition as the method '%s' is not supported yet", appDef.Spec.Method)
		return nil, nil
	}
	logger.Info("Retrieving images…")

	tmpDir, err := os.MkdirTemp("", "helm-charts")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			logger.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	// if DefaultValues is provided, use it as values file
	valuesFile := ""
	if appDef.Spec.DefaultValues != nil {
		valuesFile = tmpDir + "values.yaml"
		err = os.WriteFile(valuesFile, appDef.Spec.DefaultValues.Raw, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to create values file: %w", err)
		}
	}

	var images []string
	for _, appVer := range appDef.Spec.Versions {
		appVerLog := logger.WithField("application-version", appVer.Version)
		appVerLog.Debug("Downloading Helm chart…")
		// pull the chart
		chartPath, err := downloadAppSourceChart(&appVer.Template.Source, tmpDir, helmTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to pull app chart: %w", err)
		}
		// get images
		chartImages, err := GetImagesForHelmChart(appVerLog, nil, helmClient, chartPath, valuesFile, registryPrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to get images for chart: %w", err)
		}
		images = append(images, chartImages...)
	}

	return images, nil
}

func downloadAppSourceChart(appSource *appskubermaticv1.ApplicationSource, directory string, timeout time.Duration) (chartPath string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	sp, err := providers.NewSourceProvider(ctx, log.NewDefault().Sugar(), nil, "", directory, appSource, "")
	if err != nil {
		return "", fmt.Errorf("failed to create app source provider: %w", err)
	}
	chartPath, err = sp.DownloadSource(directory)
	if err != nil {
		return "", fmt.Errorf("failed to download app source: %w", err)
	}
	return
}
