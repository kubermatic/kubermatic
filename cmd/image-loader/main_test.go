package main

import (
	"context"
	"testing"

	kubermaticlog "github.com/kubermatic/kubermatic/pkg/log"
	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/version"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestRetagImageForAllVersions(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
	masterResources := "../../config/kubermatic/static/master/versions.yaml"
	addonPath := "../../addons"

	versions, err := version.LoadVersions(masterResources)
	if err != nil {
		t.Errorf("Error loading versions: %v", err)
	}

	// Cannot be set during go-test
	resources.KUBERMATICCOMMIT = "latest"

	imageSet := sets.NewString()
	for _, v := range versions {
		images, err := getImagesForVersion(log.Desugar(), v, addonPath)
		if err != nil {
			t.Errorf("Error calling getImagesForVersion: %v", err)
		}
		imageSet.Insert(images...)
	}

	if err := processImages(context.Background(), log.Desugar(), true, imageSet.List(), "test-registry:5000"); err != nil {
		t.Errorf("Error calling processImages: %v", err)
	}
}
