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
	"flag"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/Masterminds/semver"
	"go.uber.org/zap"

	addonutil "k8c.io/kubermatic/v2/pkg/addon"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubernetescontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/kubernetes"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/monitoring"
	containerlinux "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/container-linux"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
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
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
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
}

func main() {
	klog.InitFlags(nil)

	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)

	o := opts{}
	flag.StringVar(&o.versionsFile, "versions", "charts/kubermatic/static/master/versions.yaml", "The versions.yaml file path")
	flag.StringVar(&o.versionFilter, "version-filter", "", "Version constraint which can be used to filter for specific versions")
	flag.StringVar(&o.registry, "registry", "registry.corp.local", "Address of the registry to push to")
	flag.BoolVar(&o.dryRun, "dry-run", false, "Only print the names of found images")
	flag.StringVar(&o.addonsPath, "addons-path", "", "Path to the folder containing the addons")
	flag.Parse()

	log := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	defer func() {
		if err := log.Sync(); err != nil {
			fmt.Println(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signalCh := signals.SetupSignalHandler()
	go func() {
		<-signalCh
		cancel()
	}()

	if o.registry == "" {
		log.Fatal("Error: registry-name parameter must contain a valid registry address!")
	}

	versions, err := getVersions(log, o.versionsFile, o.versionFilter)
	if err != nil {
		log.Fatal("Error loading versions", zap.Error(err))
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
		images, err := getImagesForVersion(log, version, o.addonsPath)
		if err != nil {
			versionLog.Fatal("failed to get images", zap.Error(err))
		}
		imageSet.Insert(images...)
	}

	if err := processImages(ctx, log, o.dryRun, imageSet.List(), o.registry); err != nil {
		log.Fatal("Failed to process images", zap.Error(err))
	}
}

func processImages(ctx context.Context, log *zap.Logger, dryRun bool, images []string, registry string) error {
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

func getImagesForVersion(log *zap.Logger, version *kubermaticversion.Version, addonsPath string) (images []string, err error) {
	templateData, err := getTemplateData(version)
	if err != nil {
		return nil, err
	}

	creatorImages, err := getImagesFromCreators(templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to get images from internal creator functions: %v", err)
	}
	images = append(images, creatorImages...)

	if addonsPath != "" {
		addonImages, err := getImagesFromAddons(log, addonsPath, templateData.Cluster())
		if err != nil {
			return nil, fmt.Errorf("failed to get images from addons: %v", err)
		}
		images = append(images, addonImages...)
	}

	return images, nil
}

func getImagesFromCreators(templateData *resources.TemplateData) (images []string, err error) {
	statefulsetCreators := kubernetescontroller.GetStatefulSetCreators(templateData, false)
	statefulsetCreators = append(statefulsetCreators, monitoring.GetStatefulSetCreators(templateData)...)

	deploymentCreators := kubernetescontroller.GetDeploymentCreators(templateData, false)
	deploymentCreators = append(deploymentCreators, monitoring.GetDeploymentCreators(templateData)...)
	deploymentCreators = append(deploymentCreators, containerlinux.GetDeploymentCreators("", kubermaticv1.UpdateWindow{})...)

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

func getVersions(log *zap.Logger, versionsFile, versionFilter string) ([]*kubermaticversion.Version, error) {
	log = log.With(
		zap.String("versions-file", versionsFile),
		zap.String("versions-filter", versionFilter),
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

func getImagesFromAddons(log *zap.Logger, addonsPath string, cluster *kubermaticv1.Cluster) ([]string, error) {
	credentials := resources.Credentials{}

	addonData, err := addonutil.NewTemplateData(cluster, credentials, "", "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create addon template data: %v", err)
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

func getImagesFromAddon(log *zap.Logger, addonPath string, decoder runtime.Decoder, data *addonutil.TemplateData) ([]string, error) {
	log = log.With(zap.String("addon", path.Base(addonPath)))
	log.Debug("Processing manifests...")

	allManifests, err := addonutil.ParseFromFolder(log.Sugar(), "", addonPath, data)
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

func getImagesFromManifest(log *zap.Logger, decoder runtime.Decoder, b []byte) ([]string, error) {
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
