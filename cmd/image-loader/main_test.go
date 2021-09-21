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

	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestRetagImageForAllVersions(t *testing.T) {
	log := kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar()

	config, err := defaults.DefaultConfiguration(&operatorv1alpha1.KubermaticConfiguration{}, log)
	if err != nil {
		t.Errorf("failed to determine versions: %v", err)
	}

	kubermaticVersions := kubermatic.NewFakeVersions()
	clusterVersions := getVersionsFromKubermaticConfiguration(config)
	addonPath := "../../addons"

	caBundle, err := certificates.NewCABundleFromFile("../../charts/kubermatic-operator/static/ca-bundle.pem")
	if err != nil {
		t.Errorf("failed to load CA bundle: %v", err)
	}

	imageSet := sets.NewString()
	for _, clusterVersion := range clusterVersions {
		images, err := getImagesForVersion(log, clusterVersion, config, addonPath, kubermaticVersions, caBundle)
		if err != nil {
			t.Errorf("Error calling getImagesForVersion: %v", err)
		}
		imageSet.Insert(images...)
	}

	if err := processImages(context.Background(), log, true, imageSet.List(), "test-registry:5000"); err != nil {
		t.Errorf("Error calling processImages: %v", err)
	}
}
