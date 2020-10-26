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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Masterminds/semver"
	"go.uber.org/zap"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common/vpa"
	masteroperator "k8c.io/kubermatic/v2/pkg/controller/operator/master/resources/kubermatic"
	seedoperatorkubermatic "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/kubermatic"
	seedoperatornodeportproxy "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	kubernetescontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/kubernetes"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/monitoring"
	containerlinux "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/container-linux"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/docker"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	metricsserver "k8c.io/kubermatic/v2/pkg/resources/metrics-server"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	batchv2alpha1 "k8s.io/api/batch/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	extensionv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const mockNamespaceName = "mock-namespace"

var (
	staticImages = []string{}
)

type opts struct {
	configurationFile string
	versionsFile      string
	versionFilter     string
	registry          string
	dryRun            bool
	addonsPath        string
	addonsImage       string
	chartsPath        string
	helmValuesPath    string
	helmBinary        string
}

func main() {
	var err error

	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.Format = kubermaticlog.FormatConsole
	logOpts.AddFlags(flag.CommandLine)

	o := opts{}
	flag.StringVar(&o.configurationFile, "configuration-file", "", "Path to the KubermaticConfiguration YAML file")
	flag.StringVar(&o.versionsFile, "versions-file", "", "The versions.yaml file path (deprecated, EE-only, used only if no -configuration-file is given)")
	flag.StringVar(&o.versionFilter, "version-filter", "", "Version constraint which can be used to filter for specific versions")
	flag.StringVar(&o.registry, "registry", "", "Address of the registry to push to, for example localhost:5000")
	flag.BoolVar(&o.dryRun, "dry-run", false, "Only print the names of found images")
	flag.StringVar(&o.addonsPath, "addons-path", "", "Path to a directory containing the KKP addons, if not given, falls back to -addons-image, then the Docker image configured in the KubermaticConfiguration")
	flag.StringVar(&o.addonsImage, "addons-image", "", "Docker image containing KKP addons, if not given, falls back to the Docker image configured in the KubermaticConfiguration")
	flag.StringVar(&o.chartsPath, "charts-path", "", "Path to the folder containing all Helm charts")
	flag.StringVar(&o.helmValuesPath, "helm-values-file", "", "Use this values.yaml file when rendering the Helm charts (-charts-path)")
	flag.StringVar(&o.helmBinary, "helm-binary", "helm", "Helm 3.x binary to use for rendering the charts")
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	if (o.configurationFile == "") == (o.versionsFile == "") {
		log.Fatal("Either -configuration-file or -versions-file must be specified.")
	}

	if o.addonsPath != "" && o.addonsImage != "" {
		log.Fatal("-addons-path or -addons-image must not be specified at the same time.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signalCh := signals.SetupSignalHandler()
	go func() {
		<-signalCh
		cancel()
	}()

	if o.registry == "" {
		log.Fatal("-registry parameter must contain a valid registry address!")
	}

	// If given, load the KubermaticConfiguration. It's not yet a required
	// parameter in order to support Helm-based Enterprise setups.
	var kubermaticConfig *operatorv1alpha1.KubermaticConfiguration
	if o.configurationFile != "" {
		kubermaticConfig, err = loadKubermaticConfiguration(log, o.configurationFile)
		if err != nil {
			log.Fatalw("Failed to load KubermaticConfiguration", zap.Error(err))
		}
	}

	versions, err := getVersions(log, kubermaticConfig, o.versionsFile, o.versionFilter)
	if err != nil {
		log.Fatalw("Error loading versions", zap.Error(err))
	}

	// if no local addons path is given, use the configured addons
	// Docker image and extract the addons from there
	if o.addonsPath == "" {
		addonsImage := o.addonsImage
		if addonsImage == "" {
			if kubermaticConfig == nil {
				log.Warn("No KubermaticConfiguration, -addons-image or -addons-path given, cannot mirror images referenced in addons.")
			} else {
				v := common.NewDefaultVersions()
				addonsImage = kubermaticConfig.Spec.UserCluster.Addons.Kubernetes.DockerRepository + ":" + v.Kubermatic
			}
		}

		if addonsImage != "" {
			tempDir, err := extractAddonsFromDockerImage(ctx, log, addonsImage)
			if err != nil {
				log.Fatalw("Failed to create local addons path", zap.Error(err))
			}
			defer os.RemoveAll(tempDir)

			o.addonsPath = tempDir
		}
	}

	// Using a set here for deduplication
	imageSet := sets.NewString(staticImages...)
	for _, version := range versions {
		versionLog := log.With(
			zap.String("version", version.Version.String()),
			zap.String("cluster-type", version.Type),
		)
		if version.Type != "" && version.Type != apiv1.KubernetesClusterType {
			// TODO: Implement. https://github.com/kubermatic/kubermatic/issues/3623
			versionLog.Warn("Skipping version because its not for Kubernetes. We only support Kubernetes at the moment")
			continue
		}
		versionLog.Info("Collecting images...")
		images, err := getImagesForVersion(log, version, kubermaticConfig, o.addonsPath)
		if err != nil {
			versionLog.Fatalw("failed to get images", zap.Error(err))
		}
		imageSet.Insert(images...)
	}

	if o.chartsPath != "" {
		log.Infow("Rendering Helm charts", "directory", o.chartsPath)

		images, err := getImagesForHelmCharts(ctx, log, kubermaticConfig, o.chartsPath, o.helmValuesPath, o.helmBinary)
		if err != nil {
			log.Fatalw("Failed to get images", zap.Error(err))
		}
		imageSet.Insert(images...)
	}

	if err := processImages(ctx, log, o.dryRun, imageSet.List(), o.registry); err != nil {
		log.Fatalw("Failed to process images", zap.Error(err))
	}
}

func extractAddonsFromDockerImage(ctx context.Context, log *zap.SugaredLogger, imageName string) (string, error) {
	tempDir, err := ioutil.TempDir("", "imageloader*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}

	log.Infow("Extracting addon manifests from Docker image", "image", imageName, "temp-directory", tempDir)

	if err := docker.DownloadImages(ctx, log, false, []string{imageName}); err != nil {
		return tempDir, fmt.Errorf("failed to download addons image: %v", err)
	}

	if err := docker.Copy(ctx, log, imageName, tempDir, "/addons"); err != nil {
		return tempDir, fmt.Errorf("failed to extract addons: %v", err)
	}

	return tempDir, nil
}

func processImages(ctx context.Context, log *zap.SugaredLogger, dryRun bool, images []string, registry string) error {
	if err := docker.DownloadImages(ctx, log, dryRun, images); err != nil {
		return fmt.Errorf("failed to download all images: %v", err)
	}

	retaggedImages, err := docker.RetagImages(ctx, log, dryRun, images, registry)
	if err != nil {
		return fmt.Errorf("failed to re-tag images: %v", err)
	}

	if err := docker.PushImages(ctx, log, dryRun, retaggedImages); err != nil {
		return fmt.Errorf("failed to push images: %v", err)
	}
	return nil
}

func getImagesForVersion(log *zap.SugaredLogger, version *kubermaticversion.Version, config *operatorv1alpha1.KubermaticConfiguration, addonsPath string) (images []string, err error) {
	templateData, err := getTemplateData(version)
	if err != nil {
		return nil, err
	}

	creatorImages, err := getImagesFromCreators(log, templateData, config)
	if err != nil {
		return nil, fmt.Errorf("failed to get images from internal creator functions: %v", err)
	}
	images = append(images, creatorImages...)

	addonImages, err := getImagesFromAddons(log, addonsPath, templateData.Cluster())
	if err != nil {
		return nil, fmt.Errorf("failed to get images from addons: %v", err)
	}
	images = append(images, addonImages...)

	return images, nil
}

func getImagesFromCreators(log *zap.SugaredLogger, templateData *resources.TemplateData, config *operatorv1alpha1.KubermaticConfiguration) (images []string, err error) {
	v := common.NewDefaultVersions()

	seed, err := common.DefaultSeed(&kubermaticv1.Seed{}, log)
	if err != nil {
		return nil, fmt.Errorf("failed to default Seed: %v", err)
	}

	statefulsetCreators := kubernetescontroller.GetStatefulSetCreators(templateData, false)
	statefulsetCreators = append(statefulsetCreators, monitoring.GetStatefulSetCreators(templateData)...)

	deploymentCreators := kubernetescontroller.GetDeploymentCreators(templateData, false)
	deploymentCreators = append(deploymentCreators, monitoring.GetDeploymentCreators(templateData)...)
	deploymentCreators = append(deploymentCreators, containerlinux.GetDeploymentCreators("", kubermaticv1.UpdateWindow{})...)
	deploymentCreators = append(deploymentCreators, masteroperator.APIDeploymentCreator(config, "", v))
	deploymentCreators = append(deploymentCreators, masteroperator.MasterControllerManagerDeploymentCreator(config, "", v))
	deploymentCreators = append(deploymentCreators, masteroperator.UIDeploymentCreator(config, v))
	deploymentCreators = append(deploymentCreators, seedoperatorkubermatic.SeedControllerManagerDeploymentCreator("", v, config, seed))
	deploymentCreators = append(deploymentCreators, seedoperatornodeportproxy.EnvoyDeploymentCreator(seed, v))
	deploymentCreators = append(deploymentCreators, seedoperatornodeportproxy.UpdaterDeploymentCreator(seed, v))
	deploymentCreators = append(deploymentCreators, vpa.AdmissionControllerDeploymentCreator(config, v))
	deploymentCreators = append(deploymentCreators, vpa.RecommenderDeploymentCreator(config, v))
	deploymentCreators = append(deploymentCreators, vpa.UpdaterDeploymentCreator(config, v))

	cronjobCreators := kubernetescontroller.GetCronJobCreators(templateData)

	daemonSetCreators := containerlinux.GetDaemonSetCreators("")

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

	for _, createFunc := range daemonSetCreators {
		_, creator := createFunc()
		daemonSet, err := creator(&appsv1.DaemonSet{})
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodSpec(daemonSet.Spec.Template.Spec)...)
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

func getTemplateData(version *kubermaticversion.Version) (*resources.TemplateData, error) {
	// We need listers and a set of objects to not have our deployment/statefulset creators fail
	cloudConfigConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.CloudConfigConfigMapName,
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
	configMapList := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{cloudConfigConfigMap, prometheusConfigMap, dnsResolverConfigMap, openvpnClientConfigsConfigMap, auditConfigMap},
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
		resources.DexCASecretName,
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
	})
	objects := []runtime.Object{configMapList, secretList, serviceList}

	clusterVersion, err := ksemver.NewSemver(version.Version.String())
	if err != nil {
		return nil, err
	}
	fakeCluster := &kubermaticv1.Cluster{}
	fakeCluster.Spec.Cloud = kubermaticv1.CloudSpec{}
	fakeCluster.Spec.Version = *clusterVersion
	fakeCluster.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
	fakeCluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
	fakeCluster.Spec.ClusterNetwork.DNSDomain = "cluster.local"
	fakeCluster.Status.NamespaceName = mockNamespaceName

	fakeDynamicClient := fake.NewFakeClient(objects...)

	return resources.NewTemplateData(
		context.Background(),
		fakeDynamicClient,
		fakeCluster,
		&kubermaticv1.Datacenter{},
		&kubermaticv1.Seed{},
		"",
		"",
		"192.0.2.0/24",
		resource.Quantity{},
		"",
		"",
		false,
		false,
		"",
		"",
		"",
		"",
		true,
		// Since this is the image-loader we hardcode the default image for pulling.
		resources.DefaultKubermaticImage,
		resources.DefaultEtcdLauncherImage,
		resources.DefaultDNATControllerImage,
		false,
	), nil
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

func getVersions(log *zap.SugaredLogger, config *operatorv1alpha1.KubermaticConfiguration, versionsFile, versionFilter string) ([]*kubermaticversion.Version, error) {
	var versions []*kubermaticversion.Version

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
		versions, err = kubermaticversion.LoadVersions(versionsFile)
		if err != nil {
			return nil, err
		}
	}

	if versionFilter == "" {
		return versions, nil
	}

	log.Debug("Filtering versions")
	constraint, err := semver.NewConstraint(versionFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version filter %q: %v", versionFilter, err)
	}

	var filteredVersions []*kubermaticversion.Version
	for _, ver := range versions {
		if constraint.Check(ver.Version) {
			filteredVersions = append(filteredVersions, ver)
		}
	}

	return filteredVersions, nil
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
		return nil, fmt.Errorf("unable to decode object: %v", err)
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
	// Deployment
	case *appsv1.Deployment:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1beta1.Deployment:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1beta2.Deployment:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *extensionv1beta1.Deployment:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)

	// ReplicaSet
	case *appsv1.ReplicaSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1beta2.ReplicaSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *extensionv1beta1.ReplicaSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)

	// Statefulset
	case *appsv1.StatefulSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1beta1.StatefulSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1beta2.StatefulSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)

	// DaemonSet
	case *appsv1.DaemonSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *appsv1beta2.DaemonSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	case *extensionv1beta1.DaemonSet:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)

	// Pod
	case *corev1.Pod:
		return getImagesFromPodSpec(obj.Spec)

	// CronJob
	case *batchv1beta1.CronJob:
		return getImagesFromPodSpec(obj.Spec.JobTemplate.Spec.Template.Spec)
	case *batchv2alpha1.CronJob:
		return getImagesFromPodSpec(obj.Spec.JobTemplate.Spec.Template.Spec)

	// Job
	case *batchv1.Job:
		return getImagesFromPodSpec(obj.Spec.Template.Spec)
	}

	return nil
}
