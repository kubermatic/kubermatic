package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/Masterminds/semver"
	"go.uber.org/zap"

	addonutil "github.com/kubermatic/kubermatic/api/pkg/addon"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubernetescontroller "github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/monitoring"
	containerlinux "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/container-linux"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/docker"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	metricsserver "github.com/kubermatic/kubermatic/api/pkg/resources/metrics-server"
	ksemver "github.com/kubermatic/kubermatic/api/pkg/semver"
	kubermaticversion "github.com/kubermatic/kubermatic/api/pkg/version"

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
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

const mockNamespaceName = "mock-namespace"

var (
	staticImages = []string{}
)

type opts struct {
	versionsFile  string
	versionFilter string
	registry      string
	dryRun        bool
	addonsPath    string
	addonsImage   string
}

func main() {
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.Format = kubermaticlog.FormatConsole
	logOpts.AddFlags(flag.CommandLine)

	o := opts{}
	flag.StringVar(&o.versionsFile, "versions", "../config/kubermatic/static/master/versions.yaml", "The versions.yaml file path")
	flag.StringVar(&o.versionFilter, "version-filter", "", "Version constraint which can be used to filter for specific versions")
	flag.StringVar(&o.registry, "registry", "", "Address of the registry to push to, for example localhost:5000")
	flag.BoolVar(&o.dryRun, "dry-run", false, "Only print the names of found images")
	flag.StringVar(&o.addonsPath, "addons-path", "", "Path to a directory containing the KKP addons, if not given, falls back to -addons-image, then the Docker image configured in the KubermaticConfiguration")
	flag.StringVar(&o.addonsImage, "addons-image", "", "Docker image containing KKP addons, if not given, falls back to the Docker image configured in the KubermaticConfiguration")
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	if o.versionsFile == "" {
		log.Fatal("-versions-file must be specified.")
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

	versions, err := getVersions(log, o.versionsFile, o.versionFilter)
	if err != nil {
		log.Fatalw("Error loading versions", zap.Error(err))
	}

	// if no local addons path is given, use the configured addons
	// Docker image and extract the addons from there
	if o.addonsPath == "" {
		addonsImage := o.addonsImage
		if addonsImage == "" {
			log.Warn("No KubermaticConfiguration, -addons-image or -addons-path given, cannot mirror images referenced in addons.")
		}

			tempDir, err := extractAddonsFromDockerImage(ctx, log, addonsImage)
			if err != nil {
				log.Fatalw("Failed to create local addons path", zap.Error(err))
			}
			defer os.RemoveAll(tempDir)

			o.addonsPath = tempDir
	}

	// Using a set here for deduplication
	imageSet := sets.NewString(staticImages...)
	for _, version := range versions {
		versionLog := log.With(
			"version", version.Version.String(),
			"cluster-type", version.Type,
		)
		if version.Type != "" && version.Type != apiv1.KubernetesClusterType {
			// TODO: Implement. https://github.com/kubermatic/kubermatic/issues/3623
			versionLog.Warn("Skipping version because its not for Kubernetes. We only support Kubernetes at the moment")
			continue
		}
		versionLog.Info("Collecting images...")
		images, err := getImagesForVersion(log, version, o.addonsPath)
		if err != nil {
			versionLog.Fatalw("failed to get images", zap.Error(err))
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

func getImagesForVersion(log *zap.SugaredLogger, version *kubermaticversion.Version, addonsPath string) (images []string, err error) {
	templateData, err := getTemplateData(version)
	if err != nil {
		return nil, err
	}

	creatorImages, err := getImagesFromCreators(templateData)
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

func getImagesFromCreators(templateData *resources.TemplateData) (images []string, err error) {
	statefulsetCreators := kubernetescontroller.GetStatefulSetCreators(templateData, false)
	statefulsetCreators = append(statefulsetCreators, monitoring.GetStatefulSetCreators(templateData)...)

	deploymentCreators := kubernetescontroller.GetDeploymentCreators(templateData, false)
	deploymentCreators = append(deploymentCreators, monitoring.GetDeploymentCreators(templateData)...)
	deploymentCreators = append(deploymentCreators, containerlinux.GetDeploymentCreators("")...)

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
	apiServerExternalService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.ApiserverExternalServiceName,
			Namespace: mockNamespaceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{NodePort: 99}},
			ClusterIP: "192.0.2.10",
		},
	}
	apiserverService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.ApiserverInternalServiceName,
			Namespace: mockNamespaceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{NodePort: 98}},
			ClusterIP: "192.0.2.11",
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
			apiServerExternalService,
			apiserverService,
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

func getVersions(log *zap.SugaredLogger, versionsFile, versionFilter string) ([]*kubermaticversion.Version, error) {
	log = log.With(
		"versions-file", versionsFile,
		"versions-filter", versionFilter,
	)
	log.Debug("Loading versions")
	versions, err := kubermaticversion.LoadVersions(versionsFile)
	if err != nil {
		return nil, err
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

func getImagesFromAddons(log *zap.SugaredLogger, addonsPath string, cluster *kubermaticv1.Cluster) ([]string, error) {
	addonData := &addonutil.TemplateData{
		Cluster:           cluster,
		MajorMinorVersion: cluster.Spec.Version.MajorMinor(),
		Addon:             &kubermaticv1.Addon{},
		Variables:         map[string]interface{}{},
	}
	infos, err := ioutil.ReadDir(addonsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to list addons: %v", err)
	}

	serializer := json.NewSerializer(&json.SimpleMetaFactory{}, scheme.Scheme, scheme.Scheme, false)
	var images []string
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		addonName := info.Name()
		addonImages, err := getImagesFromAddon(log, path.Join(addonsPath, addonName), serializer, addonData)
		if err != nil {
			return nil, fmt.Errorf("failed to get images for addon %s: %v", addonName, err)
		}
		images = append(images, addonImages...)
	}

	return images, nil
}

func getImagesFromAddon(log *zap.SugaredLogger, addonPath string, decoder runtime.Decoder, data *addonutil.TemplateData) ([]string, error) {
	log = log.With("addon", path.Base(addonPath))
	log.Debug("Processing manifests...")

	allManifests, err := addonutil.ParseFromFolder(log, "", addonPath, data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addon templates in %s: %v", addonPath, err)
	}

	var images []string
	for _, manifest := range allManifests {
		manifestImages, err := getImagesFromManifest(log, decoder, manifest.Raw)
		if err != nil {
			return nil, err
		}
		images = append(images, manifestImages...)
	}
	return images, nil
}

func getImagesFromManifest(log *zap.SugaredLogger, decoder runtime.Decoder, b []byte) ([]string, error) {
	obj, err := runtime.Decode(decoder, b)
	if err != nil {
		if runtime.IsNotRegisteredError(err) {
			// We must skip custom objects. We try to look up the object info though to give a useful warning
			metaFactory := &json.SimpleMetaFactory{}
			if gvk, err := metaFactory.Interpret(b); err == nil {
				log = log.With("gvk", gvk.String())
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
