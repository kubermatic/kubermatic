package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/golang/glog"

	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	clusterv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

const mockNamespaceName = "mock-namespace"

var (
	test             bool
	versionsFile     string
	requestedVersion string
	registryName     string
	printOnly        bool
	staticImages     = []string{
		backupcontroller.DefaultBackupContainerImage,
	}
)

func main() {

	flag.StringVar(&versionsFile, "versions", "../config/kubermatic/static/master/versions.yaml", "The versions.yaml file path")
	flag.StringVar(&requestedVersion, "version", "", "")
	flag.StringVar(&registryName, "registry-name", "registry.corp.local", "Name of the registry to push to")
	flag.BoolVar(&printOnly, "print-only", false, "Only print the names of found images")
	flag.Parse()

	if registryName == "" && !printOnly {
		glog.Fatalf("Error: registry-name parameter must contain a valid registry address!")
	}

	versions, err := version.LoadVersions(versionsFile)
	if err != nil {
		glog.Fatalf("Error loading versions from %s: %v", versionsFile, err)
	}

	var imagesUnfiltered []string
	if requestedVersion == "" {
		glog.Infof("No version passed, downloading images for all available versions...")
		for _, v := range versions {
			glog.Infof("Collecting images for v%s", v.Version.String())
			returnedImages, err := getImagesForVersion(versions, v.Version.String())
			if err != nil {
				glog.Fatalf(err.Error())
			}
			imagesUnfiltered = append(imagesUnfiltered, returnedImages...)
		}
	} else {
		imagesUnfiltered, err = getImagesForVersion(versions, requestedVersion)
		if err != nil {
			glog.Fatalf(err.Error())
		}
	}
	imagesUnfiltered = append(imagesUnfiltered, staticImages...)

	var images []string
	for _, image := range imagesUnfiltered {
		if !stringListContains(images, image) && len(strings.Split(image, ":")) == 2 {
			images = append(images, image)
		}

	}

	if printOnly {
		for _, image := range images {
			glog.Infoln(image)
		}
		glog.Infoln("Existing gracefully, because -printOnly was specified...")
		os.Exit(0)
	}

	if err = downloadImages(images); err != nil {
		glog.Fatalf(err.Error())
	}
	retaggedImages, err := retagImages(registryName, images)
	if err != nil {
		glog.Fatalf(err.Error())
	}
	if err = pushImages(retaggedImages); err != nil {
		glog.Fatalf(err.Error())
	}
}

func pushImages(imageTagList []string) error {
	for _, image := range imageTagList {
		glog.Infof("Pushing image %s", image)
		if out, err := execCommand("docker", "push", image); err != nil {
			return fmt.Errorf("failed to push image: Error: %v Output: %s", err, out)
		}
	}
	return nil
}

func retagImages(registryName string, imageTagList []string) (retaggedImages []string, err error) {
	for _, image := range imageTagList {
		imageSplitted := strings.Split(image, "/")
		if len(imageSplitted) < 2 {
			return nil, fmt.Errorf("image %s does not contain a registry", image)
		}
		retaggedImageName := fmt.Sprintf("%s/%s", registryName, strings.Join(imageSplitted[1:], "/"))
		glog.Infof("Tagging image %s as %s", image, retaggedImageName)
		if out, err := execCommand("docker", "tag", image, retaggedImageName); err != nil {
			return retaggedImages, fmt.Errorf("failed to retag image: Error: %v, Output: %s", err, out)
		}
	}

	return retaggedImages, nil
}

func downloadImages(images []string) error {
	for _, image := range images {
		glog.Infof("Downloading image '%s'...\n", image)
		if out, err := execCommand("docker", "pull", image); err != nil {
			return fmt.Errorf("error pulling image: %v\nOutput: %s", err, out)
		}
	}
	return nil
}

func stringListContains(list []string, item string) bool {
	for _, listItem := range list {
		if listItem == item {
			return true
		}
	}
	return false
}

func getImagesForVersion(versions []*version.MasterVersion, requestedVersion string) ([]string, error) {
	templateData, err := getTemplateData(versions, requestedVersion)
	if err != nil {
		return nil, err
	}
	return getImagesFromCreators(templateData)
}

func getImagesFromCreators(templateData *resources.TemplateData) (images []string, err error) {
	statefulsetCreators := cluster.GetStatefulSetCreators()
	deploymentCreators := cluster.GetDeploymentCreators()

	for _, createFunc := range statefulsetCreators {
		statefulset, err := createFunc(templateData, nil)
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodTemplateSpec(statefulset.Spec.Template)...)
	}

	for _, createFunc := range deploymentCreators {
		deployment, err := createFunc(templateData, nil)
		if err != nil {
			return nil, err
		}
		images = append(images, getImagesFromPodTemplateSpec(deployment.Spec.Template)...)
	}
	return images, nil
}

func getImagesFromPodTemplateSpec(template corev1.PodTemplateSpec) (images []string) {
	for _, initContainer := range template.Spec.InitContainers {
		images = append(images, initContainer.Image)
	}

	for _, container := range template.Spec.Containers {
		images = append(images, container.Image)
	}

	return images
}

func getVersion(versions []*version.MasterVersion, requestedVersion string) (*version.MasterVersion, error) {
	semver, err := semver.NewVersion(requestedVersion)
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.Version.Equal(semver) {
			return v, nil
		}
	}
	return nil, fmt.Errorf("version not found")
}

func getTemplateData(versions []*version.MasterVersion, requestedVersion string) (*resources.TemplateData, error) {
	masterVersion, err := getVersion(versions, requestedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %s", requestedVersion)
	}

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
	configMapList := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{cloudConfigConfigMap, prometheusConfigMap, dnsResolverConfigMap, openvpnClientConfigsConfigMap},
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
	etcdclientService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.EtcdClientServiceName,
			Namespace: mockNamespaceName,
		},
		Spec: corev1.ServiceSpec{
			Ports:     []corev1.ServicePort{{NodePort: 97}},
			ClusterIP: "192.0.2.1",
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
			Ports:     []corev1.ServicePort{{NodePort: 95}},
			ClusterIP: "192.0.2.11",
		},
	}
	serviceList := &corev1.ServiceList{
		Items: []corev1.Service{
			apiServerExternalService,
			apiserverService,
			etcdclientService,
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
		resources.EtcdTLSCertificateSecretName,
		resources.MachineControllerKubeconfigSecretName,
		resources.ControllerManagerKubeconfigSecretName,
		resources.SchedulerKubeconfigSecretName,
		resources.OpenVPNServerCertificatesSecretName,
		resources.OpenVPNClientCertificatesSecretName,
	})
	objects := []runtime.Object{configMapList, secretList, serviceList}
	client := kubefake.NewSimpleClientset(objects...)

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(client, time.Second*30)
	configMapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
	configMapLister := configMapInformer.Lister()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	secretLister := secretInformer.Lister()
	serviceInformer := kubeInformerFactory.Core().V1().Services()
	serviceLister := serviceInformer.Lister()

	fakeCluster := &clusterv1.Cluster{}
	fakeCluster.Spec.Cloud = &clusterv1.CloudSpec{}
	fakeCluster.Spec.Version = masterVersion.Version.String()
	fakeCluster.Spec.ClusterNetwork.Pods.CIDRBlocks = []string{"172.25.0.0/16"}
	fakeCluster.Spec.ClusterNetwork.Services.CIDRBlocks = []string{"10.10.10.0/24"}
	fakeCluster.Spec.ClusterNetwork.DNSDomain = "cluster.local"
	fakeCluster.Status.NamespaceName = mockNamespaceName

	stopChannel := make(chan struct{})
	kubeInformerFactory.Start(stopChannel)
	kubeInformerFactory.WaitForCacheSync(stopChannel)

	return &resources.TemplateData{
		DC:                &provider.DatacenterMeta{},
		SecretLister:      secretLister,
		ServiceLister:     serviceLister,
		ConfigMapLister:   configMapLister,
		NodeAccessNetwork: "192.0.2.0/24",
		Cluster:           fakeCluster}, nil
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

func execCommand(command ...string) (string, error) {
	glog.V(2).Infof("Executing command '%s'", strings.Join(command, " "))
	var args []string
	if len(command) > 1 {
		args = command[1:]
	}
	if test {
		glog.Infof("Not executing command as testing is enabled")
		return "", nil
	}
	out, err := exec.Command(command[0], args...).CombinedOutput()
	return string(out), err
}
