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

	flag.StringVar(&masterResources, "master-resources", "../config/kubermatic/static/master/versions.yaml", "")
	flag.StringVar(&requestedVersion, "version", "", "")
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

	for _, image := range images {
		fmt.Println(image)
	}

	err = downloadImages(images)
	if err != nil {
		glog.Fatalf(err.Error())
	}
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

func downloadImages(images []Image) error {
	for _, image := range images {
		for _, tag := range image.Tags {
			if tag == "" {
				continue
			}
			imageWithTag := fmt.Sprintf("%s:%s", image.Name, tag)
			fmt.Printf("Downloading image '%s'...\n", imageWithTag)
			cmd := exec.Command("docker", "pull", imageWithTag)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("Error pulling image: %v\nOuput: %s\n", err, output)
			}
		}
	}
	return nil
}
