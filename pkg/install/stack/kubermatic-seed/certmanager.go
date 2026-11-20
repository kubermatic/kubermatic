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

package kubermaticseed

import (
	"context"
	"fmt"
	"slices"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// bundledChartMaintainerEmail is the maintainer of the bundled cert-manager chart (see its
	// Chart.yaml). A release whose chart lists it as maintainer was installed from the bundled
	// chart, independent of the chart version.
	bundledChartMaintainerEmail = "support@kubermatic.com"

	// certManagerInstanceLabel and certManagerNameLabel are set on the Deployments of every
	// cert-manager installation, no matter if it was installed with Helm, the static manifests or
	// an operator; the instance label is "cert-manager" unless the Helm release is named differently.
	certManagerInstanceLabel = "app.kubernetes.io/instance"
	certManagerNameLabel     = "app.kubernetes.io/name"
)

// certManagerComponentNames are the values of the name label of the three cert-manager Deployments.
var certManagerComponentNames = []string{"cert-manager", "cainjector", "webhook"}

// minManagedChartVersion is the oldest chart version of an existing release that the installer
// upgrades on a separate seed. DeployCertManager migrates older releases (before 2.1.0: cert-manager
// 0.x and the pre-upstream chart) by removing and recreating Certificates and Issuers, which must
// never happen implicitly on a seed.
var minManagedChartVersion = semverlib.MustParse("2.1.0")

// deployCertManager deploys the bundled cert-manager chart to a separate seed, but only if the
// installer can take ownership of cert-manager there: either it already manages the Helm release
// or cert-manager is not installed at all. An existing cert-manager installation that was not
// created from the bundled chart (other Helm chart, kubectl, an operator, ...) is left untouched.
// Leftover CRDs alone do not count as an installation, e.g. after a rolled back first installation.
func deployCertManager(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	// kubermaticmaster.DeployCertManager skips in these cases on its own; do not probe the
	// cluster for them.
	if slices.Contains(opt.SkipCharts, kubermaticmaster.CertManagerChartName) ||
		opt.KubermaticConfiguration.Spec.FeatureGates[features.HeadlessInstallation] ||
		opt.KubermaticConfiguration.Spec.Ingress.CertificateIssuer.Name == "" {
		return kubermaticmaster.DeployCertManager(ctx, logger, kubeClient, helmClient, opt)
	}

	manage, reason, err := canManageCertManager(ctx, kubeClient, helmClient)
	if err != nil {
		return fmt.Errorf("failed to check for an existing cert-manager installation: %w", err)
	}

	if !manage {
		logger.Infof("⭕ Skipping %s deployment.", kubermaticmaster.CertManagerChartName)
		sublogger := log.Prefix(logger, "   ")
		sublogger.Warnf("%s; leaving it untouched.", reason)
		sublogger.Warn("The installer only manages cert-manager on a separate seed if it was installed from the")
		sublogger.Warn("bundled chart. Use --skip-charts cert-manager to silence this warning.")

		return nil
	}

	return kubermaticmaster.DeployCertManager(ctx, logger, kubeClient, helmClient, opt)
}

// canManageCertManager returns true if there is either a Helm release installed from the bundled
// chart or no cert-manager at all (no release and no cert-manager Deployment). Otherwise it returns
// false and the reason why.
func canManageCertManager(ctx context.Context, kubeClient ctrlruntimeclient.Client, helmClient helm.Client) (bool, string, error) {
	release, err := helmClient.GetRelease(kubermaticmaster.CertManagerNamespace, kubermaticmaster.CertManagerReleaseName)
	if err != nil {
		return false, "", fmt.Errorf("failed to check Helm release: %w", err)
	}

	if release != nil {
		emails, err := helmClient.GetReleaseMaintainerEmails(kubermaticmaster.CertManagerNamespace, kubermaticmaster.CertManagerReleaseName)
		if err != nil {
			return false, "", err
		}

		if !slices.Contains(emails, bundledChartMaintainerEmail) {
			return false, fmt.Sprintf("Helm release %s in namespace %s was not installed from the bundled chart", release.Name, release.Namespace), nil
		}

		if release.Version.LessThan(minManagedChartVersion) {
			return false, "", fmt.Errorf("cert-manager release %s in namespace %s uses chart version %s, which is older than %s and cannot be upgraded automatically on a separate seed; upgrade it manually or use --skip-charts %s", release.Name, release.Namespace, release.Version, minManagedChartVersion, kubermaticmaster.CertManagerChartName)
		}

		return true, "", nil
	}

	// No release with our name: it is only installed some other way if its Deployments exist.
	// CRDs are no indicator, they remain when a first (atomic) installation was rolled back.
	deployments := &appsv1.DeploymentList{}
	if err := kubeClient.List(ctx, deployments, ctrlruntimeclient.MatchingLabels{certManagerInstanceLabel: kubermaticmaster.CertManagerReleaseName}); err != nil {
		return false, "", fmt.Errorf("failed to list cert-manager Deployments: %w", err)
	}

	for _, deployment := range deployments.Items {
		if slices.Contains(certManagerComponentNames, deployment.Labels[certManagerNameLabel]) {
			return false, fmt.Sprintf("cert-manager Deployment %s/%s exists, but there is no Helm release %s in namespace %s", deployment.Namespace, deployment.Name, kubermaticmaster.CertManagerReleaseName, kubermaticmaster.CertManagerNamespace), nil
		}
	}

	return true, "", nil
}
