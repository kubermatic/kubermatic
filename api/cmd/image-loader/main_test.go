package main

import (
	"context"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestRetagImageForAllVersions(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
	masterResources := "../../../config/kubermatic/static/master/versions.yaml"
	addonPath := "../../../addons"

	// Cannot be set during go-test
	resources.KUBERMATICCOMMIT = "latest"
	common.KUBERMATICDOCKERTAG = resources.KUBERMATICCOMMIT
	common.UIDOCKERTAG = resources.KUBERMATICCOMMIT

	versions, err := version.LoadVersions(masterResources)
	if err != nil {
		t.Errorf("Error loading versions: %v", err)
	}

	imageSet := sets.NewString()
	for _, v := range versions {
		images, err := getImagesForVersion(log, v, addonPath)
		if err != nil {
			t.Errorf("Error calling getImagesForVersion: %v", err)
		}
		imageSet.Insert(images...)
	}

	if err := processImages(context.Background(), log, true, imageSet.List(), "test-registry:5000"); err != nil {
		t.Errorf("Error calling processImages: %v", err)
	}
}
