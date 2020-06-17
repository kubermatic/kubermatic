/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"testing"

	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestRetagImageForAllVersions(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()
	masterResources := "../../../config/kubermatic/static/master/versions.yaml"
	addonPath := "../../../addons"

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
