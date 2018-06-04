package main

import (
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/version"
)

func TestRetagImageForAllVersions(t *testing.T) {
	masterResources := "../../../config/kubermatic/static/master/versions.yaml"

	versions, err := version.LoadVersions(masterResources)
	if err != nil {
		t.Errorf("Error loading versions: %v", err)
	}

	test = true

	var imageTagList []string
	for _, masterVersion := range versions {
		imageTagList, err = getImagesForVersion(versions, masterVersion.Version.String())
		if err != nil {
			t.Errorf("Error calling getImagesForVersion: %v", err)
		}
	}

	_, err = retagImages("test.registry", imageTagList)
	if err != nil {
		t.Errorf("Error calling retagImages: %v", err)
	}
}
