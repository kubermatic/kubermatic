/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package kubermaticmaster

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func deployNginxIngressController(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	if slices.Contains(opt.SkipCharts, NginxIngressControllerChartName) {
		logger.Infof("⭕ Skipping %s deployment.", NginxIngressControllerChartName)
		return nil
	}

	logger.Infof("📦 Deploying %s…", NginxIngressControllerChartName)
	sublogger := log.Prefix(logger, "   ")

	if opt.KubermaticConfiguration.Spec.FeatureGates[features.HeadlessInstallation] {
		sublogger.Info("Headless installation requested, skipping.")
		return nil
	}

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, NginxIngressControllerChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, NginxIngressControllerNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, NginxIngressControllerNamespace, NginxIngressControllerReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %w", err)
	}

	// if version older than 1.3.0 is installed, we must perform a migration
	// by deleting the old deployment object for the controller
	v13 := semverlib.MustParse("1.3.0")
	backupTS := time.Now().Format("2006-01-02T150405")

	isUpgrading := false

	if release != nil && release.Version.LessThan(v13) && !chart.Version.LessThan(v13) {
		if !opt.EnableNginxIngressMigration {
			sublogger.Warnf("To upgrade %s to a new version, the installer", NginxIngressControllerChartName)
			sublogger.Warn("will remove the old deployment object before proceeding with the upgrade.")
			sublogger.Warn("Rerun the installer with --migrate-upstream-nginx-ingress to enable the migration process.")
			sublogger.Warn("Please refer to the KKP 2.19 upgrade notes for more information.")

			return fmt.Errorf("user must acknowledge the migration using --migrate-upstream-nginx-ingress")
		}

		isUpgrading = true
		err = upgradeNginxIngress(ctx, sublogger, kubeClient, helmClient, opt, chart, release, backupTS)
		if err != nil {
			return fmt.Errorf("failed to prepare %s for upgrade: %w", NginxIngressControllerChartName, err)
		}
	}

	// do not perform an atomic installation, as this will make Helm wait for the LoadBalancer to
	// get an IP and this can require manual intervention based on the target environment
	sublogger.Info("Deploying Helm chart…")

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, NginxIngressControllerNamespace, NginxIngressControllerReleaseName, opt.HelmValues, false, opt.ForceHelmReleaseUpgrade, opt.DisableDependencyUpdate, release); err != nil {
		if isUpgrading {
			return fmt.Errorf("failed to deploy Helm release: %w\n\nuse backup file to restore the deployment or re-try the installation after fixing any errors.", err)
		}
		return fmt.Errorf("failed to deploy Helm release: %w", err)
	}

	if err := waitForNginxIngressWebhook(ctx, sublogger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to verify that the webhook is functioning: %w", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func upgradeNginxIngress(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	helmClient helm.Client,
	opt stack.DeployOptions,
	chart *helm.Chart,
	release *helm.Release,
	backupTS string,
) error {
	logger.Infof("%s: %s detected, performing upgrade to %s…", release.Name, release.Version.String(), chart.Version.String())
	// 1: find the old deployment
	logger.Info("Backing up old ingress deployment…")

	deploymentsList := &unstructured.UnstructuredList{}
	deploymentsList.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apps",
		Kind:    "DeploymentList",
		Version: "v1",
	})

	nginxMatcher := ctrlruntimeclient.MatchingLabels{
		"app.kubernetes.io/name":       "ingress-nginx",
		"app.kubernetes.io/managed-by": "Helm",
		"app.kubernetes.io/instance":   release.Name,
	}
	if err := kubeClient.List(ctx, deploymentsList, ctrlruntimeclient.InNamespace(NginxIngressControllerNamespace), nginxMatcher); err != nil {
		return fmt.Errorf("Error querying API for the existing Deployment object, aborting upgrade process.")
	}
	// 2: store the deployment for backup
	// There can be only one...
	switch foundDeployments := len(deploymentsList.Items); foundDeployments {
	case 1:
		filename := fmt.Sprintf("backup_%s_%s.yaml", NginxIngressControllerReleaseName, backupTS)
		logger.Infof("Attempting to store the deployment in file: %s", filename)
		if err := util.DumpResources(ctx, filename, deploymentsList.Items); err != nil {
			return fmt.Errorf("failed to back up the deployment, it is not removed: %w", err)
		}

		// 3: delete the deployment
		logger.Info("Deleting the deployment from the cluster")
		if err := kubeClient.Delete(ctx, &deploymentsList.Items[0]); err != nil {
			return fmt.Errorf("failed to remove the deployment: %w\n\nuse backup file to check the changes and restore if needed", err)
		}
	case 0:
		logger.Info("No deployments matching the criteria were found, assuming clean state and attempting to upgrade...")
	default:
		return fmt.Errorf("found more than one deployment (%d) matching the nginx-ingress-controller release, stopping upgrade - clean up manually before proceeding", len(deploymentsList.Items))
	}
	return nil
}

func waitForNginxIngressWebhook(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	helmClient helm.Client,
	opt stack.DeployOptions,
) error {
	ingressName := "kubermatic-installer-test"
	ingressClassName := "nginx"

	// delete any leftovers from previous installer runs
	if err := deleteIngress(ctx, kubeClient, NginxIngressControllerNamespace, ingressName); err != nil {
		return fmt.Errorf("failed to prepare webhook: %w", err)
	}

	// always clean up on a best-effort basis
	defer func() {
		// it can take a moment for the cert to appear
		time.Sleep(3 * time.Second)

		if err := deleteIngress(ctx, kubeClient, NginxIngressControllerNamespace, ingressName); err != nil {
			logger.Warnf("Failed to clean up: %v", err)
		}
	}()

	// create an Ingress object to check if the webhook is responsive
	dummyIngress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: NginxIngressControllerNamespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &ingressClassName,
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: NginxIngressControllerReleaseName,
					Port: networkingv1.ServiceBackendPort{
						Name: "http",
					},
				},
			},
		},
	}

	var lastCreateErr error
	err := wait.PollUntilContextTimeout(ctx, 3*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		lastCreateErr = kubeClient.Create(ctx, dummyIngress)
		return lastCreateErr == nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for webhook to become ready: %w", lastCreateErr)
	}

	return nil
}

func deleteIngress(ctx context.Context, kubeClient ctrlruntimeclient.Client, namespace string, name string) error {
	ingress := &networkingv1.Ingress{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := kubeClient.Get(ctx, key, ingress); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to probe for leftover test ingress: %w", err)
	}

	if err := kubeClient.Delete(ctx, ingress); err != nil {
		return fmt.Errorf("failed to delete test ingress: %w", err)
	}

	return nil
}
