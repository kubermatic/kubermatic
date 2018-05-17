package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/glog"
)

type Image struct {
	Name string
	Tags []string
}

func main() {

	var images []Image

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

			// Remove everything up until the image name
			splitted := strings.SplitAfter(line, "ImageRegistry(")
			if len(splitted) == 2 {
				imageSanitized = strings.Replace(splitted[1], ")+\"", "", -1)
			}

			// Remove trailing "+data.Version.Values" if it exists
			if strings.Contains(imageSanitized, "+data.Version.Values") {
				splitted := strings.Split(imageSanitized, "+data.Version")
				imageSanitized = splitted[0]
			}

			imageSanitized = strings.Replace(imageSanitized, "\"", "", -1)
			imageSanitized = strings.Replace(imageSanitized, ",", "", -1)

			imageAndTags := strings.Split(imageSanitized, ":")
			if len(imageAndTags) == 1 {
				images = append(images, Image{Name: imageAndTags[0]})
				continue
			}

			if len(imageAndTags) > 1 {
				images = append(images, Image{Name: imageAndTags[0], Tags: imageAndTags[1:]})
			}
		}
	}

	for _, image := range images {
		fmt.Println(image)
	}
}
