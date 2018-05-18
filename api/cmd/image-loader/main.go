package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
)

type image struct {
	Name      string
	Tags      []string
	ValueName string
}

func main() {

	var masterResources string
	var requestedVersion string
	var registryName string

	flag.StringVar(&masterResources, "master-resources", "../config/kubermatic/static/master/versions.yaml", "")
	flag.StringVar(&requestedVersion, "version", "", "")
	flag.StringVar(&registryName, "registry-name", "", "Name of the registry to push to")
	flag.Parse()

	if registryName == "" {
		glog.Fatalf("Registry name must not be empty!")
	}

	versions, err := version.LoadVersions(masterResources)
	if err != nil {
		glog.Fatalf("Error loading versions: %v", err)
	}

	var imagesUnfiltered []string
	if requestedVersion == "" {
		glog.Infof("No version passed, downloading images for all available versions...")
		for version, _ := range versions {
			imagesUnfiltered, err = getImagesForVersion(versions, version)
			if err != nil {
				glog.Fatalf(err.Error())
			}
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

func setImageTags(versions map[string]*apiv1.MasterVersion, images *[]image, requestedVersion string) error {
	version, found := versions[requestedVersion]
	if !found {
		return fmt.Errorf("version %s could not be found", requestedVersion)
	}

	imagesValue := *images
	for idx, image := range imagesValue {
		if image.ValueName == "" {
			continue
		}

		imageVersion, found := version.Values[image.ValueName]
		if !found {
			return fmt.Errorf("found no version value named %s", image.ValueName)
		}
		imagesValue[idx].Tags = append(imagesValue[idx].Tags, imageVersion)

	}
	return nil
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

func getImageTagList(images []image) (imageWithTagList []string) {
	var intermediateImageWithTagList []string
	for _, image := range images {
		for _, tag := range image.Tags {
			if tag == "" {
				continue
			}
			intermediateImageWithTagList = append(intermediateImageWithTagList, fmt.Sprintf("%s:%s", image.Name, tag))
		}
	}

	for _, newItem := range intermediateImageWithTagList {
		if !stringListContains(imageWithTagList, newItem) {
			imageWithTagList = append(imageWithTagList, newItem)
		}
	}
	return imageWithTagList
}

func stringListContains(list []string, item string) bool {
	for _, listItem := range list {
		if listItem == item {
			return true
		}
	}
	return false
}

func getImagesForVersion(versions map[string]*apiv1.MasterVersion, requestedVersion string) (images []string, err error) {
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
	return &resources.TemplateData{Version: version, DC: &provider.DatacenterMeta{}}, nil
}
