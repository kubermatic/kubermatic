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
	"path/filepath"
	"slices"

	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/log"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// certManagerCRD is a CRD that is installed by every cert-manager installation.
const certManagerCRD = "certificates.cert-manager.io"

// deployCertManager deploys the bundled cert-manager chart to a separate seed, but only if the
// installer can take ownership of cert-manager there: either it already manages the Helm release
// or cert-manager is not installed at all. An existing cert-manager installation that was not
// created from the bundled chart (other Helm chart, kubectl, an operator, ...) is left untouched.
func deployCertManager(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	// kubermaticmaster.DeployCertManager skips in these cases on its own; do not probe the
	// cluster for them.
	if slices.Contains(opt.SkipCharts, kubermaticmaster.CertManagerChartName) ||
		opt.KubermaticConfiguration.Spec.FeatureGates[features.HeadlessInstallation] ||
		opt.KubermaticConfiguration.Spec.Ingress.CertificateIssuer.Name == "" {
		return kubermaticmaster.DeployCertManager(ctx, logger, kubeClient, helmClient, opt)
	}

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, kubermaticmaster.CertManagerChartName))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %w", err)
	}

	manage, reason, err := canManageCertManager(ctx, kubeClient, helmClient, chart)
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

// canManageCertManager returns true if there is either a Helm release installed from the given
// (bundled) chart or no cert-manager at all. Otherwise it returns false and the reason why.
func canManageCertManager(ctx context.Context, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, chart *helm.Chart) (bool, string, error) {
	release, err := helmClient.GetRelease(kubermaticmaster.CertManagerNamespace, kubermaticmaster.CertManagerReleaseName)
	if err != nil {
		return false, "", fmt.Errorf("failed to check Helm release: %w", err)
	}

	if release != nil {
		metadata, err := helmClient.GetReleaseMetadata(kubermaticmaster.CertManagerNamespace, kubermaticmaster.CertManagerReleaseName)
		if err != nil {
			return false, "", err
		}

		// The release is ours if it was installed from a chart with the same dependencies as the
		// bundled one (the bundled chart wraps the upstream cert-manager chart).
		if !hasDependencies(metadata.Dependencies, chart.Dependencies) {
			return false, fmt.Sprintf("Helm release %s in namespace %s was not installed from the bundled chart", release.Name, release.Namespace), nil
		}

		return true, "", nil
	}

	// No release with our name, so cert-manager must not be installed in any other way either.
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := kubeClient.Get(ctx, types.NamespacedName{Name: certManagerCRD}, crd); err != nil {
		if apierrors.IsNotFound(err) {
			return true, "", nil
		}

		return false, "", fmt.Errorf("failed to check for CRD %s: %w", certManagerCRD, err)
	}

	return false, fmt.Sprintf("CRD %s exists, but there is no Helm release %s in namespace %s", certManagerCRD, kubermaticmaster.CertManagerReleaseName, kubermaticmaster.CertManagerNamespace), nil
}

// hasDependencies returns true if every wanted dependency is part of the installed ones,
// matched by name and repository (the versions change with every release).
func hasDependencies(installed []helm.Dependency, wanted []helm.Dependency) bool {
	if len(wanted) == 0 {
		return false
	}

	for _, w := range wanted {
		found := slices.ContainsFunc(installed, func(i helm.Dependency) bool {
			return i.Name == w.Name && i.Repository == w.Repository
		})
		if !found {
			return false
		}
	}

	return true
}
