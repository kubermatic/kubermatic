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

package kubermatic

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NginxIngressControllerChartName   = "nginx-ingress-controller"
	NginxIngressControllerReleaseName = NginxIngressControllerChartName
	NginxIngressControllerNamespace   = NginxIngressControllerChartName

	CertManagerChartName   = "cert-manager"
	CertManagerReleaseName = CertManagerChartName
	CertManagerNamespace   = CertManagerChartName

	DexChartName   = "oauth"
	DexReleaseName = DexChartName
	DexNamespace   = DexChartName

	KubermaticOperatorChartName   = "kubermatic-operator"
	KubermaticOperatorReleaseName = KubermaticOperatorChartName
	KubermaticOperatorNamespace   = "kubermatic"

	StorageClassName = "kubermatic-fast"
)

type Options struct {
	StorageClassProvider       string
	HelmValues                 *yamled.Document
	KubermaticConfiguration    *operatorv1alpha1.KubermaticConfiguration
	RawKubermaticConfiguration *unstructured.Unstructured
	ForceHelmReleaseUpgrade    bool
	ChartsDirectory            string
}

func Deploy(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	if err := deployStorageClass(ctx, logger, kubeClient, opt); err != nil {
		return fmt.Errorf("failed to deploy StorageClass: %v", err)
	}

	if err := deployNginxIngressController(ctx, logger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy nginx-ingress-controller: %v", err)
	}

	if err := deployCertManager(ctx, logger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy cert-manager: %v", err)
	}

	if err := deployDex(ctx, logger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Dex: %v", err)
	}

	if err := deployKubermaticOperator(ctx, logger, kubeClient, helmClient, opt); err != nil {
		return fmt.Errorf("failed to deploy Kubermatic Operator: %v", err)
	}

	if err := applyKubermaticConfiguration(ctx, logger, kubeClient, opt); err != nil {
		return fmt.Errorf("failed to apply Kubermatic Configuration: %v", err)
	}

	showDNSSettings(ctx, logger, kubeClient, opt)

	return nil
}

func deployStorageClass(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt Options) error {
	logger.Infof("💾 Deploying %s StorageClass…", StorageClassName)
	sublogger := log.Prefix(logger, "   ")

	cls := storagev1.StorageClass{}
	key := types.NamespacedName{Name: StorageClassName}

	err := kubeClient.Get(ctx, key, &cls)

	// storage class exists already
	if err == nil {
		logger.Info("✅ StorageClass exists, nothing to do.")
		return nil
	}

	if !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for StorageClass %s: %v", StorageClassName, err)
	}

	if opt.StorageClassProvider == "" {
		sublogger.Warnf("The %s StorageClass does not exist yet. Depending on your environment,", StorageClassName)
		sublogger.Warn("the installer can auto-create a class for you, see the --storageclass CLI flag.")
		sublogger.Warn("Alternatively, please manually create a StorageClass and then re-run the installer to continue.")

		return fmt.Errorf("no %s StorageClass found", StorageClassName)
	}

	factory := storageClassFactories[opt.StorageClassProvider]

	sc, err := factory(ctx, sublogger, kubeClient, StorageClassName)
	if err != nil {
		return fmt.Errorf("failed to define StorageClass: %v", err)
	}

	if err := kubeClient.Create(ctx, &sc); err != nil {
		return fmt.Errorf("failed to create StorageClass: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployNginxIngressController(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	logger.Info("📦 Deploying nginx-ingress-controller…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, "nginx-ingress-controller"))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, NginxIngressControllerNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, NginxIngressControllerNamespace, NginxIngressControllerReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	// do not perform an atomic installation, as this will make Helm wait for the LoadBalancer to
	// get an IP and this can require manual intervention based on the target environment
	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, NginxIngressControllerNamespace, NginxIngressControllerReleaseName, opt.HelmValues, false, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployCertManager(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
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

	sublogger.Info("Deploying Custom Resource Definitions…")
	if err := util.DeployCRDs(ctx, kubeClient, sublogger, filepath.Join(chartDir, "crd")); err != nil {
		return fmt.Errorf("failed to deploy CRDs: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, CertManagerNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	sublogger.Info("Deploying Helm chart…")
	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, CertManagerNamespace, CertManagerReleaseName)
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

func deployDex(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	logger.Info("📦 Deploying Dex…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, "oauth"))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, DexNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, DexNamespace, DexReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, DexNamespace, DexReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func deployKubermaticOperator(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, helmClient helm.Client, opt Options) error {
	logger.Info("📦 Deploying Kubermatic Operator…")
	sublogger := log.Prefix(logger, "   ")

	chart, err := helm.LoadChart(filepath.Join(opt.ChartsDirectory, "kubermatic-operator"))
	if err != nil {
		return fmt.Errorf("failed to load Helm chart: %v", err)
	}

	sublogger.Info("Deploying Custom Resource Definitions…")
	if err := util.DeployCRDs(ctx, kubeClient, sublogger, filepath.Join(opt.ChartsDirectory, "kubermatic", "crd")); err != nil {
		return fmt.Errorf("failed to deploy CRDs: %v", err)
	}

	if err := util.EnsureNamespace(ctx, sublogger, kubeClient, KubermaticOperatorNamespace); err != nil {
		return fmt.Errorf("failed to create namespace: %v", err)
	}

	sublogger.Info("Deploying Helm chart…")
	release, err := util.CheckHelmRelease(ctx, sublogger, helmClient, KubermaticOperatorNamespace, KubermaticOperatorReleaseName)
	if err != nil {
		return fmt.Errorf("failed to check to Helm release: %v", err)
	}

	if err := util.DeployHelmChart(ctx, sublogger, helmClient, chart, KubermaticOperatorNamespace, KubermaticOperatorReleaseName, opt.HelmValues, true, opt.ForceHelmReleaseUpgrade, release); err != nil {
		return fmt.Errorf("failed to deploy Helm release: %v", err)
	}

	logger.Info("✅ Success.")

	return nil
}

func applyKubermaticConfiguration(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt Options) error {
	logger.Info("📝 Applying Kubermatic Configuration…")

	existingConfig := &operatorv1alpha1.KubermaticConfiguration{}
	name := types.NamespacedName{
		Name:      opt.KubermaticConfiguration.Name,
		Namespace: opt.KubermaticConfiguration.Namespace,
	}

	err := kubeClient.Get(ctx, name, existingConfig)
	if err != nil && !kerrors.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing KubermaticConfiguration: %v", err)
	}

	if err == nil {
		// found existing config, need to patch it
		opt.RawKubermaticConfiguration.SetResourceVersion(existingConfig.ResourceVersion)
		opt.RawKubermaticConfiguration.SetAnnotations(existingConfig.Annotations)
		opt.RawKubermaticConfiguration.SetLabels(existingConfig.Labels)
		opt.RawKubermaticConfiguration.SetFinalizers(existingConfig.Finalizers)
		opt.RawKubermaticConfiguration.SetOwnerReferences(existingConfig.OwnerReferences)

		err = kubeClient.Patch(ctx, opt.RawKubermaticConfiguration, ctrlruntimeclient.MergeFrom(existingConfig))
	} else {
		// no config exists yet
		err = kubeClient.Create(ctx, opt.RawKubermaticConfiguration)
	}

	logger.Info("✅ Success.")

	return err
}

// showDNSSettings attempts to inform the user about required DNS settings
// to be made. If errors happen, only warnings areprinted, but the installation
// can still succeed.
func showDNSSettings(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt Options) {
	logger.Info("📡 Determining DNS settings…")
	sublogger := log.Prefix(logger, "   ")

	if opt.KubermaticConfiguration.Spec.Ingress.Disable {
		sublogger.Info("Ingress creation has been disabled in the KubermaticConfiguration, skipping.")
		return
	}

	hostNetwork, _ := opt.HelmValues.GetBool(yamled.Path{"nginx", "hostNetwork"})

	// in hostNetwork mode, nginx is deployed as a DaemonSet and the DNS
	// records need to point to one or more worker nodes directly;
	// normally we're not using the host network, but a regular LoadBalancer
	if hostNetwork {
		showHostNetworkDNSSettings(ctx, sublogger, kubeClient, opt)
	} else {
		showLoadBalancerDNSSettings(ctx, sublogger, kubeClient, opt)
	}
}

type nginxTargetPod struct {
	pod string
	ip  string
	dns string
}

func (t nginxTargetPod) prefererdTarget() string {
	if len(t.dns) > 0 {
		return t.dns
	}

	return t.ip
}

func showHostNetworkDNSSettings(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt Options) {
	logger.Debugf("Listing nginx-ingress-controller pods…")

	podList := v1.PodList{}
	err := kubeClient.List(ctx, &podList, &ctrlruntimeclient.ListOptions{
		Namespace: NginxIngressControllerNamespace,
	})
	if err != nil {
		logger.Warnf("Failed to find any nginx-ingress-controller pods: %v", err)
		logger.Warn("Please check the DaemonSet and, if necessary, reconfigure the")
		logger.Warn("nginx-ingress-controller Helm chart. Re-run the installer to apply")
		logger.Warn("updated configuration afterwards.")
		return
	}

	if len(podList.Items) == 0 {
		logger.Warnf("No nginx-ingress-controller pods were found in the %q namespace.", NginxIngressControllerNamespace)
		logger.Warn("Please check the DaemonSet and, if necessary, reconfigure the")
		logger.Warn("nginx-ingress-controller Helm chart. Re-run the installer to apply")
		logger.Warn("updated configuration afterwards.")
		return
	}

	nodeList := v1.NodeList{}
	err = kubeClient.List(ctx, &nodeList)
	if err != nil {
		logger.Warnf("Failed to retrieve nodes: %v", err)
		return
	}

	targets := []nginxTargetPod{}

	for _, pod := range podList.Items {
		hostIP := pod.Status.HostIP

		for _, node := range nodeList.Items {
			matches := false
			externalIP := ""
			externalDNS := ""

			for _, address := range node.Status.Addresses {
				switch address.Type {
				case v1.NodeExternalIP:
					externalIP = address.Address
				case v1.NodeExternalDNS:
					externalDNS = address.Address
				}

				if address.Address == hostIP {
					matches = true
					// do not break, so we can try more addresses
					// to find the external IP and DNS names
				}
			}

			if matches && (externalIP != "" || externalDNS != "") {
				targets = append(targets, nginxTargetPod{
					pod: pod.Name,
					ip:  externalIP,
					dns: externalDNS,
				})
			}
		}
	}

	if len(targets) == 0 {
		logger.Warnf("No nginx-ingress-controller pods in the %q namespace are scheduled onto nodes yet.", NginxIngressControllerNamespace)
		logger.Warn("Please check the DaemonSet and, if necessary, reconfigure the")
		logger.Warn("nginx-ingress-controller Helm chart. Re-run the installer to apply")
		logger.Warn("updated configuration afterwards.")
		return
	}

	logger.Info("The nginx-ingress-controller pods have been rolled out in your cluster.")
	logger.Info("")

	logger.Infof("  %-50s    IP / Hostname", "Pod")
	for _, target := range targets {
		logger.Infof("  %-50s    %s", target.pod, target.prefererdTarget())
	}

	domain := opt.KubermaticConfiguration.Spec.Ingress.Domain
	rand := targets[rand.Intn(len(targets))]

	logger.Info("")
	logger.Info("Choose a single target for your DNS or configure an external LoadBalancer")
	logger.Info("to balance between all targets listed above. For a basic setup, choose a")
	logger.Infof("random target from above, for example %s.", rand.prefererdTarget())
	logger.Infof("Then ensure your DNS settings for %q include the following records:", domain)
	logger.Info("")

	if rand.dns != "" {
		logger.Infof("   %s.    IN  CNAME  %s.", domain, rand.dns)
		logger.Infof("   *.%s.  IN  CNAME  %s.", domain, rand.dns)
	} else {
		logger.Infof("   %s.    IN  A  %s", domain, rand.ip)
		logger.Infof("   *.%s.  IN  A  %s", domain, rand.ip)
	}

	logger.Info("")
}

func showLoadBalancerDNSSettings(ctx context.Context, logger *logrus.Entry, kubeClient ctrlruntimeclient.Client, opt Options) {
	svcName := types.NamespacedName{
		Namespace: NginxIngressControllerNamespace,
		Name:      "nginx-ingress-controller",
	}

	logger.Debugf("Waiting for %q to be ready…", svcName)

	var ingresses []v1.LoadBalancerIngress
	err := wait.PollImmediate(5*time.Second, 3*time.Minute, func() (bool, error) {
		svc := v1.Service{}
		if err := kubeClient.Get(ctx, svcName, &svc); err != nil {
			return false, err
		}

		ingresses = svc.Status.LoadBalancer.Ingress

		return len(ingresses) > 0, nil
	})
	if err != nil {
		logger.Warnf("Timed out waiting for the LoadBalancer service %q to become ready.", svcName)
		logger.Warn("Please check the Service and, if necessary, reconfigure the")
		logger.Warn("nginx-ingress-controller Helm chart. Re-run the installer to apply")
		logger.Warn("updated configuration afterwards.")
		return
	}

	logger.Info("The main LoadBalancer is ready.")
	logger.Info("")
	logger.Infof("  Service             : %s / %s", svcName.Namespace, svcName.Name)

	domain := opt.KubermaticConfiguration.Spec.Ingress.Domain

	if hostname := ingresses[0].Hostname; hostname != "" {
		logger.Infof("  Ingress via hostname: %s", hostname)
		logger.Info("")
		logger.Infof("Please ensure your DNS settings for %q include the following records:", domain)
		logger.Info("")
		logger.Infof("   %s.    IN  CNAME  %s.", domain, hostname)
		logger.Infof("   *.%s.  IN  CNAME  %s.", domain, hostname)
	} else if ip := ingresses[0].IP; ip != "" {
		logger.Infof("  Ingress via IP      : %s", ip)
		logger.Info("")
		logger.Infof("Please ensure your DNS settings for %q include the following records:", domain)
		logger.Info("")
		logger.Infof("   %s.    IN  A  %s", domain, ip)
		logger.Infof("   *.%s.  IN  A  %s", domain, ip)
	}

	logger.Info("")
}
