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

package images

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common/vpa"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	masteroperator "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/kubermatic"
	seedoperatorkubermatic "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	seedoperatornodeportproxy "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	kubernetescontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/kubernetes"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/monitoring"
	nodelocaldns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/node-local-dns"
	"k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/usersshkeys"
	"k8c.io/kubermatic/v2/pkg/docker"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	metricsserver "k8c.io/kubermatic/v2/pkg/resources/metrics-server"
	"k8c.io/kubermatic/v2/pkg/resources/operatingsystemmanager"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/version"
	"k8c.io/kubermatic/v2/pkg/version/cni"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const mockNamespaceName = "mock-namespace"

func extractAddonsFromDockerImage(ctx context.Context, log *zap.SugaredLogger, imageName string) (string, error) {
	tempDir, err := os.MkdirTemp("", "imageloader*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	log.Infow("Extracting addon manifests from Docker image", "image", imageName, "temp-directory", tempDir)

	if err := docker.DownloadImages(ctx, log, false, []string{imageName}); err != nil {
		return tempDir, fmt.Errorf("failed to download addons image: %w", err)
	}

	if err := docker.Copy(ctx, log, imageName, tempDir, "/addons"); err != nil {
		return tempDir, fmt.Errorf("failed to extract addons: %w", err)
	}

	return tempDir, nil
}

func processImages(ctx context.Context, log *zap.SugaredLogger, dryRun bool, images []string, registry string) error {
	if err := docker.DownloadImages(ctx, log, dryRun, images); err != nil {
		return fmt.Errorf("failed to download all images: %w", err)
	}

	retaggedImages, err := docker.RetagImages(ctx, log, dryRun, images, registry)
	if err != nil {
		return fmt.Errorf("failed to re-tag images: %w", err)
	}

	if err := docker.PushImages(ctx, log, dryRun, retaggedImages); err != nil {
		return fmt.Errorf("failed to push images: %w", err)
	}
	return nil
}

func getImagesForVersion(log *zap.SugaredLogger, clusterVersion *version.Version, cloudSpec kubermaticv1.CloudSpec, cniPlugin *kubermaticv1.CNIPluginSettings, config *kubermaticv1.KubermaticConfiguration, addonsPath string, kubermaticVersions kubermatic.Versions, caBundle resources.CABundle) (images []string, err error) {
	templateData, err := getTemplateData(clusterVersion, cloudSpec, cniPlugin, kubermaticVersions, caBundle)
	if err != nil {
		return nil, err
	}

	creatorImages, err := getImagesFromCreators(log, templateData, config, kubermaticVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to get images from internal creator functions: %w", err)
	}
	images = append(images, creatorImages...)

	addonImages, err := getImagesFromAddons(log, addonsPath, templateData.Cluster())
	if err != nil {
		return nil, fmt.Errorf("failed to get images from addons: %w", err)
	}
	images = append(images, addonImages...)

	return images, nil
}

func getImagesFromCreators(log *zap.SugaredLogger, templateData *resources.TemplateData, config *kubermaticv1.KubermaticConfiguration, kubermaticVersions kubermatic.Versions) (images []string, err error) {
	seed, err := defaults.DefaultSeed(&kubermaticv1.Seed{}, config, log)
	if err != nil {
		return nil, fmt.Errorf("failed to default Seed: %w", err)
	}

	statefulsetCreators := kubernetescontroller.GetStatefulSetCreators(templateData, false, false)
	statefulsetCreators = append(statefulsetCreators, monitoring.GetStatefulSetCreators(templateData)...)

	deploymentCreators := kubernetescontroller.GetDeploymentCreators(templateData, false)
	deploymentCreators = append(deploymentCreators, monitoring.GetDeploymentCreators(templateData)...)
	deploymentCreators = append(deploymentCreators, masteroperator.APIDeploymentCreator(config, "", kubermaticVersions))
	deploymentCreators = append(deploymentCreators, masteroperator.MasterControllerManagerDeploymentCreator(config, "", kubermaticVersions))
	deploymentCreators = append(deploymentCreators, masteroperator.UIDeploymentCreator(config, kubermaticVersions))
	deploymentCreators = append(deploymentCreators, seedoperatorkubermatic.SeedControllerManagerDeploymentCreator("", kubermaticVersions, config, seed))
	deploymentCreators = append(deploymentCreators, seedoperatornodeportproxy.EnvoyDeploymentCreator(config, seed, false, kubermaticVersions))
	deploymentCreators = append(deploymentCreators, seedoperatornodeportproxy.UpdaterDeploymentCreator(config, seed, kubermaticVersions))
	deploymentCreators = append(deploymentCreators, vpa.AdmissionControllerDeploymentCreator(config, kubermaticVersions))
	deploymentCreators = append(deploymentCreators, vpa.RecommenderDeploymentCreator(config, kubermaticVersions))
	deploymentCreators = append(deploymentCreators, vpa.UpdaterDeploymentCreator(config, kubermaticVersions))
	deploymentCreators = append(deploymentCreators, mla.GatewayDeploymentCreator(templateData, nil))
	deploymentCreators = append(deploymentCreators, cloudcontroller.DeploymentCreator(templateData))
	deploymentCreators = append(deploymentCreators, operatingsystemmanager.DeploymentCreator(templateData))

	cronjobCreators := kubernetescontroller.GetCronJobCreators(templateData)

	var daemonsetCreators []reconciling.NamedDaemonSetCreatorGetter
	daemonsetCreators = append(daemonsetCreators, usersshkeys.DaemonSetCreator(
		kubermaticVersions,
		templateData.ImageRegistry,
	))
	daemonsetCreators = append(daemonsetCreators, nodelocaldns.DaemonSetCreator(templateData.ImageRegistry))

	for _, creatorGetter := range statefulsetCreators {
		_, creator := creatorGetter()
		statefulset, err := creator(&appsv1.StatefulSet{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(statefulset.Spec.Template.Spec)...)
	}

	for _, createFunc := range deploymentCreators {
		_, creator := createFunc()
		deployment, err := creator(&appsv1.Deployment{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(deployment.Spec.Template.Spec)...)
	}

	for _, createFunc := range cronjobCreators {
		_, creator := createFunc()
		cronJob, err := creator(&batchv1beta1.CronJob{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(cronJob.Spec.JobTemplate.Spec.Template.Spec)...)
	}

	for _, createFunc := range daemonsetCreators {
		_, creator := createFunc()
		daemonset, err := creator(&appsv1.DaemonSet{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(daemonset.Spec.Template.Spec)...)
	}

	return images, nil
}

func getImagesFromPodSpec(spec corev1.PodSpec) (images []string) {
	for _, initContainer := range spec.InitContainers {
		images = append(images, initContainer.Image)
	}

	for _, container := range spec.Containers {
		images = append(images, container.Image)
	}

	return images
}

func getTemplateData(clusterVersion *version.Version, cloudSpec kubermaticv1.CloudSpec, cniPlugin *kubermaticv1.CNIPluginSettings, kubermaticVersions kubermatic.Versions, caBundle resources.CABundle) (*resources.TemplateData, error) {
	// We need listers and a set of objects to not have our deployment/statefulset creators fail
	cloudConfigConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.CloudConfigConfigMapName,
			Namespace: mockNamespaceName,
		},
	}
	caBundleConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.CABundleConfigMapName,
			Namespace: mockNamespaceName,
		},
	}
	prometheusConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.PrometheusConfigConfigMapName,
			Namespace: mockNamespaceName,
		},
	}
	dnsResolverConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.DNSResolverConfigMapName,
			Namespace: mockNamespaceName,
		},
	}
	openvpnClientConfigsConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.OpenVPNClientConfigsConfigMapName,
			Namespace: mockNamespaceName,
		},
	}
	auditConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.AuditConfigMapName,
			Namespace: mockNamespaceName,
		},
	}
	admissionControlConfigMapName := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.AdmissionControlConfigMapName,
			Namespace: mockNamespaceName,
		},
	}
	configMapList := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			cloudConfigConfigMap,
			caBundleConfigMap,
			prometheusConfigMap,
			dnsResolverConfigMap,
			openvpnClientConfigsConfigMap,
			auditConfigMap,
			admissionControlConfigMapName,
		},
	}
	apiServerService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.ApiserverServiceName,
			Namespace: mockNamespaceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{NodePort: 99}},
			ClusterIP: "192.0.2.10",
		},
	}
	openvpnserverService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.OpenVPNServerServiceName,
			Namespace: mockNamespaceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{NodePort: 96}},
			ClusterIP: "192.0.2.2",
		},
	}
	dnsService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.DNSResolverServiceName,
			Namespace: mockNamespaceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{NodePort: 98}},
			ClusterIP: "192.0.2.11",
		},
	}
	serviceList := &corev1.ServiceList{
		Items: []corev1.Service{
			apiServerService,
			openvpnserverService,
			dnsService,
		},
	}
	secretList := createNamedSecrets([]string{
		resources.CASecretName,
		resources.TokensSecretName,
		resources.ApiserverTLSSecretName,
		resources.KubeletClientCertificatesSecretName,
		resources.ServiceAccountKeySecretName,
		resources.ApiserverEtcdClientCertificateSecretName,
		resources.ApiserverFrontProxyClientCertificateSecretName,
		resources.EtcdTLSCertificateSecretName,
		resources.MachineControllerKubeconfigSecretName,
		resources.ControllerManagerKubeconfigSecretName,
		resources.SchedulerKubeconfigSecretName,
		resources.KubeStateMetricsKubeconfigSecretName,
		resources.OpenVPNCASecretName,
		resources.OpenVPNServerCertificatesSecretName,
		resources.OpenVPNClientCertificatesSecretName,
		resources.FrontProxyCASecretName,
		resources.KubeletDnatControllerKubeconfigSecretName,
		resources.PrometheusApiserverClientCertificateSecretName,
		resources.MetricsServerKubeconfigSecretName,
		resources.MachineControllerWebhookServingCertSecretName,
		resources.InternalUserClusterAdminKubeconfigSecretName,
		resources.ClusterAutoscalerKubeconfigSecretName,
		resources.KubernetesDashboardKubeconfigSecretName,
		metricsserver.ServingCertSecretName,
		resources.UserSSHKeys,
		resources.AdminKubeconfigSecretName,
		resources.GatekeeperWebhookServerCertSecretName,
		resources.OperatingSystemManagerKubeconfigSecretName,
	})
	datacenter := &kubermaticv1.Datacenter{
		Spec: kubermaticv1.DatacenterSpec{
			VSphere:   &kubermaticv1.DatacenterSpecVSphere{},
			Openstack: &kubermaticv1.DatacenterSpecOpenstack{},
			Hetzner:   &kubermaticv1.DatacenterSpecHetzner{},
			Anexia:    &kubermaticv1.DatacenterSpecAnexia{},
			Kubevirt:  &kubermaticv1.DatacenterSpecKubevirt{},
		},
	}
	objects := []runtime.Object{configMapList, secretList, serviceList}

	clusterSemver, err := ksemver.NewSemver(clusterVersion.Version.String())
	if err != nil {
		return nil, err
	}
	fakeCluster := &kubermaticv1.Cluster{}
	fakeCluster.Spec.Cloud = cloudSpec
	fakeCluster.Spec.Version = *clusterSemver
	fakeCluster.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
	fakeCluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
	fakeCluster.Spec.ClusterNetwork.DNSDomain = "cluster.local"
	fakeCluster.Spec.CNIPlugin = cniPlugin

	if fakeCluster.Spec.Cloud.Openstack != nil {
		if fakeCluster.Spec.Features == nil {
			fakeCluster.Spec.Features = make(map[string]bool)
		}
		fakeCluster.Spec.Features[kubermaticv1.ClusterFeatureExternalCloudProvider] = true
	}

	fakeCluster.Spec.EnableUserSSHKeyAgent = pointer.Bool(true)
	fakeCluster.Spec.EnableOperatingSystemManager = true

	fakeCluster.Status.NamespaceName = mockNamespaceName
	fakeCluster.Status.Versions.ControlPlane = *clusterSemver
	fakeCluster.Status.Versions.Apiserver = *clusterSemver
	fakeCluster.Status.Versions.ControllerManager = *clusterSemver
	fakeCluster.Status.Versions.Scheduler = *clusterSemver

	fakeDynamicClient := fake.NewClientBuilder().WithRuntimeObjects(objects...).Build()

	return resources.NewTemplateDataBuilder().
		WithContext(context.Background()).
		WithClient(fakeDynamicClient).
		WithCluster(fakeCluster).
		WithDatacenter(datacenter).
		WithSeed(&kubermaticv1.Seed{}).
		WithNodeAccessNetwork("192.0.2.0/24").
		WithEtcdDiskSize(resource.Quantity{}).
		WithKubermaticImage(defaults.DefaultKubermaticImage).
		WithEtcdLauncherImage(defaults.DefaultEtcdLauncherImage).
		WithDnatControllerImage(defaults.DefaultDNATControllerImage).
		WithBackupPeriod(20 * time.Minute).
		WithFailureDomainZoneAntiaffinity(false).
		WithVersions(kubermaticVersions).
		WithCABundle(caBundle).
		Build(), nil
}

func createNamedSecrets(secretNames []string) *corev1.SecretList {
	secretList := corev1.SecretList{}
	for _, secretName := range secretNames {
		secret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: mockNamespaceName,
			},
		}
		secretList.Items = append(secretList.Items, secret)
	}
	return &secretList
}

func getVersions(log *zap.SugaredLogger, config *kubermaticv1.KubermaticConfiguration, versionsFile, versionFilter string) ([]*version.Version, error) {
	var versions []*version.Version

	log = log.With("versions-filter", versionFilter)

	if config != nil {
		log.Debug("Loading versions")
		versions = getVersionsFromKubermaticConfiguration(config)
	} else {
		if versionsFile == "" {
			return nil, errors.New("either a KubermaticConfiguration or a versions file must be specified")
		}

		var err error

		log.Debugw("Loading versions", "file", versionsFile)
		versions, err = version.LoadVersions(versionsFile)
		if err != nil {
			return nil, err
		}
	}

	if versionFilter == "" {
		return versions, nil
	}

	log.Debug("Filtering versions")
	constraint, err := semverlib.NewConstraint(versionFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version filter %q: %w", versionFilter, err)
	}

	var filteredVersions []*version.Version
	for _, ver := range versions {
		if constraint.Check(ver.Version) {
			filteredVersions = append(filteredVersions, ver)
		}
	}

	return filteredVersions, nil
}

// list all the cloudSpecs for all the Cloud providers for which we are currently using the external CCM/CSI.
func getCloudSpecs() []kubermaticv1.CloudSpec {
	return []kubermaticv1.CloudSpec{
		{
			ProviderName: string(kubermaticv1.VSphereCloudProvider),
			VSphere:      &kubermaticv1.VSphereCloudSpec{},
		},
		{
			ProviderName: string(kubermaticv1.OpenstackCloudProvider),
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Domain:   "fakeDomain",
				Username: "fakeUsername",
				Password: "fakePassword",
			},
		},
		{
			ProviderName: string(kubermaticv1.HetznerCloudProvider),
			Hetzner: &kubermaticv1.HetznerCloudSpec{
				Token:   "fakeToken",
				Network: "fakeNetwork",
			},
		},
		{
			ProviderName: string(kubermaticv1.AnexiaCloudProvider),
			Anexia: &kubermaticv1.AnexiaCloudSpec{
				Token: "fakeToken",
			},
		},
		{
			ProviderName: string(kubermaticv1.KubevirtCloudProvider),
			Kubevirt: &kubermaticv1.KubevirtCloudSpec{
				Kubeconfig:    "fakeKubeconfig",
				CSIKubeconfig: "fakeKubeconfig",
			},
		},
	}
}

// list all the supported CNI plugins along with their supported versions.
func getCNIPlugins() []*kubermaticv1.CNIPluginSettings {
	cniPluginSettings := []*kubermaticv1.CNIPluginSettings{}
	supportedCNIPlugins := cni.GetSupportedCNIPlugins()

	for _, cniPlugin := range supportedCNIPlugins.List() {
		// error cannot ever occur since we just listed the supported CNIPluginTypes
		versions, _ := cni.GetAllowedCNIPluginVersions(kubermaticv1.CNIPluginType(cniPlugin))

		for _, version := range versions.List() {
			cniPluginSettings = append(cniPluginSettings, &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginType(cniPlugin),
				Version: version,
			})
		}
	}

	return cniPluginSettings
}

func getImagesFromManifest(log *zap.SugaredLogger, decoder runtime.Decoder, b []byte) ([]string, error) {
	obj, err := runtime.Decode(decoder, b)
	if err != nil {
		if runtime.IsNotRegisteredError(err) {
			// We must skip custom objects. We try to look up the object info though to give a useful warning
			metaFactory := &json.SimpleMetaFactory{}
			if gvk, err := metaFactory.Interpret(b); err == nil {
				log = log.With(zap.String("gvk", gvk.String()))
			}

			log.Debug("Skipping object because its not known")
			return nil, nil
		}
		return nil, fmt.Errorf("unable to decode object: %w", err)
	}

	images := getImagesFromObject(obj)
	if images == nil {
		log.Debug("Object has no images or is not known to this application. If this object contains images, please manually push the image to the target registry")
		return nil, nil
	}

	return images, nil
}

func getImagesFromObject(obj runtime.Object) []string {
	// We don't have the conversion funcs available thus we must check all available Kubernetes types which can contain images
	switch obj := obj.(type) {
	case *appsv1.Deployment:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1.ReplicaSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1.StatefulSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1.DaemonSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *corev1.Pod:
		return getImagesFromPodSpec(obj.Spec)

	// CronJob
	case *batchv1.CronJob:
		return getImagesFromPodSpec(obj.Spec.JobTemplate.Spec.Template.Spec)
	case *batchv1beta1.CronJob:
		return getImagesFromPodSpec(obj.Spec.JobTemplate.Spec.Template.Spec)

	// Job
	case *batchv1.Job:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	}

	return nil
}
