package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/glog"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	clusterv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

const mockNamespaceName = "mock-namespace"

var (
	masterResources  string
	requestedVersion string
	registryName     string
	printOnly        bool
)

func main() {

	flag.StringVar(&masterResources, "master-resources", "../config/kubermatic/static/master/versions.yaml", "")
	flag.StringVar(&requestedVersion, "version", "", "")
	flag.StringVar(&registryName, "registry-name", "", "Name of the registry to push to")
	flag.BoolVar(&printOnly, "print-only", false, "Only print the names of found images")
	flag.Parse()

	if registryName == "" && !printOnly {
		glog.Fatalf("Registry name must not be empty!")
	}

	versions, err := version.LoadVersions(masterResources)
	if err != nil {
		glog.Fatalf("Error loading versions: %v", err)
	}

	var imagesUnfiltered []string
	if requestedVersion == "" {
		glog.Infof("No version passed, downloading images for all available versions...")
		for version := range versions {
			returnedImages, err := getImagesForVersion(versions, version)
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
		if out, err := exec.Command("docker", "push", image).CombinedOutput(); err != nil {
			return fmt.Errorf("Failed to push image: Error: %v Output: %s", err, string(out))
		}
	}
	return nil
}

func retagImages(registryName string, imageTagList []string) (retaggedImages []string, err error) {
	for _, image := range imageTagList {
		imageSplitted := strings.Split(image, "/")
		retaggedImageName := fmt.Sprintf("%s/%s", registryName, strings.Join(imageSplitted[1:], "/"))
		glog.Infof("Tagging image %s as %s", image, retaggedImageName)
		if out, err := exec.Command("docker", "tag", image, retaggedImageName).CombinedOutput(); err != nil {
			return retaggedImages, fmt.Errorf("Failed to retag image: Error: %v, Output: %s", err, string(out))
		}
	}

	return retaggedImages, nil
}

func downloadImages(images []string) error {
	for _, image := range images {
		glog.Infof("Downloading image '%s'...\n", image)
		if out, err := exec.Command("docker", "pull", image).CombinedOutput(); err != nil {
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

func getImagesForVersion(versions map[string]*apiv1.MasterVersion, requestedVersion string) ([]string, error) {
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

func getTemplateData(versions map[string]*apiv1.MasterVersion, requestedVersion string) (*resources.TemplateData, error) {
	version, found := versions[requestedVersion]
	if !found {
		return nil, fmt.Errorf("failed to get version %s", requestedVersion)
	}

	// We need listers and a set of objects to not have our deployment/statefulset creators fail
	cloudConfigConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-config",
			Namespace: mockNamespaceName,
		},
	}
	prometheusConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus",
			Namespace: mockNamespaceName,
		},
	}
	configMapList := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{cloudConfigConfigMap, prometheusConfigMap},
	}
	apiServerExternalService := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apiserver-external",
			Namespace: mockNamespaceName,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{NodePort: 99}},
		},
	}
	serviceList := &corev1.ServiceList{
		Items: []corev1.Service{apiServerExternalService},
	}
	secretList := createNamedSecrets([]string{"ca-cert",
		"ca-key",
		"tokens",
		"apiserver-tls",
		"kubelet-client-certificates",
		"service-account-key"})
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
	fakeCluster.Status.NamespaceName = mockNamespaceName
	fakeCluster.Address = &clusterv1.ClusterAddress{}

	go configMapInformer.Informer().Run(wait.NeverStop)
	go secretInformer.Informer().Run(wait.NeverStop)
	go serviceInformer.Informer().Run(wait.NeverStop)
	cache.WaitForCacheSync(wait.NeverStop, configMapInformer.Informer().HasSynced)
	cache.WaitForCacheSync(wait.NeverStop, secretInformer.Informer().HasSynced)
	cache.WaitForCacheSync(wait.NeverStop, serviceInformer.Informer().HasSynced)

	return &resources.TemplateData{Version: version,
		DC:              &provider.DatacenterMeta{},
		SecretLister:    secretLister,
		ServiceLister:   serviceLister,
		ConfigMapLister: configMapLister,
		Cluster:         fakeCluster}, nil
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
