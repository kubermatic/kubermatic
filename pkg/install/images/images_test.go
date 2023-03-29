/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package images

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/install/helm"
	"k8c.io/kubermatic/v3/pkg/resources/certificates"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/sets"
)

// These tests do not just verify that the Docker images can be extracted
// properly, but also that all addons can be processed for each cluster
// combination (i.e. there are no broken Go templates).
// When working on this test, remember that it also tests pkg/addon/'s
// ParseFromFolder().

func TestRetagImageForAllVersions(t *testing.T) {
	log := logrus.New()

	config, err := defaulting.DefaultConfiguration(&kubermaticv1.KubermaticConfiguration{}, zap.NewNop().Sugar())
	if err != nil {
		t.Errorf("failed to determine versions: %v", err)
	}

	kubermaticVersions := kubermatic.NewFakeVersions()
	clusterVersions := getVersionsFromKubermaticConfiguration(config)
	addonPath := "../../../addons"

	caBundle, err := certificates.NewCABundleFromFile("../../../charts/kubermatic-operator/static/ca-bundle.pem")
	if err != nil {
		t.Errorf("failed to load CA bundle: %v", err)
	}

	imageSet := sets.New[string]()
	for _, clusterVersion := range clusterVersions {
		for _, cloudSpec := range GetCloudSpecs() {
			for _, cniPlugin := range GetCNIPlugins() {
				images, err := GetImagesForVersion(log, clusterVersion, cloudSpec, cniPlugin, false, config, addonPath, kubermaticVersions, caBundle, "")
				if err != nil {
					t.Errorf("Error calling getImagesForVersion: %v", err)
				}
				imageSet.Insert(images...)
			}
		}
	}

	if _, _, err := ProcessImages(context.Background(), log, true, sets.List(imageSet), "test-registry:5000", "kubermatic-installer/test"); err != nil {
		t.Errorf("Error calling processImages: %v", err)
	}
}

func TestProcessImagesFromHelmChartsAndSystemApps(t *testing.T) {
	log := logrus.New()

	helmBinary, err := exec.LookPath("helm")
	if err != nil {
		t.Skip("Skipping test due to missing helm binary")
	}

	helmClient, err := helm.NewCLI(helmBinary, "", "", 5*time.Minute, log)
	if err != nil {
		t.Errorf("failed to create Helm client: %v", err)
	}

	config, err := defaulting.DefaultConfiguration(&kubermaticv1.KubermaticConfiguration{}, zap.NewNop().Sugar())
	if err != nil {
		t.Errorf("failed to determine versions: %v", err)
	}

	var images []string
	chartImages, err := GetImagesForHelmCharts(context.Background(), log, config, helmClient, "../../../charts/monitoring", "", "")
	if err != nil {
		t.Errorf("error calling GetImagesForHelmCharts: %v", err)
	}
	images = append(images, chartImages...)

	appImages, err := GetImagesFromSystemApplicationDefinitions(log, config, helmClient, 5*time.Minute, "")
	if err != nil {
		t.Errorf("Error calling GetImagesFromSystemApplicationDefinitions: %v", err)
	}
	images = append(images, appImages...)

	if _, _, err := ProcessImages(context.Background(), log, true, images, "test-registry:5000", "kubermatic-installer/test"); err != nil {
		t.Errorf("Error calling processImages: %v", err)
	}
}
