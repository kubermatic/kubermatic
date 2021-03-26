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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	certmanagermetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		if !opt.EnableCertManagerV2Migration {
			sublogger.Warn("cert-manager CRDs need to be migrated. This requires to temporarily remove and recreate")
			sublogger.Warn("all related resources (like Certificates, Issuers, etc.). Rerun the installer with")
			sublogger.Warn("--migrate-cert-manager to enable this mandatory migration.")
			sublogger.Warn("Please refer to the KKP 2.17 upgrade notes for more information.")

			return fmt.Errorf("user must acknowledge the migration using --migrate-cert-manager")
		}

		if err := migrateCertManagerV2(ctx, sublogger, kubeClient, helmClient, opt, chart, release); err != nil {
			return fmt.Errorf("upgrade failed: %v", err)
		}
	} else {
		sublogger.Info("Deploying Custom Resource Definitions…")
		if err := util.DeployCRDs(ctx, kubeClient, sublogger, filepath.Join(chartDir, "crd")); err != nil {
			return fmt.Errorf("failed to deploy CRDs: %v", err)
		}
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

// migrateCertManagerV2 removes all tracecs of cert-manager from the cluster,
// so that the installer can then install it cleanly.
func migrateCertManagerV2(
	ctx context.Context,
	logger *logrus.Entry,
	kubeClient ctrlruntimeclient.Client,
	helmClient helm.Client,
	opt stack.DeployOptions,
	chart *helm.Chart,
	release *helm.Release,
) error {
	logger.Infof("%s detected, performing upgrade to %s…", release.Version.String(), chart.Version.String())

	// step 1: purge the Helm release
	logger.Info("Uninstalling release…")
	if err := helmClient.UninstallRelease(CertManagerNamespace, CertManagerReleaseName); err != nil {
		return fmt.Errorf("failed to uninstall release: %v", err)
	}

	now := time.Now().Format("2006-01-02T150405")

	// for these CRDs, we not only back them up as YAML, but also restore them
	// automatically so the user doesn't have to (and we filter out resources
	// with an ownerRef, which would indicate that some other process/object
	// manages a given certificate, for example)
	restorableCRDs := []schema.GroupVersionKind{
		// restore issuers and clusterissues before certs and requests
		{Version: "v1alpha2", Group: "cert-manager.io", Kind: "clusterissuer"},
		{Version: "v1alpha2", Group: "cert-manager.io", Kind: "issuer"},
		{Version: "v1alpha2", Group: "cert-manager.io", Kind: "certificaterequest"},
		{Version: "v1alpha2", Group: "cert-manager.io", Kind: "certificate"},
	}

	allCRDs := append(
		restorableCRDs,
		schema.GroupVersionKind{Version: "v1alpha2", Group: "acme.cert-manager.io", Kind: "challenge"},
		schema.GroupVersionKind{Version: "v1alpha2", Group: "acme.cert-manager.io", Kind: "order"},
	)

	// step 2: fetch restorable resources in memory
	logger.Info("Creating backups for all Custom Resources…")
	objectsToRestore, secrets, err := getCustomResources(ctx, logger, kubeClient, restorableCRDs)
	if err != nil {
		return fmt.Errorf("failed to list resources: %v", err)
	}

	// step 3: backup resources into files
	for _, crdGVK := range allCRDs {
		logger.Infof("  dumping %s", crdGVK.Kind)

		filename := fmt.Sprintf("backup_%s_%s.yaml", now, crdGVK.Kind)
		if err := util.BackupResources(ctx, kubeClient, crdGVK, filename); err != nil {
			return fmt.Errorf("failed to backup %s resources: %v", crdGVK.Kind, err)
		}
	}

	logger.Infof("  dumping secret")
	filename := fmt.Sprintf("backup_%s_secret.yaml", now)
	if err := util.DumpResources(ctx, filename, secrets); err != nil {
		return fmt.Errorf("failed to backup secret resources: %v", err)
	}

	// step 4: remove finalizers from resources
	logger.Info("Removing finalizers from Custom Resources…")
	if err := removeFinalizersFromCustomResources(ctx, kubeClient, allCRDs, []string{"finalizer.acme.cert-manager.io"}); err != nil {
		return fmt.Errorf("failed to remove finalizers: %v", err)
	}

	// step 5: delete all cert-manager CRDs
	logger.Info("Removing Custom Resource Definitions…")
	for _, crdGVK := range allCRDs {
		crd := apiextensionsv1.CustomResourceDefinition{}
		key := types.NamespacedName{Name: crdName(crdGVK)}

		if err := kubeClient.Get(ctx, key, &crd); err != nil {
			if kerrors.IsNotFound(err) {
				continue
			}

			return fmt.Errorf("failed to retrieve CRD %s: %v", crdGVK.Kind, err)
		}

		if err := kubeClient.Delete(ctx, &crd); err != nil {
			return fmt.Errorf("failed to delete CRD %s: %v", crdGVK.Kind, err)
		}
	}

	// wait for all CRs to be gone; we do this now after deleting all
	// CRDs so that as many CRs as possible can be cleaned up, if e.g.
	// the first CRD already got stuck it doesn't block others from
	// being cleaned up
	hasErrors := false
	for _, crdGVK := range allCRDs {
		if err := util.WaitForCRDGone(ctx, kubeClient, crdName(crdGVK), 10*time.Second); err != nil {
			logger.Errorf("  %s could not be deleted, please check for remaining resources and remove any finalizers", crdName(crdGVK))
			hasErrors = true
		}
	}

	if hasErrors {
		logger.Warn("Remaining finalizers indicate third party controllers that due to the deleted")
		logger.Warn("CRDs cannot properly clean up anymore and must be resolved manually.")
		logger.Warn("After manual cleanup, ensure that all cert-manager CRDs are gone from the cluster.")
		logger.Warn("You can then re-run the installer and it will continue the migration.")
		return errors.New("cleanup failed")
	}

	// step 6: install new CRDs
	logger.Info("Deploying new Custom Resource Definitions…")
	if err := util.DeployCRDs(ctx, kubeClient, logger, filepath.Join(chart.Directory, "crd")); err != nil {
		return fmt.Errorf("failed to deploy CRDs: %v", err)
	}

	// step 7: recreate deleted resources
	logger.Info("Recreating deleted resources…")
	for _, object := range objectsToRestore {
		logger.Infof("  creating %s %s/%s", object.GroupVersionKind().Kind, object.GetNamespace(), object.GetName())

		object.SetResourceVersion("")
		object.SetUID("")
		object.SetSelfLink("")

		// only log errors, but continue, as the user can easily fix
		// problems by using the YAML backup files
		if err := kubeClient.Create(ctx, &object); err != nil {
			if kerrors.IsAlreadyExists(err) {
				logger.Warn("  already exists, please compare to backup")
			} else {
				logger.Errorf("  failed: %v", err)
			}

			hasErrors = true
		}
	}

	if hasErrors {
		logger.Warn("Use the YAML backup files to manually recreate missing resources.")
	}

	return nil
}

// crdName returns the plural name of a CRD. It assumes the given GVK
// is in singular. This function is required because for _fetching_
// custom resources we need the singular name, but to fetch the CRD
// itself we need the plural name.
func crdName(gvk schema.GroupVersionKind) string {
	// make kind plural, the cheap and easy and brittle way
	gvk.Kind += "s"

	return gvk.GroupKind().String()
}

func getCustomResources(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, crds []schema.GroupVersionKind) ([]unstructured.Unstructured, []unstructured.Unstructured, error) {
	resources := []unstructured.Unstructured{}
	secrets := []unstructured.Unstructured{}

	for _, crdGVK := range crds {
		logger.Infof("  fetching %s", crdGVK.Kind)

		items, err := util.ListResources(ctx, kubeClient, crdGVK)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list %s resources: %v", crdGVK.Kind, err)
		}

		for idx := range items {
			item := items[idx]

			// we want to only restore resources that have not been automatically
			// created by cert-manager, e.g. via Ingress annotation, so we filter
			// out anything that has an owner ref; excluded resources are still
			// backed up as YAML
			if len(item.GetOwnerReferences()) == 0 {
				resources = append(resources, item)
			}

			// for certificates, we want to also dump the Secret that contains
			// the actual certificate data
			if crdGVK.Kind == "certificate" {
				secret, err := getSecretForCertificate(ctx, kubeClient, item)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to get Secret for certificate %s/%s: %v", item.GetNamespace(), item.GetName(), err)
				}

				if secret != nil {
					// as the Secret most likely has an ownerRef to the Certificate, we
					// must restore it _after_ the Certificate has been created
					resources = append(resources, *secret)

					// dump secrets later as well
					secrets = append(secrets, *secret)
				}
			}
		}
	}

	return resources, secrets, nil
}

func getSecretForCertificate(ctx context.Context, kubeClient ctrlruntimeclient.Client, unstructuredCert unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// convert to typed certificate
	bytes, err := unstructuredCert.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to encode certificate as JSON: %v", err)
	}

	cert := &certmanagerv1alpha2.Certificate{}
	if err := json.Unmarshal(bytes, cert); err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// invalid cert
	if cert.Spec.SecretName == "" {
		return nil, nil
	}

	// just because a SecretName is set, does not mean it exists;
	// we could check the cert Status and parse the conditions, but
	// it's easier to just try to fetch the secret and see what happens
	secret := &v1.Secret{}
	if err := kubeClient.Get(ctx, types.NamespacedName{
		Name:      cert.Spec.SecretName,
		Namespace: cert.Namespace,
	}, secret); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("failed to retrieve Secret %q for certificate: %v", cert.Spec.SecretName, err)
	}

	// convert back to unstructured to make the surrounding handling
	// code easier
	bytes, err = json.Marshal(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Secret as JSON: %v", err)
	}

	result := &unstructured.Unstructured{}
	if err := result.UnmarshalJSON(bytes); err != nil {
		return nil, fmt.Errorf("failed to decode Secret: %v", err)
	}

	return result, nil
}

func removeFinalizersFromCustomResources(ctx context.Context, kubeClient ctrlruntimeclient.Client, crds []schema.GroupVersionKind, finalizers []string) error {
	for _, crdGVK := range crds {
		items, err := util.ListResources(ctx, kubeClient, crdGVK)
		if err != nil {
			return fmt.Errorf("failed to list %s resources: %v", crdGVK.Kind, err)
		}

		for idx := range items {
			item := items[idx]

			if kubernetes.HasAnyFinalizer(&item, finalizers...) {
				oldItem := item.DeepCopy()
				kubernetes.RemoveFinalizer(&item, finalizers...)

				if err := kubeClient.Patch(ctx, &item, ctrlruntimeclient.MergeFrom(oldItem)); err != nil {
					return fmt.Errorf("failed to patch %s %s/%s: %v", crdGVK.Kind, item.GetNamespace(), item.GetName(), err)
				}
			}
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
		// it can take a moment for the cert to appear
		time.Sleep(3 * time.Second)

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
			IssuerRef: certmanagermetav1.ObjectReference{
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
