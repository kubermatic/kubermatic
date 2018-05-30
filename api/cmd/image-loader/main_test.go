package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/version"
)

func TestGetImageForAllVersions(t *testing.T) {
	gopath := os.Getenv("GOPATH")
	masterResources := fmt.Sprintf("%s/%s", gopath, "src/github.com/kubermatic/kubermatic/config/kubermatic/static/master/versions.yaml")

	versions, err := version.LoadVersions(masterResources)
	if err != nil {
		t.Errorf("Error loading versions: %v", err)
	}
	for _, masterVersion := range versions {
		_, err := getImagesForVersion(versions, masterVersion.Version.String())
		if err != nil {
			t.Errorf("Error calling getImagesForVersion: %v", err)
		}
	}
}
