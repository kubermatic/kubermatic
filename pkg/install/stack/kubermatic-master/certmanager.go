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
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	v1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func deployCertManager(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt stack.DeployOptions) error {
	logger.Info("📦 Deploying cert-manager…")
	sublogger := log.Prefix(logger, "   ")

	if opt.KubermaticConfiguration.Spec.Ingress.CertificateIssuer.Name == "" {
		sublogger.Info("No CertificateIssuer configured in KubermaticConfiguration, skipping.")
		return nil
	}

	chartDir := filepath.Join(opt.ChartsDirectory, "cert-manager")

	chart, err := helm.LoadChart(chartDir)
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, CertManagerNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, CertManagerNamespace, CertManagerReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	// if a pre-2.0 version of the chart is installed, we must perform a
	// larger migration to bring the cluster from cert-manager 0.16 to 1.x
	// (and its CRD from v1alpha2 to v1)
	v2 := semver.MustParse("2.0.0")

	if release != nil && release.Version.LessThan(v2) && !chart.Version.LessThan(v2) {
		if err := purgeCertManager(ctx, sublogger, kubeClient, helmClient, opt, chart, release); err != nil {
			return fmt.Errorf("upgrade failed: %v", err)
		}
	}

	sublogger.Info("Deploying Custom Resource Definitions…")
	if err := util.DeployCRDs(ctx, kubeClient, sublogger, filepath.Join(chartDir, "crd")); err != nil {
		return fmt.Errorf("failed to deploy CRDs: %v", err)
	}

	sublogger.Info("Deploying Helm chart…")
	release, err = util.CheckHelmRelease(ctx, sublogger, helmClient, CertManagerNamespace, CertManagerReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, CertManagerNamespace, CertManagerReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %v", err)
	}

	if err := waitForCertManagerWebhook(ctx, sublogger, kubeClient); err != nil {
		return fmt.Errorf("failed to verify that the webhook is functioning: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

// purgeCertManager removes all tracecs of cert-manager from the cluster,
// so that the installer can then install it cleanly.
func purgeCertManager(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	helmClient helm.Client,
	opt stack.DeployOptions,
	chart *helm.Chart,
	release *helm.Release,
) error {
	logger.Infof("%s detected, performing upgrade to Helm chart %s…", release.Version.String(), chart.Version.String())

	// step 1: purge the Helm release
	logger.Info("Uninstalling cert-manager…")
	if err := helmClient.UninstallRelease(CertManagerNamespace, CertManagerReleaseName); err != nil {
		return fmt.Errorf("failed to uninstall release: %v", err)
	}

	// step 2: delete all cert-manager CRDs
	logger.Info("Removing cert-manager Custom Resource Definitions…")
	crdNames := []string{
		"certificaterequests.cert-manager.io",
		"certificates.cert-manager.io",
		"challenges.acme.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"issuers.cert-manager.io",
		"orders.acme.cert-manager.io",
	}
	for _, crdName := range crdNames {
		logger.Info("  %s", crdName)

		crd := apiextensionsv1beta1.CustomResourceDefinition{}
		key := types.NamespacedName{Name: crdName}

		if err := kubeClient.Get(ctx, key, &crd); err != nil {
			if kerrors.IsNotFound(err) {
				continue
			}

			return fmt.Errorf("failed to retrieve CRD %s: %v", crdName, err)
		}

		if err := kubeClient.Delete(ctx, &crd); err != nil {
			return fmt.Errorf("failed to delete CRD %s: %v", crdName, err)
		}
	}

	return nil
}

func waitForCertManagerWebhook(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client) error {
	logger.Debug("Waiting for webhook to become ready…")

	certName := "kubermatic-installer-test"

	// delete any leftovers from previous installer runs
	if err := deleteCertificate(ctx, kubeClient, CertManagerNamespace, certName); err != nil {
		return fmt.Errorf("failed to prepare webhook: %v", err)
	}

	// always clean up on a best-effort basis
	defer func() {
		if err := deleteCertificate(ctx, kubeClient, CertManagerNamespace, certName); err != nil {
			logger.Warnf("Failed to cleanup: %v", err)
		}
	}()

	// create a dummy cert to see if the webhook is alive and well
	dummyCert := &certmanagerv1alpha2.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      certName,
			Namespace: CertManagerNamespace,
		},
		Spec: certmanagerv1alpha2.CertificateSpec{
			SecretName: certName,
			DNSNames:   []string{"www.example.com"},
			IssuerRef: v1.ObjectReference{
				Name: "dummy-issuer", // does not have to actually exist
			},
		},
	}

	var lastCreateErr error
	err := wait.PollImmediate(1*time.Second, 1*time.Minute, func() (bool, error) {
		lastCreateErr = kubeClient.Create(ctx, dummyCert)
		return lastCreateErr == nil, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for webhook to become ready: %v", lastCreateErr)
	}

	return nil
}

func deleteCertificate(ctx context.Context, kubeClient ctrlruntimeclient.Client, namespace string, name string) error {
	cert := &certmanagerv1alpha2.Certificate{}
	key := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}

	if err := kubeClient.Get(ctx, key, cert); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("failed to probe for leftover test certificate: %v", err)
	}

	if err := kubeClient.Delete(ctx, cert); err != nil {
		return fmt.Errorf("failed to delete test certificate: %v", err)
	}

	return nil
}
