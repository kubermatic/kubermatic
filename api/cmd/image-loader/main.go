package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"

	"github.com/golang/glog"
)

type Image struct {
	Name      string
	Tags      []string
	ValueName string
}

func main() {

	var images []Image
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

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		glog.Fatalf("Error reading from stdin: %v", err)
	}
	dataSanitized := strings.Replace(string(data), "\t", "", -1)
	dataSanitized = strings.Replace(dataSanitized, " ", "", -1)
	lines := strings.Split(dataSanitized, "\n")
	for _, line := range lines {
		if strings.Contains(line, "ImageRegistry(") {
			var imageSanitized string
			var valueName string

			// Remove everything up until the image name
			splitted := strings.SplitAfter(line, "ImageRegistry(")
			if len(splitted) == 2 {
				imageSanitized = strings.Replace(splitted[1], ")+\"", "", -1)
			}

			// Remove trailing "+data.Version.Values" if it exists
			if strings.Contains(imageSanitized, "+data.Version.Values") {
				splitted := strings.Split(imageSanitized, "+data.Version")
				imageSanitized = splitted[0]
				splittedByQuotationSign := strings.Split(splitted[1], "\"")
				if len(splittedByQuotationSign) != 3 {
					glog.Fatalf("Can not extract name of version from string %s!", splitted[1])
				}
				valueName = splittedByQuotationSign[1]

			}

			imageSanitized = strings.Replace(imageSanitized, "\"", "", -1)
			imageSanitized = strings.Replace(imageSanitized, ",", "", -1)

			imageAndTags := strings.Split(imageSanitized, ":")
			if len(imageAndTags) == 1 {
				images = append(images, Image{Name: imageAndTags[0], ValueName: valueName})
				continue
			}

			if len(imageAndTags) > 1 {
				images = append(images, Image{Name: imageAndTags[0], Tags: imageAndTags[1:], ValueName: valueName})
			}
		}
	}
	if requestedVersion == "" {
		for version, _ := range versions {
			setImageTags(versions, &images, version)
		}
	} else {
		setImageTags(versions, &images, requestedVersion)
	}

	imageTagList := getImageTagList(images)
	//err = downloadImages(imageTagList)
	retaggedImages, err := retagImages(registryName, imageTagList)
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
		out, err := exec.Command("docker", "tag", image, retaggedImageName).CombinedOutput()
		if err != nil {
			return retaggedImages, fmt.Errorf("Failed to retag image: Error: %v, Output: %s", err, string(out))
		}
		retaggedImages = append(retaggedImages, retaggedImageName)
	}

	return retaggedImages, nil
}

func setImageTags(versions map[string]*apiv1.MasterVersion, images *[]Image, requestedVersion string) (*[]Image, error) {
	version, found := versions[requestedVersion]
	if !found {
		return nil, fmt.Errorf("Version %s could not be found!", requestedVersion)
	}

	imagesValue := *images
	for idx, image := range imagesValue {
		if image.ValueName == "" {
			continue
		}

		imageVersion, found := version.Values[image.ValueName]
		if !found {
			return nil, fmt.Errorf("Found no version value named %s!", image.ValueName)
		}
		imagesValue[idx].Tags = append(imagesValue[idx].Tags, imageVersion)

	}
	return &imagesValue, nil
}

func downloadImages(images []string) error {
	for _, image := range images {
		glog.Infof("Downloading image '%s'...\n", image)
		cmd := exec.Command("docker", "pull", image)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Error pulling image: %v\nOuput: %s\n", err, output)
		}
	}
	return nil
}

func getImageTagList(images []Image) (imageWithTagList []string) {
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
